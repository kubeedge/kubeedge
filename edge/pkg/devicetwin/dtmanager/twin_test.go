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
	"sync"
	"testing"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/golang/mock/gomock"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var mockOrmer *beego.MockOrmer
var mockQuerySeter *beego.MockQuerySeter

// mocksInit is function to mock DBAccess
func mocksInit(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockOrmer = beego.NewMockOrmer(mockCtrl)
	mockQuerySeter = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = mockOrmer
}

// sendMsg is function to send message to receiverChannel and heartbeatChannel
func (tw TwinWorker) sendMsg(msg *dttype.DTMessage, msgHeart string, actionType string, contentType interface{}) {
	if tw.ReceiverChan != nil {
		msg.Action = actionType
		msg.Msg.Content = contentType
		tw.ReceiverChan <- msg
	}
	if tw.HeartBeatChan != nil {
		tw.HeartBeatChan <- msgHeart
	}
}

// receiveMsg is function to receive message from the commChannel
func receiveMsg(commChannel chan interface{}, message *dttype.DTMessage) {
	msg, ok := <-commChannel
	if !ok {
		log.LOGGER.Errorf("No message received from communication channel")
		return
	}
	*message = *msg.(*dttype.DTMessage)
	return
}

// expectedFunc is function to declare an object of type TwinValue
func expectedFunc() *dttype.TwinValue {
	var expected dttype.TwinValue
	value := "value"
	valueMetaData := &dttype.ValueMetadata{Timestamp: time.Now().UnixNano() / 1e6}
	expected.Value = &value
	expected.Metadata = valueMetaData
	return &expected
}

// actualFunc is function to declare an object of type TwinValue
func actualFunc() *dttype.TwinValue {
	var actual dttype.TwinValue
	value := "value"
	valueMetaData := &dttype.ValueMetadata{Timestamp: time.Now().UnixNano() / 1e6}
	actual.Value = &value
	actual.Metadata = valueMetaData
	return &actual
}

// keyTwinUpdateFunc is function to declare an object of type DeviceTwinUpdate
func keyTwinUpdateFunc() dttype.DeviceTwinUpdate {
	var keyTwinUpdate dttype.DeviceTwinUpdate
	twinKey := make(map[string]*dttype.MsgTwin)
	twinKey["key1"] = &dttype.MsgTwin{
		Expected: expectedFunc(),
		Actual:   actualFunc(),
		Metadata: &dttype.TypeMetadata{"nil"},
	}
	keyTwinUpdate.Twin = twinKey
	keyTwinUpdate.BaseMessage = dttype.BaseMessage{EventID: "Event1"}
	return keyTwinUpdate
}

// twinWorkerFunc is function to return an object of type TwinWorker
func twinWorkerFunc(receiverChannel chan interface{}, confirmChannel chan interface{}, heartBeatChannel chan interface{}, context dtcontext.DTContext, group string) TwinWorker {
	return TwinWorker{
		Worker{
			receiverChannel,
			confirmChannel,
			heartBeatChannel,
			&context,
		},
		group,
	}
}

// contextFunc is function to return an object of type DTContext
func contextFunc() dtcontext.DTContext {
	context := dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.Mutex{},
	}
	var testMutex sync.Mutex
	context.DeviceMutex.Store("DeviceB", &testMutex)
	var device dttype.Device
	context.DeviceList.Store("DeviceB", &device)
	return context
}

// msgTypeFunc is function to return an object of type model.Message
func msgTypeFunc(content interface{}) *model.Message {
	return &model.Message{
		model.MessageHeader{},
		model.MessageRoute{},
		content,
	}
}

// msgTypeErrorFunc is function to return an object which is not of type model.Message
func msgTypeErrorFunc(content interface{}) model.Message {
	return model.Message{
		model.MessageHeader{},
		model.MessageRoute{},
		content,
	}
}

// TestStart is function to test Start
func TestStart(t *testing.T) {
	keyTwinUpdate := keyTwinUpdateFunc()
	contentKeyTwin, _ := json.Marshal(keyTwinUpdate)

	commChan := make(map[string]chan interface{})
	commChannel := make(chan interface{})
	commChan[dtcommon.CommModule] = commChannel

	context := dtcontext.DTContext{
		DeviceList:    &sync.Map{},
		DeviceMutex:   &sync.Map{},
		Mutex:         &sync.Mutex{},
		CommChan:      commChan,
		ModulesHealth: &sync.Map{},
	}
	var testMutex sync.Mutex
	context.DeviceMutex.Store("DeviceB", &testMutex)
	msgAttr := make(map[string]*dttype.MsgAttr)
	device := dttype.Device{
		ID:         "id1",
		Name:       "DeviceB",
		Attributes: msgAttr,
		Twin:       keyTwinUpdate.Twin,
	}
	context.DeviceList.Store("DeviceB", &device)

	msg := &dttype.DTMessage{
		Msg: &model.Message{
			Header: model.MessageHeader{
				ID:        "id1",
				ParentID:  "pid1",
				Timestamp: 0,
				Sync:      false,
			},
			Router: model.MessageRoute{
				Source:    "source",
				Resource:  "resource",
				Group:     "group",
				Operation: "op",
			},
			Content: contentKeyTwin,
		},
		Action: dtcommon.TwinGet,
		Type:   dtcommon.CommModule,
	}
	msgHeartPing := "ping"
	msgHeartStop := "stop"
	receiverChannel := make(chan interface{})
	heartbeatChannel := make(chan interface{})

	tests := []struct {
		name        string
		tw          TwinWorker
		actionType  string
		contentType interface{}
		msgType     string
	}{
		{
			name:        "TestStart(): Case 1",
			tw:          twinWorkerFunc(receiverChannel, nil, nil, context, ""),
			actionType:  dtcommon.TwinGet,
			contentType: contentKeyTwin,
		},
		{
			name:        "TestStart(): Case 2",
			tw:          twinWorkerFunc(receiverChannel, nil, nil, context, ""),
			actionType:  dtcommon.SendToEdge,
			contentType: contentKeyTwin,
		},
		{
			name:       "TestStart(): Case 3",
			tw:         twinWorkerFunc(receiverChannel, nil, nil, context, ""),
			actionType: dtcommon.TwinGet,
		},
		{
			name:    "TestStart(): Case 4",
			tw:      twinWorkerFunc(nil, nil, heartbeatChannel, context, "Group1"),
			msgType: msgHeartPing,
		},
		{
			name:    "TestStart(): Case 5",
			tw:      twinWorkerFunc(nil, nil, heartbeatChannel, context, "Group1"),
			msgType: msgHeartStop,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			go test.tw.sendMsg(msg, test.msgType, test.actionType, test.contentType)
			go test.tw.Start()
			time.Sleep(100 * time.Millisecond)
			message := &dttype.DTMessage{}
			go receiveMsg(commChannel, message)
			time.Sleep(100 * time.Millisecond)
			if (test.tw.ReceiverChan != nil) && !reflect.DeepEqual(message.Identity, msg.Identity) && !reflect.DeepEqual(message.Type, msg.Type) {
				t.Errorf("DTManager.TestStart() case failed: got = %v, Want = %v", message, msg)
			}
			if _, exist := context.ModulesHealth.Load("Group1"); test.tw.HeartBeatChan != nil && !exist {
				t.Errorf("DTManager.TestStart() case failed: HeartBeatChan received no string")
			}
		})
	}
}

// TestDealTwinSync is function to test dealTwinSync
func TestDealTwinSync(t *testing.T) {
	content, _ := json.Marshal(dttype.DeviceTwinUpdate{BaseMessage: dttype.BaseMessage{EventID: "Event1"}})
	contentKeyTwin, _ := json.Marshal(keyTwinUpdateFunc())
	context := contextFunc()

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		errWant  error
	}{
		{
			name:    "TestDealTwinSync(): Case 1: msg not Message type",
			context: &dtcontext.DTContext{},
			msg:     msgTypeErrorFunc(dttype.BaseMessage{EventID: "Event1"}),
			errWant: errors.New("msg not Message type"),
		},
		{
			name:    "TestDealTwinSync(): Case 2: invalid message content",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(dttype.BaseMessage{EventID: "Event1"}),
			errWant: errors.New("invalid message content"),
		},
		{
			name:    "TestDealTwinSync(): Case 3: Unmarshal update request body failed",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(content),
			errWant: errors.New("Update twin error, the update request body not have key:twin"),
		},
		{
			name:     "TestDealTwinSync(): Case 4: Success case",
			context:  &context,
			resource: "DeviceB",
			msg:      msgTypeFunc(contentKeyTwin),
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := dealTwinSync(test.context, test.resource, test.msg); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealTwinSync() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestDealTwinGet is function to test dealTwinGet
func TestDealTwinGet(t *testing.T) {
	contentKeyTwin, _ := json.Marshal(keyTwinUpdateFunc())
	context := contextFunc()

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		errWant  error
	}{
		{
			name:    "TestDealTwinGet(): Case 1: msg not Message type",
			context: &dtcontext.DTContext{},
			msg:     msgTypeErrorFunc(dttype.BaseMessage{EventID: "Event1"}),
			errWant: errors.New("msg not Message type"),
		},
		{
			name:    "TestDealTwinGet(): Case 2: invalid message content",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(dttype.BaseMessage{EventID: "Event1"}),
			errWant: errors.New("invalid message content"),
		},
		{
			name:     "TestDealTwinGet(): Case 3: Success case",
			context:  &context,
			resource: "DeviceB",
			msg:      msgTypeFunc(contentKeyTwin),
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := dealTwinGet(test.context, test.resource, test.msg); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealTwinGet() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestDealTwinUpdate is function to test dealTwinUpdate
func TestDealTwinUpdate(t *testing.T) {
	content, _ := json.Marshal(dttype.DeviceTwinUpdate{BaseMessage: dttype.BaseMessage{EventID: "Event1"}})
	context := contextFunc()

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		errWant  error
	}{
		{
			name:    "TestDealTwinUpdate(): Case 1: msg not Message type",
			context: &dtcontext.DTContext{},
			msg:     msgTypeErrorFunc(dttype.BaseMessage{EventID: "Event1"}),
			errWant: errors.New("msg not Message type"),
		},
		{
			name:    "TestDealTwinUpdate(): Case 2: invalid message content",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(dttype.BaseMessage{EventID: "Event1"}),
			errWant: errors.New("invalid message content"),
		},
		{
			name:     "TestDealTwinUpdate(): Case 3: Success case",
			context:  &context,
			resource: "DeviceB",
			msg:      msgTypeFunc(content),
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := dealTwinUpdate(test.context, test.resource, test.msg); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealTwinUpdate() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestUpdated is function to test Updated
func TestUpdated(t *testing.T) {
	content, _ := json.Marshal(dttype.DeviceTwinUpdate{BaseMessage: dttype.BaseMessage{EventID: "Event1"}})
	contentKeyTwin, _ := json.Marshal(keyTwinUpdateFunc())
	context := contextFunc()

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		deviceID string
		payload  []byte
	}{
		{
			name:    "TestUpdated(): Case1: Unmarshal update request body failed",
			context: &dtcontext.DTContext{},
			payload: content,
		},
		{
			name:    "TestUpdated(): Case2: Success case",
			context: &context,
			payload: contentKeyTwin,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Updated(test.context, test.deviceID, test.payload)
		})
	}
}

// TestDealDeviceTwin is function to test DealDeviceTwin
func TestDealDeviceTwin(t *testing.T) {
	mocksInit(t)
	str := "string"
	optionTrue := true
	value := "value"
	msgTwin := make(map[string]*dttype.MsgTwin)
	msgTwin["key1"] = &dttype.MsgTwin{
		Expected: expectedFunc(),
		Metadata: &dttype.TypeMetadata{"deleted"},
	}
	msgTwinValue := make(map[string]*dttype.MsgTwin)
	msgTwinValue["DeviceB"] = &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &value},
		Metadata: &dttype.TypeMetadata{"nil"},
	}
	contextDeviceB := contextFunc()
	twinDeviceB := make(map[string]*dttype.MsgTwin)
	twinDeviceB["DeviceB"] = &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &str},
		Optional: &optionTrue,
	}
	deviceBTwin := dttype.Device{Twin: twinDeviceB}
	contextDeviceB.DeviceList.Store("DeviceB", &deviceBTwin)

	contextDeviceA := contextFunc()
	twinDeviceA := make(map[string]*dttype.MsgTwin)
	twinDeviceA["DeviceA"] = &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &str},
		Actual:   &dttype.TwinValue{Value: &str},
		Optional: &optionTrue,
		Metadata: &dttype.TypeMetadata{"deleted"},
	}
	deviceATwin := dttype.Device{Twin: twinDeviceA}
	contextDeviceA.DeviceList.Store("DeviceA", &deviceATwin)

	contextDeviceC := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.Mutex{},
	}
	var testMutex sync.Mutex
	contextDeviceC.DeviceMutex.Store("DeviceC", &testMutex)
	twinDeviceC := make(map[string]*dttype.MsgTwin)
	twinDeviceC["DeviceC"] = &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &str},
		Optional: &optionTrue,
	}
	deviceCTwin := dttype.Device{Twin: twinDeviceC}
	contextDeviceC.DeviceList.Store("DeviceC", &deviceCTwin)

	tests := []struct {
		name             string
		context          *dtcontext.DTContext
		deviceID         string
		eventID          string
		msgTwin          map[string]*dttype.MsgTwin
		dealType         int
		errWant          error
		filterReturn     orm.QuerySeter
		insertReturnInt  int64
		insertReturnErr  error
		deleteReturnInt  int64
		deleteReturnErr  error
		updateReturnInt  int64
		updateReturnErr  error
		allReturnInt     int64
		allReturnErr     error
		queryTableReturn orm.QuerySeter
	}{
		{
			name:     "TestDealDeviceTwin(): Case 1: DeviceID does not exist",
			context:  &contextDeviceB,
			msgTwin:  msgTwin,
			dealType: 0,
			errWant:  errors.New("Update rejected due to the device is not existed"),
		},
		{
			name:     "TestDealDeviceTwin(): Case 2: msgTwin is nil",
			context:  &contextDeviceB,
			deviceID: "DeviceB",
			dealType: 0,
			errWant:  errors.New("Update twin error, the update request body not have key:twin"),
		},
		{
			name:             "TestDealDeviceTwin(): Case 3: dealTwinResult error",
			context:          &contextDeviceA,
			deviceID:         "DeviceB",
			msgTwin:          msgTwinValue,
			dealType:         0,
			errWant:          errors.New("The value type is not allowed"),
			filterReturn:     mockQuerySeter,
			allReturnInt:     int64(1),
			allReturnErr:     nil,
			queryTableReturn: mockQuerySeter,
		},
		{
			name:             "TestDealDeviceTwin(): Case 4: DeviceTwinTrans error",
			context:          &contextDeviceB,
			deviceID:         "DeviceB",
			msgTwin:          msgTwin,
			dealType:         0,
			errWant:          errors.New("Failed DB Operation"),
			filterReturn:     mockQuerySeter,
			insertReturnInt:  int64(1),
			insertReturnErr:  errors.New("Failed DB Operation"),
			deleteReturnInt:  int64(1),
			deleteReturnErr:  nil,
			updateReturnInt:  int64(1),
			updateReturnErr:  nil,
			allReturnInt:     int64(1),
			allReturnErr:     nil,
			queryTableReturn: mockQuerySeter,
		},
		{
			name:             "TestDealDeviceTwin(): Case 5: Success Case",
			context:          contextDeviceC,
			deviceID:         "DeviceC",
			msgTwin:          msgTwin,
			dealType:         0,
			errWant:          nil,
			filterReturn:     mockQuerySeter,
			allReturnInt:     int64(1),
			allReturnErr:     nil,
			queryTableReturn: mockQuerySeter,
		},
	}

	fakeDevice := new([]dtclient.Device)
	fakeDeviceArray := make([]dtclient.Device, 1)
	fakeDeviceArray[0] = dtclient.Device{ID: "DeviceB"}
	fakeDevice = &fakeDeviceArray

	fakeDeviceAttr := new([]dtclient.DeviceAttr)
	fakeDeviceAttrArray := make([]dtclient.DeviceAttr, 1)
	fakeDeviceAttrArray[0] = dtclient.DeviceAttr{DeviceID: "DeviceB"}
	fakeDeviceAttr = &fakeDeviceAttrArray

	fakeDeviceTwin := new([]dtclient.DeviceTwin)
	fakeDeviceTwinArray := make([]dtclient.DeviceTwin, 1)
	fakeDeviceTwinArray[0] = dtclient.DeviceTwin{DeviceID: "DeviceB"}
	fakeDeviceTwin = &fakeDeviceTwinArray

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if reflect.DeepEqual(test.name, "TestDealDeviceTwin(): Case 3: dealTwinResult error") {
				mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
				mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceAttr).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
				mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceTwin).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			}
			if reflect.DeepEqual(test.name, "TestDealDeviceTwin(): Case 4: DeviceTwinTrans error") {
				mockOrmer.EXPECT().Rollback().Return(nil).Times(5)
				mockOrmer.EXPECT().Commit().Return(nil).Times(0)
				mockOrmer.EXPECT().Begin().Return(nil).Times(5)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(0)
				mockOrmer.EXPECT().Insert(gomock.Any()).Return(test.insertReturnInt, test.insertReturnErr).Times(5)
				mockQuerySeter.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(0)
				mockQuerySeter.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(0)
				mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
				mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceAttr).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
				mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceTwin).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			}
			if reflect.DeepEqual(test.name, "TestDealDeviceTwin(): Case 5: Success Case") {
				mockOrmer.EXPECT().Rollback().Return(nil).Times(1)
				mockOrmer.EXPECT().Commit().Return(nil).Times(1)
				mockOrmer.EXPECT().Begin().Return(nil).Times(1)
				mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(0)
				mockOrmer.EXPECT().Insert(gomock.Any()).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockQuerySeter.EXPECT().Delete().Return(test.allReturnInt, test.allReturnErr).Times(0)
				mockQuerySeter.EXPECT().Update(gomock.Any()).Return(test.allReturnInt, test.allReturnErr).Times(1)
				mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(0)
			}
			if err := DealDeviceTwin(test.context, test.deviceID, test.eventID, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealDeviceTwin() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestDealUpdateResult is function to test dealUpdateResult
func TestDealUpdateResult(t *testing.T) {
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		deviceID string
		eventID  string
		code     int
		err      error
		payload  []byte
		errWant  error
	}{
		{
			name:    "TestDealUpdateResult(): Case 1: Error passed is nil",
			context: &dtcontext.DTContext{},
			code:    0,
			payload: []byte(""),
			errWant: errors.New("Not found chan to communicate"),
		},
		{
			name:     "TestDealUpdateResult(): Case 2: Parameter Reason Error",
			context:  &dtcontext.DTContext{},
			deviceID: "DeviceA",
			eventID:  "Event1",
			code:     dtcommon.BadRequestCode,
			err:      errors.New("Test Error"),
			payload:  []byte(""),
			errWant:  errors.New("Not found chan to communicate"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealUpdateResult(test.context, test.deviceID, test.eventID, test.code, test.err, test.payload); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealUpdateResult() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestDealDelta is function to test dealDelta
func TestDealDelta(t *testing.T) {
	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		deviceID string
		payload  []byte
		errWant  error
	}{
		{
			name:     "TestDealDelta(): Case 1",
			context:  &dtcontext.DTContext{},
			deviceID: "DeviceA",
			payload:  []byte(""),
			errWant:  errors.New("Not found chan to communicate"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealDelta(test.context, test.deviceID, test.payload); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealDelta() case failed: got = %+v, Want = %+v", err, test.errWant)
			}
		})
	}
}

// TestDealSyncResult is function to test dealSyncResult
func TestDealSyncResult(t *testing.T) {
	twinKey := make(map[string]*dttype.MsgTwin)
	tests := []struct {
		name        string
		context     *dtcontext.DTContext
		deviceID    string
		baseMessage dttype.BaseMessage
		twin        map[string]*dttype.MsgTwin
		errWant     error
	}{
		{
			name:        "TestDealSyncResult(): Case 1",
			context:     &dtcontext.DTContext{},
			deviceID:    "DeviceA",
			baseMessage: dttype.BaseMessage{},
			twin:        twinKey,
			errWant:     errors.New("Not found chan to communicate"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealSyncResult(test.context, test.deviceID, test.baseMessage, test.twin); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealSyncResult() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestDealDocument is function to test dealDocument
func TestDealDocument(t *testing.T) {
	twinDocKey := make(map[string]*dttype.TwinDoc)
	tests := []struct {
		name         string
		context      *dtcontext.DTContext
		deviceID     string
		baseMessage  dttype.BaseMessage
		twinDocument map[string]*dttype.TwinDoc
		errWant      error
	}{
		{
			name:         "TestDealDocument(): Case 1",
			context:      &dtcontext.DTContext{},
			deviceID:     "DeviceA",
			baseMessage:  dttype.BaseMessage{},
			twinDocument: twinDocKey,
			errWant:      errors.New("Not found chan to communicate"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealDocument(test.context, test.deviceID, test.baseMessage, test.twinDocument); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealDocument() case failed: got = %v, Want = %v", err, test.errWant)
			}
		})
	}
}

// TestDealGetTwin is function to test DealGetTwin
func TestDealGetTwin(t *testing.T) {
	context := contextFunc()
	var baseMessage dttype.BaseMessage
	bytesBaseMessage, _ := json.Marshal(baseMessage)

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		deviceID string
		payload  []byte
		errWant  error
	}{
		{
			name:     "TestDealGetTwin(): Case1: Success Case",
			context:  &context,
			deviceID: "DeviceB",
			payload:  []byte(""),
			errWant:  errors.New("Not found chan to communicate"),
		},
		{
			name:     "TestDealGetTwin(): Case 2",
			context:  &context,
			deviceID: "DeviceB",
			payload:  bytesBaseMessage,
			errWant:  errors.New("Not found chan to communicate"),
		},
		{
			name:     "TestDealGetTwin(): Case 3",
			context:  &context,
			deviceID: "DeviceC",
			payload:  bytesBaseMessage,
			errWant:  errors.New("Not found chan to communicate"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DealGetTwin(tt.context, tt.deviceID, tt.payload); !reflect.DeepEqual(err, tt.errWant) {
				t.Errorf("DTManager.TestDealGetTwin() case failed: got = %v, want = %v", err, tt.errWant)
			}
		})
	}
}

// TestDealVersion is function to test dealVersion
func TestDealVersion(t *testing.T) {
	twinCloudEdgeVersion := dttype.TwinVersion{
		CloudVersion: 1,
		EdgeVersion:  1,
	}
	twinEdgeVersion := dttype.TwinVersion{
		CloudVersion: 0,
		EdgeVersion:  1,
	}
	twinCloudVersion := dttype.TwinVersion{
		CloudVersion: 1,
		EdgeVersion:  0,
	}

	tests := []struct {
		name       string
		version    *dttype.TwinVersion
		reqVersion *dttype.TwinVersion
		dealType   int
		boolWant   bool
		errWant    error
	}{
		{
			name:       "TestDealVersion(): Case 1: Success Case",
			version:    &dttype.TwinVersion{},
			reqVersion: &dttype.TwinVersion{},
			dealType:   0,
			boolWant:   true,
			errWant:    nil,
		},
		{
			name:     "TestDealVersion(): Case 2",
			version:  &dttype.TwinVersion{},
			dealType: 1,
			boolWant: false,
			errWant:  errors.New("Version not allowed be nil while syncing"),
		},
		{
			name:     "TestDealVersion(): Case 3",
			version:  &dttype.TwinVersion{},
			dealType: 3,
			boolWant: true,
			errWant:  nil,
		},
		{
			name:       "TestDealVersion(): Case 4",
			version:    &dttype.TwinVersion{},
			reqVersion: &dttype.TwinVersion{},
			dealType:   1,
			boolWant:   true,
			errWant:    nil,
		},
		{
			name:       "TestDealVersion(): Case 5",
			version:    &twinCloudEdgeVersion,
			reqVersion: &twinEdgeVersion,
			dealType:   1,
			boolWant:   false,
			errWant:    errors.New("Version not allowed"),
		},
		{
			name:       "TestDealVersion(): Case 6",
			version:    &twinCloudEdgeVersion,
			reqVersion: &twinCloudVersion,
			dealType:   1,
			boolWant:   false,
			errWant:    errors.New("Not allowed to sync due to version conflict"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := dealVersion(test.version, test.reqVersion, test.dealType)
			if !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealVersion() case failed: got = %v, Want = %v", err, test.errWant)
				return
			}
			if !reflect.DeepEqual(got, test.boolWant) {
				t.Errorf("DTManager.TestDealVersion() case failed: got = %v, want %v", got, test.boolWant)
			}
		})
	}
}

// TestDealTwinDelete is function to test dealTwinDelete
func TestDealTwinDelete(t *testing.T) {
	optionTrue := true
	optionFalse := false
	str := "string"
	doc := make(map[string]*dttype.TwinDoc)
	doc["key1"] = &dttype.TwinDoc{}
	sync := make(map[string]*dttype.MsgTwin)
	sync["key1"] = &dttype.MsgTwin{
		Expected:        expectedFunc(),
		Actual:          actualFunc(),
		Optional:        &optionTrue,
		Metadata:        &dttype.TypeMetadata{"deleted"},
		ExpectedVersion: &dttype.TwinVersion{},
		ActualVersion:   &dttype.TwinVersion{},
	}
	result := make(map[string]*dttype.MsgTwin)
	result["key1"] = &dttype.MsgTwin{}

	tests := []struct {
		name         string
		returnResult *dttype.DealTwinResult
		deviceID     string
		key          string
		twin         *dttype.MsgTwin
		msgTwin      *dttype.MsgTwin
		dealType     int
		errWant      error
	}{
		{
			name:         "TestDealTwinDelete(): Case 1: msgTwin nil",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinDelete(): Case 2",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionFalse,
				Metadata:        &dttype.TypeMetadata{"string"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinDelete(): Case 3",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"string"},
				ExpectedVersion: &dttype.TwinVersion{CloudVersion: 1},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionFalse,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{CloudVersion: 0},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinDelete(): Case 4",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"string"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinDelete(): Case 5",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional:      &optionTrue,
				Metadata:      &dttype.TypeMetadata{"string"},
				ActualVersion: &dttype.TwinVersion{CloudVersion: 1},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionFalse,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{CloudVersion: 0},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinDelete(): Case 6",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"string"},
				ExpectedVersion: &dttype.TwinVersion{},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionFalse,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealTwinDelete(test.returnResult, test.deviceID, test.key, test.twin, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealTwinDelete() case failed: got = %+v, Want = %+v", err, test.errWant)
			}
		})
	}
}

// TestIsTwinValueDiff is function to test isTwinValueDiff
func TestIsTwinValueDiff(t *testing.T) {
	tests := []struct {
		name     string
		twin     *dttype.MsgTwin
		msgTwin  *dttype.MsgTwin
		dealType int
		boolWant bool
		errWant  error
	}{
		{
			name:     "TestIsTwinValueDiff(): Case 1",
			twin:     &dttype.MsgTwin{Expected: expectedFunc(), Metadata: &dttype.TypeMetadata{"int"}},
			msgTwin:  &dttype.MsgTwin{Expected: expectedFunc(), Metadata: &dttype.TypeMetadata{"string"}},
			dealType: 0,
			boolWant: false,
			errWant:  errors.New("The value is not int"),
		},
		{
			name:     "TestIsTwinValueDiff(): Case 2",
			twin:     &dttype.MsgTwin{Expected: expectedFunc(), Metadata: &dttype.TypeMetadata{"string"}},
			msgTwin:  &dttype.MsgTwin{Expected: expectedFunc(), Metadata: &dttype.TypeMetadata{"string"}},
			dealType: 0,
			boolWant: true,
			errWant:  nil,
		},
		{
			name:     "TestIsTwinValueDiff(): Case 3",
			twin:     &dttype.MsgTwin{Metadata: &dttype.TypeMetadata{"deleted"}},
			msgTwin:  &dttype.MsgTwin{Actual: actualFunc(), Metadata: &dttype.TypeMetadata{"deleted"}},
			dealType: 1,
			boolWant: true,
			errWant:  nil,
		},
		{
			name:     "TestIsTwinValueDiff(): Case 4",
			twin:     &dttype.MsgTwin{Metadata: &dttype.TypeMetadata{"deleted"}},
			msgTwin:  &dttype.MsgTwin{Metadata: &dttype.TypeMetadata{"deleted"}},
			dealType: 1,
			boolWant: false,
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := isTwinValueDiff(test.twin, test.msgTwin, test.dealType)
			if !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestIsTwinValueDiff() case failed: got = %v, Want = %v", err, test.errWant)
				return
			}
			if !reflect.DeepEqual(got, test.boolWant) {
				t.Errorf("DTManager.TestIsTwinValueDiff() case failed: got = %v, want %v", got, test.boolWant)
			}
		})
	}
}

// TestDealTwinCompare is function to test dealTwinCompare
func TestDealTwinCompare(t *testing.T) {
	optionTrue := true
	optionFalse := false
	str := "string"
	doc := make(map[string]*dttype.TwinDoc)
	doc["key1"] = &dttype.TwinDoc{}
	sync := make(map[string]*dttype.MsgTwin)
	sync["key1"] = &dttype.MsgTwin{
		Expected:        expectedFunc(),
		Actual:          actualFunc(),
		Optional:        &optionTrue,
		Metadata:        &dttype.TypeMetadata{"deleted"},
		ExpectedVersion: &dttype.TwinVersion{},
		ActualVersion:   &dttype.TwinVersion{},
	}
	result := make(map[string]*dttype.MsgTwin)
	result["key1"] = &dttype.MsgTwin{}

	tests := []struct {
		name         string
		returnResult *dttype.DealTwinResult
		deviceID     string
		key          string
		twin         *dttype.MsgTwin
		msgTwin      *dttype.MsgTwin
		dealType     int
		errWant      error
	}{
		{
			name:         "TestDealTwinCompare(): Case 1: msgTwin nil",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"deleted"},
			},
			dealType: 0,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinCompare(): Case 2",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Expected: &dttype.TwinValue{Value: &str},
				Actual:   &dttype.TwinValue{Value: &str},
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"deleted"},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:      &dttype.TwinValue{Value: &str},
				Actual:        &dttype.TwinValue{Value: &str},
				Optional:      &optionFalse,
				Metadata:      &dttype.TypeMetadata{"int"},
				ActualVersion: &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  errors.New("The value is not int"),
		},
		{
			name:         "TestDealTwinCompare(): Case 3",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Expected: &dttype.TwinValue{Value: &str},
				Actual:   &dttype.TwinValue{Value: &str},
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"deleted"},
			},
			msgTwin: &dttype.MsgTwin{
				Actual:   &dttype.TwinValue{Value: &str},
				Optional: &optionFalse,
				Metadata: &dttype.TypeMetadata{"int"},
			},
			dealType: 0,
			errWant:  errors.New("The value is not int"),
		},
		{
			name:         "TestDealTwinCompare(): Case 4",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"deleted"},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:      &dttype.TwinValue{Value: &str},
				Actual:        &dttype.TwinValue{Value: &str},
				Optional:      &optionFalse,
				Metadata:      &dttype.TypeMetadata{"string"},
				ActualVersion: &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinCompare(): Case 5",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Expected: &dttype.TwinValue{Value: &str},
				Actual:   &dttype.TwinValue{Value: &str},
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"string"},
			},
			msgTwin: &dttype.MsgTwin{
				Expected:      &dttype.TwinValue{Value: &str},
				Actual:        &dttype.TwinValue{Value: &str},
				Optional:      &optionFalse,
				Metadata:      &dttype.TypeMetadata{"int"},
				ActualVersion: &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinCompare(): Case 6",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Expected: &dttype.TwinValue{Value: &str},
				Actual:   &dttype.TwinValue{Value: &str},
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"deleted"},
			},
			msgTwin: &dttype.MsgTwin{
				Actual:   &dttype.TwinValue{Value: &str},
				Optional: &optionFalse,
				Metadata: &dttype.TypeMetadata{"string"},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinCompare(): Case 7",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twin: &dttype.MsgTwin{
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{"deleted"},
			},
			msgTwin: &dttype.MsgTwin{
				Optional:      &optionFalse,
				ActualVersion: &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealTwinCompare(test.returnResult, test.deviceID, test.key, test.twin, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealTwinCompare() case failed: got = %+v, Want = %+v", err, test.errWant)
			}
		})
	}
}

// TestDealTwinAdd is function to test dealTwinAdd
func TestDealTwinAdd(t *testing.T) {
	optionTrue := true
	str := "string"
	doc := make(map[string]*dttype.TwinDoc)
	doc["key1"] = &dttype.TwinDoc{}
	sync := make(map[string]*dttype.MsgTwin)
	sync["key1"] = &dttype.MsgTwin{}
	result := make(map[string]*dttype.MsgTwin)
	result["key1"] = &dttype.MsgTwin{}

	twinDelete := make(map[string]*dttype.MsgTwin)
	twinDelete["key1"] = &dttype.MsgTwin{Metadata: &dttype.TypeMetadata{"deleted"}}
	twinInt := make(map[string]*dttype.MsgTwin)
	twinInt["key1"] = &dttype.MsgTwin{Metadata: &dttype.TypeMetadata{"int"}}

	tests := []struct {
		name         string
		returnResult *dttype.DealTwinResult
		deviceID     string
		key          string
		twins        map[string]*dttype.MsgTwin
		msgTwin      *dttype.MsgTwin
		dealType     int
		errWant      error
	}{
		{
			name:         "TestDealTwinAdd(): Case 1: msgTwin nil",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			dealType:     0,
			errWant:      errors.New("The request body is wrong"),
		},
		{
			name:         "TestDealTwinAdd(): Case 2",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinAdd(): Case 3",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Expected:      &dttype.TwinValue{Value: &str},
				Actual:        &dttype.TwinValue{Value: &str},
				Optional:      &optionTrue,
				Metadata:      &dttype.TypeMetadata{"deleted"},
				ActualVersion: &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinAdd(): Case 4",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"int"},
				ExpectedVersion: &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  errors.New("The value is not int"),
		},
		{
			name:         "TestDealTwinAdd(): Case 5",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"int"},
				ExpectedVersion: &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinAdd(): Case 6",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Expected:        &dttype.TwinValue{Value: &str},
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"deleted"},
				ExpectedVersion: &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinAdd(): Case 7",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"int"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  errors.New("The value is not int"),
		},
		{
			name:         "TestDealTwinAdd(): Case 8",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinDelete,
			msgTwin: &dttype.MsgTwin{
				Actual:          &dttype.TwinValue{Value: &str},
				Optional:        &optionTrue,
				Metadata:        &dttype.TypeMetadata{"int"},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 1,
			errWant:  nil,
		},
		{
			name:         "TestDealTwinAdd(): Case 9",
			returnResult: &dttype.DealTwinResult{Document: doc, SyncResult: sync, Result: result},
			deviceID:     "DeviceA",
			key:          "key1",
			twins:        twinInt,
			msgTwin: &dttype.MsgTwin{
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: 0,
			errWant:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealTwinAdd(test.returnResult, test.deviceID, test.key, test.twins, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.errWant) {
				t.Errorf("DTManager.TestDealTwinAdd() case failed: got = %+v, Want = %+v", err, test.errWant)
			}
		})
	}
}

// TestDealMsgTwin is function to test DealMsgTwin
func TestDealMsgTwin(t *testing.T) {
	value := "value"
	str := "string"
	optionTrue := true
	optionFalse := false
	add := make([]dtclient.DeviceTwin, 0)
	deletes := make([]dtclient.DeviceDelete, 0)
	update := make([]dtclient.DeviceTwinUpdate, 0)
	result := make(map[string]*dttype.MsgTwin)
	syncResult := make(map[string]*dttype.MsgTwin)
	syncResultDevice := make(map[string]*dttype.MsgTwin)
	syncResultDevice["DeviceA"] = &dttype.MsgTwin{}
	document := make(map[string]*dttype.TwinDoc)
	documentDevice := make(map[string]*dttype.TwinDoc)
	documentDevice["DeviceA"] = &dttype.TwinDoc{LastState: nil}
	documentDeviceTwin := make(map[string]*dttype.TwinDoc)
	documentDeviceTwin["DeviceA"] = &dttype.TwinDoc{LastState: &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &str},
		Actual:   &dttype.TwinValue{Value: &str},
		Optional: &optionTrue,
		Metadata: &dttype.TypeMetadata{"deleted"},
	},
	}

	msgTwin := make(map[string]*dttype.MsgTwin)
	msgTwin["DeviceB"] = &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &value},
		Metadata: &dttype.TypeMetadata{"nil"},
	}
	msgTwinDevice := make(map[string]*dttype.MsgTwin)
	msgTwinDevice["DeviceA"] = nil
	msgTwinDeviceTwin := make(map[string]*dttype.MsgTwin)
	msgTwinDeviceTwin["DeviceA"] = &dttype.MsgTwin{
		Expected:      &dttype.TwinValue{Value: &str},
		Actual:        &dttype.TwinValue{Value: &str},
		Optional:      &optionFalse,
		Metadata:      &dttype.TypeMetadata{"int"},
		ActualVersion: &dttype.TwinVersion{},
	}

	context := contextFunc()
	twin := make(map[string]*dttype.MsgTwin)
	twin["DeviceA"] = &dttype.MsgTwin{
		Expected: &dttype.TwinValue{Value: &str},
		Actual:   &dttype.TwinValue{Value: &str},
		Optional: &optionTrue,
		Metadata: &dttype.TypeMetadata{"deleted"},
	}
	device := dttype.Device{Twin: twin}
	context.DeviceList.Store("DeviceA", &device)

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		deviceID string
		msgTwins map[string]*dttype.MsgTwin
		dealType int
		want     dttype.DealTwinResult
	}{
		{
			name:     "TestDealMsgTwin(): Case1: invalid device id",
			context:  &context,
			deviceID: "DeviceC",
			msgTwins: msgTwin,
			dealType: 0,
			want: dttype.DealTwinResult{
				Add:        add,
				Delete:     deletes,
				Update:     update,
				Result:     result,
				SyncResult: syncResult,
				Document:   document,
				Err:        errors.New("invalid device id"),
			},
		},
		{
			name:     "TestDealMsgTwin(): Case 2: dealTwinAdd return error",
			context:  &context,
			deviceID: "DeviceB",
			msgTwins: msgTwin,
			dealType: 0,
			want: dttype.DealTwinResult{
				Add:        add,
				Delete:     deletes,
				Update:     update,
				Result:     result,
				SyncResult: syncResult,
				Document:   document,
				Err:        errors.New("The value type is not allowed"),
			},
		},
		{
			name:     "TestDealMsgTwin(): Case 3: dealTwinCompare return error",
			context:  &context,
			deviceID: "DeviceA",
			msgTwins: msgTwinDeviceTwin,
			dealType: 0,
			want: dttype.DealTwinResult{
				Add:        add,
				Delete:     deletes,
				Update:     update,
				Result:     result,
				SyncResult: syncResultDevice,
				Document:   documentDevice,
				Err:        errors.New("The value is not int"),
			},
		},
		{
			name:     "TestDealMsgTwin(): Case 4: Success case",
			context:  &context,
			deviceID: "DeviceA",
			msgTwins: msgTwinDevice,
			dealType: 0,
			want: dttype.DealTwinResult{
				Add:        add,
				Delete:     deletes,
				Update:     update,
				Result:     result,
				SyncResult: syncResultDevice,
				Document:   documentDeviceTwin,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DealMsgTwin(tt.context, tt.deviceID, tt.msgTwins, tt.dealType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DTManager.DealMsgTwin() case failed: got = %+v, want = %+v", got, tt.want)
			}
		})
	}
}
