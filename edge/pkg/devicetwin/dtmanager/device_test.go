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

package dtmanager

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

var called bool

// testAction is a dummy function for testing Start
func testAction(*dtcontext.DTContext, string, interface{}) error {
	called = true
	return errors.New("called the dummy function for testing")
}

// TestDeviceStartAction is function to test Start() when value is passed in ReceiverChan.
func TestDeviceStartAction(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	content := dttype.DeviceUpdate{}
	bytes, _ := json.Marshal(content)
	msg := model.Message{Content: bytes}

	receiveChanActionNotPresent := GenerateReceiveChanAction(Action, Identity, Message, Msg)

	tests := []CaseWorkerStr{
		GenerateStartActionCase(ActionNotPresent, receiveChanActionNotPresent),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dw := DeviceWorker{
				Worker: test.Worker,
			}
			go dw.Start()
			time.Sleep(1 * time.Millisecond)
			//Adding a dummy function to callback to ensure Start is successful.
			deviceActionCallBack[TestAction] = testAction
			dw.ReceiverChan <- &dttype.DTMessage{
				Action:   TestAction,
				Identity: Identity,
				Msg:      &msg,
			}
			time.Sleep(1 * time.Millisecond)
			if !called {
				t.Errorf("Start failed")
			}
		})
	}
}

// TestDeviceHeartBeat is function to test Start() when value is passed in HeartBeatChan.
func TestDeviceHeartBeat(t *testing.T) {
	heartChanStop := make(chan interface{}, 1)
	heartChanPing := make(chan interface{}, 1)
	heartChanStop <- "stop"
	heartChanPing <- "ping"
	tests := []CaseHeartBeatWorkerStr{
		GenerateHeartBeatCase(PingHeartBeat, Group, heartChanPing),
		GenerateHeartBeatCase(StopHeartBeat, Group, heartChanStop),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dw := DeviceWorker{
				Worker: test.Worker,
				Group:  test.Group,
			}
			go dw.Start()
			time.Sleep(1 * time.Millisecond)
			if test.Worker.HeartBeatChan == heartChanPing {
				_, exist := test.Worker.DTContexts.ModulesHealth.Load("group")
				if !exist {
					t.Errorf("Start Failed to add module in beehiveContext")
				}
			}
		})
	}
}

func TestDealDeviceStateUpdate(t *testing.T) {
	var emptyDevUpdate dttype.DeviceUpdate
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	dtContexts, err := dtcontext.InitDTContext()
	if err != nil {
		t.Errorf("InitDTContext error %v", err)
		return
	}

	dtContexts.DeviceList.Store("DeviceC", "DeviceC")
	deviceD := &dttype.Device{}
	dtContexts.DeviceList.Store("DeviceD", deviceD)
	bytesEmptyDevUpdate, err := json.Marshal(emptyDevUpdate)
	if err != nil {
		t.Errorf("marshal error %v", err)
		return
	}
	devUpdate := &dttype.DeviceUpdate{State: "online"}
	bytesDevUpdate, err := json.Marshal(devUpdate)
	if err != nil {
		t.Errorf("marshal error %v", err)
		return
	}

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		wantErr  error
	}{
		{
			name:     "dealDeviceStateUpdateTest-WrongMessageType",
			context:  dtContexts,
			resource: "DeviceA",
			msg:      "",
			wantErr:  errors.New("msg not Message type"),
		},
		{
			name:     "dealDeviceStateUpdateTest-DeviceDoesNotExist",
			context:  dtContexts,
			resource: "DeviceB",
			msg:      &model.Message{Content: bytesEmptyDevUpdate},
			wantErr:  nil,
		},
		{
			name:     "dealDeviceStateUpdateTest-DeviceExist",
			context:  dtContexts,
			resource: "DeviceC",
			msg:      &model.Message{Content: bytesEmptyDevUpdate},
			wantErr:  nil,
		},
		{
			name:     "dealDeviceStateUpdateTest-CorrectDeviceType",
			context:  dtContexts,
			resource: "DeviceD",
			msg:      &model.Message{Content: bytesEmptyDevUpdate},
			wantErr:  nil,
		},
		{
			name:     "dealDeviceStateUpdateTest-UpdatePresent",
			context:  dtContexts,
			resource: "DeviceD",
			msg:      &model.Message{Content: bytesDevUpdate},
			wantErr:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dealDeviceStateUpdate(test.context, test.resource, test.msg)
			// Compare error messages instead of error objects
			if test.wantErr != nil {
				if err == nil || err.Error() != test.wantErr.Error() {
					t.Errorf("dealDeviceStateUpdate() error = %v, wantErr %v", err, test.wantErr)
					return
				}
			} else {
				if err != nil {
					t.Errorf("dealDeviceStateUpdate() error = %v, expected no error", err)
					return
				}
			}
		})
	}
}

func TestDealUpdateDeviceAttr(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	dtContexts, _ := dtcontext.InitDTContext()
	content := dttype.DeviceUpdate{}
	bytes, err := json.Marshal(content)
	if err != nil {
		t.Errorf("marshal error %v", err)
		return
	}
	msg := model.Message{Content: bytes}
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		wantErr  error
	}{
		{
			name:     "DealUpdateDeviceAttrTest-Wrong Message Type",
			context:  dtContexts,
			resource: "Device",
			msg:      "",
			wantErr:  errors.New("msg not Message type"),
		},
		{
			name:     "DealUpdateDeviceAttrTest-Correct Message Type",
			context:  dtContexts,
			resource: "DeviceA",
			msg:      &msg,
			wantErr:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dealDeviceAttrUpdate(test.context, test.resource, test.msg)
			// Compare error messages instead of error objects
			if test.wantErr != nil {
				if err == nil || err.Error() != test.wantErr.Error() {
					t.Errorf("dealUpdateDeviceAttr() error = %v, wantErr %v", err, test.wantErr)
					return
				}
			} else {
				if err != nil {
					t.Errorf("dealUpdateDeviceAttr() error = %v, expected no error", err)
					return
				}
			}
		})
	}
}

func TestUpdateDeviceAttr(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})

	dtContexts, _ := dtcontext.InitDTContext()
	dtContexts.DeviceList.Store("EmptyDevice", "Device")

	devA := &dttype.Device{ID: "DeviceA"}
	dtContexts.DeviceList.Store("DeviceA", devA)

	messageAttributes := generateTestMessageAttributes()
	baseMessage := dttype.BuildBaseMessage()

	tests := []struct {
		name        string
		context     *dtcontext.DTContext
		deviceID    string
		attributes  map[string]*dttype.MsgAttr
		baseMessage dttype.BaseMessage
		dealType    int
		want        interface{}
		wantErr     error
	}{
		{
			name:        "Test1",
			context:     dtContexts,
			deviceID:    "Device",
			attributes:  messageAttributes,
			baseMessage: baseMessage,
			want:        nil,
			wantErr:     nil,
		},
		{
			name:        "Test2",
			context:     dtContexts,
			deviceID:    "EmptyDevice",
			attributes:  messageAttributes,
			baseMessage: baseMessage,
			want:        nil,
			wantErr:     nil,
		},
		{
			name:        "Test3",
			context:     dtContexts,
			deviceID:    "DeviceA",
			attributes:  messageAttributes,
			baseMessage: baseMessage,
			wantErr:     nil,
			want:        nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UpdateDeviceAttr(test.context, test.deviceID, test.attributes, test.baseMessage, test.dealType)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("UpdateDeviceAttr() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("UpdateDeviceAttr() failed Got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDealMsgAttr(t *testing.T) {
	dtContextsEmptyAttributes, err := dtcontext.InitDTContext()
	if err != nil {
		t.Errorf("initDtcontext error %v", err)
		return
	}
	dtContextsNonEmptyAttributes, err := dtcontext.InitDTContext()
	if err != nil {
		t.Errorf("initDtcontext error %v", err)
		return
	}
	//Creating want and message attributes when device attribute is not present
	devA := &dttype.Device{ID: "DeviceA"}
	dtContextsEmptyAttributes.DeviceList.Store("DeviceA", devA)

	messageAttributes := make(map[string]*dttype.MsgAttr)
	optional := true

	msgattr := &dttype.MsgAttr{
		Value:    "ON",
		Optional: &optional,
		Metadata: &dttype.TypeMetadata{
			Type: "device",
		},
	}
	messageAttributes["DeviceA"] = msgattr
	add := []models.DeviceAttr{}
	add = append(add, models.DeviceAttr{
		ID:       0,
		DeviceID: "DeviceA",
		Name:     "DeviceA",
		Value:    "ON",
		Optional: true,
		AttrType: "device",
		Metadata: "{}",
	})
	result := make(map[string]*dttype.MsgAttr)
	result["DeviceA"] = msgattr
	wantDealAttrResult := dttype.DealAttrResult{
		Add:    add,
		Delete: []models.DeviceDelete{},
		Update: []models.DeviceAttrUpdate{},
		Result: result,
		Err:    nil,
	}
	//Creating want and message attributes when device attribute is present
	attributes := map[string]*dttype.MsgAttr{}
	attributes["DeviceB"] = msgattr
	attr := map[string]*dttype.MsgAttr{}
	opt := false
	attr["DeviceB"] = &dttype.MsgAttr{
		Value:    "OFF",
		Optional: &opt,
		Metadata: &dttype.TypeMetadata{
			Type: "device",
		},
	}
	devB := &dttype.Device{ID: "DeviceB", Attributes: attr}
	update := []models.DeviceAttrUpdate{}
	cols := make(map[string]interface{})
	cols["value"] = "ON"
	upd := models.DeviceAttrUpdate{
		Name:     "DeviceB",
		DeviceID: "DeviceB",
		Cols:     cols,
	}
	update = append(update, upd)
	dtContextsNonEmptyAttributes.DeviceList.Store("DeviceB", devB)
	want := dttype.DealAttrResult{
		Add:    []models.DeviceAttr{},
		Delete: []models.DeviceDelete{},
		Update: update,
	}

	tests := []struct {
		name          string
		context       *dtcontext.DTContext
		deviceID      string
		msgAttributes map[string]*dttype.MsgAttr
		dealType      int
		want          dttype.DealAttrResult
	}{
		{
			name:          "DealMsgAttrTest-DeviceAttribute not present",
			context:       dtContextsEmptyAttributes,
			deviceID:      "DeviceA",
			msgAttributes: messageAttributes,
			want:          wantDealAttrResult,
		},
		{
			name:          "DealMsgAttrTest-DeviceAttribute present",
			context:       dtContextsNonEmptyAttributes,
			deviceID:      "DeviceB",
			msgAttributes: attributes,
			dealType:      1,
			want:          want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DealMsgAttr(tt.context, tt.deviceID, tt.msgAttributes, tt.dealType)
			if !reflect.DeepEqual(got.Add, tt.want.Add) {
				t.Errorf("Add error , Got = %v, Want = %v", got.Add, tt.want.Add)
				return
			}
			if !reflect.DeepEqual(got.Delete, tt.want.Delete) {
				t.Errorf("Delete error , Got = %v, Want = %v", got.Delete, tt.want.Delete)
				return
			}
			if !reflect.DeepEqual(got.Update, tt.want.Update) {
				t.Errorf("Update error , Got = %v, Want = %v", got.Update, tt.want.Update)
				return
			}
			if !reflect.DeepEqual(got.Err, tt.want.Err) {
				t.Errorf("Error error , Got = %v, Want = %v", got.Err, tt.want.Err)
				return
			}
			for key, value := range tt.want.Result {
				check := false
				for key1, value1 := range got.Result {
					if key == key1 {
						if value == value1 {
							check = true
							break
						}
					}
					if !check {
						t.Errorf("Wrong Map")
						return
					}
				}
			}
		})
	}
}

// generateTestMessageAttributes creates test message attributes
func generateTestMessageAttributes() map[string]*dttype.MsgAttr {
	optional := true
	return map[string]*dttype.MsgAttr{
		"attr1": {
			Value:    "value1",
			Optional: &optional,
		},
	}
}
