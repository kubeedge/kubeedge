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

Source: database/sql/sql.go (interfaces: Result)

Package beego is a generated GoMock package.
*/

package beego

import (
	"reflect"

	"github.com/golang/mock/gomock"
)

// MockSQLResult is a mock of SQL Result
type MockSQLResult struct {
	ctrl     *gomock.Controller
	recorder *MockSQLResultMockRecorder
}

// MockSQLResultMockRecorder is the mock recorder for MockSQLResult
type MockSQLResultMockRecorder struct {
	mock *MockSQLResult
}

// NewMockDriverRes creates a new mock instance
func NewMockDriverRes(ctrl *gomock.Controller) *MockSQLResult {
	mock := &MockSQLResult{ctrl: ctrl}
	mock.recorder = &MockSQLResultMockRecorder{mock: mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockSQLResult) EXPECT() *MockSQLResultMockRecorder {
	return m.recorder
}

// RowsAffected indicates an expected call of RowsAffected
func (mr *MockSQLResultMockRecorder) RowsAffected() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RowsAffected", reflect.TypeOf((*MockSQLResult)(nil).RowsAffected))
}

// LastInsertId mocks base LastInsertId method
func (m *MockSQLResult) LastInsertId() (int64, error) {
	return 1, nil
}

// RowsAffected mocks base RowsAffected method
func (m *MockSQLResult) RowsAffected() (int64, error) {
	ret := m.ctrl.Call(m, "RowsAffected")
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}
