/*
Copyright 2018 The KubeEdge Authors.

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

package dtcontext

import (
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/kubeedge/beehive/pkg/common/config"
	_ "github.com/kubeedge/beehive/pkg/common/config"
	"github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

// TestInitDTContext is function to test InitDTContext().
func TestInitDTContext(t *testing.T) {
	nodeID, err := config.CONFIG.GetValue("edgehub.controller.node-id").ToString()
	if err != nil {
		t.Errorf("Error in getting node id %v", err)
		return
	}
	tests := []struct {
		name    string
		context *context.Context
		want    *DTContext
		wantErr error
	}{
		{
			name:    "ActualContextArgumentTest",
			context: context.GetContext(context.MsgCtxTypeChannel),
			want: &DTContext{
				NodeID:         nodeID,
				CommChan:       make(map[string]chan interface{}),
				ConfirmChan:    make(chan interface{}, 1000),
				ModulesContext: context.GetContext(context.MsgCtxTypeChannel),
				State:          dtcommon.Disconnected,
			},
			wantErr: nil,
		},
		{
			name:    "EmptyContextArgumentTest",
			context: context.GetContext("test"),
			want: &DTContext{
				NodeID:         nodeID,
				CommChan:       make(map[string]chan interface{}),
				ConfirmChan:    make(chan interface{}, 1000),
				ModulesContext: context.GetContext("test"),
				State:          dtcommon.Disconnected,
			},
			wantErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := InitDTContext(test.context)
			if err != test.wantErr {
				t.Errorf("InitDTContext() error = %v, wantError %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got.NodeID, test.want.NodeID) {
				t.Errorf("NodeID = %v, Want = %v", got.NodeID, test.want.NodeID)
				return
			}
			if !reflect.DeepEqual(got.CommChan, test.want.CommChan) {
				t.Errorf("CommunicationChannel = %v, Want =%v", got.CommChan, test.want.CommChan)
				return
			}
			if cap(got.ConfirmChan) != cap(test.want.ConfirmChan) {
				t.Errorf("ConfirmChan size = %v, Want = %v", cap(got.ConfirmChan), cap(test.want.ConfirmChan))
				return
			}
			if !reflect.DeepEqual(got.ModulesContext, test.want.ModulesContext) {
				t.Errorf("ModulesContext = %v, Want = %v", got.ModulesContext, test.want.ModulesContext)
				return
			}
			if !reflect.DeepEqual(got.State, test.want.State) {
				t.Errorf("State = %v, Want = %v", got.State, test.want.State)
				return
			}
		})
	}
}

//TestCommTo is function to test CommTo().
func TestCommTo(t *testing.T) {
	commChan := make(map[string]chan interface{})
	testInterface := make(chan interface{}, 1)
	commChan["ModuleB"] = testInterface
	dtContext := &DTContext{
		CommChan: commChan,
	}
	var returnValue interface{}
	tests := []struct {
		name       string
		modulename string
		content    interface{}
		wantErr    error
	}{
		{
			//Failure Case
			name:       "ModuleNotPresent",
			modulename: "ModuleA",
			content:    nil,
			wantErr:    errors.New("Not found chan to communicate"),
		},
		{
			//Success Case
			name:       "ModulePresent",
			modulename: "ModuleB",
			content:    dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}}},
			wantErr:    nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dtContext.CommTo(test.modulename, test.content)
			if err == nil {
				returnValue = <-testInterface
			} else {
				returnValue = nil
			}
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("Error in CommTo %v, Want %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(returnValue, test.content) {
				t.Errorf("Got %v on channel, Want % v on Channel", returnValue, test.content)
				return
			}
		})
	}
}

//TestHeartBeat is function to test HeartBeat().
func TestHeartBeat(t *testing.T) {
	tests := []struct {
		name       string
		dtc        *DTContext
		moduleName string
		content    interface{}
		wantError  error
	}{
		{
			//Success Case
			name: "PingTest",
			dtc: &DTContext{
				ModulesHealth: &sync.Map{},
			},
			moduleName: "ModuleA",
			content:    "ping",
			wantError:  nil,
		},
		{
			//Failure Case
			name: "StopTest",
			dtc: &DTContext{
				ModulesHealth: &sync.Map{},
			},
			moduleName: "ModuleB",
			content:    "stop",
			wantError:  errors.New("stop"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dtc := &DTContext{
				ModulesHealth: test.dtc.ModulesHealth,
			}
			if err := dtc.HeartBeat(test.moduleName, test.content); !reflect.DeepEqual(err, test.wantError) {
				t.Errorf("DTContext.HeartBeat() error = %v, Want = %v", err, test.wantError)
			}
		})
	}
}

//TestGetMutex is function to test GetMutex().
func TestGetMutex(t *testing.T) {
	dtc := &DTContext{
		DeviceMutex: &sync.Map{},
	}
	var testMutex *sync.Mutex
	dtc.DeviceMutex.Store("DeviceB", "")
	dtc.DeviceMutex.Store("DeviceC", testMutex)
	tests := []struct {
		name     string
		want     *sync.Mutex
		wantBool bool
		deviceID string
	}{
		{
			//Failure Case-No device present
			name:     "UnknownDevice",
			wantBool: false,
			deviceID: "DeviceA",
		},
		{
			//Failure Case-Device present but unable to get mutex
			name:     "UnableToGetMutex",
			wantBool: false,
			deviceID: "DeviceB",
		},
		{
			//Success Case
			name:     "KnownDevice",
			wantBool: true,
			deviceID: "DeviceC",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotBool := dtc.GetMutex(test.deviceID)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("DTContext.GetMutex() got = %v, want = %v", got, test.want)
				return
			}
			if gotBool != test.wantBool {
				t.Errorf("DTContext.GetMutex() gotBool = %v, wantError = %v", gotBool, test.wantBool)
				return
			}
		})
	}
}

//TestLock is function to test Lock().
func TestLock(t *testing.T) {
	dtc := &DTContext{
		Mutex:       &sync.RWMutex{},
		DeviceMutex: &sync.Map{},
	}
	var testMutex sync.Mutex
	dtc.DeviceMutex.Store("DeviceB", &testMutex)
	tests := []struct {
		name     string
		deviceID string
		wantBool bool
	}{
		{
			//Failure Case
			name:     "UnknownDevice",
			deviceID: "DeviceA",
			wantBool: false,
		},
		{
			//Success Case
			name:     "KnownDevice",
			deviceID: "DeviceB",
			wantBool: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := dtc.Lock(test.deviceID); got != test.wantBool {
				t.Errorf("DTContext.Lock() = %v, want = %v", got, test.wantBool)
			}
		})
	}
}

//TestUnlock is function to test Unlock().
func TestUnlock(t *testing.T) {
	dtc := &DTContext{
		Mutex:       &sync.RWMutex{},
		DeviceMutex: &sync.Map{},
	}
	// Creating a mutex variable and getting a lock over it.
	var testMutex sync.Mutex
	dtc.DeviceMutex.Store("DeviceB", &testMutex)
	dtc.Lock("DeviceB")
	tests := []struct {
		name     string
		deviceID string
		wantBool bool
	}{
		{
			//Failure Case
			name:     "UnknownDevice",
			deviceID: "DeviceA",
			wantBool: false,
		},
		{
			//Success Case
			name:     "KnownDevice",
			deviceID: "DeviceB",
			wantBool: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dtc.Unlock(tt.deviceID); got != tt.wantBool {
				t.Errorf("DTContext.Unlock() = %v, Want = %v", got, tt.wantBool)
			}
		})
	}
}

//TestIsDeviceExist is to test IsDeviceExist().
func TestIsDeviceExist(t *testing.T) {
	dtc := &DTContext{
		DeviceList: &sync.Map{},
	}
	dtc.DeviceList.Store("DeviceB", "")
	tests := []struct {
		name     string
		deviceID string
		wantBool bool
	}{
		{
			//Failure Case
			name:     "UnknownDevice",
			deviceID: "DeviceA",
			wantBool: false,
		},
		{
			//Success Case
			name:     "KnownDevice",
			deviceID: "DeviceB",
			wantBool: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := dtc.IsDeviceExist(test.deviceID); got != test.wantBool {
				t.Errorf("DTContext.IsDeviceExist() = %v, Want = %v", got, test.wantBool)
			}
		})
	}
}

//Function TestGetDevice is to test GetDevice().
func TestGetDevice(t *testing.T) {
	dtc := &DTContext{
		DeviceList: &sync.Map{},
	}
	var device dttype.Device
	dtc.DeviceList.Store("DeviceA", "")
	dtc.DeviceList.Store("DeviceB", &device)
	tests := []struct {
		name     string
		deviceID string
		want     *dttype.Device
		wantBool bool
	}{
		{
			//Failure Case-DeviceID not present
			name:     "UnknownDevice",
			deviceID: "",
			want:     nil,
			wantBool: false,
		},
		{
			//Failure Case-DeviceID present but unable to get device
			name:     "DeviceError",
			deviceID: "DeviceA",
			want:     nil,
			wantBool: false,
		},
		{
			//Success Case
			name:     "KnownDevice",
			deviceID: "DeviceB",
			want:     &dttype.Device{},
			wantBool: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, gotBool := dtc.GetDevice(test.deviceID)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("DTContext.GetDevice() got = %v, Want = %v", got, test.want)
				return
			}
			if gotBool != test.wantBool {
				t.Errorf("DTContext.GetDevice() gotBool = %v, wantError %v", gotBool, test.wantBool)
				return
			}
		})
	}
}

//Function TestSend is function to test Send().
func TestSend(t *testing.T) {
	payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}}}
	content, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("Got error on marshalling: %v", err)
	}
	commChan := make(map[string]chan interface{})
	receiveCh := make(chan interface{}, 1)
	commChan[dtcommon.TwinModule] = receiveCh
	var msg = &model.Message{
		Content: content,
	}
	dtc := &DTContext{
		CommChan: commChan,
	}
	tests := []struct {
		name      string
		identity  string
		action    string
		module    string
		msg       *model.Message
		wantError error
	}{
		{
			//Failure Case
			name:      "UnknownModule",
			action:    dtcommon.SendToCloud,
			module:    dtcommon.CommModule,
			msg:       msg,
			wantError: errors.New("Not found chan to communicate"),
		},
		{
			//Success Case
			name:      "KnownModule",
			action:    dtcommon.SendToCloud,
			module:    dtcommon.TwinModule,
			msg:       msg,
			wantError: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dtc.Send(test.identity, test.action, test.module, test.msg); !reflect.DeepEqual(err, test.wantError) {
				t.Errorf("DTContext.Send() error = %v, Want =  %v", err, test.wantError)
			}
		})
	}
}

//TestBuildModelMessage is to test BuildModelMessage().
func TestBuildModelMessage(t *testing.T) {
	dtc := &DTContext{}
	payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router", State: "unknown"}}}
	content, err := json.Marshal(payload)
	if err != nil {
		t.Errorf("Error on Marshalling: %v", err)
	}
	tests := []struct {
		name      string
		group     string
		parentID  string
		resource  string
		operation string
		content   interface{}
		want      *model.Message
	}{
		{
			name:      "BuildModelMessageTest",
			group:     "resource",
			resource:  "membership/detail",
			operation: "get",
			content:   content,
			want:      &model.Message{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := dtc.BuildModelMessage(test.group, test.parentID, test.resource, test.operation, test.content)
			if got.Header.ParentID != test.parentID {
				t.Errorf("DTContext.BuildModelMessage failed due to wrong parentID, Got = %v Want = %v", got.Header.ParentID, test.parentID)
				return
			}
			if got.Router.Source != modules.TwinGroup {
				t.Errorf("DtContext.BuildModelMessage failed due to wrong source, Got= %v Want = %v", got.Router.Source, modules.TwinGroup)
				return
			}
			if got.Router.Group != test.group {
				t.Errorf("DTContext.BuildModelMessage due to wrong group, Got = %v Want = %v", got.Router.Group, test.group)
				return
			}
			if got.Router.Resource != test.resource {
				t.Errorf("DTContext.BuildModelMessage failed due to wrong resource, Got = %v Want =%v ", got.Router.Resource, test.resource)
				return
			}
			if got.Router.Operation != test.operation {
				t.Errorf("DTContext.BuildModelMessage failed due to wrong operation, Got = %v Want = %v ", got.Router.Operation, test.operation)
				return
			}
			if !reflect.DeepEqual(got.Content, test.content) {
				t.Errorf("DTContext.buildModelMessage failed due to wrong content, Got= %v Want = %v", got.Content, test.content)
				return
			}
		})
	}
}
