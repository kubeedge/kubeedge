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
	"github.com/256dpi/gomqtt/broker"
	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/topic"
	"github.com/256dpi/gomqtt/transport"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/dao"
)

// Server serve as an internal mqtt broker.
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
	klog.Infof("OnSubscribe recevie msg from topic: %s", msg.Topic)
	NewMessageMux().Dispatch(msg.Topic, msg.Payload)
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
	if err := m.backend.Publish(client, msg, nil); err != nil {
		// TODO: handle error
		klog.Error(err)
	}
}
