/*
Copyright 2020 The KubeEdge Authors.

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
	"database/sql"
	"errors"
	"testing"

	"github.com/astaxie/beego/orm"
	"github.com/golang/mock/gomock"

	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

const (
	// FailedDBOperation is common Database operation fail message
	FailedDBOperation = "Failed DB Operation"
)

var errFailedDBOperation = errors.New(FailedDBOperation)

var testTopic = SubTopics{
	Topic: "TestTopic",
}

// TestInsertTopics is function to test InsertTopics
func TestInsertTopics(t *testing.T) {
	// Initialize Global Variables (Mocks)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	rawSeterMock := beego.NewMockRawSeter(mockCtrl)
	dbm.DBAccess = ormerMock

	cases := []struct {
		// name is name of the testcase
		name string
		// returnSQL is first return of mock interface rawSeterMock's Exec function
		returnSQL sql.Result
		// returnErr is second return of mock interface rawSeterMock's Exec function which is also expected error
		returnErr error
		// returnRaw is the return of mock interface ormerMock's Raw function
		returnRaw orm.RawSeter
	}{{
		// Success Case
		name:      "SuccessCase",
		returnSQL: nil,
		returnErr: nil,
		returnRaw: rawSeterMock,
	}, {
		// Failure Case
		name:      "FailureCase",
		returnSQL: nil,
		returnErr: errFailedDBOperation,
		returnRaw: rawSeterMock,
	},
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			rawSeterMock.EXPECT().Exec().Return(test.returnSQL, test.returnErr).Times(1)
			ormerMock.EXPECT().Raw(gomock.Any(), gomock.Any()).Return(test.returnRaw).Times(1)
			err := InsertTopics(testTopic.Topic)
			if test.returnErr != err {
				t.Errorf("Insert or Update Meta Case failed : wanted %v and got %v", test.returnErr, err)
			}
		})
	}
}

// TestDeleteTopicsByKey is function to test DeleteTopicsByKey
func TestDeleteTopicsByKey(t *testing.T) {
	// Initialize Global Variables (Mocks)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

	cases := []struct {
		// name is name of the testcase
		name string
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// deleteReturnInt is the first return of mock interface querySeterMock's delete function
		deleteReturnInt int64
		// deleteReturnErr is the second return of mock interface querySeterMocks's delete function also expected error
		deleteReturnErr error
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
	}{{
		// Failure Case
		name:             "FailureCase",
		filterReturn:     querySeterMock,
		deleteReturnInt:  int64(0),
		deleteReturnErr:  errFailedDBOperation,
		queryTableReturn: querySeterMock,
	},
		{
			// Success Case
			name:             "SuccessCase",
			filterReturn:     querySeterMock,
			deleteReturnInt:  int64(1),
			deleteReturnErr:  nil,
			queryTableReturn: querySeterMock,
		}}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := DeleteTopicsByKey("test")
			if test.deleteReturnErr != err {
				t.Errorf("Delete Meta By Key Case failed : wanted %v and got %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestQueryAllTopics is function to test QueryAllTopics
func TestQueryAllTopics(t *testing.T) {
	// Initialize Global Variables (Mocks)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

	cases := []struct {
		// name is name of the testcase
		name string
		// allReturnInt is the first return of mock interface querySeterMock's all function
		allReturnInt int64
		// allReturnErr is the second return of mock interface querySeterMocks's all function also expected error
		allReturnErr error
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
	}{{
		// Success Case
		name:             "SuccessCase",
		allReturnInt:     int64(1),
		allReturnErr:     nil,
		queryTableReturn: querySeterMock,
	}, {
		// Failure Case
		name:             "FailureCase",
		allReturnInt:     int64(0),
		allReturnErr:     errFailedDBOperation,
		queryTableReturn: querySeterMock,
	},
	}

	// fakeDao is used to set the argument of All function
	fakeDao := new([]SubTopics)
	fakeDaoArray := make([]SubTopics, 1)
	fakeDaoArray[0] = SubTopics{Topic: "Test"}
	fakeDao = &fakeDaoArray

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDao).Return(test.allReturnInt, test.allReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			event, err := QueryAllTopics()
			if test.allReturnErr != err {
				t.Errorf("Query Event Case Failed : wanted error %v and got error %v", test.allReturnErr, err)
				return
			}

			if err == nil {
				if len(*event) != 1 {
					t.Errorf("Query Event Case failed: wanted length 1 and got length %v", len(*event))
				}
			}
		})
	}
}
