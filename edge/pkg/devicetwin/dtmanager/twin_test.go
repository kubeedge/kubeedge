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
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var (
	deviceA = "default/DeviceA"
	deviceB = "default/DeviceB"
	deviceC = "default/DeviceC"
	event1  = "Event1"
	key1    = "key1"

	typeDeleted = "deleted"
	typeInt     = "int"
	typeString  = "string"

	valueType = "value"
)

// sendMsg sends message to receiverChannel and heartbeatChannel
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

// receiveMsg receives message from the commChannel
func receiveMsg(commChannel chan interface{}, message *dttype.DTMessage) {
	msg, ok := <-commChannel
	if !ok {
		klog.Errorf("No message received from communication channel")
		return
	}
	*message = *msg.(*dttype.DTMessage)
}

// twinValueFunc returns a new TwinValue
func twinValueFunc() v1alpha2.TwinProperty {
	var twinValue v1alpha2.TwinProperty
	value := valueType
	twinValue.Value = value
	twinValue.Metadata = make(map[string]string)
	twinValue.Metadata["timestamp"] = strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	return twinValue
}

// keyTwinUpdateFunc returns a new DeviceTwinUpdate
func keyTwinUpdateFunc() v1alpha2.Device {
	var device v1alpha2.Device
	device.Status.Twins = make([]v1alpha2.Twin, 0)
	twin := v1alpha2.Twin{
		PropertyName: key1,
		Desired:      twinValueFunc(),
		Reported:     twinValueFunc(),
	}
	device.Status.Twins = append(device.Status.Twins, twin)
	return device
}

// twinWorkerFunc returns a new TwinWorker
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

// contextFunc returns a new DTContext
func contextFunc(deviceID string) dtcontext.DTContext {
	context := dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
	}
	var testMutex sync.Mutex
	context.DeviceMutex.Store(deviceID, &testMutex)
	var device v1alpha2.Device
	context.DeviceList.Store(deviceID, &device)
	return context
}

// msgTypeFunc returns a new Message
func msgTypeFunc(content interface{}) *model.Message {
	return &model.Message{
		Content: content,
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
		Mutex:         &sync.RWMutex{},
		CommChan:      commChan,
		ModulesHealth: &sync.Map{},
	}
	var testMutex sync.Mutex
	context.DeviceMutex.Store(deviceB, &testMutex)

	device := v1alpha2.Device{}
	device.Namespace = "default"
	device.Name = deviceB
	device.Status.Twins = keyTwinUpdate.Status.Twins

	context.DeviceList.Store(deviceB, &device)

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
			name:        "TestStart(): Case 1: ReceiverChan case when error is nil",
			tw:          twinWorkerFunc(receiverChannel, nil, nil, context, ""),
			actionType:  dtcommon.TwinGet,
			contentType: contentKeyTwin,
		},
		{
			name:        "TestStart(): Case 2: ReceiverChan case error log; TwinModule deal event failed, not found callback",
			tw:          twinWorkerFunc(receiverChannel, nil, nil, context, ""),
			actionType:  dtcommon.SendToEdge,
			contentType: contentKeyTwin,
		},
		{
			name:       "TestStart(): Case 3: ReceiverChan case error log; TwinModule deal event failed",
			tw:         twinWorkerFunc(receiverChannel, nil, nil, context, ""),
			actionType: dtcommon.TwinGet,
		},
		{
			name:    "TestStart(): Case 4: HeartBeatChan case when error is nil",
			tw:      twinWorkerFunc(nil, nil, heartbeatChannel, context, "Group1"),
			msgType: msgHeartPing,
		},
		{
			name:    "TestStart(): Case 5: HeartBeatChan case when error is not nil",
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
	var ormerMock *beego.MockOrmer
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	dbm.DBAccess = ormerMock

	content, _ := json.Marshal(v1alpha2.Device{})
	contentKeyTwin, _ := json.Marshal(keyTwinUpdateFunc())
	context := contextFunc(deviceB)

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		err      error
	}{
		{
			name:    "TestDealTwinSync(): Case 1: msg not Message type",
			context: &dtcontext.DTContext{},
			msg: model.Message{
				Content: dttype.BaseMessage{EventID: event1},
			},
			err: errors.New("msg not Message type"),
		},
		{
			name:    "TestDealTwinSync(): Case 2: invalid message content",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(dttype.BaseMessage{EventID: event1}),
			err:     errors.New("invalid message content"),
		},
		{
			name:    "TestDealTwinSync(): Case 3: Unmarshal update request body failed",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(content),
			err:     dttype.ErrorUpdate,
		},
		{
			name:     "TestDealTwinSync(): Case 4: Success case",
			context:  &context,
			resource: deviceB,
			msg:      msgTypeFunc(contentKeyTwin),
			err:      nil,
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if index == 3 {
				ormerMock.EXPECT().Begin().Return(nil)
				ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
				ormerMock.EXPECT().Commit().Return(nil)
			}
			if _, err := dealTwinSync(test.context, test.resource, test.msg); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealTwinSync() case failed: got = %v, Want = %v", err, test.err)
			}
		})
	}
}

// TestDealTwinGet is function to test dealTwinGet
func TestDealTwinGet(t *testing.T) {
	contentKeyTwin, _ := json.Marshal(keyTwinUpdateFunc())
	context := contextFunc(deviceB)

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		err      error
	}{
		{
			name:    "TestDealTwinGet(): Case 1: msg not Message type",
			context: &dtcontext.DTContext{},
			msg: model.Message{
				Content: dttype.BaseMessage{EventID: event1},
			},
			err: errors.New("msg not Message type"),
		},
		{
			name:    "TestDealTwinGet(): Case 2: invalid message content",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(dttype.BaseMessage{EventID: event1}),
			err:     errors.New("invalid message content"),
		},
		{
			name:     "TestDealTwinGet(): Case 3: Success; Unmarshal twin info fails in DealGetTwin()",
			context:  &context,
			resource: deviceB,
			msg:      msgTypeFunc([]byte("")),
			err:      nil,
		},
		{
			name:     "TestDealTwinGet(): Case 4: Success; Device not found while getting twin in DealGetTwin()",
			context:  &context,
			resource: deviceB,
			msg:      msgTypeFunc(contentKeyTwin),
			err:      nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := dealTwinGet(test.context, test.resource, test.msg); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealTwinGet() case failed: got = %v, Want = %v", err, test.err)
			}
		})
	}
}

// TestDealTwinUpdate is function to test dealTwinUpdate
func TestDealTwinUpdate(t *testing.T) {
	var ormerMock *beego.MockOrmer
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	dbm.DBAccess = ormerMock

	content, _ := json.Marshal(v1alpha2.Device{})
	contentKeyTwin, _ := json.Marshal(keyTwinUpdateFunc())
	context := contextFunc(deviceB)

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		resource string
		msg      interface{}
		err      error
		times    int
	}{
		{
			name:    "TestDealTwinUpdate(): Case 1: msg not Message type",
			context: &dtcontext.DTContext{},
			msg: model.Message{
				Content: dttype.BaseMessage{EventID: event1},
			},
			err: errors.New("msg not Message type"),
		},
		{
			name:    "TestDealTwinUpdate(): Case 2: invalid message content",
			context: &dtcontext.DTContext{},
			msg:     msgTypeFunc(dttype.BaseMessage{EventID: event1}),
			err:     errors.New("invalid message content"),
		},
		{
			name:     "TestDealTwinUpdate(): Case 3: Success; Unmarshal update request body fails in Updated()",
			context:  &context,
			resource: deviceB,
			msg:      msgTypeFunc(content),
			err:      nil,
		},
		{
			name:     "TestDealTwinUpdate(): Case 4: Success; Begin to update twin of the device in Updated()",
			context:  &context,
			resource: deviceB,
			msg:      msgTypeFunc(contentKeyTwin),
			err:      nil,
			times:    3,
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if index == 3 {
				ormerMock.EXPECT().Begin().Return(nil)
				ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
				ormerMock.EXPECT().Commit().Return(nil)
			}
			if _, err := dealTwinUpdate(test.context, test.resource, test.msg); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealTwinUpdate() case failed: got = %v, Want = %v", err, test.err)
			}
		})
	}
}

// TestDealDeviceTwin is function to test DealDeviceTwin
func TestDealDeviceTwin(t *testing.T) {
	var mockOrmer *beego.MockOrmer
	var mockQuerySeter *beego.MockQuerySeter
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockOrmer = beego.NewMockOrmer(mockCtrl)
	mockQuerySeter = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = mockOrmer

	str := typeString

	msgTwin := make([]v1alpha2.Twin, 1)
	msgTwin[0] = v1alpha2.Twin{
		PropertyName: key1,
		Desired:      twinValueFunc(),
	}
	msgTwin[0].Desired.Metadata["type"] = typeDeleted

	contextDeviceB := contextFunc(deviceB)
	twinDeviceB := make([]v1alpha2.Twin, 1)
	twinDeviceB[0] = v1alpha2.Twin{
		PropertyName: deviceB,
		Desired: v1alpha2.TwinProperty{
			Value: str,
		},
	}

	b := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: twinDeviceB,
		},
	}
	contextDeviceB.DeviceList.Store(deviceB, &b)

	contextDeviceC := dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
	}
	var testMutex sync.Mutex
	contextDeviceC.DeviceMutex.Store(deviceC, &testMutex)
	twinDeviceC := make([]v1alpha2.Twin, 1)
	twinDeviceC[0] = v1alpha2.Twin{
		PropertyName: deviceC,
		Desired: v1alpha2.TwinProperty{
			Value: str,
		},
	}

	deviceCTwin := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: twinDeviceC,
		},
	}
	contextDeviceC.DeviceList.Store(deviceC, &deviceCTwin)

	tests := []struct {
		name             string
		context          *dtcontext.DTContext
		deviceID         string
		eventID          string
		msgTwin          []v1alpha2.Twin
		dealType         int
		err              error
		filterReturn     orm.QuerySeter
		allReturnInt     int64
		allReturnErr     error
		queryTableReturn orm.QuerySeter

		rollbackNums int
		beginNums    int
		commitNums   int
		filterNums   int
		insertNums   int
		deleteNums   int
		updateNums   int
		queryNums    int
	}{
		{
			name:         "TestDealDeviceTwin(): Case 1: msgTwin is nil",
			context:      &contextDeviceB,
			deviceID:     deviceB,
			dealType:     RestDealType,
			err:          dttype.ErrorUpdate,
			rollbackNums: 0,
			beginNums:    1,
			commitNums:   1,
			filterNums:   0,
			insertNums:   1,
			deleteNums:   0,
			updateNums:   0,
			queryNums:    0,
		},
		{
			name:             "TestDealDeviceTwin(): Case 2: Success Case",
			context:          &contextDeviceC,
			deviceID:         deviceC,
			msgTwin:          msgTwin,
			dealType:         RestDealType,
			err:              nil,
			filterReturn:     mockQuerySeter,
			allReturnInt:     int64(1),
			allReturnErr:     nil,
			queryTableReturn: mockQuerySeter,
			rollbackNums:     0,
			beginNums:        1,
			commitNums:       1,
			filterNums:       0,
			insertNums:       1,
			deleteNums:       0,
			updateNums:       0,
			queryNums:        0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockOrmer.EXPECT().Rollback().Return(nil).Times(test.rollbackNums)
			mockOrmer.EXPECT().Begin().Return(nil).MaxTimes(test.beginNums)
			mockOrmer.EXPECT().Commit().Return(nil).MaxTimes(test.commitNums)
			mockOrmer.EXPECT().Insert(gomock.Any()).Return(test.allReturnInt, test.allReturnErr).MaxTimes(test.insertNums)
			mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.filterNums)
			mockQuerySeter.EXPECT().Delete().Return(test.allReturnInt, test.allReturnErr).Times(test.deleteNums)
			mockQuerySeter.EXPECT().Update(gomock.Any()).Return(test.allReturnInt, test.allReturnErr).Times(test.updateNums)
			mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.queryNums)
			if err := DealDeviceTwin(test.context, test.deviceID, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealDeviceTwin() case failed: got = %v, Want = %v", err, test.err)
			}
		})
	}
}

// TestDealDeviceTwinResult is function to test DealDeviceTwin when dealTwinResult.Err is not nil
func TestDealDeviceTwinResult(t *testing.T) {
	var mockOrmer *beego.MockOrmer
	var mockQuerySeter *beego.MockQuerySeter
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockOrmer = beego.NewMockOrmer(mockCtrl)
	mockQuerySeter = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = mockOrmer

	str := typeString
	//optionTrue := true
	value := valueType
	msgTwinValue := make([]v1alpha2.Twin, 1)
	msgTwinValue[0] = v1alpha2.Twin{
		PropertyName: deviceB,
		Desired: v1alpha2.TwinProperty{
			Value:    value,
			Metadata: make(map[string]string),
		},
	}
	msgTwinValue[0].Desired.Metadata["type"] = "nil"

	contextDeviceA := contextFunc(deviceB)
	twinDeviceA := make([]v1alpha2.Twin, 1)
	twinDeviceA[0] = v1alpha2.Twin{
		PropertyName: deviceA,
		Desired: v1alpha2.TwinProperty{
			Value:    str,
			Metadata: make(map[string]string),
		},
		Reported: v1alpha2.TwinProperty{
			Value: str,
		},
	}
	twinDeviceA[0].Desired.Metadata["type"] = typeDeleted

	deviceATwin := &v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: twinDeviceA,
		},
	}
	contextDeviceA.DeviceList.Store(deviceA, deviceATwin)

	tests := []struct {
		name             string
		context          *dtcontext.DTContext
		deviceID         string
		eventID          string
		msgTwin          []v1alpha2.Twin
		dealType         int
		err              error
		filterReturn     orm.QuerySeter
		allReturnInt     int64
		allReturnErr     error
		queryTableReturn orm.QuerySeter
	}{
		{
			name:             "TestDealDeviceTwinResult(): dealTwinResult error",
			context:          &contextDeviceA,
			deviceID:         deviceB,
			msgTwin:          msgTwinValue,
			dealType:         RestDealType,
			err:              nil,
			filterReturn:     mockQuerySeter,
			allReturnInt:     int64(1),
			allReturnErr:     nil,
			queryTableReturn: mockQuerySeter,
		},
	}

	fakeDevice := new([]dtclient.Device)
	fakeDeviceArray := make([]dtclient.Device, 1)
	fakeDeviceArray[0] = dtclient.Device{Name: "DeviceB", Namespace: "default"}
	fakeDevice = &fakeDeviceArray

	fakeDeviceTwin := new([]dtclient.DeviceTwin)
	fakeDeviceTwinArray := make([]dtclient.DeviceTwin, 1)
	fakeDeviceTwinArray[0] = dtclient.DeviceTwin{DeviceName: "DeviceB", DeviceNamespace: "default"}
	fakeDeviceTwin = &fakeDeviceTwinArray

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(0)
			mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(0)
			mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(0)
			mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceTwin).Return(test.allReturnInt, test.allReturnErr).Times(0)
			mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(0)
			mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(0)
			mockOrmer.EXPECT().Begin().Return(nil)
			mockOrmer.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
			mockOrmer.EXPECT().Commit().Return(nil)
			if err := DealDeviceTwin(test.context, test.deviceID, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealDeviceTwinResult() case failed: got = %v, Want = %v", err, test.err)
			}
		})
	}
}

// TestDealDeviceTwinTrans is function to test DealDeviceTwin when DeviceTwinTrans() return error
func TestDealDeviceTwinTrans(t *testing.T) {
	var mockOrmer *beego.MockOrmer
	var mockQuerySeter *beego.MockQuerySeter
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockOrmer = beego.NewMockOrmer(mockCtrl)
	mockQuerySeter = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = mockOrmer

	str := typeString
	//optionTrue := true
	msgTwin := make([]v1alpha2.Twin, 1)
	msgTwin[0] = v1alpha2.Twin{
		PropertyName: key1,
		Desired:      twinValueFunc(),
	}
	msgTwin[0].Desired.Metadata["type"] = typeDeleted

	contextDeviceB := contextFunc(deviceB)
	twinDeviceB := make([]v1alpha2.Twin, 1)
	twinDeviceB[0] = v1alpha2.Twin{
		PropertyName: deviceB,
		Desired: v1alpha2.TwinProperty{
			Value: str,
		},
	}
	deviceBTwin := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: twinDeviceB,
		},
	}
	contextDeviceB.DeviceList.Store(deviceB, &deviceBTwin)

	tests := []struct {
		name             string
		context          *dtcontext.DTContext
		deviceID         string
		eventID          string
		msgTwin          []v1alpha2.Twin
		dealType         int
		err              error
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
			name:             "TestDealDeviceTwinTrans(): DeviceTwinTrans error",
			context:          &contextDeviceB,
			deviceID:         deviceB,
			msgTwin:          msgTwin,
			dealType:         RestDealType,
			err:              errors.New("Failed DB Operation"),
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
	}

	fakeDevice := new([]dtclient.Device)
	fakeDeviceArray := make([]dtclient.Device, 1)
	fakeDeviceArray[0] = dtclient.Device{Name: "DeviceB", Namespace: "default"}
	fakeDevice = &fakeDeviceArray

	fakeDeviceTwin := new([]dtclient.DeviceTwin)
	fakeDeviceTwinArray := make([]dtclient.DeviceTwin, 1)
	fakeDeviceTwinArray[0] = dtclient.DeviceTwin{DeviceName: "DeviceB", DeviceNamespace: "default"}
	fakeDeviceTwin = &fakeDeviceTwinArray

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockOrmer.EXPECT().Rollback().Return(nil).Times(5)
			mockOrmer.EXPECT().Commit().Return(nil).Times(0)
			mockOrmer.EXPECT().Begin().Return(nil).Times(5)
			mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(0)
			mockOrmer.EXPECT().Insert(gomock.Any()).Return(test.insertReturnInt, test.insertReturnErr).Times(5)
			mockQuerySeter.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(0)
			mockQuerySeter.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(0)

			mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(1)
			mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)

			mockQuerySeter.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceTwin).Return(test.allReturnInt, test.allReturnErr).Times(1)
			mockQuerySeter.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			mockOrmer.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			if err := DealDeviceTwin(test.context, test.deviceID, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealDeviceTwinTrans() case failed: got = %v, Want = %v", err, test.err)
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
	twinCloudVersion := dttype.TwinVersion{
		CloudVersion: 1,
		EdgeVersion:  0,
	}

	tests := []struct {
		name       string
		version    *dttype.TwinVersion
		reqVersion *dttype.TwinVersion
		dealType   int
		errorWant  bool
		err        error
	}{
		{
			name:      "TestDealVersion(): Case 1: dealType=3",
			version:   &dttype.TwinVersion{},
			dealType:  SyncTwinDeleteDealType,
			errorWant: true,
			err:       nil,
		},
		{
			name:       "TestDealVersion(): Case 2: dealType>=1 && version.EdgeVersion>reqVersion.EdgeVersion",
			version:    &twinCloudEdgeVersion,
			reqVersion: &twinCloudVersion,
			dealType:   SyncDealType,
			errorWant:  false,
			err:        errors.New("Not allowed to sync due to version conflict"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := dealVersion(test.version, test.reqVersion, test.dealType)
			if !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealVersion() case failed: got = %v, Want = %v", err, test.err)
				return
			}
			if !reflect.DeepEqual(got, test.errorWant) {
				t.Errorf("DTManager.TestDealVersion() case failed: got = %v, want %v", got, test.errorWant)
			}
		})
	}
}

// TestDealTwinDelete is function to test dealTwinDelete
func TestDealTwinDelete(t *testing.T) {
	str := typeString

	sync := v1alpha2.Device{}
	sync.Status.Twins = make([]v1alpha2.Twin, 1)
	sync.Status.Twins[0] = v1alpha2.Twin{
		PropertyName: key1,
		Desired:      twinValueFunc(),
		Reported:     twinValueFunc(),
	}
	sync.Status.Twins[0].Desired.Metadata["type"] = typeDeleted
	result := v1alpha2.Device{}
	result.Status.Twins = make([]v1alpha2.Twin, 1)
	result.Status.Twins[0] = v1alpha2.Twin{
		PropertyName: key1,
	}

	msgTwin := &v1alpha2.Twin{
		Desired: v1alpha2.TwinProperty{
			Value:    str,
			Metadata: make(map[string]string),
		},
		Reported: v1alpha2.TwinProperty{
			Value: str,
		},
	}
	msgTwin.Desired.Metadata["type"] = typeDeleted

	context := contextFunc(deviceB)
	twin := make([]v1alpha2.Twin, 1)
	twin[0] = v1alpha2.Twin{
		PropertyName: deviceA,
		Desired: v1alpha2.TwinProperty{
			Value: str,
		},
		Reported: v1alpha2.TwinProperty{
			Value: str,
		},
	}
	device := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: twin,
		},
	}
	context.DeviceList.Store(deviceA, &device)

	typeStringMap := make(map[string]string)
	typeStringMap["type"] = typeString

	tests := []struct {
		Context      *dtcontext.DTContext
		name         string
		returnResult *dttype.DealTwinResult
		deviceID     string
		//key          string
		//twin         v1alpha2.Twin
		cacheTwin *[]v1alpha2.Twin
		msgTwin   *v1alpha2.Twin
		dealType  int
		err       error
	}{
		{
			Context: &context,
			name:    "TestDealTwinDelete(): Case 1: msgTwin is not nil; isChange is false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID: deviceA,
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Metadata: typeStringMap,
					},
				},
			},
			msgTwin:  msgTwin,
			dealType: SyncDealType,
			err:      nil,
		},
		{
			Context: &context,
			name:    "TestDealTwinDelete(): Case 5: hasTwinExpected is true; hasTwinActual is false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID: deviceA,
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Metadata: typeStringMap,
					},
				},
			},
			msgTwin:  msgTwin,
			dealType: SyncDealType,
			err:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 如何构造context
			if err := dealTwinDelete(test.returnResult, test.deviceID, test.cacheTwin, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealTwinDelete() case failed: got = %+v, Want = %+v", err, test.err)
			}
		})
	}
}

// TestDealTwinCompare is function to test dealTwinCompare
func TestDealTwinCompare(t *testing.T) {
	str := typeString

	syncResult := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: []v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired:      twinValueFunc(),
					Reported:     twinValueFunc(),
				},
			},
		},
	}
	syncResult.Status.Twins[0].Desired.Metadata["type"] = typeDeleted

	typeIntMap := make(map[string]string)
	typeIntMap["type"] = typeInt

	typeStringMap := make(map[string]string)
	typeStringMap["type"] = typeString

	typeDeletedMap := make(map[string]string)
	typeStringMap["type"] = typeDeleted

	tests := []struct {
		name         string
		returnResult *dttype.DealTwinResult
		deviceID     string
		cacheTwin    *[]v1alpha2.Twin
		msgTwin      *v1alpha2.Twin
		dealType     int
		err          error
	}{
		{
			name: "TestDealTwinCompare(): Case 1: msgTwin nil",
			returnResult: &dttype.DealTwinResult{
				SyncResult: syncResult,
			},
			deviceID: deviceA,
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Metadata: typeDeletedMap,
					},
				},
			},
			dealType: RestDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinCompare(): Case 3: expectedOk is true; dealVersion() returns false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: syncResult,
				//Result: result,
			},
			deviceID: deviceA,
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Value:    str,
						Metadata: typeStringMap,
					},
					Reported: v1alpha2.TwinProperty{
						Value: str,
					},
				},
			},
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
				Desired: v1alpha2.TwinProperty{
					Value:    str,
					Metadata: typeIntMap,
				},
			},

			dealType: SyncDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinCompare(): Case 4: actualOk is true; dealVersion() returns false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: syncResult,
				//Result: result,
			},
			deviceID: deviceA,
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Value:    str,
						Metadata: typeDeletedMap,
					},
					Reported: v1alpha2.TwinProperty{
						Value: str,
					},
				},
			},
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
				Desired: v1alpha2.TwinProperty{
					Metadata: typeStringMap,
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinCompare(): Case 5: expectedOk is true; actualOk is true",
			returnResult: &dttype.DealTwinResult{
				SyncResult: syncResult,
				//Result: result,
			},
			deviceID: deviceA,
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Metadata: typeDeletedMap,
					},
				},
			},
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
				Desired: v1alpha2.TwinProperty{
					Value:    str,
					Metadata: typeStringMap,
				},
			},
			dealType: RestDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinCompare(): Case 6: expectedOk is false; actualOk is false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: syncResult,
				//Result: result,
			},
			deviceID: deviceA,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
			},
			cacheTwin: &[]v1alpha2.Twin{
				{
					PropertyName: key1,
					Desired: v1alpha2.TwinProperty{
						Metadata: typeDeletedMap,
					},
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealTwinCompare(test.returnResult, test.deviceID, test.cacheTwin, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealTwinCompare() case failed: got = %+v, Want = %+v", err, test.err)
			}
		})
	}
}

// TestDealTwinAdd is function to test dealTwinAdd
func TestDealTwinAdd(t *testing.T) {
	str := typeString
	typeDeletedMap := make(map[string]string)
	typeDeletedMap["type"] = typeDeleted

	typeIntMap := make(map[string]string)
	typeIntMap["type"] = typeInt

	sync := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: []v1alpha2.Twin{
				{
					PropertyName: key1,
				},
			},
		},
	}

	twinDelete := &[]v1alpha2.Twin{
		{
			PropertyName: key1,
			Desired: v1alpha2.TwinProperty{
				Metadata: typeDeletedMap,
			},
		},
	}

	twinInt := &[]v1alpha2.Twin{
		{
			PropertyName: key1,
			Desired: v1alpha2.TwinProperty{
				Metadata: typeIntMap,
			},
		},
	}

	tests := []struct {
		name         string
		Context      *dtcontext.DTContext
		returnResult *dttype.DealTwinResult
		deviceID     string
		cacheTwin    *[]v1alpha2.Twin
		msgTwin      *v1alpha2.Twin
		dealType     int
		err          error
	}{
		{
			name: "TestDealTwinAdd(): Case 1: msgTwin nil",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID: deviceA,
			dealType: RestDealType,
			err:      errors.New("The request body is wrong"),
		},
		{
			name: "TestDealTwinAdd(): Case 2: msgTwin.Expected is not nil; dealVersion() returns false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID:  deviceA,
			cacheTwin: twinDelete,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Desired: v1alpha2.TwinProperty{
					Value:    str,
					Metadata: typeDeletedMap,
				},
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinAdd(): Case 3: msgTwin.Expected is not nil; ValidateValue() returns error",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID:  deviceA,
			cacheTwin: twinDelete,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Desired: v1alpha2.TwinProperty{
					Value:    str,
					Metadata: typeIntMap,
				},
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinAdd(): Case 4: msgTwin.Actual is not nil; dealVersion() returns false",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID:  deviceA,
			cacheTwin: twinDelete,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Desired: v1alpha2.TwinProperty{
					Value:    str,
					Metadata: typeDeletedMap,
				},
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
		{
<<<<<<< HEAD
=======
			name: "TestDealTwinAdd(): Case 5: msgTwin.Actual is not nil; ValidateValue() returns error; dealType=0",
			returnResult: &dttype.DealTwinResult{
				Document:   doc,
				SyncResult: sync,
				Result:     result,
			},
			deviceID: deviceA,
			key:      key1,
			twins:    twinDelete,
			msgTwin: &dttype.MsgTwin{
				Actual: &dttype.TwinValue{
					Value: &str,
				},
				Optional: &optionTrue,
				Metadata: &dttype.TypeMetadata{
					Type: typeInt,
				},
				ExpectedVersion: &dttype.TwinVersion{},
				ActualVersion:   &dttype.TwinVersion{},
			},
			dealType: RestDealType,
			err:      errors.New("the value is not int or integer"),
		},
		{
>>>>>>> upstream/master
			name: "TestDealTwinAdd(): Case 6: msgTwin.Actual is not nil; ValidateValue() returns error; dealType=1",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID:  deviceA,
			cacheTwin: twinDelete,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Desired: v1alpha2.TwinProperty{
					Metadata: typeIntMap,
				},
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinAdd(): Case 7: msgTwin.Expected is nil; msgTwin.Actual is nil",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID:  deviceA,
			cacheTwin: twinInt,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
			},
			dealType: RestDealType,
			err:      nil,
		},
		{
			name: "TestDealTwinAdd(): Case 8: msgTwin.Expected is not nil; msgTwin.Actual is not nil",
			returnResult: &dttype.DealTwinResult{
				SyncResult: sync,
				//Result:     result,
			},
			deviceID:  deviceA,
			cacheTwin: twinDelete,
			msgTwin: &v1alpha2.Twin{
				PropertyName: key1,
				Desired: v1alpha2.TwinProperty{
					Value:    str,
					Metadata: typeDeletedMap,
				},
				Reported: v1alpha2.TwinProperty{
					Value: str,
				},
			},
			dealType: SyncDealType,
			err:      nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := dealTwinAdd(test.returnResult, test.deviceID, test.cacheTwin, test.msgTwin, test.dealType); !reflect.DeepEqual(err, test.err) {
				t.Errorf("DTManager.TestDealTwinAdd() case failed: got = %+v, Want = %+v", err, test.err)
			}
		})
	}
}

// TestDealMsgTwin is function to test DealMsgTwin
func TestDealMsgTwin(t *testing.T) {
	value := valueType
	str := typeString

	add := make([]dtclient.DeviceTwin, 0)
	deletes := make([]dtclient.DeviceTwinPrimaryKey, 0)
	update := make([]dtclient.DeviceTwinUpdate, 0)

	syncResult := v1alpha2.Device{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "DeviceC",
		},
		Status: v1alpha2.DeviceStatus{
			Twins: make([]v1alpha2.Twin, 0),
		},
	}
	syncResultDevice := v1alpha2.Device{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "DeviceA",
		},
		Status: v1alpha2.DeviceStatus{
			Twins: make([]v1alpha2.Twin, 0),
		},
	}

	msgTwin := []*v1alpha2.Twin{
		{
			PropertyName: deviceB,
			Desired: v1alpha2.TwinProperty{
				Value: value,
			},
		},
	}

	msgTwinDevice := []*v1alpha2.Twin{
		{
			PropertyName: deviceA,
		},
	}

	typeIntMap := make(map[string]string)
	typeIntMap["type"] = typeInt

	typeDeletedMap := make(map[string]string)
	typeDeletedMap["type"] = typeDeleted

	context := contextFunc(deviceB)
	device := v1alpha2.Device{
		Status: v1alpha2.DeviceStatus{
			Twins: []v1alpha2.Twin{
				{
					PropertyName: deviceA,
					Desired: v1alpha2.TwinProperty{
						Value:    str,
						Metadata: typeDeletedMap,
					},
					Reported: v1alpha2.TwinProperty{
						Value: str,
					},
				},
			},
		},
	}
	context.DeviceList.Store(deviceA, &device)

	tests := []struct {
		name     string
		context  *dtcontext.DTContext
		deviceID string
		msgTwins []*v1alpha2.Twin
		dealType int
		want     dttype.DealTwinResult
	}{
		{
			name:     "TestDealMsgTwin(): Case1: invalid device id",
			context:  &context,
			deviceID: deviceC,
			msgTwins: msgTwin,
			dealType: RestDealType,
			want: dttype.DealTwinResult{
				Add:    add,
				Delete: deletes,
				Update: update,
				//Result:     deviceCResult,
				SyncResult: syncResult,
				Err:        errors.New("invalid device id"),
			},
		},
		{
<<<<<<< HEAD
=======
			name:     "TestDealMsgTwin(): Case 2: dealTwinCompare error",
			context:  &context,
			deviceID: deviceA,
			msgTwins: msgTwinDeviceTwin,
			dealType: RestDealType,
			want: dttype.DealTwinResult{
				Add:        add,
				Delete:     deletes,
				Update:     update,
				Result:     result,
				SyncResult: syncResultDevice,
				Document:   documentDevice,
				Err:        errors.New("the value is not int or integer"),
			},
		},
		{
>>>>>>> upstream/master
			name:     "TestDealMsgTwin(): Case 3: Success case",
			context:  &context,
			deviceID: deviceA,
			msgTwins: msgTwinDevice,
			dealType: RestDealType,
			want: dttype.DealTwinResult{
				Add:    add,
				Delete: deletes,
				Update: update,
				//Result:     result,
				SyncResult: syncResultDevice,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DealMsgTwin(tt.context, tt.deviceID, tt.msgTwins, tt.dealType)
			gotByte, _ := json.Marshal(got)
			wantByte, _ := json.Marshal(tt.want)
			if string(gotByte) != string(wantByte) {
				t.Errorf("DTManager.DealMsgTwin() case failed: got = %+v, want = %+v", string(gotByte), string(wantByte))
			}
		})
	}
}
