package mappercommon

import (
	"crypto/tls"
	"encoding/json"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Joint the topic like topic := fmt.Sprintf(TopicTwinUpdateDelta, deviceID)
const (
	TopicTwinUpdateDelta = "$hw/events/device/%s/twin/update/delta"
	TopicTwinUpdate      = "$hw/events/device/%s/twin/update"
	TopicDataUpdate      = "$ke/events/device/%s/data/update"
	TopicStateUpdate     = "$hw/events/device/%s/state/update"
)

type MqttClient struct {
	Qos      byte
	Retained bool
	IP       string
	User     string
	Passwd   string
	Cert     string
	Client   mqtt.Client
}

func (mc *MqttClient) Connect() error {
	opts := mqtt.NewClientOptions().AddBroker(mc.IP).SetClientID("").SetCleanSession(true)
	if mc.Cert != "" {
		tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
		opts.SetTLSConfig(tlsConfig)
	} else {
		opts.SetUsername(mc.User)
		opts.SetPassword(mc.Passwd)
	}

	mc.Client = mqtt.NewClient(opts)
	// The token is used to indicate when actions have completed.
	if tc := mc.Client.Connect(); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}

	mc.Qos = 0          // At most 1 time
	mc.Retained = false // Not retained
	return nil
}

func (mc *MqttClient) GetConnection() error {
	opts := mqtt.NewClientOptions().AddBroker(mc.IP).SetClientID("").SetCleanSession(true)
	if mc.Cert != "" {
		tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
		opts.SetTLSConfig(tlsConfig)
	} else {
		opts.SetUsername(mc.User)
		opts.SetPassword(mc.Passwd)
	}

	client := mqtt.NewClient(opts)
	// The token is used to indicate when actions have completed.
	if tc := client.Connect(); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}

	return nil
}
func (mc *MqttClient) Publish(topic string, payload interface{}) error {
	if tc := mc.Client.Publish(topic, mc.Qos, mc.Retained, payload); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}
	return nil
}

func (mc *MqttClient) Subscribe(topic string, onMessage mqtt.MessageHandler) error {
	if tc := mc.Client.Subscribe(topic, mc.Qos, onMessage); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}
	return nil
}

func getTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}

func CreateMessageTwinUpdate(name string, valueType string, value string) []byte {
	var updateMsg DeviceTwinUpdate
	updateMsg.BaseMessage.Timestamp = getTimestamp()
	updateMsg.Twin = map[string]*MsgTwin{}
	updateMsg.Twin[name] = &MsgTwin{}
	updateMsg.Twin[name].Actual = &TwinValue{Value: &value}
	updateMsg.Twin[name].Metadata = &TypeMetadata{Type: valueType}

	msg, err := json.Marshal(updateMsg)
	if err != nil {
		return make([]byte, 0)
	}
	return msg
}

func CreateMessageData(name string, valueType string, value string) []byte {
	var dataMsg DeviceData
	dataMsg.Data = map[string]*DataValue{}
	dataMsg.Data[name] = &DataValue{}
	dataMsg.Data[name].Value = value
	dataMsg.Data[name].Metadata.Type = valueType
	dataMsg.Data[name].Metadata.Timestamp = getTimestamp()

	msg, err := json.Marshal(dataMsg)
	if err != nil {
		return make([]byte, 0)
	}
	return msg
}

func CreateMessageState(state string) []byte {
	var stateMsg DeviceUpdate
	stateMsg.State = state

	msg, err := json.Marshal(stateMsg)
	if err != nil {
		return make([]byte, 0)
	}
	return msg
}
