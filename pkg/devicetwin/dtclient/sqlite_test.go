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

// TestSaveTwin is function to test SaveTwin
func TestSaveTwin(t *testing.T) {
	//Initialize Global Variables (Mocks)
	initMocks(t)
	InitDBTable()
	twin := Twin{
		DeviceID:   "TestDeviceID",
		DeviceName: "TestName",
		Expected:   "TestExpected",
		Actual:     "TestActual",
	}

	// Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	err := SaveTwin(&twin)
	t.Run("SaveTwinSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Save Twin Case failed : wanted error nil and got error %v", err)
		}
	})

	// Failure Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = SaveTwin(&twin)
	t.Run("SaveTwinFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Save Twin Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestDeleteTwinByID is function to test DeleteTwinByID
func TestDeleteTwinByID(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteTwinByID("test")
	t.Run("DeleteTwinByIDFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("DeleteTwinByID Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err = DeleteTwinByID("test")
	t.Run("DeleteTwinByIDSuccess", func(t *testing.T) {
		if err != nil {
			t.Errorf("DeleteTwinByID Case failed : wanted err nil and got err %v", err)
		}
	})
}

// TestUpdateTwinField is function to test UpdateTwinField
func TestUpdateTwinField(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateTwinField("test", "test", "test")
	t.Run("UpdateTwinFieldFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateTwinField Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestUpdateTwinFields is function to test UpdateTwinFields
func TestUpdateTwinFields(t *testing.T) {
	// Failure Case
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateTwinFields("test", make(map[string]interface{}))
	t.Run("UpdateTwinFieldsFailure", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("UpdateTwinFields Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestQueryTwin is function to test QueryTwin
func TestQueryTwin(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err := QueryTwin("test", "test")
	t.Run("QueryTwinCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("QueryTwin Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success case
	fakeTwin := new([]Twin)
	fakeTwinArray := make([]Twin, 1)
	fakeTwinArray[0] = Twin{DeviceID: "Test"}
	fakeTwin = &fakeTwinArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeTwin).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err = QueryTwin("test", "test")
	t.Run("QueryTwinSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("QueryTwin Case Failed,Expected a error nil and got %v", err)
		}
		want := 1
		if want != len(*device) {
			t.Errorf("QueryTwin Case failed wanted length %v and got length %v", want, len(*device))
		}
	})
}

// TestQueryTwinAll is function to test QueryTwinAll
func TestQueryTwinAll(t *testing.T) {
	// Failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err := QueryTwinAll()
	t.Run("QueryTwinAllCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("QueryTwinAll Case failed : wanted %v and got %v", want, err)
		}
	})

	// Success case
	fakeTwin := new([]Twin)
	fakeTwinArray := make([]Twin, 1)
	fakeTwinArray[0] = Twin{DeviceID: "Test"}
	fakeTwin = &fakeTwinArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeTwin).Return(int64(1), nil).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	device, err = QueryTwinAll()
	t.Run("QueryTwinAllSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("QueryTwinAll Case Failed,Expected a error nil and got %v", err)
		}
		want := 1
		if want != len(*device) {
			t.Errorf("QueryTwinAll Case failed wanted length %v and got length %v", want, len(*device))
		}
	})
}
