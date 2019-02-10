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

	"github.com/astaxie/beego/orm"
	"github.com/golang/mock/gomock"
)

// TestSaveDeviceTwin is function to test SaveDeviceTwin
func TestSaveDeviceTwin(t *testing.T) {
	//Initialize Global Variables (Mocks)
	initMocks(t)

	cases := []struct {
		// name is name of the testcase
		name string
		// returnInt is first return of mock interface ormerMock
		returnInt int64
		// returnErr is second return of mock interface ormerMock which is also expected error
		returnErr error
	}{{
		// Success Case
		name:      "SuccessCase",
		returnInt: int64(1),
		returnErr: nil,
	}, {
		// Failure Case
		name:      "FailureCase",
		returnInt: int64(1),
		returnErr: errFailedDBOperation,
	},
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().Insert(gomock.Any()).Return(test.returnInt, test.returnErr).Times(1)
			err := SaveDeviceTwin(&DeviceTwin{})
			if test.returnErr != err {
				t.Errorf("Save Device Twin Case failed: wanted error %v and got error %v", test.returnErr, err)
			}
		})
	}
}

// TestDeleteDeviceTwinByDeviceID is function to test DeleteDeviceTwinByDeviceID
func TestDeleteDeviceTwinByDeviceID(t *testing.T) {
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

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := DeleteDeviceTwinByDeviceID("test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceTwinByDeviceID Case failed: wanted error %v and got error %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestDeleteDeviceTwin is function to test DeleteDeviceTwin
func TestDeleteDeviceTwin(t *testing.T) {
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

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := DeleteDeviceTwin("test", "test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceTwin Case failed: wanted error %v and got error %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceTwinField is function to test UpdateDeviceTwinField
func TestUpdateDeviceTwinField(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// updateReturnInt is the first return of mock interface querySeterMock's update function
		updateReturnInt int64
		// updateReturnErr is the second return of mock interface querySeterMocks's update function also expected error
		updateReturnErr error
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
	}{{
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

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceTwinField("test", "test", "test", "test")
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceTwinField Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceTwinFields is function to test UpdateDeviceTwinFields
func TestUpdateDeviceTwinFields(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// updateReturnInt is the first return of mock interface querySeterMock's update function
		updateReturnInt int64
		// updateReturnErr is the second return of mock interface querySeterMocks's update function also expected error
		updateReturnErr error
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
	}{{
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

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceTwinFields("test", "test", make(map[string]interface{}))
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceTwinFields Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestQueryDeviceTwin is function to test QueryDeviceTwin
func TestQueryDeviceTwin(t *testing.T) {
	cases := []struct {
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
	}{{
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

	// fakeDeviceTwin is used to set the argument of All function
	fakeDeviceTwin := new([]DeviceTwin)
	fakeDeviceTwinArray := make([]DeviceTwin, 1)
	fakeDeviceTwinArray[0] = DeviceTwin{DeviceID: "Test"}
	fakeDeviceTwin = &fakeDeviceTwinArray

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceTwin).Return(test.allReturnInt, test.allReturnErr).Times(1)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			device, err := QueryDeviceTwin("test", "test")
			if test.allReturnErr != err {
				t.Errorf("QueryDeviceTwin Case failed: wanted error %v and got error %v", test.allReturnErr, err)
			}

			if err == nil {
				if len(*device) != 1 {
					t.Errorf("QueryDeviceTwin Case failed: wanted length 1 and got length %v", len(*device))
				}
			}
		})
	}
}

// TestUpdateDeviceTwinMulti is function to test UpdateDeviceTwinMulti
func TestUpdateDeviceTwinMulti(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// updateReturnInt is the first return of mock interface querySeterMock's update function
		updateReturnInt int64
		// updateReturnErr is the second return of mock interface querySeterMocks's update function also expected error
		updateReturnErr error
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
	}{{
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

	// updateDevice is argument to UpdateDeviceTwinMulti function
	updateDevice := make([]DeviceTwinUpdate, 0)
	updateDevice = append(updateDevice, DeviceTwinUpdate{DeviceID: "test"})

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceTwinMulti(updateDevice)
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceTwinMulti Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestDeviceTwinTrans is function to test DeviceTwinTrans
func TestDeviceTwinTrans(t *testing.T) {
	cases := []struct {
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
	}{{
		// Failure Case SaveDeviceTwin
		name:             "DeviceTwinTransSaveDeviceTwinFailureCase",
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
		// Failure Case DeleteDeviceTwin
		name:             "DeviceTwinTransDeleteDeviceTwinFailureCase",
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
		// Failure Case UpdateDeviceTwinFields
		name:             "DeviceTwinTransUpdateDeviceTwinFieldsFailureCase",
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
		name:             "DeviceTwinTransSuccessCase",
		rollBackTimes:    0,
		commitTimes:      1,
		beginTimes:       1,
		filterReturn:     querySeterMock,
		filterTimes:      6,
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

	// adds is fake DeviceTwin used as argument
	adds := make([]DeviceTwin, 0)
	// deletes is fake DeviceDelete used as argument
	deletes := make([]DeviceDelete, 0)
	// updates is fake DeviceTwinUpdate used as argument
	updates := make([]DeviceTwinUpdate, 0)
	adds = append(adds, DeviceTwin{DeviceID: "Test"})
	deletes = append(deletes, DeviceDelete{DeviceID: "test", Name: "test"})
	updates = append(updates, DeviceTwinUpdate{DeviceID: "test", Name: "test", Cols: make(map[string]interface{})})

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().Rollback().Return(nil).Times(test.rollBackTimes)
			ormerMock.EXPECT().Commit().Return(nil).Times(test.commitTimes)
			ormerMock.EXPECT().Begin().Return(nil).Times(test.beginTimes)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.filterTimes)
			ormerMock.EXPECT().Insert(gomock.Any()).Return(test.insertReturnInt, test.insertReturnErr).Times(test.insertTimes)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(test.deleteTimes)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(test.updateTimes)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.queryTableTimes)
			err := DeviceTwinTrans(adds, deletes, updates)
			if test.wantErr != err {
				t.Errorf("TestDeviceTwinTrans Case failed: wanted error %v and got error %v", test.wantErr, err)
			}
		})
	}
}
