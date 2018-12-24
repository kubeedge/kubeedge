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

package dao

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

// rawSeterMock is mocked RawSeter implementation
var rawSeterMock *beego.MockRawSeter

// meta is global variable for passing as test parameter
var meta = Meta{
	Key:   "TestKey",
	Value: "TestValue",
	Type:  "TestType",
}

// initMocks is function to initialize mocks
func initMocks(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock = beego.NewMockOrmer(mockCtrl)
	querySeterMock = beego.NewMockQuerySeter(mockCtrl)
	rawSeterMock = beego.NewMockRawSeter(mockCtrl)
	dbm.DBAccess = ormerMock
}

//TestSaveMeta is function to initialize all global variable and test SaveMeta
func TestSaveMeta(t *testing.T) {
	//Initialize Global Variables (Mocks)
	initMocks(t)
	//SaveMeta Success Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), nil).Times(1)
	err := SaveMeta(&meta)
	t.Run("SaveMetaSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Save Meta Case failed : wanted error nil and got error %v", err)
		}
	})

	//SaveMeta Failure Case
	ormerMock.EXPECT().Insert(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err = SaveMeta(&meta)
	t.Run("SaveMetaFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Save Meta Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestDeleteMetaByKey is function to test DeleteMetaByKey
func TestDeleteMetaByKey(t *testing.T) {
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Delete().Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := DeleteMetaByKey("test")
	t.Run("DeleteMetaByKeyFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Delete Meta By Key Case failed : wanted %v and got %v", want, err)
		}
	})
}

//TestUpdateMeta is function to test UpdateMeta
func TestUpdateMeta(t *testing.T) {
	ormerMock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	err := UpdateMeta(&meta)
	t.Run("UpdateMetaFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Update Meta Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestInsertOrUpdate is function to test InsertOrUpdate
func TestInsertOrUpdate(t *testing.T) {
	rawSeterMock.EXPECT().Exec().Return(nil, failedDBOperationErr).Times(1)
	ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(rawSeterMock).Times(1)
	err := InsertOrUpdate(&meta)
	t.Run("InsertOrUpdateFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Insert or Update Meta Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestUpdateMetaField is function to test UpdateMetaField
func TestUpdateMetaField(t *testing.T) {
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateMetaField("test", "test", "test")
	t.Run("UpdateMetaFieldFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Update Meta Field Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestUpdateMetaFields is function to test UpdateMetaFields
func TestUpdateMetaFields(t *testing.T) {
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	querySeterMock.EXPECT().Update(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	err := UpdateMetaFields("test", nil)
	t.Run("UpdateMetaFieldsFailCase", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Update Meta Fields Case failed : wanted %v and got %v", want, err)
		}
	})
}

// TestQueryMeta is function to test QueryMeta
func TestQueryMeta(t *testing.T) {
	//failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	meta, err := QueryMeta("test", "test")
	t.Run("QueryMetaCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Query Meta Case failed : wanted %v and got %v", want, err)
		}
	})

	//success case
	fakeDao := new([]Meta)
	fakeDaoArray := make([]Meta, 1)
	fakeDaoArray[0] = Meta{Key: "Test"}
	fakeDao = &fakeDaoArray
	querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDao).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	meta, err = QueryMeta("test", "test")
	t.Run("QueryMetaSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Query Meta Case Failed, wanted error nil and got error %v", err)
		}
		want := 1
		if want != len(*meta) {
			t.Errorf("Query Meta Case failed wanted length %v and got length %v", want, len(*meta))
		}
	})
}

// TestQueryAllMeta is function to test QueryAllMeta
func TestQueryAllMeta(t *testing.T) {
	//failure case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), failedDBOperationErr).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	meta, err := QueryAllMeta("test", "test")
	t.Run("QueryAllMetaCheckError", func(t *testing.T) {
		want := failedDBOperationErr
		if want != err {
			t.Errorf("Query All Meta Case failed : wanted %v and got %v", want, err)
		}
	})

	//success case
	querySeterMock.EXPECT().All(gomock.Any()).Return(int64(1), nil).Times(1)
	querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(querySeterMock).Times(1)
	ormerMock.EXPECT().QueryTable(gomock.Any()).Return(querySeterMock).Times(1)
	meta, err = QueryAllMeta("test", "test")
	t.Run("QueryAllMetaSuccessCase", func(t *testing.T) {
		if err != nil {
			t.Errorf("Query Meta All Case Failed, wanted error nil and got error %v", err)
		}
		want := 0
		if want != len(*meta) {
			t.Errorf("Query All Meta Case failed wanted length %v and got length %v", want, len(*meta))
		}
	})
}
