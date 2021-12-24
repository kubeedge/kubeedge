/*
Copyright 2021 The KubeEdge Authors.

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

var testURL = TargetUrls{
	URL: "http://127.0.0.1/test",
}

// TestInsertUrls is function to test InsertTopics
func TestInsertUrls(t *testing.T) {
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
			err := InsertUrls(testURL.URL)
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
			err := DeleteUrlsByKey("http://127.0.0.1/test")
			if test.deleteReturnErr != err {
				t.Errorf("Delete Meta By Key Case failed : wanted %v and got %v", test.deleteReturnErr, err)
			}
		})
	}
}

func TestIsTableEmpty(t *testing.T) {
	// Initialize Global Variables (Mocks)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

	cases := []struct {
		// name is name of the testcase
		name string
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
		// result of the function
		Result bool
	}{{
		// not empty
		name:             "count > 0",
		queryTableReturn: querySeterMock,
		Result:           false,
	},
		{
			// empty
			name:             "count = 0",
			queryTableReturn: querySeterMock,
			Result:           true,
		}}
	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "count > 0" {
				ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
				querySeterMock.EXPECT().Count().Return(int64(1), nil)
				if test.Result != IsTableEmpty() {
					t.Errorf("except false but get true")
				}
			} else {
				ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
				querySeterMock.EXPECT().Count().Return(int64(0), nil)
				if test.Result != IsTableEmpty() {
					t.Errorf("except true but get false")
				}
			}
		})
	}
}

func TestGetUrlsByKey(t *testing.T) {
	// Initialize Global Variables (Mocks)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	ormerMock := beego.NewMockOrmer(mockCtrl)
	querySeterMock := beego.NewMockQuerySeter(mockCtrl)
	dbm.DBAccess = ormerMock

	cases := []struct {
		// name is name of the testcase
		name string
		// key os the param of the functoion
		key string
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// returnErr is second return of mock interface rawSeterMock's Exec function which is also expected error
		returnErr error
	}{{
		name:             "SuccessCase",
		key:              "test",
		queryTableReturn: querySeterMock,
		filterReturn:     querySeterMock,
		returnErr:        nil,
	},
		{
			name:             "FailureCase",
			key:              "test",
			queryTableReturn: querySeterMock,
			filterReturn:     querySeterMock,
			returnErr:        errFailedDBOperation,
		}}
	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().One(gomock.Any()).Return(test.returnErr).Times(1)
			if _, err := GetUrlsByKey(test.key); test.returnErr != err {
				t.Errorf("get url By key case failed : wanted %v and got %v", test.returnErr, err)
			}
		})
	}
}
