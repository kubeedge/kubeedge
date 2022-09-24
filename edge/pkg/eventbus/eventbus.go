package eventbus

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/astaxie/beego/orm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/dao"
	mqttBus "github.com/kubeedge/kubeedge/edge/pkg/eventbus/mqtt"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

var mqttServer *mqttBus.Server

// eventbus struct
type eventbus struct {
	enable bool
}

var _ core.Module = (*eventbus)(nil)

func newEventbus(enable bool) *eventbus {
	return &eventbus{
		enable: enable,
	}
}

// Register register eventbus
func Register(eventbus *v1alpha2.EventBus, nodeName string) {
	eventconfig.InitConfigure(eventbus, nodeName)
	core.Register(newEventbus(eventbus.Enable))
	orm.RegisterModel(new(dao.SubTopics))
}

func (*eventbus) Name() string {
	return modules.EventBusModuleName
}

func (*eventbus) Group() string {
	return modules.BusGroup
}

// Enable indicates whether this module is enabled
func (eb *eventbus) Enable() bool {
	return eb.enable
}

func (eb *eventbus) Start() {
	mqttBus.RegisterMsgHandler()

	if eventconfig.Config.MqttMode >= v1alpha2.MqttModeBoth {
		hub := &mqttBus.Client{
			MQTTUrl:     eventconfig.Config.MqttServerExternal,
			SubClientID: eventconfig.Config.MqttSubClientID,
			PubClientID: eventconfig.Config.MqttPubClientID,
			Username:    eventconfig.Config.MqttUsername,
			Password:    eventconfig.Config.MqttPassword,
		}
		mqttBus.MQTTHub = hub
		hub.InitSubClient()
		hub.InitPubClient()
		klog.Infof("Init Sub And Pub Client for external mqtt broker %v successfully", eventconfig.Config.MqttServerExternal)
	}

	if eventconfig.Config.MqttMode <= v1alpha2.MqttModeBoth {
		// launch an internal mqtt server only
		mqttServer = mqttBus.NewMqttServer(
			int(eventconfig.Config.MqttSessionQueueSize),
			eventconfig.Config.MqttServerInternal,
			eventconfig.Config.MqttRetain,
			int(eventconfig.Config.MqttQOS))
		mqttServer.InitInternalTopics()
		err := mqttServer.Run()
		if err != nil {
			klog.Errorf("Launch internal mqtt broker failed, %s", err.Error())
			os.Exit(1)
		}
		klog.Infof("Launch internal mqtt broker %v successfully", eventconfig.Config.MqttServerInternal)
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
		case messagepkg.OperationSubscribe:
			eb.subscribe(resource)
			klog.Infof("Edge-hub-cli subscribe topic to %s", resource)
		case messagepkg.OperationUnsubscribe:
			eb.unsubscribe(resource)
			klog.Infof("Edge-hub-cli unsubscribe topic to %s", resource)
		case messagepkg.OperationMessage:
			body, ok := accessInfo.GetContent().(map[string]interface{})
			if !ok {
				klog.Errorf("Message type is %T and not map type", accessInfo.GetContent())
				continue
			}
			message, ok := body["message"].(map[string]interface{})
			if !ok {
				klog.Errorf("Message body type is %T and not map type", body["message"])
				continue
			}
			topic, ok := message["topic"].(string)
			if !ok {
				klog.Errorf("Message topic body type is %T and not string type", message["topic"])
				continue
			}
			payload, err := json.Marshal(&message)
			if err != nil {
				klog.Errorf("marshal message %v error: %v", topic, err)
				continue
			}
			eb.publish(topic, payload)
		case messagepkg.OperationPublish:
			topic := resource
			// cloud and edge will send different type of content, need to check
			payload, ok := accessInfo.GetContent().([]byte)
			if !ok {
				content, ok := accessInfo.GetContent().(string)
				if !ok {
					klog.Errorf("Message is not []byte or string")
					continue
				}
				payload = []byte(content)
			}
			eb.publish(topic, payload)
		case messagepkg.OperationGetResult:
			if resource != "auth_info" {
				klog.Info("Skip none auth_info get_result message")
				continue
			}
			topic := fmt.Sprintf("$hw/events/node/%s/authInfo/get/result", eventconfig.Config.NodeName)
			payload, _ := json.Marshal(accessInfo.GetContent())
			eb.publish(topic, payload)
		default:
			klog.Warningf("Action not found")
		}
	}
}

func (eb *eventbus) publish(topic string, payload []byte) {
	if eventconfig.Config.MqttMode >= v1alpha2.MqttModeBoth {
		// pub msg to external mqtt broker.
		pubMQTT(topic, payload)
	}

	if eventconfig.Config.MqttMode <= v1alpha2.MqttModeBoth {
		// pub msg to internal mqtt broker.
		mqttServer.Publish(topic, payload)
	}
}

func (eb *eventbus) subscribe(topic string) {
	if eventconfig.Config.MqttMode <= v1alpha2.MqttModeBoth {
		// set topic to internal mqtt broker.
		mqttServer.SetTopic(topic)
	}

	if eventconfig.Config.MqttMode >= v1alpha2.MqttModeBoth {
		// subscribe topic to external mqtt broker.
		token := mqttBus.MQTTHub.SubCli.Subscribe(topic, 1, mqttBus.OnSubMessageReceived)
		if rs, err := util.CheckClientToken(token); !rs {
			klog.Errorf("Edge-hub-cli subscribe topic: %s, %v", topic, err)
			return
		}
	}

	err := dao.InsertTopics(topic)
	if err != nil {
		klog.Errorf("Insert topic %s failed, %v", topic, err)
	}
}

func (eb *eventbus) unsubscribe(topic string) {
	if eventconfig.Config.MqttMode <= v1alpha2.MqttModeBoth {
		mqttServer.RemoveTopic(topic)
	}

	if eventconfig.Config.MqttMode >= v1alpha2.MqttModeBoth {
		token := mqttBus.MQTTHub.SubCli.Unsubscribe(topic)
		if rs, err := util.CheckClientToken(token); !rs {
			klog.Errorf("Edge-hub-cli unsubscribe topic: %s, %v", topic, err)
			return
		}
	}

	err := dao.DeleteTopicsByKey(topic)
	if err != nil {
		klog.Errorf("Delete topic %s failed, %v", topic, err)
	}
}
