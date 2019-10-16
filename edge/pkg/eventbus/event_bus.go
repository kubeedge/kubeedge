package eventbus

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/256dpi/gomqtt/packet"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	eventbusconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
	mqttBus "github.com/kubeedge/kubeedge/edge/pkg/eventbus/mqtt"
	edgecoreconfig "github.com/kubeedge/kubeedge/pkg/edgecore/apis/config"
)

var mqttServer *mqttBus.Server

const (
	internalMqttMode = iota // 0: launch an internal mqtt broker.
	bothMqttMode            // 1: launch an internal and external mqtt broker.
	externalMqttMode        // 2: launch an external mqtt broker.

	defaultInternalMqttURL  = "tcp://127.0.0.1:1884"
	defaultQos              = 0
	defaultRetain           = false
	defaultSessionQueueSize = 100
)

// eventbus struct
type eventbus struct {
	context  *context.Context
	mqttMode int
}

// Register register eventbus
func Register(e *edgecoreconfig.EdgedConfig, m *edgecoreconfig.MqttConfig) {
	eventbusconfig.InitEventbusConfig(e, m)
	edgeEventHubModule := eventbus{mqttMode: int(m.Mode)}
	core.Register(&edgeEventHubModule)
}

func (*eventbus) Name() string {
	return "eventbus"
}

func (*eventbus) Group() string {
	return modules.BusGroup
}

func (eb *eventbus) Start(c *context.Context) {
	// no need to call TopicInit now, we have fixed topic
	eb.context = c

	mqttBus.NodeID = eventbusconfig.Conf().Edged.HostnameOverride
	mqttBus.ModuleContext = c

	if eb.mqttMode >= bothMqttMode {
		hub := &mqttBus.Client{
			MQTTUrl: eventbusconfig.Conf().Mqtt.Server,
		}
		mqttBus.MQTTHub = hub
		hub.InitSubClient()
		hub.InitPubClient()
	}

	if eb.mqttMode <= bothMqttMode {
		internalMqttURL := eventbusconfig.Conf().Mqtt.InternalServer
		qos := int(eventbusconfig.Conf().Mqtt.QOS)
		retain := eventbusconfig.Conf().Mqtt.Retain

		sessionQueueSize := int(eventbusconfig.Conf().Mqtt.SessionQueueSize)

		if qos < int(packet.QOSAtMostOnce) || qos > int(packet.QOSExactlyOnce) || sessionQueueSize <= 0 {
			klog.Errorf("mqtt.qos must be one of [0,1,2] or mqtt.session-queue-size must > 0")
			os.Exit(1)
		}
		// launch an internal mqtt server only
		mqttServer = mqttBus.NewMqttServer(sessionQueueSize, internalMqttURL, retain, qos)
		mqttServer.InitInternalTopics()
		err := mqttServer.Run()
		if err != nil {
			klog.Errorf("Launch mqtt broker failed, %s", err.Error())
			os.Exit(1)
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
		klog.Errorf("Error in pubMQTT with topic: %s, %v", topic, token.Error())
	} else {
		klog.Infof("Success in pubMQTT with topic: %s", topic)
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
				klog.Infof("Edge-hub-cli subscribe topic to %s", resource)
			case "message":
				body, ok := accessInfo.GetContent().(map[string]interface{})
				if !ok {
					klog.Errorf("Message is not map type")
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
					klog.Info("Skip none auth_info get_result message")
					return
				}
				topic := fmt.Sprintf("$hw/events/node/%s/authInfo/get/result", mqttBus.NodeID)
				payload, _ := json.Marshal(accessInfo.GetContent())
				eb.publish(topic, payload)
			default:
				klog.Warningf("Action not found")
			}
		} else {
			klog.Errorf("Fail to get a message from channel: %v", err)
		}
	}
}

func (eb *eventbus) publish(topic string, payload []byte) {
	if eb.mqttMode >= bothMqttMode {
		// pub msg to external mqtt broker.
		pubMQTT(topic, payload)
	}

	if eb.mqttMode <= bothMqttMode {
		// pub msg to internal mqtt broker.
		mqttServer.Publish(topic, payload)
	}
}

func (eb *eventbus) subscribe(topic string) {
	if eb.mqttMode >= bothMqttMode {
		// subscribe topic to external mqtt broker.
		token := mqttBus.MQTTHub.SubCli.Subscribe(topic, 1, mqttBus.OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			klog.Errorf("Edge-hub-cli subscribe topic: %s, %v", topic, err)
		}
	}

	if eb.mqttMode <= bothMqttMode {
		// set topic to internal mqtt broker.
		mqttServer.SetTopic(topic)
	}
}
