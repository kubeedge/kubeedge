/*
Copyright 2020 The KubeEdge Authors.

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

package mappercommon

import (
	"crypto/tls"
	"encoding/json"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
)

// Joint the topic like topic := fmt.Sprintf(TopicTwinUpdateDelta, deviceID)
const (
	TopicTwinUpdateDelta = "$hw/events/device/%s/twin/update/delta"
	TopicTwinUpdate      = "$hw/events/device/%s/twin/update"
	TopicStateUpdate     = "$hw/events/device/%s/state/update" // TODO: 这个设备状态需要删除，因为cloud侧没有任何处理
	TopicDataUpdate      = "$ke/events/device/%s/data/update"  //todo: 这个消息没有处理啊？？？？？
)

// MqttClient is parameters for Mqtt client.
type MqttClient struct {
	Qos        byte
	Retained   bool
	IP         string
	User       string
	Passwd     string
	Cert       string
	PrivateKey string
	Client     mqtt.Client
}

// newTLSConfig new TLS configuration.
// Only one side check. Mqtt broker check the cert from client.
func newTLSConfig(certfile string, privateKey string) (*tls.Config, error) {
	// Import client certificate/key pair
	cert, err := tls.LoadX509KeyPair(certfile, privateKey)
	if err != nil {
		return nil, err
	}

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}, nil
}

// Connect connect to the Mqtt server.
func (mc *MqttClient) Connect() error {
	opts := mqtt.NewClientOptions().AddBroker(mc.IP).SetClientID("").SetCleanSession(true)
	if mc.Cert != "" {
		tlsConfig, err := newTLSConfig(mc.Cert, mc.PrivateKey)
		if err != nil {
			return err
		}
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

// Publish publish Mqtt message.
func (mc *MqttClient) Publish(topic string, payload interface{}) error {
	if tc := mc.Client.Publish(topic, mc.Qos, mc.Retained, payload); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}
	return nil
}

// Subscribe subsribe a Mqtt topic.
func (mc *MqttClient) Subscribe(topic string, onMessage mqtt.MessageHandler) error {
	if tc := mc.Client.Subscribe(topic, mc.Qos, onMessage); tc.Wait() && tc.Error() != nil {
		return tc.Error()
	}
	return nil
}

// getTimestamp get current timestamp.
func getTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}

// CreateMessageTwinUpdate create twin update message.
func CreateMessageTwinUpdate(name string, valueType string, value string) (msg []byte, err error) {
	device := v1alpha2.Device{}
	device.Status.Twins = make([]v1alpha2.Twin, 1)
	var twin v1alpha2.Twin

	twin.PropertyName = name
	twin.Reported = v1alpha2.TwinProperty{
		Value: value,
	}
	twin.Reported.Metadata = make(map[string]string)
	twin.Reported.Metadata["type"] = valueType
	device.Status.Twins[0] = twin

	msg, err = json.Marshal(device)
	return
}

// CreateMessageData create data message.

// data 是干嘛的topic是这样的$ke/events/device/%s/data/update
func CreateMessageData(name string, valueType string, value string) (msg []byte, err error) {
	//var dataMsg1 DeviceData
	var dataMsg v1alpha2.DeviceData

	dataMsg.DataProperties = make([]v1alpha2.DataProperty, 1)

	dataMsg.DataProperties[0].PropertyName = name
	dataMsg.DataProperties[0].Metadata = make(map[string]string)
	dataMsg.DataProperties[0].Metadata["type"] = valueType
	dataMsg.DataProperties[0].Metadata["timestamp"] = strconv.FormatInt(getTimestamp(), 10)

	// TODO：value 如何保存到结构体中？？？？

	//dataMsg.BaseMessage.Timestamp = getTimestamp()
	//dataMsg.Data = map[string]*DataValue{}
	//dataMsg.Data[name] = &DataValue{}
	//dataMsg.Data[name].Value = value
	//dataMsg.Data[name].Metadata.Type = valueType
	//dataMsg.Data[name].Metadata.Timestamp = getTimestamp()

	msg, err = json.Marshal(dataMsg)
	return
}

// CreateMessageState create device status message.
func CreateMessageState(state string) (msg []byte, err error) {
	var stateMsg DeviceUpdate

	stateMsg.BaseMessage.Timestamp = getTimestamp()
	stateMsg.State = state

	msg, err = json.Marshal(stateMsg)
	return
}
