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
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
)

type TestMessage struct {
	topic   string
	payload []byte
}

func (m *TestMessage) Duplicate() bool   { return false }
func (m *TestMessage) Qos() byte         { return 0 }
func (m *TestMessage) Retained() bool    { return false }
func (m *TestMessage) Topic() string     { return m.topic }
func (m *TestMessage) MessageID() uint16 { return 0 }
func (m *TestMessage) Payload() []byte   { return m.payload }
func (m *TestMessage) Ack()              {}

type TestToken struct {
	err error
}

func (t *TestToken) Wait() bool                     { return true }
func (t *TestToken) WaitTimeout(time.Duration) bool { return true }
func (t *TestToken) Done() <-chan struct{}          { c := make(chan struct{}); close(c); return c }
func (t *TestToken) Error() error                   { return t.err }

type TestMQTTClient struct {
	connected        bool
	subscribeTopics  map[string]bool
	subscribeError   error
	publishTopics    map[string][]byte
	publishError     error
	connectionLostCb MQTT.ConnectionLostHandler
	onConnectHandler MQTT.OnConnectHandler
}

func NewTestMQTTClient() *TestMQTTClient {
	return &TestMQTTClient{
		subscribeTopics: make(map[string]bool),
		publishTopics:   make(map[string][]byte),
	}
}

func (c *TestMQTTClient) IsConnected() bool      { return c.connected }
func (c *TestMQTTClient) IsConnectionOpen() bool { return c.connected }
func (c *TestMQTTClient) Connect() MQTT.Token {
	c.connected = true
	if c.onConnectHandler != nil {
		c.onConnectHandler(c)
	}
	return &TestToken{}
}
func (c *TestMQTTClient) Disconnect(quiesce uint) { c.connected = false }
func (c *TestMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) MQTT.Token {
	if c.publishError != nil {
		return &TestToken{err: c.publishError}
	}
	if payload != nil {
		c.publishTopics[topic] = payload.([]byte)
	} else {
		c.publishTopics[topic] = []byte{}
	}
	return &TestToken{}
}
func (c *TestMQTTClient) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) MQTT.Token {
	if c.subscribeError != nil {
		return &TestToken{err: c.subscribeError}
	}
	c.subscribeTopics[topic] = true
	return &TestToken{}
}
func (c *TestMQTTClient) SubscribeMultiple(filters map[string]byte, callback MQTT.MessageHandler) MQTT.Token {
	if c.subscribeError != nil {
		return &TestToken{err: c.subscribeError}
	}
	for topic := range filters {
		c.subscribeTopics[topic] = true
	}
	return &TestToken{}
}
func (c *TestMQTTClient) Unsubscribe(topics ...string) MQTT.Token             { return &TestToken{} }
func (c *TestMQTTClient) AddRoute(topic string, callback MQTT.MessageHandler) {}
func (c *TestMQTTClient) OptionsReader() MQTT.ClientOptionsReader             { return MQTT.ClientOptionsReader{} }

func TestInitPubClient(t *testing.T) {
	client := &Client{
		MQTTUrl:  "tcp://localhost:1883",
		Username: "user",
		Password: "password",
	}

	mockClient := NewTestMQTTClient()

	patchNewClient := gomonkey.ApplyFunc(MQTT.NewClient, func(o *MQTT.ClientOptions) MQTT.Client {
		return mockClient
	})
	defer patchNewClient.Reset()

	patchHubClientInit := gomonkey.ApplyFunc(util.HubClientInit, func(server, clientID, username, password string) *MQTT.ClientOptions {
		assert.Contains(t, clientID, "hub-client-pub-", "Client ID should have correct prefix")
		return &MQTT.ClientOptions{}
	})
	defer patchHubClientInit.Reset()

	var connectCalled bool
	patchLoopConnect := gomonkey.ApplyFunc(util.LoopConnect, func(clientID string, client MQTT.Client) {
		connectCalled = true
		assert.Contains(t, clientID, "hub-client-pub-", "ClientID in LoopConnect should have correct prefix")
		assert.Equal(t, mockClient, client)
	})
	defer patchLoopConnect.Reset()

	client.InitPubClient()

	assert.NotEmpty(t, client.PubClientID)
	assert.Contains(t, client.PubClientID, "hub-client-pub-")
	assert.True(t, connectCalled)
}

func TestInitPubClientWithExistingID(t *testing.T) {
	client := &Client{
		MQTTUrl:     "tcp://localhost:1883",
		PubClientID: "existing-pub-id",
		Username:    "user",
		Password:    "password",
	}

	mockClient := NewTestMQTTClient()

	patchNewClient := gomonkey.ApplyFunc(MQTT.NewClient, func(o *MQTT.ClientOptions) MQTT.Client {
		return mockClient
	})
	defer patchNewClient.Reset()

	patchHubClientInit := gomonkey.ApplyFunc(util.HubClientInit, func(server, clientID, username, password string) *MQTT.ClientOptions {
		assert.Equal(t, "existing-pub-id", clientID, "Client ID should remain unchanged")
		return &MQTT.ClientOptions{}
	})
	defer patchHubClientInit.Reset()

	var connectCalled bool
	patchLoopConnect := gomonkey.ApplyFunc(util.LoopConnect, func(clientID string, client MQTT.Client) {
		connectCalled = true
		assert.Equal(t, "existing-pub-id", clientID, "ClientID in LoopConnect should match existing ID")
		assert.Equal(t, mockClient, client)
	})
	defer patchLoopConnect.Reset()

	client.InitPubClient()

	assert.Equal(t, "existing-pub-id", client.PubClientID)
	assert.True(t, connectCalled)
}

func TestInitSubClient(t *testing.T) {
	client := &Client{
		MQTTUrl:  "tcp://localhost:1883",
		Username: "user",
		Password: "password",
	}

	mockClient := NewTestMQTTClient()

	patchNewClient := gomonkey.ApplyFunc(MQTT.NewClient, func(o *MQTT.ClientOptions) MQTT.Client {
		mockClient.onConnectHandler = o.OnConnect
		return mockClient
	})
	defer patchNewClient.Reset()

	patchHubClientInit := gomonkey.ApplyFunc(util.HubClientInit, func(server, clientID, username, password string) *MQTT.ClientOptions {
		assert.Contains(t, clientID, "hub-client-sub-", "Client ID should have correct prefix")
		opts := &MQTT.ClientOptions{}
		opts.OnConnect = onSubConnect
		return opts
	})
	defer patchHubClientInit.Reset()

	var connectCalled bool
	patchLoopConnect := gomonkey.ApplyFunc(util.LoopConnect, func(clientID string, client MQTT.Client) {
		connectCalled = true
		if mockClient.onConnectHandler != nil {
			mockClient.onConnectHandler(mockClient)
		}
	})
	defer patchLoopConnect.Reset()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	customTopics := []string{"custom/topic1", "custom/topic2"}
	patchQueryAllTopics := gomonkey.ApplyFunc(dbclient.NewEventBusService().QueryAllTopics, func() (*[]string, error) {
		return &customTopics, nil
	})
	defer patchQueryAllTopics.Reset()

	client.InitSubClient()

	assert.NotEmpty(t, client.SubClientID)
	assert.Contains(t, client.SubClientID, "hub-client-sub-")
	assert.True(t, connectCalled)

	for _, topic := range SubTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}

	for _, topic := range customTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}
}

func TestOnSubMessageReceived(t *testing.T) {
	message := &TestMessage{
		topic:   "SYS/dis/upload_records",
		payload: []byte("test payload"),
	}

	patchHandler := gomonkey.ApplyFunc(handleUploadTopic, func(topic string, payload []byte) {
		assert.Equal(t, "SYS/dis/upload_records", topic)
		assert.Equal(t, []byte("test payload"), payload)
	})
	defer patchHandler.Reset()

	patchSendToGroup := gomonkey.ApplyFunc(beehiveContext.SendToGroup, func(groupName string, message interface{}) {})
	defer patchSendToGroup.Reset()

	OnSubMessageReceived(nil, message)
}

func TestOnSubMessageReceivedDeviceTwin(t *testing.T) {
	message := &TestMessage{
		topic:   "$hw/events/device/test-device/twin/update",
		payload: []byte("test twin payload"),
	}

	patchHandler := gomonkey.ApplyFunc(handleDeviceTwin, func(topic string, payload []byte) {
		assert.Equal(t, "$hw/events/device/test-device/twin/update", topic)
		assert.Equal(t, []byte("test twin payload"), payload)
	})
	defer patchHandler.Reset()

	patchSendToGroup := gomonkey.ApplyFunc(beehiveContext.SendToGroup, func(groupName string, message interface{}) {})
	defer patchSendToGroup.Reset()

	OnSubMessageReceived(nil, message)
}

func TestOnPubConnectionLost(t *testing.T) {
	origMQTTHub := MQTTHub
	defer func() { MQTTHub = origMQTTHub }()

	initCalled := false
	MQTTHub = &Client{}

	patch := gomonkey.ApplyMethod(reflect.TypeOf(MQTTHub), "InitPubClient",
		func(_ *Client) {
			initCalled = true
		})
	defer patch.Reset()

	onPubConnectionLost(nil, errors.New("connection lost"))

	time.Sleep(50 * time.Millisecond)

	assert.True(t, initCalled, "InitPubClient should be called")
}

func TestOnSubConnectionLost(t *testing.T) {
	origMQTTHub := MQTTHub
	defer func() { MQTTHub = origMQTTHub }()

	initCalled := false
	MQTTHub = &Client{}

	patch := gomonkey.ApplyMethod(reflect.TypeOf(MQTTHub), "InitSubClient",
		func(_ *Client) {
			initCalled = true
		})
	defer patch.Reset()

	onSubConnectionLost(nil, errors.New("connection lost"))

	time.Sleep(50 * time.Millisecond)

	assert.True(t, initCalled, "InitSubClient should be called")
}

func TestOnSubConnect(t *testing.T) {
	mockClient := NewTestMQTTClient()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	customTopics := []string{"custom/topic1", "custom/topic2"}
	patchQueryAllTopics := gomonkey.ApplyFunc(dbclient.NewEventBusService().QueryAllTopics, func() (*[]string, error) {
		return &customTopics, nil
	})
	defer patchQueryAllTopics.Reset()

	onSubConnect(mockClient)

	for _, topic := range SubTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}

	for _, topic := range customTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}
}

func TestOnSubConnectWithQueryError(t *testing.T) {
	mockClient := NewTestMQTTClient()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	patchQueryAllTopics := gomonkey.ApplyFunc(dbclient.NewEventBusService().QueryAllTopics, func() (*[]string, error) {
		return nil, errors.New("database error")
	})
	defer patchQueryAllTopics.Reset()

	onSubConnect(mockClient)

	for _, topic := range SubTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}

	assert.False(t, mockClient.subscribeTopics["custom/topic1"])
}

func TestOnSubConnectWithEmptyTopics(t *testing.T) {
	mockClient := NewTestMQTTClient()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	emptyTopics := []string{}
	patchQueryAllTopics := gomonkey.ApplyFunc(dbclient.NewEventBusService().QueryAllTopics, func() (*[]string, error) {
		return &emptyTopics, nil
	})
	defer patchQueryAllTopics.Reset()

	onSubConnect(mockClient)

	for _, topic := range SubTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}

	assert.Equal(t, len(SubTopics), len(mockClient.subscribeTopics), "Should only have default topics")
}

func TestSubTopicsConstants(t *testing.T) {
	expectedTopics := []string{
		"$hw/events/upload/#",
		"$hw/events/device/+/+/state/update",
		"$hw/events/device/+/+/twin/+",
		"$hw/events/node/+/membership/get",
		"SYS/dis/upload_records",
		"+/user/#",
	}

	assert.Equal(t, len(expectedTopics), len(SubTopics), "SubTopics should have the expected number of entries")

	for _, expectedTopic := range expectedTopics {
		found := false
		for _, actualTopic := range SubTopics {
			if actualTopic == expectedTopic {
				found = true
				break
			}
		}
		assert.True(t, found, fmt.Sprintf("Expected topic %s should be in SubTopics", expectedTopic))
	}
}

func TestTopicFormats(t *testing.T) {
	clientID := "test-client"
	expectedConnected := fmt.Sprintf(ConnectedTopic, clientID)
	assert.Equal(t, "$hw/events/connected/test-client", expectedConnected)

	expectedDisconnected := fmt.Sprintf(DisconnectedTopic, clientID)
	assert.Equal(t, "$hw/events/disconnected/test-client", expectedDisconnected)

	groupID := "group1"
	expectedMemberGet := fmt.Sprintf(MemberGet, groupID)
	assert.Equal(t, "$hw/events/edgeGroup/group1/membership/get", expectedMemberGet)
}

func TestAccessInfoStruct(t *testing.T) {
	info := AccessInfo{
		Name:    "test-name",
		Type:    "test-type",
		Topic:   "test/topic",
		Content: []byte("test-content"),
	}

	assert.Equal(t, "test-name", info.Name)
	assert.Equal(t, "test-type", info.Type)
	assert.Equal(t, "test/topic", info.Topic)
	assert.Equal(t, []byte("test-content"), info.Content)
}
