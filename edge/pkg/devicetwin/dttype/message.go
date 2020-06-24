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

package dttype

import (
	"github.com/kubeedge/beehive/pkg/core/model"
)

//MsgSubPub the struct of message for subscribe or publish
type MsgSubPub struct {
	Action  string `json:"action"`
	Payload string `json:"payload"`
	Qos     int    `json:"qos"`
}

//DTMessage the struct of message for commutinating between cloud and edge
type DTMessage struct {
	Msg      *model.Message
	Identity string
	Action   string
	Type     string
}

//GetDetailNode the info existed in req body
type GetDetailNode struct {
	EventType string `json:"event_type,omitempty"`
	EventID   string `json:"event_id,omitempty"`
	GroupID   string `json:"group_id,omitempty"`
	Operation string `json:"operation,omitempty"`
	TimeStamp int64  `json:"timestamp,omitempty"`
}

//BuildDTMessage build devicetwin message
func BuildDTMessage(identity string, action string, actionType string, msg *model.Message) *DTMessage {
	return &DTMessage{
		Msg:      msg,
		Identity: identity,
		Action:   action,
		Type:     actionType}
}
