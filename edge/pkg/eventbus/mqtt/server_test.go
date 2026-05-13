/*
Copyright 2025 The KubeEdge Authors.

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
	"errors"
	"reflect"
	"testing"

	"github.com/256dpi/gomqtt/broker"
	"github.com/256dpi/gomqtt/packet"
	"github.com/256dpi/gomqtt/topic"
	"github.com/256dpi/gomqtt/transport"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/dao"
)

const (
	testSessionQueueSize = 200
	testMqttURL          = "tcp://localhost:1883"
	testQoS              = 1
	testTopic            = "test/topic"
	testPayload          = "test payload"
	defaultUploadTopic   = "$hw/events/upload/#"
	defaultSysTopic      = "SYS/dis/upload_records"
	customTopic1         = "custom/topic1"
	customTopic2         = "custom/topic2"
)

func TestNewMqttServer(t *testing.T) {
	retain := true

	server := NewMqttServer(testSessionQueueSize, testMqttURL, retain, testQoS)

	assert.Equal(t, testSessionQueueSize, server.sessionQueueSize)
	assert.Equal(t, testMqttURL, server.url)
	assert.Equal(t, retain, server.retain)
	assert.Equal(t, testQoS, server.qos)
	assert.NotNil(t, server.tree)
}

func TestServerRun(t *testing.T) {
	server := &Server{
		url: testMqttURL,
	}

	mockTransportServer := &struct {
		transport.Server
	}{}

	patchLaunch := gomonkey.ApplyFuncSeq(transport.Launch, []gomonkey.OutputCell{
		{Values: gomonkey.Params{mockTransportServer, nil}},
	})
	defer patchLaunch.Reset()

	mockBackend := broker.NewMemoryBackend()
	patchNewMemoryBackend := gomonkey.ApplyFunc(broker.NewMemoryBackend, func() *broker.MemoryBackend {
		return mockBackend
	})
	defer patchNewMemoryBackend.Reset()

	mockEngine := &broker.Engine{}
	patchNewEngine := gomonkey.ApplyFunc(broker.NewEngine, func(backend broker.Backend) *broker.Engine {
		return mockEngine
	})
	defer patchNewEngine.Reset()

	patchAccept := gomonkey.ApplyMethod(reflect.TypeOf(mockEngine), "Accept",
		func(_ *broker.Engine, _ transport.Server) {
		})
	defer patchAccept.Reset()

	err := server.Run()

	assert.NoError(t, err)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.backend)
}

func TestServerRunError(t *testing.T) {
	server := &Server{
		url: testMqttURL,
	}

	expectedErr := errors.New("launch error")

	patchLaunch := gomonkey.ApplyFuncSeq(transport.Launch, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, expectedErr}},
	})
	defer patchLaunch.Reset()

	err := server.Run()

	assert.Error(t, err)
}

func TestInitInternalTopics(t *testing.T) {
	server := &Server{
		tree: topic.NewTree(),
		qos:  testQoS,
	}

	customTopics := []string{customTopic1, customTopic2}
	patchQueryAllTopics := gomonkey.ApplyFunc(dao.QueryAllTopics, func() (*[]string, error) {
		return &customTopics, nil
	})
	defer patchQueryAllTopics.Reset()

	server.InitInternalTopics()

	assert.NotEmpty(t, server.tree.Match(defaultUploadTopic), "Default topic should be in the tree")
	assert.NotEmpty(t, server.tree.Match(defaultSysTopic), "Default topic should be in the tree")

	for _, topicStr := range customTopics {
		assert.NotEmpty(t, server.tree.Match(topicStr), "Custom topic %s should be in the tree", topicStr)
	}
}

func TestInitInternalTopicsDbError(t *testing.T) {
	server := &Server{
		tree: topic.NewTree(),
		qos:  testQoS,
	}

	patchQueryAllTopics := gomonkey.ApplyFunc(dao.QueryAllTopics, func() (*[]string, error) {
		return nil, errors.New("database error")
	})
	defer patchQueryAllTopics.Reset()

	server.InitInternalTopics()

	assert.NotEmpty(t, server.tree.Match(defaultUploadTopic), "Default topic should be in the tree despite DB error")
	assert.NotEmpty(t, server.tree.Match(defaultSysTopic), "Default topic should be in the tree despite DB error")
}

func TestInitInternalTopicsEmptyList(t *testing.T) {
	server := &Server{
		tree: topic.NewTree(),
		qos:  testQoS,
	}

	emptyList := []string{}
	patchQueryAllTopics := gomonkey.ApplyFunc(dao.QueryAllTopics, func() (*[]string, error) {
		return &emptyList, nil
	})
	defer patchQueryAllTopics.Reset()

	server.InitInternalTopics()

	assert.NotEmpty(t, server.tree.Match(defaultUploadTopic), "Default topic should be in the tree")
	assert.NotEmpty(t, server.tree.Match(defaultSysTopic), "Default topic should be in the tree")

	assert.Empty(t, server.tree.Match(customTopic1), "Custom topics should not be added when list is empty")
}

func TestSetAndRemoveTopic(t *testing.T) {
	server := &Server{
		tree: topic.NewTree(),
	}

	matches := server.tree.Match(testTopic)
	assert.Empty(t, matches, "Topic should not be in the tree before SetTopic")

	server.SetTopic(testTopic)

	matches = server.tree.Match(testTopic)
	assert.NotEmpty(t, matches, "Topic should be in the tree after SetTopic")

	server.RemoveTopic(testTopic)

	matches = server.tree.Match(testTopic)
	assert.Empty(t, matches, "Topic should not be in the tree after RemoveTopic")
}

func TestPublish(t *testing.T) {
	mockBackend := broker.NewMemoryBackend()

	server := &Server{
		tree:    topic.NewTree(),
		retain:  true,
		qos:     testQoS,
		backend: mockBackend,
	}

	publishCalled := false
	var capturedClient *broker.Client
	var capturedMsg *packet.Message

	patchPublish := gomonkey.ApplyMethod(reflect.TypeOf(mockBackend), "Publish",
		func(_ *broker.MemoryBackend, client *broker.Client, msg *packet.Message, _ broker.Ack) error {
			publishCalled = true
			capturedClient = client
			capturedMsg = msg
			return nil
		})
	defer patchPublish.Reset()

	server.Publish(testTopic, []byte(testPayload))

	assert.True(t, publishCalled, "Backend Publish should be called")
	assert.NotNil(t, capturedClient, "Client should be provided")
	assert.Equal(t, testTopic, capturedMsg.Topic, "Topic should match")
	assert.Equal(t, []byte(testPayload), capturedMsg.Payload, "Payload should match")
	assert.Equal(t, server.retain, capturedMsg.Retain, "Retain flag should match")
	assert.Equal(t, packet.QOS(server.qos), capturedMsg.QOS, "QOS should match")
}
