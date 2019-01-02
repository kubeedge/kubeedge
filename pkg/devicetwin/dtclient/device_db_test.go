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

package dtclient

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/kubeedge/kubeedge/mocks/beego"
	"github.com/kubeedge/kubeedge/pkg/common/dbm"
)

// FailedDBOperation is common DB operation fail error
var failedDBOperationErr = errors.New("Failed DB Operation")

// ormerMock is mocked Ormer implementation
var ormerMock *beego.MockOrmer

// querySeterMock is mocked QuerySeter implementation
var querySeterMock *beego.MockQuerySeter

// initMocks is function to initialize mocks
func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock
}

// TestSaveDevice is function to test SaveDevice
func TestSaveDevice(t *testing.T) {
	//Initialize Global Variables (Mocks)
	initMocks(t)
	device := Device{
		ID:          "TestID",
		Name:        "TestName",
		Description: "TestDescription",
		State:       "TestState",
		LastOnline:  "TestLastOnline",
	}
	// Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	err := SaveDevice(&device)
	t.Run("SaveDeviceSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Save Device Case failed : wanted error nil and got error %v", err)
		}
	})

	// Failure Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = SaveDevice(&device)
	t.Run("SaveDeviceFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Save Device Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestDeleteDeviceByID is function to test DeleteDeviceByID
func TestDeleteDeviceByID(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteDeviceByID("test")
	t.Run("DeleteDeviceByIDFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceByID Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeleteDeviceByID("test")
	t.Run("DeleteDeviceByIDSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteDeviceByID Case failed : wanted err nil and got err %v", err)
		}
	})
}

// TestUpdateDeviceField is function to test UpdateDeviceField
func TestUpdateDeviceField(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceField("test", "test", "test")
	t.Run("UpdateDeviceFieldFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceField Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestUpdateDeviceFields is function to test UpdateDeviceFields
func TestUpdateDeviceFields(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceFields("test", make(map[string]interface{}))
	t.Run("UpdateDeviceFieldsFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceFields Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestQueryDevice is function to test QueryDevice
func TestQueryDevice(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err := QueryDevice("test", "test")
	t.Run("QueryDeviceCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("QueryDevice Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success case
	fakeDevice := new([]Device)
	fakeDeviceArray := make([]Device, 1)
	fakeDeviceArray[0] = Device{ID: "Test"}
	fakeDevice = &fakeDeviceArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err = QueryDevice("test", "test")
	t.Run("QueryDeviceSuccessCase", func(t *testing.T) {
		want := 1
		if want != len(*device) {
			t.Errorf("QueryDevice Case failed wanted length %v and got length %v", want, len(*device))
		}
	})
}

// TestQueryDeviceAll is function to test QueryDeviceAll
func TestQueryDeviceAll(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err := QueryDeviceAll()
	t.Run("QueryDeviceAllCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("QueryDeviceAll Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success case
	fakeDevice := new([]Device)
	fakeDeviceArray := make([]Device, 1)
	fakeDeviceArray[0] = Device{ID: "Test"}
	fakeDevice = &fakeDeviceArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err = QueryDeviceAll()
	t.Run("QueryDeviceAllSuccessCase", func(t *testing.T) {
		want := 1
		if want != len(*device) {
			t.Errorf("QueryDeviceAll Case failed wanted length %v and got length %v", want, len(*device))
		}
	})
}

// TestUpdateDeviceMulti is function to test UpdateDeviceMulti
func TestUpdateDeviceMulti(t *testing.T) {
	// Failure Case
	updateDevice := make([]DeviceUpdate, 0)
	updateDevice = append(updateDevice, DeviceUpdate{DeviceID: "test"})
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceMulti(updateDevice)
	t.Run("UpdateDeviceMultiCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceMulti Case failed : wanted %v and got %v", want, err)
		}
	})

	//Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = UpdateDeviceMulti(updateDevice)
	t.Run("UpdateDeviceMultiCheckError", func(t *testing.T) {
		if err != nil {
			t.Errorf("UpdateDeviceMulti Case failed : wanted error nil and got error %v", err)
		}
	})
}

// TestAddDeviceTrans is function to test AddDeviceTrans
func TestAddDeviceTrans(t *testing.T) {

	adds := make([]Device, 0)
	adds = append(adds, Device{ID: "test"})
	addAttrs := make([]DeviceAttr, 0)
	addAttrs = append(addAttrs, DeviceAttr{DeviceID: "test"})
	addTwins := make([]DeviceTwin, 0)
	addTwins = append(addTwins, DeviceTwin{DeviceID: "test"})
	// Failure Case SaveDevice
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	err := AddDeviceTrans(adds, addAttrs, addTwins)
	t.Run("AddDeviceTransSaveDeviceFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("AddDeviceTrans Case failed : wanted %v and got %v", want, err)
		}
	})

	// Failure Case SaveDeviceAttr
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	err = AddDeviceTrans(adds, addAttrs, addTwins)
	t.Run("AddDeviceTransSaveDeviceAttrFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("AddDeviceTrans Case failed : wanted %v and got %v", want, err)
		}
	})

	// Failure Case SaveDeviceTwin
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(2)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	err = AddDeviceTrans(adds, addAttrs, addTwins)
	t.Run("AddDeviceTransSaveDeviceTwinFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("AddDeviceTrans Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(3)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().Commit().Return(nil).Times(1)
	err = AddDeviceTrans(adds, addAttrs, addTwins)
	t.Run("AddDeviceTransSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("AddDeviceTrans Case failed : wanted error nil and got error %v", err)
		}
	})
}

// TestDeleteDeviceTrans is function to test DeleteDeviceTrans
func TestDeleteDeviceTrans(t *testing.T) {
	deletes := []string{"test"}
	// Failure Case DeleteDeviceByID
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteDeviceTrans(deletes)
	t.Run("DeleteDeviceTransDeleteDeviceTwinByDeviceIDFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceTrans Case failed : wanted %v and got %v", want, err)
		}
	})

	// Failure Case DeleteDeviceAttrByDeviceID
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	err = DeleteDeviceTrans(deletes)
	t.Run("DeleteDeviceTransDeleteDeviceTwinByDeviceIDFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceTrans Case failed : wanted %v and got %v", want, err)
		}
	})

	// Failure Case DeleteDeviceTwinByDeviceID
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(3)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(2)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(3)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	err = DeleteDeviceTrans(deletes)
	t.Run("DeleteDeviceTransDeleteDeviceTwinByDeviceIDFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceTrans Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Commit().Return(nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(3)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(3)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(3)
	err = DeleteDeviceTrans(deletes)
	t.Run("DeleteDeviceTransSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteDeviceTrans Case failed : wanted error nil and got error %v", err)
		}
	})
}
