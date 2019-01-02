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
	"testing"

	"github.com/golang/mock/gomock"
)

// TestSaveDeviceTwin is function to test SaveDeviceTwin
func TestSaveDeviceTwin(t *testing.T) {
	//Initialize Global Variables (Mocks)
	initMocks(t)
	deviceTwin := DeviceTwin{
		ID:       123,
		DeviceID: "TestDeviceID",
		Name:     "TestName",
		Expected: "TestExpected",
		Actual:   "TestActual",
	}

	// Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	err := SaveDeviceTwin(&deviceTwin)
	t.Run("SaveDeviceTwinSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Save Device Twin Case failed : wanted error nil and got error %v", err)
		}
	})

	// Failure Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = SaveDeviceTwin(&deviceTwin)
	t.Run("SaveDeviceTwinFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Save Device Twin Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestDeleteDeviceTwinByDeviceID is function to test DeleteDeviceTwinByDeviceID
func TestDeleteDeviceTwinByDeviceID(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteDeviceTwinByDeviceID("test")
	t.Run("DeleteDeviceTwinByDeviceIDFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceTwinByDeviceID Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeleteDeviceTwinByDeviceID("test")
	t.Run("DeleteDeviceTwinByDeviceIDSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteDeviceTwinByDeviceID Case failed : wanted err nil and got err %v", err)
		}
	})
}

// TestDeleteDeviceTwin is function to test DeleteDeviceTwin
func TestDeleteDeviceTwin(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteDeviceTwin("test", "test")
	t.Run("DeleteDeviceTwinFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceTwin Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeleteDeviceTwin("test", "test")
	t.Run("DeleteDeviceTwinSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteDeviceTwin Case failed : wanted err nil and got err %v", err)
		}
	})
}

// TestUpdateDeviceTwinField is function to test UpdateDeviceTwinField
func TestUpdateDeviceTwinField(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceTwinField("test", "test", "test", "test")
	t.Run("UpdateDeviceTwinFieldFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceTwinField Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestUpdateDeviceTwinFields is function to test UpdateDeviceTwinFields
func TestUpdateDeviceTwinFields(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceTwinFields("test", "test", make(map[string]interface{}))
	t.Run("UpdateDeviceTwinFieldsFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceTwinFields Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestQueryDeviceTwin is function to test QueryDeviceTwin
func TestQueryDeviceTwin(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err := QueryDeviceTwin("test", "test")
	t.Run("QueryDeviceTwinCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("QueryDeviceTwin Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success case
	faleDeviceTwin := new([]DeviceTwin)
	faleDeviceTwinArray := make([]DeviceTwin, 1)
	faleDeviceTwinArray[0] = DeviceTwin{DeviceID: "Test"}
	faleDeviceTwin = &faleDeviceTwinArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *faleDeviceTwin).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err = QueryDeviceTwin("test", "test")
	t.Run("QueryDeviceTwinSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("QueryDeviceTwin Case Failed,Expected a error nil and got %v", err)
		}
		want := 1
		if want != len(*device) {
			t.Errorf("QueryDeviceTwin Case failed wanted length %v and got length %v", want, len(*device))
		}
	})
}

// TestUpdateDeviceTwinMulti is function to test UpdateDeviceTwinMulti
func TestUpdateDeviceTwinMulti(t *testing.T) {
	// Failure Case
	updateDevice := make([]DeviceTwinUpdate, 0)
	updateDevice = append(updateDevice, DeviceTwinUpdate{DeviceID: "test"})
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceTwinMulti(updateDevice)
	t.Run("UpdateDeviceTwinMultiCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceTwinMulti Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = UpdateDeviceTwinMulti(updateDevice)
	t.Run("UpdateDeviceTwinMultiSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("UpdateDeviceTwinMulti Case failed : wanted error nil and got error %v", err)
		}
	})
}

// TestDeviceTwinTrans is function to test DeviceTwinTrans
func TestDeviceTwinTrans(t *testing.T) {
	adds := make([]DeviceTwin, 0)
	deletes := make([]DeviceDelete, 0)
	updates := make([]DeviceTwinUpdate, 0)
	adds = append(adds, DeviceTwin{DeviceID: "Test"})
	deletes = append(deletes, DeviceDelete{DeviceID: "test", Name: "test"})
	updates = append(updates, DeviceTwinUpdate{DeviceID: "test", Name: "test", Cols: make(map[string]interface{})})

	// Failure Case SaveDeviceTwin
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err := DeviceTwinTrans(adds, deletes, updates)
	t.Run("TestDeviceTwinTransSaveDeviceTwinFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("TestDeviceTwinTrans Case failed: wanted %v and got %v", want, err)
		}
	})

	// Failure Case DeleteDeviceTwin
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeviceTwinTrans(adds, deletes, updates)
	t.Run("TestDeviceTwinTransSaveDeviceTwinFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("TestDeviceTwinTrans Case failed: wanted %v and got %v", want, err)
		}
	})

	// Failure Case UpdateDeviceTwinFields
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(4)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = DeviceTwinTrans(adds, deletes, updates)
	t.Run("TestDeviceTwinTransSaveDeviceTwinFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("TestDeviceTwinTrans Case failed: wanted %v and got %v", want, err)
		}
	})

	// Success Case
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Commit().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(6)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(3)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), nil).Times(1)
	err = DeviceTwinTrans(adds, deletes, updates)
	t.Run("TestDeviceTwinTransSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("TestDeviceTwinTrans Case failed: wanted error nil and got error %v", err)
		}
	})
}
