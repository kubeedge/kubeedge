package eventbus

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/context"

	"github.com/kubeedge/kubeedge/pkg/eventbus/common/util"
	mqttBus "github.com/kubeedge/kubeedge/pkg/eventbus/mqtt"
)

// eventbus struct
type eventbus struct {
	context *context.Context
}

func init() {
	edgeEventHubModule := eventbus{}
	core.Register(&edgeEventHubModule)
}

func (*eventbus) Name() string {
	return "eventbus"
}

func (*eventbus) Group() string {
	return core.BusGroup
}

func (eb *eventbus) Start(c *context.Context) {
	// no need to call TopicInit now, we have fixed topic
	eb.context = c

	mqttURL := config.CONFIG.GetConfigurationByKey("mqtt.server")
	nodeID := config.CONFIG.GetConfigurationByKey("edgehub.controller.node-id")
	if mqttURL == nil || nodeID == nil {
		panic("mqtt url or node id not configured")
	}
	hub := &mqttBus.MQTTClient{
		MQTTUrl: mqttURL.(string),
	}
	mqttBus.MQTTHub = hub
	mqttBus.NodeID = nodeID.(string)
	mqttBus.ModuleContext = c
	hub.InitSubClient()
	hub.InitPubClient()

	eb.pubCloudMsgToEdge()
}

func (eb *eventbus) Cleanup() {
	eb.context.Cleanup(eb.Name())
}

func pubMQTT(topic string, payload []byte) {
	token := mqttBus.MQTTHub.PubCli.Publish(topic, 1, false, payload)
	if token.WaitTimeout(util.TokenWaitTime) && token.Error() != nil {
		log.LOGGER.Errorf("error in pubCloudMsgToEdge with topic: %s", topic)
	} else {
		log.LOGGER.Infof("success in pubCloudMsgToEdge with topic: %s", topic)
	}
}

func (eb *eventbus) pubCloudMsgToEdge() {
	for {
		if accessInfo, err := eb.context.Receive(eb.Name()); err == nil {
			operation := accessInfo.GetOperation()
			resource := accessInfo.GetResource()
			switch operation {
			case "subscribe":
				token := mqttBus.MQTTHub.SubCli.Subscribe(resource, 1, mqttBus.OnSubMessageReceived)
				if rs, err := util.CheckClientToken(token); !rs {
					log.LOGGER.Errorf("edge-hub-cli subscribe topic:%s, %v", resource, err)
					return
				}
				log.LOGGER.Infof("edge-hub-cli subscribe topic to %s", resource)
			case "message":
				body, ok := accessInfo.GetContent().(map[string]interface{})
				if !ok {
					log.LOGGER.Errorf("message is not map type")
					return
				}
				message := body["message"].(map[string]interface{})
				topic := message["topic"].(string)
				payload, _ := json.Marshal(&message)
				pubMQTT(topic, payload)
			case "publish":
				topic := resource
				var ok bool
				// cloud and edge will send different type of content, need to check
				payload, ok := accessInfo.GetContent().([]byte)
				if !ok {
					content := accessInfo.GetContent().(string)
					payload = []byte(content)
				}
				pubMQTT(topic, payload)
			case "get_result":
				if resource != "auth_info" {
					log.LOGGER.Info("skip none auth_info get_result message")
					return
				}
				topic := fmt.Sprintf("$hw/events/node/%s/authInfo/get/result", mqttBus.NodeID)
				payload, _ := json.Marshal(accessInfo.GetContent())
				pubMQTT(topic, payload)
			default:
				log.LOGGER.Warnf("action not found")
			}
		} else {
			log.LOGGER.Errorf("fail to get a message from channel: %v", err)
		}
	}
}
