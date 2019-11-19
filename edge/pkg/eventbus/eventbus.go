package eventbus

import (
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
	mqttBus "github.com/kubeedge/kubeedge/edge/pkg/eventbus/mqtt"
)

var mqttServer *mqttBus.Server

// eventbus struct
type eventbus struct {
}

func newEventbus() *eventbus {
	return &eventbus{}
}

// Register register eventbus
func Register() {
	eventconfig.InitConfigure()
	core.Register(newEventbus())
}

func (*eventbus) Name() string {
	return "eventbus"
}

func (*eventbus) Group() string {
	return modules.BusGroup
}

func (eb *eventbus) Start() {

	if eventconfig.Get().Mode >= eventconfig.BothMqttMode {

		hub := &mqttBus.Client{
			MQTTUrl: eventconfig.Get().ExternalMqttURL,
		}
		mqttBus.MQTTHub = hub
		hub.InitSubClient()
		hub.InitPubClient()
	}

	if eventconfig.Get().Mode <= eventconfig.BothMqttMode {
		// launch an internal mqtt server only
		mqttServer = mqttBus.NewMqttServer(
			eventconfig.Get().SessionQueueSize,
			eventconfig.Get().InternalMqttURL,
			eventconfig.Get().Retain,
			eventconfig.Get().QOS)
		mqttServer.InitInternalTopics()
		err := mqttServer.Run()
		if err != nil {
			klog.Errorf("Launch mqtt broker failed, %s", err.Error())
			os.Exit(1)
		}
	}

	eb.pubCloudMsgToEdge()
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
		select {
		case <-beehiveContext.Done():
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
			topic := fmt.Sprintf("$hw/events/node/%s/authInfo/get/result", eventconfig.Get().NodeID)
			payload, _ := json.Marshal(accessInfo.GetContent())
			eb.publish(topic, payload)
		default:
			klog.Warningf("Action not found")
		}
	}
}

func (eb *eventbus) publish(topic string, payload []byte) {
	if eventconfig.Get().Mode >= eventconfig.BothMqttMode {
		// pub msg to external mqtt broker.
		pubMQTT(topic, payload)
	}

	if eventconfig.Get().Mode <= eventconfig.BothMqttMode {
		// pub msg to internal mqtt broker.
		mqttServer.Publish(topic, payload)
	}
}

func (eb *eventbus) subscribe(topic string) {
	if eventconfig.Get().Mode >= eventconfig.BothMqttMode {
		// subscribe topic to external mqtt broker.
		token := mqttBus.MQTTHub.SubCli.Subscribe(topic, 1, mqttBus.OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			klog.Errorf("Edge-hub-cli subscribe topic: %s, %v", topic, err)
		}
	}

	if eventconfig.Get().Mode <= eventconfig.BothMqttMode {
		// set topic to internal mqtt broker.
		mqttServer.SetTopic(topic)
	}
}
