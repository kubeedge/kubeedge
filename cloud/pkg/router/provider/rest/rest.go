package rest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"k8s.io/klog/v2"

	v1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/rules/v1"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/provider"
	httpUtils "github.com/kubeedge/kubeedge/cloud/pkg/router/utils/http"
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
	return "rest"
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
	return "rest"
}

func (r *Rest) RegisterListener(handle listener.Handle) error {
	listener.RestHandlerInstance.AddListener(fmt.Sprintf("/%s/%s", r.Namespace, r.Path), handle)
	return nil
}

func (r *Rest) UnregisterListener() {
	listener.RestHandlerInstance.RemoveListener(fmt.Sprintf("/%s/%s", r.Namespace, r.Path))
}

func (*Rest) Forward(target provider.Target, data interface{}) (response interface{}, err error) {
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
		return nil, errors.New("invalid conver to time.Duration")
	}
	res := make(map[string]interface{})
	messageID := d["messageID"].(string)
	res["messageID"] = messageID
	res["param"] = ""
	res["data"] = d["data"]
	res["nodeName"] = strings.Split(request.RequestURI, "/")[1]
	stop := make(chan struct{})
	respch := make(chan interface{})
	errch := make(chan error)
	go func() {
		//resp, err := target.GoToTarget(res, stop)
		//if err != nil {
		//	errch <- err
		//	return
		//}
		resp, err := target.GoToTarget(res, nil)
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
	var respBody string
	select {
	case response, ok = <-respch:
		if !ok {
			klog.Error("failed to get res Channel")
		}
		timer.Stop()
		httpResponse.StatusCode = http.StatusOK
		respBody = "message delivered"
		klog.Infof("response from client, msg id: %s, write result success", messageID)
	case err := <-errch:
		timer.Stop()
		httpResponse.StatusCode = http.StatusInternalServerError
		respBody = err.Error()
		klog.Infof("failed to get response, msg id: %s, write result: %v", messageID, err)
	case _, ok := <-timer.C:
		if !ok {
			klog.Error("failed to get timer channel")
		}
		stop <- struct{}{}
		httpResponse.StatusCode = http.StatusRequestTimeout
		respBody = err.Error()
		klog.Warningf("operation timeout, msg id: %s, write result: get response timeout", messageID)
	case _, ok := <-request.Context().Done():
		if !ok {
			klog.Error("failed to get request close channel")
		}
		timer.Stop()
		err = errors.New("client disconnected for handling resource")
		klog.Warningf("Client disconnected for handling resource, msg id: %s", messageID)
		stop <- struct{}{}
		return
	}
	httpResponse.Body = ioutil.NopCloser(strings.NewReader(respBody))
	response = httpResponse
	return
}

func (r *Rest) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	//No need to send ACK in the moment
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
