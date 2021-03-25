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

package helper

import (
	"crypto/tls"
	"encoding/json"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
)

var (
	DeviceETPrefix            = "$hw/events/device/"
	DeviceETStateUpdateSuffix = "/state/update"
	TwinETUpdateSuffix        = "/twin/update"
	TwinETCloudSyncSuffix     = "/twin/cloud_updated"
	TwinETGetSuffix           = "/twin/get"
	TwinETGetResultSuffix     = "/twin/get/result"
)

var TwinResult v1alpha2.Device
var Wg sync.WaitGroup
var ControllerWg sync.WaitGroup
var TwinPropertyNames []string

var TokenClient Token
var ClientOpts *MQTT.ClientOptions
var Client MQTT.Client

//Token interface to validate the MQTT connection.
type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

//DeviceStateUpdate is the structure used in updating the device state
type DeviceStateUpdate struct {
	State string `json:"state,omitempty"`
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value,omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp,omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//DeviceTwinUpdate the struct of device twin update
//type DeviceTwinUpdate struct {
//	BaseMessage
//	Twin map[string]*MsgTwin `json:"twin"`
//}

//DeviceTwinResult device get result
//type DeviceTwinResult struct {
//	BaseMessage
//	Twin map[string]*MsgTwin `json:"twin"`
//}

// HubclientInit create mqtt client config
func HubClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	opts.SetTLSConfig(tlsConfig)
	return opts
}

//MqttConnect function felicitates the MQTT connection
func MqttConnect(mqttMode int, mqttInternalServer, mqttServer string) {
	// Initiate the MQTT connection
	if mqttMode == 0 {
		ClientOpts = HubClientInit(mqttInternalServer, "eventbus", "", "")
	} else if mqttMode == 1 {
		ClientOpts = HubClientInit(mqttServer, "eventbus", "", "")
	}
	Client = MQTT.NewClient(ClientOpts)
	if TokenClient = Client.Connect(); TokenClient.Wait() && TokenClient.Error() != nil {
		klog.Errorf("client.Connect() Error is %s", TokenClient.Error())
	}
}

//ChangeTwinValue sends the updated twin value to the edge through the MQTT broker
func ChangeTwinValue(updateMessage []v1alpha2.Twin, deviceID string) {
	s := strings.Split(deviceID, "/")
	deviceInfo := v1alpha2.Device{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s[0],
			Name:      s[1],
		},
		Status: v1alpha2.DeviceStatus{
			Twins: updateMessage,
		},
	}
	// 使用K8s CRD结构体
	twinUpdateBody, err := json.Marshal(deviceInfo)
	if err != nil {
		klog.Errorf("Error in marshalling: %s", err)
	}
	deviceTwinUpdate := DeviceETPrefix + deviceID + TwinETUpdateSuffix
	TokenClient = Client.Publish(deviceTwinUpdate, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		klog.Errorf("client.publish() Error in device twin update is %s", TokenClient.Error())
	}
}

//SyncToCloud function syncs the updated device twin information to the cloud
// TODO: 感觉不需要这个，ChangeTwinValue这个函数也会报上消息到云端的
func SyncToCloud(updateMessage []v1alpha2.Twin, deviceID string) {
	s := strings.Split(deviceID, "/")
	deviceInfo := v1alpha2.Device{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s[0],
			Name:      s[1],
		},
		Status: v1alpha2.DeviceStatus{
			Twins: updateMessage,
		},
	}
	deviceTwinResultUpdate := DeviceETPrefix + deviceID + TwinETCloudSyncSuffix
	twinUpdateBody, err := json.Marshal(deviceInfo)
	if err != nil {
		klog.Errorf("Error in marshalling: %s", err)
	}
	TokenClient = Client.Publish(deviceTwinResultUpdate, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		klog.Errorf("client.publish() Error in device twin update is: %s", TokenClient.Error())
	}
}

//GetTwin function is used to get the device twin details from the edge
func GetTwin(updateMessage []v1alpha2.Twin, deviceID string) {
	s := strings.Split(deviceID, "/")
	deviceInfo := v1alpha2.Device{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s[0],
			Name:      s[1],
		},
		Status: v1alpha2.DeviceStatus{
			Twins: updateMessage,
		},
	}
	// 这个地方也使用K8s CRD统一结构体
	getTwin := DeviceETPrefix + deviceID + TwinETGetSuffix
	twinUpdateBody, err := json.Marshal(deviceInfo)
	if err != nil {
		klog.Errorf("Error in marshalling: %s", err)
	}
	TokenClient = Client.Publish(getTwin, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		klog.Errorf("client.publish() Error in device twin get  is: %s ", TokenClient.Error())
	}
}

//subscribe function subscribes  the device twin information through the MQTT broker
func TwinSubscribe(deviceID string) {
	getTwinResult := DeviceETPrefix + deviceID + TwinETGetResultSuffix
	TokenClient = Client.Subscribe(getTwinResult, 0, OnTwinMessageReceived)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		klog.Errorf("subscribe() Error in device twin result get  is: %s", TokenClient.Error())
	}
	for {
		time.Sleep(1 * time.Second)
		if TwinResult.Status.Twins != nil {
			for _, twin := range TwinResult.Status.Twins {
				TwinPropertyNames = append(TwinPropertyNames, twin.PropertyName)
			}
			Wg.Done()
			break
		}
	}
}

// OnTwinMessageReceived callback function which is called when message is received
func OnTwinMessageReceived(client MQTT.Client, message MQTT.Message) {
	err := json.Unmarshal(message.Payload(), &TwinResult)
	if err != nil {
		klog.Errorf("Error in unmarshalling:  %s", err)
	}
}

//CreateActualUpdateMessage function is used to create the device twin update message
// 这个入参是propertyname和实际值的映射map
func CreateActualUpdateMessage(updatedTwinPropertyNames map[string]string) []v1alpha2.Twin {
	deviceTwinUpdateMessage := make([]v1alpha2.Twin, 0)

	for _, propertyName := range TwinPropertyNames {
		if actualValue, ok := updatedTwinPropertyNames[propertyName]; ok {
			updatedTwin := v1alpha2.Twin{
				PropertyName: propertyName,
				Reported: v1alpha2.TwinProperty{
					Value:    actualValue,
					Metadata: make(map[string]string),
				},
			}
			updatedTwin.Reported.Metadata["type"] = "updated"
			//deviceTwinUpdateMessage.Twin[propertyName] = &MsgTwin{}
			//deviceTwinUpdateMessage.Twin[propertyName].Actual = &TwinValue{Value: &actualValue}
			//deviceTwinUpdateMessage.Twin[propertyName].Metadata = &TypeMetadata{Type: "Updated"}
			deviceTwinUpdateMessage = append(deviceTwinUpdateMessage, updatedTwin)
		}
	}
	return deviceTwinUpdateMessage
}
