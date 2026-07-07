package servicebus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	servicebusConfig "github.com/kubeedge/kubeedge/edge/pkg/servicebus/config"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus/util"
	"github.com/kubeedge/kubeedge/pkg/features"
	utilvalidation "github.com/kubeedge/kubeedge/pkg/util/validation"
)

var (
	inited int32

	serverMu sync.Mutex
	active   *http.Server
)

const (
	sourceType  = "router_servicebus"
	maxBodySize = 5 * 1e6
)

type servicebus struct {
	enable  bool
	server  string
	port    int
	timeout int
	sbs     *dbclient.ServiceBusService
}

type serverRequest struct {
	Method    string      `json:"method"`
	TargetURL string      `json:"targetURL"`
	Payload   interface{} `json:"payload"`
}

type serverResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Body string `json:"body"`
}

var htc = new(http.Client)
var uc = new(util.URLClient)

func newServicebus(enable bool, server string, port, timeout int) *servicebus {
	return &servicebus{
		enable:  enable,
		server:  server,
		port:    port,
		timeout: timeout,
		sbs:     dbclient.NewServiceBusService(),
	}
}

func Register(s *v1alpha2.ServiceBus) {
	servicebusConfig.InitConfigure(s)
	core.Register(newServicebus(s.Enable, s.Server, s.Port, s.Timeout))
}

func (*servicebus) Name() string {
	return modules.ServiceBusModuleName
}

func (*servicebus) Group() string {
	return modules.BusGroup
}

func (sb *servicebus) Enable() bool {
	return sb.enable
}

func (sb *servicebus) RestartPolicy() *core.ModuleRestartPolicy {
	if !features.DefaultFeatureGate.Enabled(features.ModuleRestart) {
		return nil
	}
	return &core.ModuleRestartPolicy{
		RestartType:            core.RestartTypeOnFailure,
		IntervalTimeGrowthRate: 2.0,
	}
}

func (sb *servicebus) Start() {
	htc.Timeout = time.Second * 10
	uc.Client = htc
	if !sb.sbs.IsTableEmpty() {
		startServer()
	}
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("servicebus stop")
			stopServer()
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.ServiceBusModuleName)
		if err != nil {
			klog.Warningf("servicebus receive msg error %v", err)
			continue
		}

		klog.V(4).Info("servicebus receive msg")
		go processMessage(&msg)
	}
}

func processMessage(msg *beehiveModel.Message) {
	source := msg.GetSource()
	if source != sourceType {
		return
	}
	resource := msg.GetResource()
	dbc := dbclient.NewServiceBusService()
	switch msg.GetOperation() {
	case message.OperationStart:
		if err := dbc.InsertUrls(resource); err != nil {
			klog.Error(err)
		}
		startServer()
	case message.OperationStop:
		if err := dbc.DeleteUrlsByKey(resource); err != nil {
			klog.Error(err)
		}

		if dbc.IsTableEmpty() {
			stopServer()
		}
	default:
		r := strings.Split(resource, ":")
		if len(r) != 2 {
			m := "the format of resource " + resource + " is incorrect"
			klog.Warning(m)
			code := http.StatusBadRequest
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}
		content, err := msg.GetContentData()
		if err != nil {
			klog.Errorf("marshall message content failed %v", err)
			m := "error to marshal request msg content"
			code := http.StatusBadRequest
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		var httpRequest commonType.HTTPRequest
		if err := json.Unmarshal(content, &httpRequest); err != nil {
			m := "error to parse http request"
			code := http.StatusBadRequest
			klog.Errorf(m, err)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		operation := httpRequest.Method
		targetURL := "http://127.0.0.1:" + r[0] + r[1]
		resp, err := uc.HTTPDo(operation, targetURL, httpRequest.Header, httpRequest.Body)
		if err != nil {
			m := "error to call service"
			code := http.StatusNotFound
			klog.Errorf(m, err)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}
		defer resp.Body.Close()
		resBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
		if err != nil {
			if err.Error() == "http: request body too large" {
				err = fmt.Errorf("response body too large")
			}
			m := "error to receive response, err: " + err.Error()
			code := http.StatusInternalServerError
			klog.Errorf(m, err)
			if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
				beehiveContext.SendToGroup(modules.HubGroup, response)
			}
			return
		}

		response := commonType.HTTPResponse{Header: resp.Header, StatusCode: resp.StatusCode, Body: resBody}
		responseMsg := beehiveModel.NewMessage(msg.GetID()).SetRoute(modules.ServiceBusModuleName, modules.UserGroup).
			SetResourceOperation("", beehiveModel.UploadOperation).FillBody(response)
		beehiveContext.SendToGroup(modules.HubGroup, *responseMsg)
	}
}

func startServer() {
	serverMu.Lock()
	if active != nil {
		serverMu.Unlock()
		return
	}

	srv, listener, err := newTLSServer(servicebusConfig.Config.ServiceBus)
	if err != nil {
		serverMu.Unlock()
		atomic.StoreInt32(&inited, 0)
		utilruntime.HandleError(err)
		return
	}
	active = srv
	atomic.StoreInt32(&inited, 1)
	serverMu.Unlock()

	go serveTLS(srv, listener)
}

func stopServer() {
	serverMu.Lock()
	srv := active
	active = nil
	atomic.StoreInt32(&inited, 0)
	serverMu.Unlock()

	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		klog.Errorf("Server shutdown failed: %s", err)
	}
}

func serveTLS(srv *http.Server, listener net.Listener) {
	defer func() {
		_ = listener.Close()
		serverMu.Lock()
		if active == srv {
			active = nil
		}
		atomic.StoreInt32(&inited, 0)
		serverMu.Unlock()
	}()

	klog.Infof("[servicebus] start to listen and serve at %v", srv.Addr)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		utilruntime.HandleError(err)
	}
}

func newTLSServer(cfg v1alpha2.ServiceBus) (*http.Server, net.Listener, error) {
	timeout, err := time.ParseDuration(fmt.Sprintf("%vs", cfg.Timeout))
	if err != nil {
		klog.Errorf("can't format timeout and the default value will be set")
		timeout = 10 * time.Second
	}

	address := fmt.Sprintf("%s:%d", cfg.Server, cfg.Port)
	if !utilvalidation.IsLoopbackHost(cfg.Server) {
		return nil, nil, fmt.Errorf("servicebus without client authentication must bind to a loopback address")
	}
	tlsConfig, err := loadTLSConfig(cfg.TLSCertFile, cfg.TLSPrivateKeyFile)
	if err != nil {
		return nil, nil, err
	}
	srv := &http.Server{
		Addr:              address,
		Handler:           buildBasicHandler(timeout),
		ReadHeaderTimeout: 10 * time.Second,
		TLSConfig:         tlsConfig,
	}
	rawListener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	return srv, tls.NewListener(rawListener, tlsConfig), nil
}

func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	if _, err := loadCertificate(certFile, keyFile); err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := loadCertificate(certFile, keyFile)
			if err != nil {
				return nil, err
			}
			return cert, nil
		},
	}, nil
}

func loadCertificate(certFile, keyFile string) (*tls.Certificate, error) {
	host := servicebusConfig.Config.Server
	if host == "" {
		host = "127.0.0.1"
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	if errs := utilvalidation.ValidateServerTLSFiles(certFile, keyFile, host, time.Now()); len(errs) > 0 {
		return nil, fmt.Errorf("invalid servicebus tls files: %s", strings.Join(errs, "; "))
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	if cert.Leaf == nil && len(cert.Certificate) > 0 {
		cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, err
		}
	}
	return &cert, nil
}

func buildBasicHandler(timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		sReq := &serverRequest{}
		sResp := &serverResponse{}
		req.Body = http.MaxBytesReader(w, req.Body, maxBodySize)
		byteData, err := io.ReadAll(req.Body)
		if err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = "can't read data from body of the http's request"
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				klog.Error(err)
			}
			return
		}
		if err = json.Unmarshal(byteData, sReq); err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = "invalid params"
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				klog.Error(err)
			}
			return
		}
		if targetURL, _ := dbclient.NewServiceBusService().GetUrlsByKey(sReq.TargetURL); targetURL == nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = fmt.Sprintf("url %s is not allowed and please make a rule for this url in the cloud", sReq.TargetURL)
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				klog.Error(err)
			}
			return
		}
		msg := beehiveModel.NewMessage("").BuildRouter(modules.ServiceBusModuleName, modules.UserGroup,
			sReq.TargetURL, beehiveModel.UploadOperation).FillBody(byteData)
		responseMessage, err := beehiveContext.SendSync(modules.EdgeHubModuleName, *msg, timeout)
		if err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = err.Error()
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				klog.Error(err)
			}
			return
		}
		resp, err := responseMessage.GetContentData()
		if err != nil {
			sResp.Code = http.StatusInternalServerError
			sResp.Msg = err.Error()
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				klog.Error(err)
			}
			return
		}

		sResp.Code = http.StatusOK
		sResp.Msg = "receive response from cloud successfully"
		sResp.Body = string(resp)
		if _, err := w.Write(marshalResult(sResp)); err != nil {
			klog.Error(err)
		}
	})
}

func buildErrorResponse(parentID string, content string, statusCode int) (beehiveModel.Message, error) {
	h := http.Header{}
	h.Add("Server", "kubeedge-edgecore")
	c := commonType.HTTPResponse{Header: h, StatusCode: statusCode, Body: []byte(content)}
	message := beehiveModel.NewMessage(parentID).
		SetRoute(modules.ServiceBusModuleName, modules.UserGroup).FillBody(c)
	return *message, nil
}

func marshalResult(sResp *serverResponse) (resp []byte) {
	resp, _ = json.Marshal(sResp)
	return
}
