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

var mqttServer *mqttBus.MqttServer

const (
	internalMqttMode = iota
	doubleMqttMode
	externalMqttMode
)

// eventbus struct
type eventbus struct {
	context  *context.Context
	mqttMode int
}

func init() {
	mode := config.CONFIG.GetConfigurationByKey("mqtt.mode")
	if mode == nil || mode.(int) > externalMqttMode || mode.(int) < internalMqttMode {
		panic("mqtt mode must be 0,1,2")
	}
	edgeEventHubModule := eventbus{mqttMode: mode.(int)}
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

	nodeID := config.CONFIG.GetConfigurationByKey("edgehub.controller.node-id")
	if nodeID == nil {
		panic("mqtt url or node id not configured")
	}

	mqttBus.NodeID = nodeID.(string)
	mqttBus.ModuleContext = c

	if eb.mqttMode >= doubleMqttMode {
		// launch an external mqtt server
		externalMqttUrl := config.CONFIG.GetConfigurationByKey("mqtt.external-mqtt")
		if externalMqttUrl == nil {
			panic("third party mqtt url not configured")
		}

		hub := &mqttBus.MQTTClient{
			MQTTUrl: externalMqttUrl.(string),
		}
		mqttBus.MQTTHub = hub
		hub.InitSubClient()
		hub.InitPubClient()
	}

	if eb.mqttMode <= doubleMqttMode {
		internalMqttURL := config.CONFIG.GetConfigurationByKey("mqtt.internal-mqtt")
		if internalMqttURL == nil {
			panic("mqtt url is not configured")
		}
		// launch an internal mqtt server only
		// launch a mqtt server
		mqttServer = mqttBus.NewMqttServer(100, internalMqttURL.(string))
		err := mqttServer.Run()
		if err != nil {
			panic(fmt.Sprintf("Launch mqtt borker failed, %s", err.Error()))
		}
	}

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
				eb.subscribe(resource)
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
				eb.publish(topic, payload)
			case "publish":
				topic := resource
				var ok bool
				// cloud and edge will send different type of content, need to check
				payload, ok := accessInfo.GetContent().([]byte)
				if !ok {
					content := accessInfo.GetContent().(string)
					payload = []byte(content)
				}
				eb.publish(topic, payload)
			case "get_result":
				if resource != "auth_info" {
					log.LOGGER.Info("skip none auth_info get_result message")
					return
				}
				topic := fmt.Sprintf("$hw/events/node/%s/authInfo/get/result", mqttBus.NodeID)
				payload, _ := json.Marshal(accessInfo.GetContent())
				eb.publish(topic, payload)
			default:
				log.LOGGER.Warnf("action not found")
			}
		} else {
			log.LOGGER.Errorf("fail to get a message from channel: %v", err)
		}
	}
}

func (eb *eventbus) publish(topic string, payload []byte) {
	if eb.mqttMode >= doubleMqttMode {
		// pub msg to external mqtt broker.
		pubMQTT(topic, payload)
	}

	if eb.mqttMode <= doubleMqttMode {
		// pub msg to internal mqtt broker.
		mqttServer.Publish(topic, payload)
	}
}

func (eb *eventbus) subscribe(topic string) {
	if eb.mqttMode >= doubleMqttMode {
		// subscribe topic to thirdparty mqtt broker.
		token := mqttBus.MQTTHub.SubCli.Subscribe(topic, 1, mqttBus.OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			log.LOGGER.Errorf("edge-hub-cli subscribe topic:%s, %v", topic, err)
		}
	}

	if eb.mqttMode <= doubleMqttMode {
		// set topic to internal mqtt broker.
		mqttServer.SetTopic(topic)
	}
}
