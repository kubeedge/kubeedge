package streamruleendpoint

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/gorilla/websocket"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	streamruleendpointConfig "github.com/kubeedge/kubeedge/edge/pkg/streamruleendpoint/config"
	"github.com/kubeedge/kubeedge/edge/pkg/streamruleendpoint/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/streamruleendpoint/util"
	"k8s.io/klog/v2"
)

const (
	sourceType = "streamrule_endpoint"
)

type streamruleendpoint struct {
	enable   bool
	sessions map[string]*util.TunnelSession
	mu       sync.RWMutex
}

func newStreamRuleEndpoint(enable bool) *streamruleendpoint {
	return &streamruleendpoint{
		enable:   enable,
		sessions: make(map[string]*util.TunnelSession),
	}
}

// Register register streamruleendpoint
func Register(s *v1alpha2.StreamRuleEndpoint, nodeName string) {
	streamruleendpointConfig.InitConfigure(s, nodeName)
	core.Register(newStreamRuleEndpoint(s.Enable))
	orm.RegisterModel(new(dao.EndpointUrls))
}

func (s *streamruleendpoint) Name() string {
	return modules.StreamRuleEndpointModuleName
}

func (s *streamruleendpoint) Group() string {
	return modules.BusGroup
}

func (s *streamruleendpoint) Enable() bool {
	return s.enable
}

func (s *streamruleendpoint) Start() {
	if !dao.IsTableEmpty() {
		epurls, err := dao.GetAllEpUrls()
		if err != nil {
			klog.Errorf("failed to get endpoints from db: %v", err)
		} else {
			for _, e := range epurls {
				go func(ep, url string) {
					done := make(chan error, 1)
					go s.startVideoTunnel(ep, url, done)

					err := <-done
					if err != nil {
						klog.Errorf("failed to start tunnel for ep=%s, url=%s: %v", ep, url, err)
						dao.DeleteEpUrlsByKey(ep)
					} else {
						klog.Infof("tunnel started for ep=%s, url=%s", ep, url)
					}
				}(e.Endpoint, e.URL)
			}
		}
	}

	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("streamruleendpoint stop")
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.StreamRuleEndpointModuleName)
		if err != nil {
			klog.Warningf("servicebus receive msg error %v", err)
			continue
		}

		go s.processMessage(&msg)
	}
}

func (s *streamruleendpoint) processMessage(msg *beehiveModel.Message) {
	source := msg.GetSource()
	if source != sourceType {
		return
	}

	resource := msg.GetResource()
	parts := strings.SplitN(resource, "/", 2)
	if len(parts) < 2 {
		klog.Errorf("[streamruleEp] invalid resource format: %s", resource)
		return
	}
	endpointName := parts[0]
	url := parts[1]

	switch msg.GetOperation() {
	case "start":
		if epurl, _ := dao.GetEpUrlsByKey(endpointName); epurl != nil {
			code := http.StatusInternalServerError
			m := fmt.Sprintf("endpoint %s already exists", endpointName)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		resultCh := make(chan error, 1)
		go s.startVideoTunnel(endpointName, url, resultCh)

		select {
		case err := <-resultCh:
			if err != nil {
				if msg, e := buildErrorResponse(msg.GetID(), err.Error(), http.StatusInternalServerError); e == nil {
					beehiveContext.SendToGroup(modules.HubGroup, msg)
				}
			} else {
				if err := dao.InsertEpUrls(endpointName, url); err != nil {
					klog.Error(err)
				}

				if response, err := buildSuccessResponse(msg.GetID(), `{"code":200,"message":"OK"}`); err == nil {
					beehiveContext.SendToGroup(modules.HubGroup, response)
				}
			}

		case <-time.After(30 * time.Second):
			klog.Errorf("[streamruleEp] startVideoTunnel timeout after 30s")
			code := http.StatusGatewayTimeout
			m := "error to start video tunnel, err: Timeout waiting for tunnel"
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
		}

	case "stop":
		if err := dao.DeleteEpUrlsByKey(endpointName); err != nil {
			klog.Errorf("failed to delete ep: %v", err)
			code := http.StatusInternalServerError
			m := "error to stop video tunnel, err: " + err.Error()
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
		} else {
			s.stopVideoTunnel(endpointName)
			if response, err := buildSuccessResponse(msg.GetID(), `{"code":200,"message":"OK"}`); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
		}

	default:
		klog.Warningf("Action not found")
		code := http.StatusNotFound
		m := fmt.Sprintf("action %s not found for resource %s", msg.GetOperation(), resource)
		if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
			beehiveContext.SendToGroup(modules.HubGroup, response)
		}
	}
}

func (s *streamruleendpoint) startVideoTunnel(ep string, resourceUrl string, done chan error) {
	params := url.Values{}
	params.Add("ep", ep)
	params.Add("url", resourceUrl)

	serverURL := url.URL{
		Scheme:   "wss",
		Host:     streamruleendpointConfig.Config.TunnelServer,
		Path:     "/v1/kubeedge/videoconnect",
		RawQuery: params.Encode(),
	}

	cert, err := tls.LoadX509KeyPair(streamruleendpointConfig.Config.TLSTunnelCertFile, streamruleendpointConfig.Config.TLSTunnelPrivateKeyFile)
	if err != nil {
		klog.Exitf("Failed to load x509 key pair: %v", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		select {
		case <-beehiveContext.Done():
			return
		case <-ticker.C:
			err := s.TLSClientConnect(ep, serverURL, tlsConfig)
			if err != nil {
				klog.Errorf("TLSClientConnect error %v", err)
			} else {
				klog.Infof("TLSClientConnect success")
				done <- nil
				return
			}
		}
	}
	done <- err
}

func (s *streamruleendpoint) stopVideoTunnel(ep string) {

	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.sessions[ep]; ok {
		sess.Close()
		delete(s.sessions, ep)
	}
}

func (s *streamruleendpoint) TLSClientConnect(ep string, url url.URL, tlsConfig *tls.Config) error {
	dial := websocket.Dialer{
		TLSClientConfig:  tlsConfig,
		HandshakeTimeout: time.Duration(30) * time.Second,
	}
	header := http.Header{}

	con, _, err := dial.Dial(url.String(), header)
	if err != nil {
		klog.Errorf("dial %v error %v", url.String(), err)
		return err
	}

	session := util.NewTunnelSession(con)
	s.mu.Lock()
	s.sessions[ep] = session
	s.mu.Unlock()

	go session.Serve()
	return nil
}

func buildSuccessResponse(parentID string, content string) (beehiveModel.Message, error) {
	h := http.Header{}
	h.Add("Content-Type", "application/json")
	c := commonType.HTTPResponse{
		Header:     h,
		StatusCode: http.StatusOK,
		Body:       []byte(content),
	}
	message := beehiveModel.NewMessage(parentID).
		SetRoute(modules.StreamRuleEndpointModuleName, modules.UserGroup).
		SetResourceOperation("", beehiveModel.UploadOperation).
		FillBody(c)
	return *message, nil
}

func buildErrorResponse(parentID string, content string, statusCode int) (beehiveModel.Message, error) {
	h := http.Header{}
	h.Add("Server", "kubeedge-edgecore")
	c := commonType.HTTPResponse{Header: h, StatusCode: statusCode, Body: []byte(content)}
	message := beehiveModel.NewMessage(parentID).
		SetRoute(modules.StreamRuleEndpointModuleName, modules.UserGroup).
		SetResourceOperation("", beehiveModel.UploadOperation).
		FillBody(c)
	return *message, nil
}
