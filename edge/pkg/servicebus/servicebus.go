package servicebus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus/util"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
)

const (
	sourceType  = "router_rest"
	maxBodySize = 5 * 1e6
)

// servicebus struct
type servicebus struct {
	enable bool
}

func newServicebus(enable bool) *servicebus {
	return &servicebus{
		enable: enable,
	}
}

// Register register servicebus
func Register(s *v1alpha1.ServiceBus) {
	core.Register(newServicebus(s.Enable))
}

func (*servicebus) Name() string {
	return modules.ServiceBusModuleName
}

func (*servicebus) Group() string {
	return modules.UserGroup
}

func (sb *servicebus) Enable() bool {
	return sb.enable
}

func (sb *servicebus) Start() {
	// no need to call TopicInit now, we have fixed topic
	var htc = new(http.Client)
	htc.Timeout = time.Second * 10

	var uc = new(util.URLClient)
	uc.Client = htc

	//Get message from channel
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("ServiceBus stop")
			return
		default:
		}
		msg, err := beehiveContext.Receive("servicebus")
		if err != nil {
			klog.Warningf("servicebus receive msg error %v", err)
			continue
		}
		go func() {
			klog.Infof("ServiceBus receive msg")
			source := msg.GetSource()
			if source != sourceType {
				return
			}
			resource := msg.GetResource()
			r := strings.Split(resource, ":")
			if len(r) != 2 {
				m := "the format of resource " + resource + " is incorrect"
				klog.Warningf(m)
				code := http.StatusBadRequest
				if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
					beehiveContext.SendToGroup(modules.HubGroup, response)
				}
				return
			}
			content, err := json.Marshal(msg.GetContent())
			if err != nil {
				klog.Errorf("marshall message content failed %v", err)
				m := "error to marshal request msg content"
				code := http.StatusBadRequest
				if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
					beehiveContext.SendToGroup(modules.HubGroup, response)
				}
				return
			}
			var httpRequest util.HTTPRequest
			if err := json.Unmarshal(content, &httpRequest); err != nil {
				m := "error to parse http request"
				code := http.StatusBadRequest
				klog.Errorf(m, err)
				if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
					beehiveContext.SendToGroup(modules.HubGroup, response)
				}
				return
			}
			operation := msg.GetOperation()
			targetURL := "http://127.0.0.1:" + r[0] + "/" + r[1]
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
			resp.Body = http.MaxBytesReader(nil, resp.Body, maxBodySize)
			resBody, err := ioutil.ReadAll(resp.Body)
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

			response := util.HTTPResponse{Header: resp.Header, StatusCode: resp.StatusCode, Body: resBody}
			responseMsg := model.NewMessage(msg.GetID())
			responseMsg.Content = response
			responseMsg.SetRoute("servicebus", modules.UserGroup)
			beehiveContext.SendToGroup(modules.HubGroup, *responseMsg)
		}()
	}
}

func buildErrorResponse(parentID string, content string, statusCode int) (model.Message, error) {
	responseMsg := model.NewMessage(parentID)
	h := http.Header{}
	h.Add("Server", "kubeedge-edgecore")
	c := util.HTTPResponse{Header: h, StatusCode: statusCode, Body: []byte(content)}
	responseMsg.Content = c
	responseMsg.SetRoute("servicebus", modules.UserGroup)
	return *responseMsg, nil
}
