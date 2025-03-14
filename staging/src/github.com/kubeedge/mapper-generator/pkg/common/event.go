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

package common

import (
	"encoding/json"
	"time"
)

// Joint the topic like topic := fmt.Sprintf(TopicTwinUpdateDelta, deviceID)
const (
	TopicTwinUpdateDelta = "$hw/events/device/%s/twin/update/delta"
	TopicTwinUpdate      = "$hw/events/device/%s/twin/update"
	TopicStateUpdate     = "$hw/events/device/%s/state/update"
	TopicDataUpdate      = "$ke/events/device/%s/data/update"
)

// getTimestamp get current timestamp.
func getTimestamp() int64 {
	return time.Now().UnixNano() / 1e6
}

// CreateMessageTwinUpdate create twin update message.
func CreateMessageTwinUpdate(name string, valueType string, value string) (msg []byte, err error) {
	var updateMsg DeviceTwinUpdate

	updateMsg.BaseMessage.Timestamp = getTimestamp()
	updateMsg.Twin = map[string]*MsgTwin{}
	updateMsg.Twin[name] = &MsgTwin{}
	updateMsg.Twin[name].Actual = &TwinValue{Value: &value}
	updateMsg.Twin[name].Metadata = &TypeMetadata{Type: valueType}

	msg, err = json.Marshal(updateMsg)
	return
}

// CreateMessageData create data message.
func CreateMessageData(name string, valueType string, value string) (msg []byte, err error) {
	var dataMsg DeviceData

	dataMsg.BaseMessage.Timestamp = getTimestamp()
	dataMsg.Data = map[string]*DataValue{}
	dataMsg.Data[name] = &DataValue{}
	dataMsg.Data[name].Value = value
	dataMsg.Data[name].Metadata.Type = valueType
	dataMsg.Data[name].Metadata.Timestamp = getTimestamp()

	msg, err = json.Marshal(dataMsg)
	return
}
