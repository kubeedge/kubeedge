package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/256dpi/gomqtt/packet"
	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	mqttBus "github.com/kubeedge/kubeedge/edge/pkg/eventbus/mqtt"
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
	cancel   context.CancelFunc
	mqttMode int
}

// Register register eventbus
func Register() {
	mode, err := config.CONFIG.GetValue("mqtt.mode").ToInt()
	if err != nil || mode > externalMqttMode || mode < internalMqttMode {
		mode = internalMqttMode
	}
	edgeEventHubModule := eventbus{mqttMode: mode}
	core.Register(&edgeEventHubModule)
}

func (*eventbus) Name() string {
	return "eventbus"
}

func (*eventbus) Group() string {
	return modules.BusGroup
}

func (eb *eventbus) Start() {
	// no need to call TopicInit now, we have fixed topic
	var ctx context.Context
	ctx, eb.cancel = context.WithCancel(context.Background())

	nodeID := config.CONFIG.GetConfigurationByKey("edgehub.controller.node-id")
	if nodeID == nil {
		klog.Errorf("node id not configured")
		os.Exit(1)
	}

	mqttBus.NodeID = nodeID.(string)

	if eb.mqttMode >= bothMqttMode {
		// launch an external mqtt server
		externalMqttURL := config.CONFIG.GetConfigurationByKey("mqtt.server")
		if externalMqttURL == nil {
			panic(" mqtt server url not configured")
		}

		hub := &mqttBus.Client{
			MQTTUrl: externalMqttURL.(string),
		}
		mqttBus.MQTTHub = hub
		hub.InitSubClient()
		hub.InitPubClient()
	}

	if eb.mqttMode <= bothMqttMode {
		internalMqttURL := config.CONFIG.GetConfigurationByKey("mqtt.internal-server")
		if internalMqttURL == nil {
			internalMqttURL = defaultInternalMqttURL
		}

		qos := config.CONFIG.GetConfigurationByKey("mqtt.qos")
		if qos == nil {
			qos = defaultQos
		}

		retain := config.CONFIG.GetConfigurationByKey("mqtt.retain")
		if retain == nil {
			retain = defaultRetain
		}

		sessionQueueSize := config.CONFIG.GetConfigurationByKey("mqtt.session-queue-size")
		if sessionQueueSize == nil {
			sessionQueueSize = defaultSessionQueueSize
		}

		if qos.(int) < int(packet.QOSAtMostOnce) || qos.(int) > int(packet.QOSExactlyOnce) || sessionQueueSize.(int) <= 0 {
			klog.Errorf("mqtt.qos must be one of [0,1,2] or mqtt.session-queue-size must > 0")
			os.Exit(1)
		}
		// launch an internal mqtt server only
		mqttServer = mqttBus.NewMqttServer(sessionQueueSize.(int), internalMqttURL.(string), retain.(bool), qos.(int))
		mqttServer.InitInternalTopics()
		err := mqttServer.Run()
		if err != nil {
			klog.Errorf("Launch mqtt broker failed, %s", err.Error())
			os.Exit(1)
		}
	}

	eb.pubCloudMsgToEdge(ctx)
}

func (eb *eventbus) Cleanup() {
	eb.cancel()
	beehiveContext.Cleanup(eb.Name())
}

func pubMQTT(topic string, payload []byte) {
	token := mqttBus.MQTTHub.PubCli.Publish(topic, 1, false, payload)
	if token.WaitTimeout(util.TokenWaitTime) && token.Error() != nil {
		klog.Errorf("Error in pubMQTT with topic: %s, %v", topic, token.Error())
	} else {
		klog.Infof("Success in pubMQTT with topic: %s", topic)
	}
}

func (eb *eventbus) pubCloudMsgToEdge(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			klog.Warning("EventBus PubCloudMsg To Edge stop")
			return
		default:
		}
		accessInfo, err := beehiveContext.Receive(eb.Name())
		if err != nil {
			klog.Errorf("Fail to get a message from channel: %v", err)
			continue
		}
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
