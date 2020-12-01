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
func modelMessage(ID string, PID string, Timestamp int64, Source string, Group string, Operation string, Resource string, Content interface{}) model.Message {
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
		Content: Content,
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

	msgResource := modelMessage("", "", 0, "", "", "", "Resource1", nil)
	msgOpDelete := modelMessage("", "", 0, "", "", model.DeleteOperation, "node/edge-node/default/node/Node1", nil)
	msgNoContent := modelMessage("", "", 0, "", "", "", "node/edge-node/default/node/Node1", nil)
	msgContent := modelMessage("", "", 0, "", "", model.UpdateOperation, "node/edge-node/default/node/Node1", content)
	msgNoAction := modelMessage("", "", 0, "", "", model.UpdateOperation, "node/edge-node/default/node/Node1", body)
	tests := []struct {
		name      string
		msg       *model.Message
		errorWant bool
	}{
		{
			name:      "TestIsNodeStopped(): Case 1: UserGroup.Resource!=ResNode",
			msg:       &msgResource,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 2: UserGroup.Operation=OpDelete",
			msg:       &msgOpDelete,
			errorWant: true,
		},
		{
			name:      "TestIsNodeStopped(): Case 3: msg.Content=nil",
			msg:       &msgNoContent,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 4: msg.Content!=nil",
			msg:       &msgContent,
			errorWant: false,
		},
		{
			name:      "TestIsNodeStopped(): Case 5: msg.Content[action] is nil",
			msg:       &msgNoAction,
			errorWant: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorWant := IsNodeStopped(test.msg); !reflect.DeepEqual(errorWant, test.errorWant) {
				t.Errorf("Model.TestIsNodeStopped() case failed: got = %v, Want = %v", errorWant, test.errorWant)
			}
		})
	}
}

// TestIsFromEdge is function to test IsFromEdge
func TestIsFromEdge(t *testing.T) {
	tests := []struct {
		name      string
		msg       model.Message
		errorWant bool
	}{
		{
			name:      "TestIsFromEdge(): when the msg is sent from edge",
			msg:       modelMessage("", "", 0, "Source1", "", "", "", nil),
			errorWant: true,
		},
		{
			name:      "TestIsFromEdge(): when the msg is also sent from edge",
			msg:       modelMessage("", "", 0, "edged", "", "", "", nil),
			errorWant: true,
		},
		{
			name:      "TestIsFromEdge(): when the msg is not sent from edge",
			msg:       modelMessage("", "", 0, "edgecontroller", "", "", "", nil),
			errorWant: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorWant := IsFromEdge(&test.msg); !reflect.DeepEqual(errorWant, test.errorWant) {
				t.Errorf("Model.TestIsFromEdge() case failed: got = %v, Want = %v", errorWant, test.errorWant)
			}
		})
	}
}

// TestIsToEdge is function to test IsToEdge
func TestIsToEdge(t *testing.T) {
	msgSource := modelMessage("", "", 0, "Source1", "", "", "", nil)
	msgResource := modelMessage("", "", 0, SrcManager, "", "", "node/Node1/node/res1", nil)
	msgOperation := modelMessage("", "", 0, SrcManager, "", "get", "membership", nil)
	tests := []struct {
		name      string
		msg       *model.Message
		errorWant bool
	}{
		{
			name:      "TestIsToEdge(): Case 1: when the msg is sent from edge",
			msg:       &msgSource,
			errorWant: true,
		},
		{
			name:      "TestIsToEdge(): Case 2: msg.Source!=SrcManager",
			msg:       &msgResource,
			errorWant: true,
		},
		{
			name:      "TestIsToEdge(): Case 3: when the msg equals to resOpMap",
			msg:       &msgOperation,
			errorWant: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if errorWant := IsToEdge(test.msg); !reflect.DeepEqual(errorWant, test.errorWant) {
				t.Errorf("Model.TestIsToEdge() case failed: got = %v, Want = %v", errorWant, test.errorWant)
			}
		})
	}
}
