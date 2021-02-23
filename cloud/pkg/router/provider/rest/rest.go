package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
	httpUtils "github.com/kubeedge/kubeedge/cloud/pkg/router/utils/http"
	commonType "github.com/kubeedge/kubeedge/common/types"
)

var inited int32

type restFactory struct {
}

type Rest struct {
	Endpoint  string
	Path      string
	Namespace string
}

func init() {
	factory := &restFactory{}
	provider.RegisterSource(factory)
	provider.RegisterTarget(factory)
}

func (factory *restFactory) Type() string {
	return constants.RestEndpoint
}

func (*restFactory) GetSource(ep *v1.RuleEndpoint, sourceResource map[string]string) provider.Source {
	path, exist := sourceResource["path"]
	if !exist {
		klog.Errorf("source resource attributes \"path\" does not exist")
		return nil
	}
	cli := &Rest{Namespace: ep.Namespace, Path: normalizeResource(path)}
	if atomic.CompareAndSwapInt32(&inited, 0, 1) {
		listener.InitHandler()
		// guarantee that it will be executed only once
		go listener.RestHandlerInstance.Serve()
	}
	return cli
}

func (*restFactory) GetTarget(ep *v1.RuleEndpoint, targetResource map[string]string) provider.Target {
	endpoint, exist := targetResource["resource"]
	if !exist {
		klog.Errorf("target resource attributes \"resource\" does not exist")
		return nil
	}
	cli := &Rest{
		Namespace: ep.Namespace,
		Endpoint:  endpoint,
	}
	return cli
}

func (*Rest) Name() string {
	return constants.RestProvider
}

func (r *Rest) RegisterListener(handle listener.Handle) error {
	listener.RestHandlerInstance.AddListener(fmt.Sprintf("/%s/%s", r.Namespace, r.Path), handle)
	return nil
}

func (r *Rest) UnregisterListener() {
	listener.RestHandlerInstance.RemoveListener(fmt.Sprintf("/%s/%s", r.Namespace, r.Path))
}

func (r *Rest) Forward(target provider.Target, data interface{}) (interface{}, error) {
	d := data.(map[string]interface{})
	v, exist := d["request"]
	if !exist {
		return nil, errors.New("input data does not exist value \"request\"")
	}
	request, ok := v.(*http.Request)
	if !ok {
		return nil, errors.New("invalid convert to http.Request")
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
	res["param"] = strings.TrimLeft(strings.SplitN(request.RequestURI, "/", 4)[3], r.Path)
	res["data"] = d["data"]
	res["nodeName"] = strings.Split(request.RequestURI, "/")[1]
	res["header"] = request.Header
	res["method"] = request.Method
	stop := make(chan struct{})
	respch := make(chan interface{})
	errch := make(chan error)
	go func() {
		resp, err := target.GoToTarget(res, stop)
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
			httpResponse.Body = ioutil.NopCloser(strings.NewReader("message delivered"))
		} else {
			msg, ok := resp.(*model.Message)
			if !ok {
				klog.Error("response can not convert to Message")
				httpResponse.StatusCode = http.StatusInternalServerError
				httpResponse.Body = ioutil.NopCloser(strings.NewReader("invalid response"))
				return httpResponse, nil
			}
			content, err := json.Marshal(msg.GetContent())
			if err != nil {
				klog.Error("message content can not convert to json")
				httpResponse.StatusCode = http.StatusInternalServerError
				httpResponse.Body = ioutil.NopCloser(strings.NewReader("invalid response"))
				return httpResponse, nil
			}
			var response commonType.HTTPResponse
			if err := json.Unmarshal(content, &response); err != nil {
				klog.Error("message content can not convert to HTTPResponse")
				httpResponse.StatusCode = http.StatusInternalServerError
				httpResponse.Body = ioutil.NopCloser(strings.NewReader("invalid response"))
				return httpResponse, nil
			}
			httpResponse.StatusCode = response.StatusCode
			httpResponse.Body = ioutil.NopCloser(bytes.NewReader(response.Body))
			httpResponse.Header = response.Header
		}
		klog.Infof("response from client, msg id: %s, write result success", messageID)
	case err := <-errch:
		timer.Stop()
		httpResponse.StatusCode = http.StatusInternalServerError
		httpResponse.Body = ioutil.NopCloser(strings.NewReader(err.Error()))
		klog.Errorf("failed to get response, msg id: %s, write result: %v", messageID, err)
	case _, ok := <-timer.C:
		if !ok {
			return nil, errors.New("failed to get timer channel")
		}
		stop <- struct{}{}
		httpResponse.StatusCode = http.StatusRequestTimeout
		httpResponse.Body = ioutil.NopCloser(strings.NewReader("wait to get response time out"))
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

func (r *Rest) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	//TODO: need to get ACK
	v, exist := data["data"]
	if !exist {
		return nil, errors.New("input data does not exist value \"data\"")
	}
	content, ok := v.([]byte)
	if !ok || len(content) == 0 {
		return nil, errors.New("invalid convert to []byte")
	}
	req, err := httpUtils.BuildRequest(http.MethodPost, r.Endpoint, bytes.NewReader(content), "", "")
	if err != nil {
		return nil, err
	}

	client := httpUtils.NewHTTPClient()
	return httpUtils.SendRequest(req, client)
}

func normalizeResource(resource string) string {
	finalResource := resource
	if strings.HasPrefix(finalResource, "/") {
		finalResource = strings.TrimLeft(finalResource, "/")
	}
	if strings.HasSuffix(finalResource, "/") {
		finalResource = strings.TrimRight(finalResource, "/")
	}
	return finalResource
}
