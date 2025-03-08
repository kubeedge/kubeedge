package servicebus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/beego/beego/v2/client/orm"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	servicebusConfig "github.com/kubeedge/kubeedge/edge/pkg/servicebus/config"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus/dao"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus/util"
)

var (
	inited int32
	c      = make(chan struct{})
)

const (
	sourceType  = "router_servicebus"
	maxBodySize = 5 * 1e6
)

// servicebus struct
type servicebus struct {
	enable bool
	// default 127.0.0.1
	server  string
	port    int
	timeout int
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
	}
}

// Register register servicebus
func Register(s *v1alpha2.ServiceBus) {
	servicebusConfig.InitConfigure(s)
	core.Register(newServicebus(s.Enable, s.Server, s.Port, s.Timeout))
	orm.RegisterModel(new(dao.TargetUrls))
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

func (sb *servicebus) Start() {
	// no need to call TopicInit now, we have fixed topic
	htc.Timeout = time.Second * 10
	uc.Client = htc
	if !dao.IsTableEmpty() {
		if atomic.CompareAndSwapInt32(&inited, 0, 1) {
			go server(c)
		}
	}
	//Get message from channel
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("servicebus stop")
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.ServiceBusModuleName)
		if err != nil {
			klog.Warningf("servicebus receive msg error %v", err)
			continue
		}

		// build new message with required field & send message to servicebus
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
	switch msg.GetOperation() {
	case message.OperationStart:
		if err := dao.InsertUrls(resource); err != nil {
			// TODO: handle err
			klog.Error(err)
		}
		if atomic.CompareAndSwapInt32(&inited, 0, 1) {
			go server(c)
		}
	case message.OperationStop:
		if err := dao.DeleteUrlsByKey(resource); err != nil {
			// TODO: handle err
			klog.Error(err)
		}
		if dao.IsTableEmpty() {
			c <- struct{}{}
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

		//send message with resource to the edge part
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

func server(stopChan <-chan struct{}) {
	var (
		timeout time.Duration
		err     error
	)
	if timeout, err = time.ParseDuration(fmt.Sprintf("%vs", servicebusConfig.Config.Timeout)); err != nil {
		klog.Errorf("can't format timeout and the default value will be set")
		timeout, _ = time.ParseDuration("10s")
	}

	h := buildBasicHandler(timeout)
	// TODO we should add tls for servicebus http server later
	s := http.Server{
		Addr:    fmt.Sprintf("%s:%d", servicebusConfig.Config.Server, servicebusConfig.Config.Port),
		Handler: h,
	}
	go func() {
		<-stopChan
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Shutdown(ctx); err != nil {
			klog.Errorf("Server shutdown failed: %s", err)
		}
		atomic.StoreInt32(&inited, 0)
	}()

	klog.Infof("[servicebus]start to listen and server at %v", s.Addr)
	utilruntime.HandleError(s.ListenAndServe())
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
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		if err = json.Unmarshal(byteData, sReq); err != nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = "invalid params"
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		if targetURL, _ := dao.GetUrlsByKey(sReq.TargetURL); targetURL == nil {
			sResp.Code = http.StatusBadRequest
			sResp.Msg = fmt.Sprintf("url %s is not allowed and please make a rule for this url in the cloud", sReq.TargetURL)
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
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
				// TODO: handle err
				klog.Error(err)
			}
			return
		}
		resp, err := responseMessage.GetContentData()
		if err != nil {
			sResp.Code = http.StatusInternalServerError
			sResp.Msg = err.Error()
			if _, err := w.Write(marshalResult(sResp)); err != nil {
				// TODO: handle err
				klog.Error(err)
			}
			return
		}

		sResp.Code = http.StatusOK
		sResp.Msg = "receive response from cloud successfully"
		sResp.Body = string(resp)
		if _, err := w.Write(marshalResult(sResp)); err != nil {
			// TODO: handle err
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
