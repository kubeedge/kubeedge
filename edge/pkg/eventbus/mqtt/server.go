/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mqtt

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/256dpi/gomqtt/broker"
	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/topic"
	"github.com/256dpi/gomqtt/transport"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	messagepkg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/dao"
)

//Server serve as an internal mqtt broker.
type Server struct {
	// Internal mqtt url
	url string

	// Used to save and match topic, it is thread-safe tree.
	tree *topic.Tree

	// A server accepts incoming connections.
	server transport.Server

	// A MemoryBackend stores all in memory.
	backend *broker.MemoryBackend

	// Qos has three types: QOSAtMostOnce, QOSAtLeastOnce, QOSExactlyOnce.
	// now we use QOSAtMostOnce as default.
	qos int

	// If set retain to true, the topic message will be saved in memory and
	// the future subscribers will receive the msg whose subscriptions match
	// its topic.
	// If set retain to false, then will do nothing.
	retain bool

	// A sessionQueueSize will default to 100
	sessionQueueSize int
}

// NewMqttServer create an internal mqtt server.
func NewMqttServer(sqz int, url string, retain bool, qos int) *Server {
	return &Server{
		sessionQueueSize: sqz,
		url:              url,
		tree:             topic.NewTree(),
		retain:           retain,
		qos:              qos,
	}
}

// Run launch a server and accept connections.
func (m *Server) Run() error {
	var err error

	m.server, err = transport.Launch(m.url)
	if err != nil {
		klog.Errorf("Launch transport failed %v", err)
		return err
	}

	m.backend = broker.NewMemoryBackend()
	m.backend.SessionQueueSize = m.sessionQueueSize

	m.backend.Logger = func(event broker.LogEvent, client *broker.Client, pkt packet.Generic, msg *packet.Message, err error) {
		if event == broker.MessagePublished {
			if len(m.tree.Match(msg.Topic)) > 0 {
				m.onSubscribe(msg)
			}
		}
	}

	engine := broker.NewEngine(m.backend)
	engine.Accept(m.server)

	return nil
}

// onSubscribe will be called if the topic is matched in topic tree.
func (m *Server) onSubscribe(msg *packet.Message) {
	// for "$hw/events/device/+/twin/+", "$hw/events/node/+/membership/get", send to twin
	// for other, send to hub
	// for "SYS/dis/upload_records", no need to base64 topic
	var target string
	var message *model.Message
	if strings.HasPrefix(msg.Topic, "$hw/events/device") || strings.HasPrefix(msg.Topic, "$hw/events/node") {
		target = modules.TwinGroup
		resource := base64.URLEncoding.EncodeToString([]byte(msg.Topic))
		// routing key will be $hw.<project_id>.events.user.bus.response.cluster.<cluster_id>.node.<node_id>.<base64_topic>
		message = model.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
			resource, messagepkg.OperationResponse).FillBody(string(msg.Payload))
	} else {
		target = modules.HubGroup
		message = model.NewMessage("").BuildRouter(modules.BusGroup, modules.UserGroup,
			msg.Topic, "upload").FillBody(string(msg.Payload))
	}
	klog.Info(fmt.Sprintf("Received msg from mqttserver, deliver to %s with resource %s", target, message.GetResource()))
	beehiveContext.SendToGroup(target, *message)
}

// InitInternalTopics sets internal topics to server by default.
func (m *Server) InitInternalTopics() {
	for _, v := range SubTopics {
		m.tree.Set(v, packet.Subscription{Topic: v, QOS: packet.QOS(m.qos)})
		klog.Infof("Subscribe internal topic to %s", v)
	}
	topics, err := dao.QueryAllTopics()
	if err != nil {
		klog.Errorf("list edge-hub-cli-topics failed: %v", err)
		return
	}
	if len(*topics) <= 0 {
		klog.Infof("list edge-hub-cli-topics status, no record, skip sync")
		return
	}
	for _, t := range *topics {
		m.tree.Set(t, packet.Subscription{Topic: t, QOS: packet.QOS(m.qos)})
		klog.Infof("Subscribe internal topic to %s", t)
	}
}

// SetTopic set the topic to internal mqtt broker.
func (m *Server) SetTopic(topic string) {
	m.tree.Set(topic, packet.Subscription{Topic: topic, QOS: packet.QOSAtMostOnce})
}

// RemoveTopic remove the topic from internal mqtt broker.
func (m *Server) RemoveTopic(topic string) {
	m.tree.Remove(topic, packet.Subscription{Topic: topic, QOS: packet.QOSAtMostOnce})
}

// Publish will dispatch topic msg to its subscribers directly.
func (m *Server) Publish(topic string, payload []byte) {
	client := &broker.Client{}

	msg := &packet.Message{
		Topic:   topic,
		Retain:  m.retain,
		Payload: payload,
		QOS:     packet.QOS(m.qos),
	}
	m.backend.Publish(client, msg, nil)
}
