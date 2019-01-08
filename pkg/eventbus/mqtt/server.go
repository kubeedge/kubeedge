package mqtt

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/256dpi/gomqtt/broker"
	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/topic"
	"github.com/256dpi/gomqtt/transport"

	"github.com/kubeedge/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/beehive/pkg/core/model"
)

type MqttServer struct {
	url              string
	tree             *topic.Tree
	server           transport.Server
	backend          *MemoryBackendExtend
	sessionQueueSize int
}

func NewMqttServer(sqz int, url string) *MqttServer {
	tree := topic.NewTree()
	for _, v := range SubTopics {
		tree.Set(v, packet.Subscription{Topic: v, QOS: packet.QOSAtMostOnce})
	}
	return &MqttServer{
		sessionQueueSize: sqz,
		url:              url,
		tree:             tree,
	}
}

func (m *MqttServer) Run() error {
	var err error

	m.server, err = transport.Launch(m.url)
	if err != nil {
		log.LOGGER.Errorf("launch transport failed.", err)
		return err
	}

	m.backend = NewMemoryBackendExtend()
	m.backend.Backend.SessionQueueSize = m.sessionQueueSize

	m.backend.Backend.Logger = func(event broker.LogEvent, client *broker.Client, pkt packet.Generic, msg *packet.Message, err error) {
		if event == broker.MessagePublished {
			log.LOGGER.Infof("message publish: clientId: [%s], topic: [%s], payload: [%s]", client.ID(), msg.Topic, string(msg.Payload))
			if len(m.tree.Match(msg.Topic)) > 0 {
				log.LOGGER.Infof("topic: [%s] matched.", msg.Topic)
				m.onSubscribe(msg)
			}
		}
	}

	engine := broker.NewEngine(m.backend.Backend)
	engine.Accept(m.server)

	return nil
}

func (m *MqttServer) onSubscribe(msg *packet.Message) {

	// for "$hw/events/device/+/twin/+", "$hw/events/node/+/membership/get", send to twin
	// for other, send to hub
	// for "SYS/dis/upload_records", no need to base64 topic
	var target string
	resource := base64.URLEncoding.EncodeToString([]byte(msg.Topic))
	if strings.HasPrefix(msg.Topic, "$hw/events/device") || strings.HasPrefix(msg.Topic, "$hw/events/node") {
		target = core.TwinGroup
	} else {
		target = core.HubGroup
		if msg.Topic == "SYS/dis/upload_records" {
			resource = "SYS/dis/upload_records"
		}
	}
	// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
	message := model.NewMessage("").BuildRouter(core.BusGroup, "user",
		resource, "response").FillBody(string(msg.Payload))
	log.LOGGER.Info(fmt.Sprintf("received msg from mqttserver, deliver to %s with resource %s", target, resource))
	ModuleContext.Send2Group(target, *message)
}

func (m *MqttServer) SetTopic(topic string) {
	m.tree.Set(topic, packet.Subscription{Topic: topic, QOS: packet.QOSAtMostOnce})
}

func (m *MqttServer) Publish(topic string, payload []byte) {
	m.backend.Publish(topic, payload)
}
