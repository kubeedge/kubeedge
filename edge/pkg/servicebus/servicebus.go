package servicebus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/servicebus/util"
)

const (
	sourceType  = "router_rest"
	maxBodySize = 5 * 1e6
)

// servicebus struct
type servicebus struct {
	context *context.Context
}

func init() {
	edgeServiceBusModule := servicebus{}
	core.Register(&edgeServiceBusModule)
}

func (*servicebus) Name() string {
	return "servicebus"
}

func (*servicebus) Group() string {
	return modules.BusGroup
}

func (sb *servicebus) Start(c *context.Context) {
	// no need to call TopicInit now, we have fixed topic
	sb.context = c
	var htc = new(http.Client)
	htc.Timeout = time.Second * 10

	var uc = new(util.URLClient)
	uc.Client = htc

	//Get message from channel
	for {
		if msg, ok := sb.context.Receive("servicebus"); ok == nil {
			go func() {
				log.LOGGER.Infof("ServiceBus receive msg")
				source := msg.GetSource()
				if source != sourceType {
					return
				}
				resource := msg.GetResource()
				r := strings.Split(resource, ":")
				if len(r) != 2 {
					m := "the format of resource " + resource + " is incorrect"
					log.LOGGER.Warnf(m)
					code := http.StatusBadRequest
					if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
						sb.context.Send2Group(modules.HubGroup, response)
					}
					return
				}
				content, err := json.Marshal(msg.GetContent())
				if err != nil {
					log.LOGGER.Errorf("marshall message content failed", err)
					m := "error to marshal request msg content"
					code := http.StatusBadRequest
					if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
						sb.context.Send2Group(modules.HubGroup, response)
					}
					return
				}
				var httpRequest util.HTTPRequest
				if err := json.Unmarshal(content, &httpRequest); err != nil {
					m := "error to parse http request"
					code := http.StatusBadRequest
					log.LOGGER.Errorf(m, err)
					if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
						sb.context.Send2Group(modules.HubGroup, response)
					}
					return
				}
				operation := msg.GetOperation()
				targetURL := "http://127.0.0.1:" + r[0] + "/" + r[1]
				resp, err := uc.HTTPDo(operation, targetURL, httpRequest.Header, httpRequest.Body)
				if err != nil {
					m := "error to call service"
					code := http.StatusNotFound
					log.LOGGER.Errorf(m, err)
					if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
						sb.context.Send2Group(modules.HubGroup, response)
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
					log.LOGGER.Errorf(m, err)
					if response, err := buildErrorResponse(msg.GetID(), m, code); err == nil {
						sb.context.Send2Group(modules.HubGroup, response)
					}
					return
				}

				response := util.HTTPResponse{Header: resp.Header, StatusCode: resp.StatusCode, Body: resBody}
				responseMsg := model.NewMessage(msg.GetID())
				responseMsg.Content = response
				responseMsg.SetRoute("servicebus", modules.UserGroup)
				sb.context.Send2Group(modules.HubGroup, *responseMsg)
			}()
		}
	}
}

func (sb *servicebus) Cleanup() {
	sb.context.Cleanup(sb.Name())
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
