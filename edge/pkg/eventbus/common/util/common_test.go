package util

import (
	"testing"

	"github.com/bouk/monkey"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"

	common "github.com/kubeedge/kubeedge/edge/pkg/edgehub/common"
)

type fakeMqttClient struct{}

func (f *fakeMqttClient) IsConnected() bool {
	return true
}

func (f *fakeMqttClient) Connect() MQTT.Token {

	return nil
}

func (f *fakeMqttClient) Disconnect(quiesce uint) {
}

func (f *fakeMqttClient) Publish(topic string, qos byte, retained bool, payload interface{}) MQTT.Token {
	return nil
}

func (f *fakeMqttClient) Subscribe(topic string, qos byte, callback MQTT.MessageHandler) MQTT.Token {
	return nil
}

func (f *fakeMqttClient) SubscribeMultiple(filters map[string]byte, callback MQTT.MessageHandler) MQTT.Token {
	return nil
}

func (f *fakeMqttClient) Unsubscribe(topics ...string) MQTT.Token {
	return nil
}

func (f *fakeMqttClient) AddRoute(topic string, callback MQTT.MessageHandler) {
}

func (f *fakeMqttClient) OptionsReader() MQTT.ClientOptionsReader {
	return MQTT.ClientOptionsReader{}
}

func TestPathExist(t *testing.T) {
	fakePath := "/"
	result := PathExist(fakePath)
	common.AssertTrue(t, result, "result is not true")
}

func TestCheckKeyExist(t *testing.T) {
	fakeKey := []string{"key1"}
	fakeInfo := map[string]interface{}{
		"key1": "value1",
	}
	result := CheckKeyExist(fakeKey, fakeInfo)
	assert.Nil(t, result)
}

func TestCheckKeyFailed(t *testing.T) {
	fakeKey := []string{"key1"}
	fakeInfo := map[string]interface{}{
		"key2": "value1",
	}
	result := CheckKeyExist(fakeKey, fakeInfo)
	assert.NotNil(t, result)
}

func TestHubclientInit(t *testing.T) {
	HubclientInit("tcp://127.0.0.1:1883", "123", "123", "123")
}

func TestLoopConnect(t *testing.T) {
	//connectChan := make(chan bool, 1)
	fakeClient := &fakeMqttClient{}
	result := true
	var err error
	monkey.Patch(CheckClientToken, func(token MQTT.Token) (bool, error) {
		return result, err
	})
	defer monkey.Unpatch(CheckClientToken)
	LoopConnect("1234", fakeClient)
}
