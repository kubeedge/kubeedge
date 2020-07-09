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

	"github.com/astaxie/beego/orm"
	"github.com/golang/mock/gomock"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var called bool

//testAction is a dummy function for testing Start
func testAction(context *dtcontext.DTContext, resource string, msg interface{}) (interface{}, error) {
	called = true
	return called, errors.New("Called the dummy function for testing")
}

// TestDeviceStartAction is function to test Start() when value is passed in ReceiverChan.
func TestDeviceStartAction(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	dtContextStateConnected, _ := dtcontext.InitDTContext()
	dtContextStateConnected.State = dtcommon.Connected
	content := dttype.DeviceUpdate{}
	bytes, _ := json.Marshal(content)
	msg := model.Message{Content: bytes}

	receiveChanActionPresent := make(chan interface{}, 1)
	receiveChanActionPresent <- &dttype.DTMessage{
		Action:   "testAction",
		Identity: "identity",
		Msg:      &msg,
	}

	receiveChanActionNotPresent := make(chan interface{}, 1)
	receiveChanActionNotPresent <- &dttype.DTMessage{
		Action:   "action",
		Identity: "identity",
		Msg: &model.Message{
			Content: "msg",
		},
	}
	tests := []struct {
		name   string
		Worker Worker
	}{
		{
			name: "StartTest-ActionNotPresentInActionCallback",
			Worker: Worker{
				ReceiverChan: receiveChanActionNotPresent,
				DTContexts:   dtContextStateConnected,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dw := DeviceWorker{
				Worker: test.Worker,
			}
			go dw.Start()
			time.Sleep(1 * time.Millisecond)
			//Adding a dummy function to callback to ensure Start is successful.
			deviceActionCallBack["testAction"] = testAction
			dw.ReceiverChan <- &dttype.DTMessage{
				Action:   "testAction",
				Identity: "identity",
				Msg:      &msg,
			}
			time.Sleep(1 * time.Millisecond)
			if !called {
				t.Errorf("Start failed")
			}
		})
	}
}

// TestDevicetHeartBeat is function to test Start() when value is passed in HeartBeatChan.
func TestDeviceStartHeartBeat(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	dtContexts, _ := dtcontext.InitDTContext()
	heartChanStop := make(chan interface{}, 1)
	heartChanPing := make(chan interface{}, 1)
	heartChanStop <- "stop"
	heartChanPing <- "ping"
	tests := []struct {
		name   string
		Worker Worker
		Group  string
	}{
		{
			name: "StartTest-PingInHeartBeatChannel",
			Worker: Worker{
				HeartBeatChan: heartChanPing,
				DTContexts:    dtContexts,
			},
			Group: "group",
		},
		{
			name: "StartTest-StopInHeartBeatChannel",
			Worker: Worker{
				HeartBeatChan: heartChanStop,
				DTContexts:    dtContexts,
			},
			Group: "group",
		},
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

// TestDealDeviceStatusUpdate test dealDeviceStatusUpdate
func TestDealDeviceStateUpdate(t *testing.T) {
	var ormerMock *beego.MockOrmer
	var querySeterMock *beego.MockQuerySeter
	var emptyDevUpdate dttype.DeviceUpdate
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

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
		want     interface{}
		wantErr  error
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// updateReturnInt is the first return of mock interface querySeterMock's update function
		updateReturnInt int64
		// updateReturnErr is the second return of mock interface querySeterMocks's update function also expected error
		updateReturnErr error
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
		times            int
	}{
		{
			name:     "dealDeviceStateUpdateTest-WrongMessageType",
			context:  dtContexts,
			resource: "DeviceA",
			msg:      "",
			want:     nil,
			wantErr:  errors.New("msg not Message type"),
		},
		{
			name:     "dealDeviceStateUpdateTest-DeviceDoesNotExist",
			context:  dtContexts,
			resource: "DeviceB",
			msg:      &model.Message{Content: bytesEmptyDevUpdate},
			want:     nil,
			wantErr:  nil,
		},
		{
			name:     "dealDeviceStateUpdateTest-DeviceExist",
			context:  dtContexts,
			resource: "DeviceC",
			msg:      &model.Message{Content: bytesEmptyDevUpdate},
			want:     nil,
			wantErr:  nil,
		},
		{
			name:     "dealDeviceStateUpdateTest-CorrectDeviceType",
			context:  dtContexts,
			resource: "DeviceD",
			msg:      &model.Message{Content: bytesEmptyDevUpdate},
			want:     nil,
			wantErr:  nil,
		},
		{
			name:             "dealDeviceStateUpdateTest-UpdatePresent",
			context:          dtContexts,
			resource:         "DeviceD",
			msg:              &model.Message{Content: bytesDevUpdate},
			want:             nil,
			wantErr:          nil,
			filterReturn:     querySeterMock,
			updateReturnInt:  int64(1),
			updateReturnErr:  nil,
			queryTableReturn: querySeterMock,
			times:            2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.times)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(test.times)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.times)
			got, err := dealDeviceStateUpdate(test.context, test.resource, test.msg)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("dealDeviceStateUpdate() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("dealDeviceStateUpdate() = %v, want %v", got, test.want)
			}
		})
	}
}

//TestDealUpdateDeviceAttr is function to test dealUpdateDeviceAttr().
func TestDealUpdateDeviceAttr(t *testing.T) {
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
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
		want     interface{}
		wantErr  error
	}{
		{
			name:     "DealUpdateDeviceAttrTest-Wrong Message Type",
			context:  dtContexts,
			resource: "Device",
			msg:      "",
			want:     nil,
			wantErr:  errors.New("msg not Message type"),
		},
		{
			name:     "DealUpdateDeviceAttrTest-Correct Message Type",
			context:  dtContexts,
			resource: "DeviceA",
			msg:      &msg,
			want:     nil,
			wantErr:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := dealUpdateDeviceAttr(test.context, test.resource, test.msg)
			if !reflect.DeepEqual(err, test.wantErr) {
				t.Errorf("dealUpdateDeviceAttr() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("dealUpdateDeviceAttr() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestUpdateDeviceAttr is function to test UpdateDeviceAttr().
func TestUpdateDeviceAttr(t *testing.T) {
	var ormerMock *beego.MockOrmer
	var querySeterMock *beego.MockQuerySeter
	beehiveContext.InitContext(beehiveContext.MsgCtxTypeChannel)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

	// adds is fake DeviceAttr used as argument
	adds := make([]dtclient.DeviceAttr, 0)
	// deletes is fake DeviceDelete used as argument
	deletes := make([]dtclient.DeviceDelete, 0)
	// updates is fake DeviceAttrUpdate used as argument
	updates := make([]dtclient.DeviceAttrUpdate, 0)
	adds = append(adds, dtclient.DeviceAttr{
		DeviceID: "Test",
	})
	deletes = append(deletes, dtclient.DeviceDelete{
		DeviceID: "test",
		Name:     "test",
	})
	updates = append(updates, dtclient.DeviceAttrUpdate{
		DeviceID: "test",
		Name:     "test",
		Cols:     make(map[string]interface{}),
	})

	dtContexts, _ := dtcontext.InitDTContext()
	dtContexts.DeviceList.Store("EmptyDevice", "Device")

	devA := &dttype.Device{ID: "DeviceA"}
	dtContexts.DeviceList.Store("DeviceA", devA)

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
		// commitTimes is number of times commit is expected
		commitTimes int
		// beginTimes is number of times begin is expected
		beginTimes int
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// filterTimes is the number of times filter is called
		filterTimes int
		// insertReturnInt is the first return of mock interface ormerMock's Insert function
		insertReturnInt int64
		// insertReturnErr is the second return of mock interface ormerMock's Insert function
		insertReturnErr error
		// insertTimes is number of times Insert is expected
		insertTimes int
		// deleteReturnInt is the first return of mock interface ormerMock's Delete function
		deleteReturnInt int64
		// deleteReturnErr is the second return of mock interface ormerMock's Delete function
		deleteReturnErr error
		// deleteTimes is number of times Delete is expected
		deleteTimes int
		// updateReturnInt is the first return of mock interface ormerMock's Update function
		updateReturnInt int64
		// updateReturnErr is the second return of mock interface ormerMock's Update function
		updateReturnErr error
		// updateTimes is number of times Update is expected
		updateTimes int
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
		// queryTableTimes is the number of times queryTable is called
		queryTableTimes int
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
			name:             "Test3",
			context:          dtContexts,
			deviceID:         "DeviceA",
			attributes:       messageAttributes,
			baseMessage:      baseMessage,
			wantErr:          nil,
			want:             nil,
			commitTimes:      1,
			beginTimes:       1,
			filterReturn:     querySeterMock,
			filterTimes:      0,
			insertReturnInt:  int64(1),
			insertReturnErr:  nil,
			insertTimes:      1,
			deleteReturnInt:  int64(1),
			deleteReturnErr:  nil,
			deleteTimes:      0,
			updateReturnInt:  int64(1),
			updateReturnErr:  nil,
			updateTimes:      0,
			queryTableReturn: querySeterMock,
			queryTableTimes:  0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().Commit().Return(nil).Times(test.commitTimes)
			ormerMock.EXPECT().Begin().Return(nil).Times(test.beginTimes)
			ormerMock.EXPECT().Insert(gomock.Any()).Return(test.insertReturnInt, test.insertReturnErr).Times(test.insertTimes)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.queryTableTimes)

			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.filterTimes)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(test.deleteTimes)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(test.updateTimes)

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

// TestDealMsgAttr is function to test DealMsgAttr().
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
	add := []dtclient.DeviceAttr{}
	add = append(add, dtclient.DeviceAttr{
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
		Delete: []dtclient.DeviceDelete{},
		Update: []dtclient.DeviceAttrUpdate{},
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
	update := []dtclient.DeviceAttrUpdate{}
	cols := make(map[string]interface{})
	cols["value"] = "ON"
	upd := dtclient.DeviceAttrUpdate{
		Name:     "DeviceB",
		DeviceID: "DeviceB",
		Cols:     cols,
	}
	update = append(update, upd)
	dtContextsNonEmptyAttributes.DeviceList.Store("DeviceB", devB)
	want := dttype.DealAttrResult{
		Add:    []dtclient.DeviceAttr{},
		Delete: []dtclient.DeviceDelete{},
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
				t.Errorf("Error error , Got = %v, Want = %v", got.Update, tt.want.Update)
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
					if check == false {
						t.Errorf("Wrong Map")
						return
					}
				}
			}
		})
	}
}
