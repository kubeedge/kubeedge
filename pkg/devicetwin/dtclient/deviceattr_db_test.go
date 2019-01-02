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

// TestSaveDeviceAttr is function to test SaveDeviceAttr
func TestSaveDeviceAttr(t *testing.T) {
	//Initialize Global Variables (Mocks)
	initMocks(t)
	deviceAttr := DeviceAttr{
		ID:          123,
		DeviceID:    "TestDeviceID",
		Name:        "TestName",
		Description: "TestDescription",
		Value:       "TestValue",
	}

	// Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	err := SaveDeviceAttr(&deviceAttr)
	t.Run("SaveDeviceAttrSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Save Device Attr Case failed : wanted error nil and got error %v", err)
		}
	})

	// Failure Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = SaveDeviceAttr(&deviceAttr)
	t.Run("SaveDeviceAttrFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Save Device Attr Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestDeleteDeviceAttrByDeviceID is function to test DeleteDeviceAttrByDeviceID
func TestDeleteDeviceAttrByDeviceID(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteDeviceAttrByDeviceID("test")
	t.Run("DeleteDeviceAttrByDeviceIDFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceAttrByDeviceID Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeleteDeviceAttrByDeviceID("test")
	t.Run("DeleteDeviceAttrByDeviceIDSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteDeviceAttrByDeviceID Case failed : wanted err nil and got err %v", err)
		}
	})
}

// TestDeleteDeviceAttr is function to test DeleteDeviceAttr
func TestDeleteDeviceAttr(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteDeviceAttr("test", "test")
	t.Run("DeleteDeviceAttrFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteDeviceAttr Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeleteDeviceAttr("test", "test")
	t.Run("DeleteDeviceAttrSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteDeviceAttr Case failed : wanted err nil and got err %v", err)
		}
	})
}

// TestUpdateDeviceAttrField is function to test UpdateDeviceAttrField
func TestUpdateDeviceAttrField(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceAttrField("test", "test", "test", "test")
	t.Run("UpdateDeviceAttrFieldFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceAttrField Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestUpdateDeviceAttrFields is function to test UpdateDeviceAttrFields
func TestUpdateDeviceAttrFields(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceAttrFields("test", "test", make(map[string]interface{}))
	t.Run("UpdateDeviceAttrFieldsFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceAttrFields Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestQueryDeviceAttr is function to test QueryDeviceAttr
func TestQueryDeviceAttr(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err := QueryDeviceAttr("test", "test")
	t.Run("QueryDeviceAttrCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("QueryDeviceAttr Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success case
	fakeDeviceAttr := new([]DeviceAttr)
	fakeDeviceAttrArray := make([]DeviceAttr, 1)
	fakeDeviceAttrArray[0] = DeviceAttr{DeviceID: "Test"}
	fakeDeviceAttr = &fakeDeviceAttrArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceAttr).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err = QueryDeviceAttr("test", "test")
	t.Run("QueryDeviceAttrSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("QueryDeviceAttr Case Failed,Expected a error nil and got %v", err)
		}
		want := 1
		if want != len(*device) {
			t.Errorf("QueryDeviceAttr Case failed wanted length %v and got length %v", want, len(*device))
		}
	})
}

// TestUpdateDeviceAttrMulti is function to test UpdateDeviceAttrMulti
func TestUpdateDeviceAttrMulti(t *testing.T) {
	// Failure Case
	updateDevice := make([]DeviceAttrUpdate, 0)
	updateDevice = append(updateDevice, DeviceAttrUpdate{DeviceID: "test"})
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateDeviceAttrMulti(updateDevice)
	t.Run("UpdateDeviceAttrMultiCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateDeviceAttrMulti Case failed : wanted %v and got %v", want, err)
		}
	})

	//Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = UpdateDeviceAttrMulti(updateDevice)
	t.Run("UpdateDeviceAttrMultiSuccesCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("UpdateDeviceAttrMulti Case failed : wanted error nil and got error %v", err)
		}
	})
}

// TestDeviceAttrTrans is function to test DeviceAttrTrans
func TestDeviceAttrTrans(t *testing.T) {
	adds := make([]DeviceAttr, 0)
	deletes := make([]DeviceDelete, 0)
	updates := make([]DeviceAttrUpdate, 0)
	adds = append(adds, DeviceAttr{DeviceID: "Test"})
	deletes = append(deletes, DeviceDelete{DeviceID: "test", Name: "test"})
	updates = append(updates, DeviceAttrUpdate{DeviceID: "test", Name: "test", Cols: make(map[string]interface{})})

	// Failure Case SaveDeviceAttr
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err := DeviceAttrTrans(adds, deletes, updates)
	t.Run("TestDeviceAttrTransSaveDeviceAttrFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("TestDeviceAttrTrans Case failed: wanted %v and got %v", want, err)
		}
	})

	// Failure Case DeleteDeviceAttr
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeviceAttrTrans(adds, deletes, updates)
	t.Run("TestDeviceAttrTransSaveDeviceAttrFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("TestDeviceAttrTrans Case failed: wanted %v and got %v", want, err)
		}
	})

	// Failure Case UpdateDeviceAttrFields
	ormerMock.EXPECT().Begin().Return(nil).Times(1)
	ormerMock.EXPECT().Rollback().Return(nil).Times(1)
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(4)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(2)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = DeviceAttrTrans(adds, deletes, updates)
	t.Run("TestDeviceAttrTransSaveDeviceAttrFailureCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("TestDeviceAttrTrans Case failed: wanted %v and got %v", want, err)
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
	err = DeviceAttrTrans(adds, deletes, updates)
	t.Run("TestDeviceAttrTransSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("TestDeviceAttrTrans Case failed: wanted error nil and got error %v", err)
		}
	})
}
