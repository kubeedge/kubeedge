/*
Copyright 2023 The KubeEdge Authors.

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

	"github.com/beego/beego/v2/client/orm"

	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/pkg/testtools"
)

// errFailedDBOperation is common DB operation fail error
var errFailedDBOperation = errors.New("Failed DB Operation")

// CasesSaveStr is a struct for cases of save
type CasesSaveStr []struct {
	// name is name of the testcase
	name string
	// doTXReturnErr is return of mock interface ormerMock's DoTX function
	doTXReturnErr error
}

// CasesDeleteStr is a struct for cases of delete
type CasesDeleteStr []struct {
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
	// doTXReturnErr is return of mock interface ormerMock's DoTX function
	doTXReturnErr error
}

// CasesUpdateStr is a struct for cases of update
type CasesUpdateStr []struct {
	// name is name of the testcase
	name string
	// filterReturn is the return of mock interface querySeterMock's filter function
	filterReturn orm.QuerySeter
	// updateReturnInt is the first return of mock interface querySeterMock's update function
	updateReturnInt int64
	// updateReturnErr is the second return of mock interface querySeterMock's update function also expected error
	updateReturnErr error
	// queryTableReturn is the return of mock interface ormerMock's QueryTable function
	queryTableReturn orm.QuerySeter
}

// CasesQueryStr is a struct for cases of query
type CasesQueryStr []struct {
	// name is name of the testcase
	name string
	// filterReturn is the return of mock interface querySeterMock's filter function
	filterReturn orm.QuerySeter
	// allReturnInt is the first return of mock interface querySeterMock's all function
	allReturnInt int64
	// allReturnErr is the second return of mock interface querySeterMocks's all function also expected error
	allReturnErr error
	// queryTableReturn is the return of mock interface ormerMock's QueryTable function
	queryTableReturn orm.QuerySeter
	// beginReturn is the return of mock interface ormerMock's Begin function
	beginReturn orm.TxOrmer
}

// CasesTransStr is a struct for cases of trans
type CasesTransStr []struct {
	// name is name of the testcase
	name string
	// rollBackTimes is number of times rollback is expected
	rollBackTimes int
	// commitTimes is number of times commit is expected
	commitTimes int
	// beginTimes is number of times begin is expected
	beginTimes int
	// filterReturn is the return of mock interface querySeterMock's filter function
	filterReturn orm.QuerySeter
	// filterTimes is the number of times filter is called
	filterTimes int
	// insertReturnInt is the first return of mock interface ormerMock's Insert function
	insertReturnInt int64
	// insertReturnErr is the second return of mock interface ormerMock's Insert function
	insertReturnErr error
	// insertTimes is number of times Insert is expected
	insertTimes int
	// deleteReturnInt is the first return of mock interface ormerMock's Delete function
	deleteReturnInt int64
	// deleteReturnErr is the second return of mock interface ormerMock's Delete function
	deleteReturnErr error
	// deleteTimes is number of times Delete is expected
	deleteTimes int
	// updateReturnInt is the first return of mock interface ormerMock's Update function
	updateReturnInt int64
	// updateReturnErr is the second return of mock interface ormerMock's Update function
	updateReturnErr error
	// updateTimes is number of times Update is expected
	updateTimes int
	// queryTableReturn is the return of mock interface ormerMock's QueryTable function
	queryTableReturn orm.QuerySeter
	// queryTableTimes is the number of times queryTable is called
	queryTableTimes int
	// wantErr is expected error
	wantErr error
}

// GetCasesSave get cases for save
func GetCasesSave(t *testing.T) (*beego.MockOrmer, CasesSaveStr) {
	ormerMock, _ := testtools.InitOrmerMock(t)
	return ormerMock, CasesSaveStr{{
		// Success Case
		name:          "SuccessCase",
		doTXReturnErr: nil,
	}, {
		// Failure Case
		name:          "FailureCase",
		doTXReturnErr: errFailedDBOperation,
	},
	}
}

// GetCasesDelete get cases for delete
func GetCasesDelete(t *testing.T) (*beego.MockOrmer, *beego.MockQuerySeter, CasesDeleteStr) {
	ormerMock, querySeterMock := testtools.InitOrmerMock(t)
	return ormerMock, querySeterMock, CasesDeleteStr{{
		// Success Case
		name:             "SuccessCase",
		filterReturn:     querySeterMock,
		deleteReturnInt:  int64(1),
		deleteReturnErr:  nil,
		queryTableReturn: querySeterMock,
	}, {
		// Failure Case
		name:             "FailureCase",
		filterReturn:     querySeterMock,
		deleteReturnInt:  int64(0),
		deleteReturnErr:  errFailedDBOperation,
		queryTableReturn: querySeterMock,
	},
	}
}

// GetCasesUpdate get cases for update
func GetCasesUpdate(t *testing.T) (*beego.MockOrmer, *beego.MockQuerySeter, CasesUpdateStr) {
	ormerMock, querySeterMock := testtools.InitOrmerMock(t)
	return ormerMock, querySeterMock, CasesUpdateStr{{
		// Success Case
		name:             "SuccessCase",
		filterReturn:     querySeterMock,
		updateReturnInt:  int64(1),
		updateReturnErr:  nil,
		queryTableReturn: querySeterMock,
	}, {
		// Failure Case
		name:             "FailureCase",
		filterReturn:     querySeterMock,
		updateReturnInt:  int64(0),
		updateReturnErr:  errFailedDBOperation,
		queryTableReturn: querySeterMock,
	},
	}
}

// GetCasesQuery get cases for query
func GetCasesQuery(t *testing.T) (*beego.MockOrmer, *beego.MockQuerySeter, CasesQueryStr) {
	ormerMock, querySeterMock := testtools.InitOrmerMock(t)
	return ormerMock, querySeterMock, CasesQueryStr{{
		// Success Case
		name:             "SuccessCase",
		filterReturn:     querySeterMock,
		allReturnInt:     int64(1),
		allReturnErr:     nil,
		queryTableReturn: querySeterMock,
	}, {
		// Failure Case
		name:             "FailureCase",
		filterReturn:     querySeterMock,
		allReturnInt:     int64(0),
		allReturnErr:     errFailedDBOperation,
		queryTableReturn: querySeterMock,
	},
	}
}

// GetCasesTrans get cases for trans
func GetCasesTrans(caseName string, t *testing.T) (*beego.MockOrmer, *beego.MockQuerySeter, CasesTransStr) {
	ormerMock, querySeterMock := testtools.InitOrmerMock(t)
	return ormerMock, querySeterMock, CasesTransStr{{
		// Failure Case SaveDeviceAttr
		name:             caseName + "TransSaveFailureCase",
		rollBackTimes:    1,
		commitTimes:      0,
		beginTimes:       1,
		filterReturn:     nil,
		filterTimes:      0,
		insertReturnInt:  int64(1),
		insertReturnErr:  errFailedDBOperation,
		insertTimes:      1,
		deleteReturnInt:  int64(1),
		deleteReturnErr:  nil,
		deleteTimes:      0,
		updateReturnInt:  int64(1),
		updateReturnErr:  nil,
		updateTimes:      0,
		queryTableReturn: nil,
		queryTableTimes:  0,
		wantErr:          errFailedDBOperation,
	}, {
		// Failure Case DeleteDeviceAttr
		name:             caseName + "TransDeleteFailureCase",
		rollBackTimes:    1,
		commitTimes:      0,
		beginTimes:       1,
		filterReturn:     querySeterMock,
		filterTimes:      2,
		insertReturnInt:  int64(1),
		insertReturnErr:  nil,
		insertTimes:      1,
		deleteReturnInt:  int64(1),
		deleteReturnErr:  errFailedDBOperation,
		deleteTimes:      1,
		updateReturnInt:  int64(1),
		updateReturnErr:  nil,
		updateTimes:      0,
		queryTableReturn: querySeterMock,
		queryTableTimes:  1,
		wantErr:          errFailedDBOperation,
	}, {
		// Failure Case UpdateDeviceAttrFields
		name:             caseName + "TransUpdateFieldsFailureCase",
		rollBackTimes:    1,
		commitTimes:      0,
		beginTimes:       1,
		filterReturn:     querySeterMock,
		filterTimes:      4,
		insertReturnInt:  int64(1),
		insertReturnErr:  nil,
		insertTimes:      1,
		deleteReturnInt:  int64(1),
		deleteReturnErr:  nil,
		deleteTimes:      1,
		updateReturnInt:  int64(1),
		updateReturnErr:  errFailedDBOperation,
		updateTimes:      1,
		queryTableReturn: querySeterMock,
		queryTableTimes:  2,
		wantErr:          errFailedDBOperation,
	}, {
		// Success Case
		name:             caseName + "TransSuccessCase",
		rollBackTimes:    0,
		commitTimes:      1,
		beginTimes:       1,
		filterReturn:     querySeterMock,
		filterTimes:      4,
		insertReturnInt:  int64(1),
		insertReturnErr:  nil,
		insertTimes:      1,
		deleteReturnInt:  int64(1),
		deleteReturnErr:  nil,
		deleteTimes:      1,
		updateReturnInt:  int64(1),
		updateReturnErr:  nil,
		updateTimes:      1,
		queryTableReturn: querySeterMock,
		queryTableTimes:  2,
		wantErr:          nil,
	},
	}
}
