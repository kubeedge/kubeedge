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
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
)

var ormerMock *beego.MockOrmer
var querySeterMock *beego.MockQuerySeter

func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock
}

func TestInitMemActionCallBack(t *testing.T) {
	initMemActionCallBack()
}

func TestGetRemoveList(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
	}

	var device dttype.Device
	dtc.DeviceList.Store("DeviceB", &device)
	dArray := []dttype.Device{}
	d := dttype.Device{
		ID: "123",
	}
	dArray = append(dArray, d)
	value := getRemoveList(dtc, dArray)
	for i := range value {
		assert.Equal(t, "DeviceB", value[i].ID)
	}
}

func TestGetRemoveListProperDevideID(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
	}
	var device dttype.Device
	dtc.DeviceList.Store("123", &device)
	dArray := []dttype.Device{}
	d := dttype.Device{
		ID: "123",
	}
	dArray = append(dArray, d)
	value := getRemoveList(dtc, dArray)
	for i := range value {
		assert.Equal(t, "123", value[i].ID)
	}
}

func TestDealMembershipDetailInvalidEmptyMessage(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}
	value, err := dealMembershipDetail(dtc, "t", "invalid")
	assert.Error(t, err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipDetailInvalidMsg(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}

	var m = &model.Message{
		Content: "invalidmsg",
	}

	value, err := dealMembershipDetail(dtc, "t", m)
	assert.Error(t, err)
	assert.Equal(t, errors.New("assertion failed"), err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipDetailInvalidContent(t *testing.T) {

	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}
	var cnt []uint8
	cnt = append(cnt, 1)
	var m = &model.Message{
		Content: cnt,
	}

	value, err := dealMembershipDetail(dtc, "t", m)
	assert.Error(t, err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipDetailValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		Mutex:      &sync.RWMutex{},
		GroupID:    "1",
	}

	payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router",
		State: "unknown"}}, BaseMessage: dttype.BaseMessage{EventID: "eventid"}}
	content, _ := json.Marshal(payload)
	var m = &model.Message{
		Content: content,
	}
	value, err := dealMembershipDetail(dtc, "t", m)
	assert.NoError(t, err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipUpdatedEmptyMessage(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}
	value, err := dealMembershipDetail(dtc, "t", "invalid")
	assert.Error(t, err)
	assert.Equal(t, errors.New("msg not Message type"), err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipUpdatedInvalidMsg(t *testing.T) {

	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}

	var m = &model.Message{
		Content: "invalidmessage",
	}

	value, err := dealMembershipUpdated(dtc, "t", m)
	assert.Error(t, err)
	assert.Equal(t, errors.New("assertion failed"), err)
	assert.Equal(t, nil, value)
}
func TestDealMembershipUpdatedInvalidContent(t *testing.T) {

	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}

	var cnt []uint8
	cnt = append(cnt, 1)
	var m = &model.Message{
		Content: cnt,
	}

	value, err := dealMembershipUpdated(dtc, "t", m)
	assert.Error(t, err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipUpdatedValidAddedDevice(t *testing.T) {
	initMocks(t)
	ormerMock.EXPECT().Begin().Return(nil).Times(6)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(3)
	ormerMock.EXPECT().Commit().Return(nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router",
		State: "unknown"}}, BaseMessage: dttype.BaseMessage{EventID: "eventid"}}
	content, _ := json.Marshal(payload)
	var m = &model.Message{
		Content: content,
	}
	value, err := dealMembershipUpdated(dtc, "t", m)
	assert.NoError(t, err)
	assert.Equal(t, nil, value)
}

func TestDealMembershipUpdatedValidRemovedDevice(t *testing.T) {
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().Commit().Return(nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	payload := dttype.MembershipUpdate{RemoveDevices: []dttype.Device{{ID: "DeviceA", Name: "Router",
		State: "unknown"}}, BaseMessage: dttype.BaseMessage{EventID: "eventid"}}
	content, _ := json.Marshal(payload)
	var m = &model.Message{
		Content: content,
	}
	value, err := dealMembershipUpdated(dtc, "t", m)
	assert.NoError(t, err)
	assert.Equal(t, nil, value)
}

func TestDealMerbershipGetEmptyMsg(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}
	value, err := dealMerbershipGet(dtc, "t", "invalid")
	assert.Error(t, err)
	assert.Equal(t, errors.New("msg not Message type"), err)
	assert.Equal(t, nil, value)
}

func TestDealMerbershipGetInvalidMsg(t *testing.T) {

	dtc := &dtcontext.DTContext{
		DeviceList: &sync.Map{},
		GroupID:    "1",
	}

	var m = &model.Message{
		Content: "hello",
	}

	value, err := dealMerbershipGet(dtc, "t", m)
	assert.Error(t, err)
	assert.Equal(t, errors.New("assertion failed"), err)
	assert.Equal(t, nil, value)
}

func TestDealMerbershipGetValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router",
		State: "unknown"}}, BaseMessage: dttype.BaseMessage{EventID: "eventid"}}
	content, _ := json.Marshal(payload)
	var m = &model.Message{
		Content: content,
	}
	value, err := dealMerbershipGet(dtc, "t", m)
	assert.Equal(t, nil, value)
	assert.NoError(t, err)
}

func TestDealGetMembershipValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	payload := dttype.MembershipUpdate{AddDevices: []dttype.Device{{ID: "DeviceA", Name: "Router",
		State: "unknown"}}, BaseMessage: dttype.BaseMessage{EventID: "eventid"}}
	content, _ := json.Marshal(payload)

	err := DealGetMembership(dtc, content)
	assert.NoError(t, err)
}

func TestDealGetMembershipInValid(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.RWMutex{},
		GroupID:     "1",
	}

	err := DealGetMembership(dtc, []byte("invalid"))
	assert.NoError(t, err)
}

/* Commented As we are not considering about the coverage incase for coverage we can uncomment below cases.
func TestAdded(t *testing.T) {
	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.Mutex{},
		GroupID:     "1",
	}

	var d = []dttype.Device{{
		ID:    "DeviceA",
		Name:  "Router",
		State: "unknown",
	}}
	var b = dttype.BaseMessage{
		EventID: "eventid",
	}
	Added(dtc, d, b, true)
}

func TestRemoved(t *testing.T) {
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(0)
	ormerMock.EXPECT().Commit().Return(nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(3)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(3)
	// success delete
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(3)
	// fail delete
	querySeterMock.EXPECT().Delete().Return(int64(1), errors.New("failed to delete")).Times(0)

	dtc := &dtcontext.DTContext{
		DeviceList:  &sync.Map{},
		DeviceMutex: &sync.Map{},
		Mutex:       &sync.Mutex{},
		GroupID:     "1",
	}

	var device dttype.Device
	dtc.DeviceList.Store("DeviceA", &device)
	var d = []dttype.Device{{
		ID:    "DeviceA",
		Name:  "Router",
		State: "unknown",
	}}
	var b = dttype.BaseMessage{
		EventID: "eventid",
	}

	Removed(dtc, d, b, true)
}*/
