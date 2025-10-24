package streamruleendpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/streamrules/v1alpha1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/streamrulecontroller/provider"
	commonType "github.com/kubeedge/kubeedge/common/types"
)

var inited int32

type streamruleEndpointFactory struct{}

type StreamruleEndpoint struct {
	nodeName     string
	resourceUrl  string
	EndpointName string
	Path         string
	Namespace    string
}

func init() {
	factory := &streamruleEndpointFactory{}
	provider.RegisterTarget(factory)
}

func (sf *streamruleEndpointFactory) Type() v1alpha1.ProtocolType {
	return v1alpha1.ProtocolTypeWebSocket
}

func (sf *streamruleEndpointFactory) GetTarget(ep *v1alpha1.StreamRuleEndpoint, targetResource map[string]string) provider.Target {
	nodeName, exist := targetResource["node_name"]
	if !exist {
		klog.Errorf("source resource attributes \"node_name\" does not exist")
		return nil
	}
	path, exist := targetResource["path"]
	if !exist {
		klog.Errorf("source resource attributes \"path\" does not exist")
		return nil
	}

	cli := &StreamruleEndpoint{
		nodeName:     nodeName,
		resourceUrl:  ep.Spec.URL,
		EndpointName: ep.Name,
		Path:         normalizeResource(path),
		Namespace:    ep.Namespace,
	}
	if atomic.CompareAndSwapInt32(&inited, 0, 1) {
		listener.InitHandler()
		go listener.StreamruleHandlerInstance.Serve()
	}
	return cli
}

func (s *StreamruleEndpoint) Name() string {
	return constants.StreamRuleEndpointProvider
}

func (s *StreamruleEndpoint) RegisterListener(handle listener.Handle) error {
	listener.StreamruleHandlerInstance.AddListener(fmt.Sprintf("/%s/%s/%s", s.nodeName, s.Namespace, s.Path), handle)
	return nil
}

func (s *StreamruleEndpoint) UnregisterListener() {
	listener.StreamruleHandlerInstance.RemoveListener(fmt.Sprintf("/%s/%s", s.Namespace, s.Path))
}

func (s *StreamruleEndpoint) SendMsg(data interface{}) (response interface{}, err error) {
	d := data.(map[string]interface{})
	v, exist := d["request"]
	if !exist {
		return nil, errors.New("input data does not exist value \"request\"")
	}
	request, ok := v.(*http.Request)
	if !ok {
		return nil, errors.New("invalid convert to http.Request")
	}
	uri := strings.SplitN(request.RequestURI, "/", 4)
	if len(uri) < 4 {
		return nil, errors.New("invalid format of http.Request")
	}
	v, exist = d["timeout"]
	if !exist {
		return nil, errors.New("input data does not exist value \"timeout\"")
	}
	timeout, ok := v.(time.Duration)
	if !ok {
		return nil, errors.New("invalid convert to time.Duration")
	}
	res := make(map[string]interface{})
	messageID := d["messageID"].(string)
	res["messageID"] = messageID
	res["param"] = strings.TrimPrefix(uri[3], s.Path)
	res["data"] = d["data"]
	res["nodeName"] = strings.Split(request.RequestURI, "/")[1]

	stop := make(chan struct{})
	respch := make(chan interface{})
	errch := make(chan error)
	go func() {
		resp, err := s.SendToEdge(res, stop)
		if err != nil {
			errch <- err
			return
		}
		respch <- resp
	}()
	timer := time.NewTimer(timeout)
	var httpResponse = &http.Response{
		Request: request,
		Header:  http.Header{},
	}
	select {
	case resp, ok := <-respch:
		if !ok {
			return nil, errors.New("failed to get res Channel")
		}
		timer.Stop()
		if resp == nil {
			httpResponse.StatusCode = http.StatusOK
			httpResponse.Body = io.NopCloser(strings.NewReader("message delivered"))
		} else {
			msg, ok := resp.(*model.Message)
			if !ok {
				klog.Error("response is not message type")
				httpResponse.StatusCode = http.StatusInternalServerError
				httpResponse.Body = io.NopCloser(strings.NewReader("invalid response"))
				return httpResponse, nil
			}
			content, err := msg.GetContentData()
			if err != nil {
				klog.Errorf("get message %s data err: %v", msg.GetID(), err)
				httpResponse.StatusCode = http.StatusInternalServerError
				httpResponse.Body = io.NopCloser(strings.NewReader("invalid response"))
				return httpResponse, nil
			}
			var response commonType.HTTPResponse
			if err := json.Unmarshal(content, &response); err != nil {
				klog.Errorf("message %s content can not convert to HTTPResponse: %v", msg.GetID(), err)
				httpResponse.StatusCode = http.StatusInternalServerError
				httpResponse.Body = io.NopCloser(strings.NewReader("invalid response"))
				return httpResponse, nil
			}
			httpResponse.StatusCode = response.StatusCode
			httpResponse.Body = io.NopCloser(bytes.NewReader(response.Body))
			httpResponse.Header = response.Header
		}
		klog.Infof("response from client, msg id: %s, write result success", messageID)
	case err := <-errch:
		timer.Stop()
		httpResponse.StatusCode = http.StatusInternalServerError
		httpResponse.Body = io.NopCloser(strings.NewReader(err.Error()))
		klog.Errorf("failed to get response, msg id: %s, write result: %v", messageID, err)
	case _, ok := <-timer.C:
		if !ok {
			return nil, errors.New("failed to get timer channel")
		}
		stop <- struct{}{}
		httpResponse.StatusCode = http.StatusRequestTimeout
		httpResponse.Body = io.NopCloser(strings.NewReader("wait to get response time out"))
		klog.Warningf("operation timeout, msg id: %s, write result: get response timeout", messageID)
	case _, ok := <-request.Context().Done():
		if !ok {
			return nil, errors.New("failed to get request close channel")
		}
		timer.Stop()
		klog.Warningf("Client disconnected for handling resource, msg id: %s", messageID)
		stop <- struct{}{}
		return nil, errors.New("client disconnected for handling resource")
	}

	return httpResponse, nil
}

func (s *StreamruleEndpoint) SendToEdge(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	var response *model.Message
	messageID, ok := data["messageID"].(string)
	if !ok {
		return nil, buildAndLogError("messageID")
	}
	nodeName, ok := data["nodeName"].(string)
	if !ok {
		return nil, buildAndLogError("nodeName")
	}

	dataBytes, ok := data["data"].([]byte)
	if !ok {
		return nil, buildAndLogError("data body")
	}

	var msgData map[string]interface{}
	err := json.Unmarshal(dataBytes, &msgData)
	if err != nil {
		klog.Errorf("json unmarshal failed: %v", err)
		return nil, err
	}

	operation, ok := msgData["operation"].(string)
	if !ok {
		return nil, buildAndLogError("data body operation")
	}

	msg := model.NewMessage("")
	msg.BuildHeader(messageID, "", msg.GetTimestamp())
	resource := "node/" + nodeName + "/" + s.EndpointName + "/" + s.resourceUrl

	msg.SetResourceOperation(resource, operation)
	msg.FillBody(string(dataBytes))
	msg.SetRoute(modules.StreamRuleEndpointProvider, modules.UserGroup)
	beehiveContext.Send(modules.CloudHubModuleName, *msg)
	if stop != nil {
		listener.MessageHandlerInstance.SetCallback(messageID, func(message *model.Message) {
			response = message
			stop <- struct{}{}
		})
		<-stop
		listener.MessageHandlerInstance.DelCallback(messageID)
	}
	return response, nil
}

func buildAndLogError(key string) error {
	err := fmt.Errorf("data transform failed, %s type is not matched or value is nil", key)
	klog.Error(err.Error())
	return err
}

func normalizeResource(resource string) string {
	finalResource := resource

	finalResource = strings.TrimPrefix(finalResource, "/")
	finalResource = strings.TrimSuffix(finalResource, "/")

	return finalResource
}
