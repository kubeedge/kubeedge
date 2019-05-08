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

package model

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// modelMessage returns a new model.Message
func modelMessage(ID string, PID string, Timestamp int64, Source string, Group string, Operation string, Resource string) model.Message {
	return model.Message{
		Header: model.MessageHeader{
			ID:        ID,
			ParentID:  PID,
			Timestamp: Timestamp,
		},
		Router: model.MessageRoute{
			Source:    Source,
			Group:     Group,
			Operation: Operation,
			Resource:  Resource,
		},
	}
}

// modelEvent returns a new Event
func modelEvent(ID string, PID string, Timestamp int64, Source string, Group string, Operation string, Resource string, Content interface{}) Event {
	return Event{
		Group:  Group,
		Source: Source,
		UserGroup: UserGroupInfo{
			Resource:  Resource,
			Operation: Operation,
		},
		ID:        ID,
		ParentID:  PID,
		Timestamp: Timestamp,
		Content:   Content,
	}
}

// TestEventToMessage is function to test EventToMessage
func TestEventToMessage(t *testing.T) {
	msg := modelMessage("ID1", "PID1", time.Now().UnixNano()/1e6, "Source1", "Group1", "Operation1", "Resource1")
	event := modelEvent("ID1", "PID1", time.Now().UnixNano()/1e6, "Source1", "Group1", "Operation1", "Resource1", nil)
	tests := []struct {
		name  string
		event *Event
		msg   model.Message
	}{
		{
			name:  "TestEventToMessage(): converts an event to a model message",
			event: &event,
			msg:   msg,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if msg := EventToMessage(test.event); !reflect.DeepEqual(msg, test.msg) {
				t.Errorf("Model.TestEventToMessage() case failed: got = %v, Want = %v", msg, test.msg)
			}
		})
	}
}

// TestMessageToEvent is function to test MessageToEvent
func TestMessageToEvent(t *testing.T) {
	msg := modelMessage("ID1", "PID1", time.Now().UnixNano()/1e6, "Source1", "Group1", "Operation1", "Resource1")
	event := modelEvent("ID1", "PID1", time.Now().UnixNano()/1e6, "Source1", "Group1", "Operation1", "Resource1", nil)
	tests := []struct {
		name  string
		msg   *model.Message
		event Event
	}{
		{
			name:  "TestMessageToEvent(): converts a model message to an event",
			msg:   &msg,
			event: event,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if event := MessageToEvent(test.msg); !reflect.DeepEqual(event, test.event) {
				t.Errorf("Model.TestMessageToEvent() case failed: got = %v, Want = %v", event, test.event)
			}
		})
	}
}

// TestNewResource is function to test NewResource
func TestNewResource(t *testing.T) {
	tests := []struct {
		name    string
		resType string
		resID   string
		info    *HubInfo
		str     string
	}{
		{
			name:    "TestNewResource(): Case 1: resID is empty",
			resType: ResNode,
			resID:   "",
			info:    &HubInfo{ProjectID: "Project1", NodeID: "Node1"},
			str:     "node/Node1/node",
		},
		{
			name:    "TestNewResource(): Case 2: resID is not empty",
			resType: ResNode,
			resID:   "res1",
			info:    &HubInfo{ProjectID: "Project1", NodeID: "Node1"},
			str:     "node/Node1/node/res1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if str := NewResource(test.resType, test.resID, test.info); !reflect.DeepEqual(str, test.str) {
				t.Errorf("Model.TestNewResource() case failed: got = %v, Want = %v", str, test.str)
			}
		})
	}
}

// TestIsNodeStopped is function to test IsNodeStopped
func TestIsNodeStopped(t *testing.T) {
	body := map[string]interface{}{
		"event_type": OpConnect,
		"timestamp":  time.Now().Unix(),
	}
	content, _ := json.Marshal(body)
	bodyAction := map[string]interface{}{
		"event_type": OpConnect,
		"timestamp":  time.Now().Unix(),
		"action":     "stop",
	}
	eventResource := modelEvent("", "", 0, "", "", "", "Resource1", nil)
	eventOpDelete := modelEvent("", "", 0, "", "", OpDelete, "node/Node1", nil)
	eventNoContent := modelEvent("", "", 0, "", "", "", "node/Node1", nil)
	eventContent := modelEvent("", "", 0, "", "", OpUpdate, "node/Node1", content)
	eventNoAction := modelEvent("", "", 0, "", "", OpUpdate, "node/Node1", body)
	eventActionStop := modelEvent("", "", 0, "", "", OpUpdate, "node/Node1", bodyAction)
	tests := []struct {
		name      string
		event     *Event
		errorWant bool
	}{
		{
			name:      "TestIsNodeStopped(): Case 1: UserGroup.Resource!=ResNode",
			event:     &eventResource,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 2: UserGroup.Operation=OpDelete",
			event:     &eventOpDelete,
			errorWant: true,
		},
		{
			name:      "TestIsNodeStopped(): Case 3: event.Content=nil",
			event:     &eventNoContent,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 4: event.Content!=nil",
			event:     &eventContent,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 5: event.Content[action] is nil",
			event:     &eventNoAction,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 6: event.Content[action]=stop",
			event:     &eventActionStop,
			errorWant: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorWant := test.event.IsNodeStopped(); !reflect.DeepEqual(errorWant, test.errorWant) {
				t.Errorf("Model.TestIsNodeStopped() case failed: got = %v, Want = %v", errorWant, test.errorWant)
			}
		})
	}
}

// TestIsFromEdge is function to test IsFromEdge
func TestIsFromEdge(t *testing.T) {
	event := modelEvent("", "", 0, "Source1", "", "", "", nil)
	tests := []struct {
		name      string
		event     *Event
		errorWant bool
	}{
		{
			name:      "TestIsFromEdge(): when the event is sent from edge",
			event:     &event,
			errorWant: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorWant := test.event.IsFromEdge(); !reflect.DeepEqual(errorWant, test.errorWant) {
				t.Errorf("Model.TestIsFromEdge() case failed: got = %v, Want = %v", errorWant, test.errorWant)
			}
		})
	}
}

// TestIsToEdge is function to test IsToEdge
func TestIsToEdge(t *testing.T) {
	eventSource := modelEvent("", "", 0, "Source1", "", "", "", nil)
	eventResource := modelEvent("", "", 0, SrcManager, "", "", "node/Node1/node/res1", nil)
	eventOperation := modelEvent("", "", 0, SrcManager, "", "get", "membership", nil)
	tests := []struct {
		name      string
		event     *Event
		errorWant bool
	}{
		{
			name:      "TestIsToEdge(): Case 1: when the event is sent from edge",
			event:     &eventResource,
			errorWant: true,
		},
		{
			name:      "TestIsToEdge(): Case 2: event.Source!=SrcManager",
			event:     &eventSource,
			errorWant: true,
		},
		{
			name:      "TestIsToEdge(): Case 3: when the event equals to resOpMap",
			event:     &eventOperation,
			errorWant: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorWant := test.event.IsToEdge(); !reflect.DeepEqual(errorWant, test.errorWant) {
				t.Errorf("Model.TestIsToEdge() case failed: got = %v, Want = %v", errorWant, test.errorWant)
			}
		})
	}
}
