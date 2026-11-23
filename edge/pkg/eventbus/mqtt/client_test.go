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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/edge/pkg/eventbus/common/util"
	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/mocks"
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

var originalEventBusServiceFactory = EventBusServiceFactory

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

	assert.NoError(t, client.InitPubClient())

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

	assert.NoError(t, client.InitPubClient())

	assert.Equal(t, "existing-pub-id", client.PubClientID)
	assert.True(t, connectCalled)
}

func TestInitSubClient(t *testing.T) {
	defer func() {
		EventBusServiceFactory = originalEventBusServiceFactory
	}()

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

	// Setup EventBusServiceFactory mock using MockEventBusService
	customTopics := []string{"custom/topic1", "custom/topic2"}
	mockEventBusService := mocks.NewMockEventBusService()
	mockEventBusService.QueryAllTopicsFunc = func() (*[]string, error) {
		return &customTopics, nil
	}
	EventBusServiceFactory = func() interface {
		InsertTopics(topic string) error
		DeleteTopicsByKey(key string) error
		QueryAllTopics() (*[]string, error)
	} {
		return mockEventBusService
	}

	assert.NoError(t, client.InitSubClient())

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

// TestInitClientWithInvalidTLS checks that the init functions return an error,
// instead of dereferencing nil client options, when the MQTT TLS files cannot be loaded.
func TestInitClientWithInvalidTLS(t *testing.T) {
	origTLS := eventconfig.Config.TLS
	defer func() { eventconfig.Config.TLS = origTLS }()
	missingDir := t.TempDir()
	eventconfig.Config.TLS = &v1alpha2.EventBusTLS{
		Enable:                true,
		TLSMqttCAFile:         filepath.Join(missingDir, "ca.crt"),
		TLSMqttCertFile:       filepath.Join(missingDir, "server.crt"),
		TLSMqttPrivateKeyFile: filepath.Join(missingDir, "server.key"),
	}

	var connectCalled bool
	patchLoopConnect := gomonkey.ApplyFunc(util.LoopConnect, func(clientID string, client MQTT.Client) {
		connectCalled = true
	})
	defer patchLoopConnect.Reset()

	tests := []struct {
		name string
		init func(c *Client) error
		cli  func(c *Client) MQTT.Client
	}{
		{
			name: "sub client",
			init: func(c *Client) error { return c.InitSubClient() },
			cli:  func(c *Client) MQTT.Client { return c.SubCli },
		},
		{
			name: "pub client",
			init: func(c *Client) error { return c.InitPubClient() },
			cli:  func(c *Client) MQTT.Client { return c.PubCli },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{MQTTUrl: "tcp://localhost:1883"}

			var err error
			assert.NotPanics(t, func() { err = tt.init(client) })

			assert.Error(t, err)
			assert.Nil(t, tt.cli(client), "no client should be created")
			assert.False(t, connectCalled, "no connection should be attempted")
		})
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
		func(_ *Client) error {
			initCalled = true
			return nil
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
		func(_ *Client) error {
			initCalled = true
			return nil
		})
	defer patch.Reset()

	onSubConnectionLost(nil, errors.New("connection lost"))

	time.Sleep(50 * time.Millisecond)

	assert.True(t, initCalled, "InitSubClient should be called")
}

// logBuffer is a goroutine-safe buffer to capture klog output.
type logBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestConnectionLostLogsInitError checks that the reconnect callbacks log the
// error returned by the init functions, for example when the TLS files cannot
// be loaded any more, instead of panicking.
func TestConnectionLostLogsInitError(t *testing.T) {
	origMQTTHub := MQTTHub
	defer func() { MQTTHub = origMQTTHub }()
	MQTTHub = &Client{}

	logs := &logBuffer{}
	klog.LogToStderr(false)
	klog.SetOutput(logs)
	defer func() {
		klog.SetOutput(os.Stderr)
		klog.LogToStderr(true)
	}()

	hubType := reflect.TypeOf(MQTTHub)
	patchPub := gomonkey.ApplyMethod(hubType, "InitPubClient",
		func(_ *Client) error { return errors.New("pub init failed") })
	defer patchPub.Reset()
	patchSub := gomonkey.ApplyMethod(hubType, "InitSubClient",
		func(_ *Client) error { return errors.New("sub init failed") })
	defer patchSub.Reset()

	tests := []struct {
		name string
		lost func(MQTT.Client, error)
		want string
	}{
		{"pub", onPubConnectionLost, "reconnect hub-client pub failed: pub init failed"},
		{"sub", onSubConnectionLost, "reconnect hub-client sub failed: sub init failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.lost(nil, errors.New("connection lost"))

			assert.Eventually(t, func() bool {
				klog.Flush()
				return strings.Contains(logs.String(), tt.want)
			}, time.Second, 10*time.Millisecond)
		})
	}
}

func TestOnSubConnect(t *testing.T) {
	defer func() {
		EventBusServiceFactory = originalEventBusServiceFactory
	}()

	mockClient := NewTestMQTTClient()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	// Setup EventBusServiceFactory mock using MockEventBusService
	customTopics := []string{"custom/topic1", "custom/topic2"}
	mockEventBusService := mocks.NewMockEventBusService()
	mockEventBusService.QueryAllTopicsFunc = func() (*[]string, error) {
		return &customTopics, nil
	}
	EventBusServiceFactory = func() interface {
		InsertTopics(topic string) error
		DeleteTopicsByKey(key string) error
		QueryAllTopics() (*[]string, error)
	} {
		return mockEventBusService
	}

	onSubConnect(mockClient)

	for _, topic := range SubTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}

	for _, topic := range customTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}
}

func TestOnSubConnectWithQueryError(t *testing.T) {
	defer func() {
		EventBusServiceFactory = originalEventBusServiceFactory
	}()

	mockClient := NewTestMQTTClient()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	// Setup EventBusServiceFactory mock using MockEventBusService with error
	mockEventBusService := mocks.NewMockEventBusService()
	mockEventBusService.QueryAllTopicsFunc = func() (*[]string, error) {
		return nil, errors.New("database error")
	}
	EventBusServiceFactory = func() interface {
		InsertTopics(topic string) error
		DeleteTopicsByKey(key string) error
		QueryAllTopics() (*[]string, error)
	} {
		return mockEventBusService
	}

	onSubConnect(mockClient)

	for _, topic := range SubTopics {
		assert.True(t, mockClient.subscribeTopics[topic], fmt.Sprintf("Topic %s should be subscribed", topic))
	}

	assert.False(t, mockClient.subscribeTopics["custom/topic1"])
}

func TestOnSubConnectWithEmptyTopics(t *testing.T) {
	defer func() {
		EventBusServiceFactory = originalEventBusServiceFactory
	}()

	mockClient := NewTestMQTTClient()

	patchCheckClientToken := gomonkey.ApplyFunc(util.CheckClientToken, func(token MQTT.Token) (bool, error) {
		return true, nil
	})
	defer patchCheckClientToken.Reset()

	// Setup EventBusServiceFactory mock using MockEventBusService with empty topics
	emptyTopics := []string{}
	mockEventBusService := mocks.NewMockEventBusService()
	mockEventBusService.QueryAllTopicsFunc = func() (*[]string, error) {
		return &emptyTopics, nil
	}
	EventBusServiceFactory = func() interface {
		InsertTopics(topic string) error
		DeleteTopicsByKey(key string) error
		QueryAllTopics() (*[]string, error)
	} {
		return mockEventBusService
	}

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
