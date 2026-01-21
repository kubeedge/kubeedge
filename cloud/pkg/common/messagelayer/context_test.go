/*
Copyright 2022 The KubeEdge Authors.

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

package messagelayer

import (
	"reflect"
	"sync"
	"testing"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
)

const (
	sendModuleName     = "testcore"
	receiveModuleName  = "testcore"
	responseModuleName = "testcore"
	routerModuleName   = "testrouter"
)

var once sync.Once

func setupTestContext() {
	once.Do(func() {
		beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
		modules := []string{receiveModuleName, routerModuleName}
		for _, module := range modules {
			add := &common.ModuleInfo{
				ModuleName: module,
				ModuleType: common.MsgCtxTypeChannel,
			}
			beehiveContext.AddModule(add)
			beehiveContext.AddModuleGroup(module, module)
		}
	})
}

// compareMessages compares two messages ignoring the UUID field
func compareMessages(a, b model.Message) bool {
	if !reflect.DeepEqual(a.Router, b.Router) {
		return false
	}

	if !reflect.DeepEqual(a.Content, b.Content) {
		return false
	}

	if a.Header.ParentID != b.Header.ParentID ||
		a.Header.Timestamp != b.Header.Timestamp ||
		a.Header.ResourceVersion != b.Header.ResourceVersion ||
		a.Header.Sync != b.Header.Sync ||
		a.Header.MessageType != b.Header.MessageType {
		return false
	}

	return true
}

func TestContextMessageLayer_Send_Receive_Response(t *testing.T) {
	setupTestContext()

	tests := []struct {
		name             string
		message          *model.Message
		sendRouterModule string
		wantErr          bool
		isRouterMessage  bool
	}{
		{
			name: "Test normal message flow",
			message: model.NewMessage("").
				BuildRouter(sendModuleName, receiveModuleName, "default/resource", model.UpdateOperation).
				FillBody("Hello Kubeedge"),
			wantErr:         false,
			isRouterMessage: false,
		},
		{
			name: "Test rule router message",
			message: model.NewMessage("").
				BuildRouter(sendModuleName, routerModuleName, "rule/testrule", model.UpdateOperation).
				FillBody("Rule Message"),
			sendRouterModule: routerModuleName,
			wantErr:          false,
			isRouterMessage:  true,
		},
		{
			name: "Test rule endpoint router message",
			message: model.NewMessage("").
				BuildRouter(sendModuleName, routerModuleName, "ruleendpoint/testendpoint", model.UpdateOperation).
				FillBody("Rule Endpoint Message"),
			sendRouterModule: routerModuleName,
			wantErr:          false,
			isRouterMessage:  true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			cml := &ContextMessageLayer{
				SendModuleName:       sendModuleName,
				SendRouterModuleName: tt.sendRouterModule,
				ReceiveModuleName:    tt.message.GetGroup(),
				ResponseModuleName:   responseModuleName,
			}

			// Test Send
			if err := cml.Send(*tt.message); (err != nil) != tt.wantErr {
				t.Errorf("ContextMessageLayer.Send() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Test isRouterMsg
			if got := isRouterMsg(*tt.message); got != tt.isRouterMessage {
				t.Errorf("isRouterMsg() = %v, want %v", got, tt.isRouterMessage)
			}

			// Test Receive with timeout
			got, err := cml.Receive()
			if err != nil {
				t.Errorf("ContextMessageLayer.Receive() failed. err: %v", err)
				return
			}

			// Compare messages ignoring IDs
			if !compareMessages(got, *tt.message) {
				t.Errorf("ContextMessageLayer.Receive() message mismatch:\ngot  = %+v\nwant = %+v", got, *tt.message)
			}

			// Test Response
			err = cml.Response(got)
			if (err != nil) != tt.wantErr {
				t.Errorf("ContextMessageLayer.Response() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageLayerImplementations(t *testing.T) {
	tests := []struct {
		name     string
		factory  func() MessageLayer
		expected ContextMessageLayer
	}{
		{
			name:    "EdgeController Message Layer",
			factory: EdgeControllerMessageLayer,
			expected: ContextMessageLayer{
				SendModuleName:       modules.CloudHubModuleName,
				SendRouterModuleName: modules.RouterModuleName,
				ReceiveModuleName:    modules.EdgeControllerModuleName,
				ResponseModuleName:   modules.CloudHubModuleName,
			},
		},
		{
			name:    "DeviceController Message Layer",
			factory: DeviceControllerMessageLayer,
			expected: ContextMessageLayer{
				SendModuleName:     modules.CloudHubModuleName,
				ReceiveModuleName:  modules.DeviceControllerModuleName,
				ResponseModuleName: modules.CloudHubModuleName,
			},
		},
		{
			name:    "DynamicController Message Layer",
			factory: DynamicControllerMessageLayer,
			expected: ContextMessageLayer{
				SendModuleName:     modules.CloudHubModuleName,
				ReceiveModuleName:  modules.DynamicControllerModuleName,
				ResponseModuleName: modules.CloudHubModuleName,
			},
		},
		{
			name:    "TaskManager Message Layer",
			factory: TaskManagerMessageLayer,
			expected: ContextMessageLayer{
				SendModuleName:     modules.CloudHubModuleName,
				ReceiveModuleName:  modules.TaskManagerModuleName,
				ResponseModuleName: modules.CloudHubModuleName,
			},
		},
		{
			name:    "PolicyController Message Layer",
			factory: PolicyControllerMessageLayer,
			expected: ContextMessageLayer{
				SendModuleName:     modules.CloudHubModuleName,
				ReceiveModuleName:  modules.PolicyControllerModuleName,
				ResponseModuleName: modules.CloudHubModuleName,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			got := tt.factory()
			cml, ok := got.(*ContextMessageLayer)
			if !ok {
				t.Errorf("%s: expected ContextMessageLayer, got %T", tt.name, got)
				return
			}
			if !reflect.DeepEqual(*cml, tt.expected) {
				t.Errorf("%s: got = %v, want %v", tt.name, *cml, tt.expected)
			}
		})
	}
}
