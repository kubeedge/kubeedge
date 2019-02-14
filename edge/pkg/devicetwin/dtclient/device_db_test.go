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

	"github.com/astaxie/beego/orm"
	"github.com/golang/mock/gomock"

	"github.com/kubeedge/kubeedge/edge/mocks/beego"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

// errFailedDBOperation is common DB operation fail error
var errFailedDBOperation = errors.New("Failed DB Operation")

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
			err := SaveDevice(&Device{})
			if test.returnErr != err {
				t.Errorf("SaveDevice case failed: wanted error %v and got error %v", test.returnErr, err)
			}
		})
	}
}

// TestDeleteDeviceByID is function to test DeleteDeviceByID
func TestDeleteDeviceByID(t *testing.T) {
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
			err := DeleteDeviceByID("test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceByID case failed: wanted %v and got %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceField is function to test UpdateDeviceField
func TestUpdateDeviceField(t *testing.T) {
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
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceField("test", "test", "test")
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceField case failed: wanted %v and got %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceFields is function to test UpdateDeviceFields
func TestUpdateDeviceFields(t *testing.T) {
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
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceFields("test", make(map[string]interface{}))
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceFields case failed: wanted %v and got %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestQueryDevice is function to test QueryDevice
func TestQueryDevice(t *testing.T) {
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

	// fakeDevice is used to set the argument of All function
	fakeDevice := new([]Device)
	fakeDeviceArray := make([]Device, 1)
	fakeDeviceArray[0] = Device{ID: "Test"}
	fakeDevice = &fakeDeviceArray

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(1)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			device, err := QueryDevice("test", "test")
			if test.allReturnErr != err {
				t.Errorf("QueryDevice case failed: wanted error %v and got error %v", test.allReturnErr, err)
				return
			}

			if err == nil {
				if len(*device) != 1 {
					t.Errorf("QueryDevice case failed: wanted length 1 and got length %v", len(*device))
				}
			}
		})
	}
}

// TestQueryDeviceAll is function to test QueryDeviceAll
func TestQueryDeviceAll(t *testing.T) {
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

	// fakeDevice is used to set the argument of All function
	fakeDevice := new([]Device)
	fakeDeviceArray := make([]Device, 1)
	fakeDeviceArray[0] = Device{ID: "Test"}
	fakeDevice = &fakeDeviceArray

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(1)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			device, err := QueryDeviceAll()
			if test.allReturnErr != err {
				t.Errorf("QueryDeviceAll case failed: wanted error %v and got error %v", test.allReturnErr, err)
				return
			}

			if err == nil {
				if len(*device) != 1 {
					t.Errorf("QueryDeviceAll case failed: wanted length 1 and got length %v", len(*device))
				}
			}
		})
	}
}

// TestUpdateDeviceMulti is function to test UpdateDeviceMulti
func TestUpdateDeviceMulti(t *testing.T) {
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

	// updateDevice is argument to UpdateDeviceMulti function
	updateDevice := make([]DeviceUpdate, 0)
	updateDevice = append(updateDevice, DeviceUpdate{DeviceID: "test"})

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceMulti(updateDevice)
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceMulti case failed: wanted %v and got %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestAddDeviceTrans is function to test AddDeviceTrans
func TestAddDeviceTrans(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// rollBackTimes is number of times rollback is expected
		rollBackTimes int
		// commitTimes is number of times commit is expected
		commitTimes int
		// beginTimes is number of times begin is expected
		beginTimes int
		// successInsertReturnInt is the first return of mock interface ormerMock's Insert function success case
		successInsertReturnInt int64
		// successInsertReturnErr is the second return of mock interface ormerMock's Insert function success case
		successInsertReturnErr error
		// successInsertTimes is number of times successful insert is expected
		successInsertTimes int
		// failInsertReturnInt is the first return of mock interface ormerMock's Insert function error case
		failInsertReturnInt int64
		// failInsertReturnErr is the second return of mock interface ormerMock's Insert function error case
		failInsertReturnErr error
		// failInsertTimes is number of times fail insert is expected
		failInsertTimes int
		// wantErr is expected error
		wantErr error
	}{{
		// Failure Case SaveDevice
		name:                   "FailureCaseSaveDevice",
		rollBackTimes:          1,
		commitTimes:            0,
		beginTimes:             1,
		successInsertReturnInt: int64(1),
		successInsertReturnErr: nil,
		successInsertTimes:     0,
		failInsertReturnInt:    int64(1),
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        1,
		wantErr:                errFailedDBOperation,
	}, {
		// Failure Case SaveDeviceAttr
		name:                   "FailureCaseSaveDeviceAttr",
		rollBackTimes:          1,
		commitTimes:            0,
		beginTimes:             1,
		successInsertReturnInt: int64(1),
		successInsertReturnErr: nil,
		successInsertTimes:     1,
		failInsertReturnInt:    int64(1),
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        1,
		wantErr:                errFailedDBOperation,
	}, {
		// Failure Case SaveDeviceTwin
		name:                   "FailureCaseSaveDeviceAttr",
		rollBackTimes:          1,
		commitTimes:            0,
		beginTimes:             1,
		successInsertReturnInt: int64(1),
		successInsertReturnErr: nil,
		successInsertTimes:     2,
		failInsertReturnInt:    int64(1),
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        1,
		wantErr:                errFailedDBOperation,
	}, {
		// Success Case SaveDeviceTwin
		name:                   "SuccessCaseSaveDeviceAttr",
		rollBackTimes:          0,
		commitTimes:            1,
		beginTimes:             1,
		successInsertReturnInt: int64(1),
		successInsertReturnErr: nil,
		successInsertTimes:     3,
		failInsertReturnInt:    int64(1),
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        0,
		wantErr:                nil,
	},
	}

	// adds is fake Device used as argument
	adds := make([]Device, 0)
	adds = append(adds, Device{ID: "test"})
	// addAttrs is fake DeviceAttr used as argument
	addAttrs := make([]DeviceAttr, 0)
	addAttrs = append(addAttrs, DeviceAttr{DeviceID: "test"})
	// addTwins is fake DeviceTwin used as argument
	addTwins := make([]DeviceTwin, 0)
	addTwins = append(addTwins, DeviceTwin{DeviceID: "test"})

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().Begin().Return(nil).Times(test.beginTimes)
			// success insert
			ormerMock.EXPECT().Insert(gomock.Any()).Return(test.successInsertReturnInt, test.successInsertReturnErr).Times(test.successInsertTimes)
			// fail insert
			ormerMock.EXPECT().Insert(gomock.Any()).Return(test.failInsertReturnInt, test.failInsertReturnErr).Times(test.failInsertTimes)
			ormerMock.EXPECT().Commit().Return(nil).Times(test.commitTimes)
			ormerMock.EXPECT().Rollback().Return(nil).Times(test.rollBackTimes)
			err := AddDeviceTrans(adds, addAttrs, addTwins)
			if test.wantErr != err {
				t.Errorf("AddDeviceTrans case failed: wanted error %v and got error %v", test.wantErr, err)
			}
		})
	}
}

// TestDeleteDeviceTrans is function to test DeleteDeviceTrans
func TestDeleteDeviceTrans(t *testing.T) {
	cases := []struct {
		// name is name of the testcase
		name string
		// rollBackTimes is number of times rollback is expected
		rollBackTimes int
		// commitTimes is number of times commit is expected
		commitTimes int
		// beginTimes is number of times begin is expected
		beginTimes int
		// successDeleteReturnInt is the first return of mock interface ormerMock's delete function success case
		successDeleteReturnInt int64
		// successDeleteReturnErr is the second return of mock interface ormerMock's delete function success case
		successDeleteReturnErr error
		// successDeleteTimes is number of times successful delete is expected
		successDeleteTimes int
		// failDeleteReturnInt is the first return of mock interface ormerMock's delete function error case
		failDeleteReturnInt int64
		// failDeleteReturnErr is the second return of mock interface ormerMock's delete function error case
		failDeleteReturnErr error
		// failDeleteTimes is number of times fail Delete is expected
		failDeleteTimes int
		// queryTableReturn is the return of mock interface ormerMock's QueryTable function
		queryTableReturn orm.QuerySeter
		// queryTableTimes is the number of times queryTable is called
		queryTableTimes int
		// filterReturn is the return of mock interface querySeterMock's filter function
		filterReturn orm.QuerySeter
		// filterTimes is the number of times filter is called
		filterTimes int
		// wantErr is expected error
		wantErr error
	}{{
		// Failure Case DeleteDeviceByID
		name:                   "FailureCaseDeleteDeviceByID",
		rollBackTimes:          1,
		commitTimes:            0,
		beginTimes:             1,
		successDeleteReturnInt: int64(1),
		successDeleteReturnErr: nil,
		successDeleteTimes:     0,
		failDeleteReturnInt:    int64(1),
		failDeleteReturnErr:    errFailedDBOperation,
		failDeleteTimes:        1,
		queryTableReturn:       querySeterMock,
		queryTableTimes:        1,
		filterReturn:           querySeterMock,
		filterTimes:            1,
		wantErr:                errFailedDBOperation,
	}, {
		// Failure Case DeleteDeviceAttrByDeviceID
		name:                   "FailureCaseDeleteDeviceAttrByDeviceID",
		rollBackTimes:          1,
		commitTimes:            0,
		beginTimes:             1,
		successDeleteReturnInt: int64(1),
		successDeleteReturnErr: nil,
		successDeleteTimes:     1,
		failDeleteReturnInt:    int64(1),
		failDeleteReturnErr:    errFailedDBOperation,
		failDeleteTimes:        1,
		queryTableReturn:       querySeterMock,
		queryTableTimes:        2,
		filterReturn:           querySeterMock,
		filterTimes:            2,
		wantErr:                errFailedDBOperation,
	}, {
		// Failure Case DeleteDeviceTwinByDeviceID
		name:                   "FailureCaseDeleteDeviceTwinByDeviceID",
		rollBackTimes:          1,
		commitTimes:            0,
		beginTimes:             1,
		successDeleteReturnInt: int64(1),
		successDeleteReturnErr: nil,
		successDeleteTimes:     2,
		failDeleteReturnInt:    int64(1),
		failDeleteReturnErr:    errFailedDBOperation,
		failDeleteTimes:        1,
		queryTableReturn:       querySeterMock,
		queryTableTimes:        3,
		filterReturn:           querySeterMock,
		filterTimes:            3,
		wantErr:                errFailedDBOperation,
	}, {
		// Success Case
		name:                   "SuccessCase",
		rollBackTimes:          0,
		commitTimes:            1,
		beginTimes:             1,
		successDeleteReturnInt: int64(1),
		successDeleteReturnErr: nil,
		successDeleteTimes:     3,
		failDeleteReturnInt:    int64(1),
		failDeleteReturnErr:    errFailedDBOperation,
		failDeleteTimes:        0,
		queryTableReturn:       querySeterMock,
		queryTableTimes:        3,
		filterReturn:           querySeterMock,
		filterTimes:            3,
		wantErr:                nil,
	},
	}

	// deletes is argument to DeleteDeviceTrans function
	deletes := []string{"test"}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().Begin().Return(nil).Times(test.beginTimes)
			ormerMock.EXPECT().Rollback().Return(nil).Times(test.rollBackTimes)
			ormerMock.EXPECT().Commit().Return(nil).Times(test.commitTimes)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.filterTimes)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.queryTableTimes)
			// success delete
			querySeterMock.EXPECT().Delete().Return(test.successDeleteReturnInt, test.successDeleteReturnErr).Times(test.successDeleteTimes)
			// fail delete
			querySeterMock.EXPECT().Delete().Return(test.failDeleteReturnInt, test.failDeleteReturnErr).Times(test.failDeleteTimes)
			err := DeleteDeviceTrans(deletes)
			if test.wantErr != err {
				t.Errorf("DeleteDeviceTrans Case failed : wanted %v and got %v", test.wantErr, err)
			}
		})
	}
}
