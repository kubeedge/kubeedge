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
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"
)

var (
	DeviceETPrefix            = "$hw/events/device/"
	DeviceETStateUpdateSuffix = "/state/update"
	TwinETUpdateSuffix        = "/twin/update"
	TwinETCloudSyncSuffix     = "/twin/cloud_updated"
	TwinETGetSuffix           = "/twin/get"
	TwinETGetResultSuffix     = "/twin/get/result"
)

var TwinResult DeviceTwinResult
var Wg sync.WaitGroup
var ControllerWg sync.WaitGroup
var TwinAttributes []string

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
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

//DeviceTwinResult device get result
type DeviceTwinResult struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

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
func ChangeTwinValue(updateMessage DeviceTwinUpdate, deviceID string) {
	twinUpdateBody, err := json.Marshal(updateMessage)
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
func SyncToCloud(updateMessage DeviceTwinUpdate, deviceID string) {
	deviceTwinResultUpdate := DeviceETPrefix + deviceID + TwinETCloudSyncSuffix
	twinUpdateBody, err := json.Marshal(updateMessage)
	if err != nil {
		klog.Errorf("Error in marshalling: %s", err)
	}
	TokenClient = Client.Publish(deviceTwinResultUpdate, 0, false, twinUpdateBody)
	if TokenClient.Wait() && TokenClient.Error() != nil {
		klog.Errorf("client.publish() Error in device twin update is: %s", TokenClient.Error())
	}
}

//GetTwin function is used to get the device twin details from the edge
func GetTwin(updateMessage DeviceTwinUpdate, deviceID string) {
	getTwin := DeviceETPrefix + deviceID + TwinETGetSuffix
	twinUpdateBody, err := json.Marshal(updateMessage)
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
		if TwinResult.Twin != nil {
			for k := range TwinResult.Twin {
				TwinAttributes = append(TwinAttributes, k)
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
func CreateActualUpdateMessage(updatedTwinAttributes map[string]string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	deviceTwinUpdateMessage.Twin = map[string]*MsgTwin{}
	for _, twinAttribute := range TwinAttributes {
		if actualValue, ok := updatedTwinAttributes[twinAttribute]; ok {
			deviceTwinUpdateMessage.Twin[twinAttribute] = &MsgTwin{}
			deviceTwinUpdateMessage.Twin[twinAttribute].Actual = &TwinValue{Value: &actualValue}
			deviceTwinUpdateMessage.Twin[twinAttribute].Metadata = &TypeMetadata{Type: "Updated"}
		}
	}
	return deviceTwinUpdateMessage
}
