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

	"github.com/beego/beego/v2/client/orm"
	"github.com/golang/mock/gomock"

	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"github.com/kubeedge/kubeedge/pkg/testtools"
)

// TestSaveDevice is used to test SaveDevice
func TestSaveDevice(t *testing.T) {
	ormerMock, cases := GetCasesSave(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.doTXReturnErr).Times(1)
			err := SaveDevice(ormerMock, &Device{})
			if test.doTXReturnErr != err {
				t.Errorf("SaveDevice case failed: wanted error %v and got error %v", test.doTXReturnErr, err)
			}
		})
	}
}

// TestDeleteDeviceByID is function to test DeleteDeviceByID
func TestDeleteDeviceByID(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesDelete(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(1)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(1)
			err := DeleteDeviceByID(ormerMock, "test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceByID case failed: wanted %v and got %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceField is function to test UpdateDeviceField
func TestUpdateDeviceField(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

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
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

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
	ormerMock, querySeterMock, cases := GetCasesQuery(t)

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
	ormerMock, querySeterMock, cases := GetCasesQuery(t)

	// fakeDevice is used to set the argument of All function
	fakeDevice := new([]Device)
	fakeDeviceArray := make([]Device, 1)
	fakeDeviceArray[0] = Device{ID: "Test"}
	fakeDevice = &fakeDeviceArray

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDevice).Return(test.allReturnInt, test.allReturnErr).Times(1)
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
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

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
	ormerMock, _ := testtools.InitOrmerMock(t)

	cases := []struct {
		// name is name of the testcase
		name string
		// successInsertReturnErr is the second return of mock interface ormerMock's Insert function success case
		successInsertReturnErr error
		// successInsertTimes is number of times successful insert is expected
		successInsertTimes int
		// failInsertReturnErr is the second return of mock interface ormerMock's Insert function error case
		failInsertReturnErr error
		// failInsertTimes is number of times fail insert is expected
		failInsertTimes int
		// wantErr is expected error
		wantErr error
		// doTXReturnErr is return of mock interface ormerMock's DoTX function
		doTXReturnErr error
	}{{
		// Failure Case SaveDevice
		name:                   "FailureCaseSaveDevice",
		successInsertReturnErr: nil,
		successInsertTimes:     0,
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        1,
		wantErr:                errFailedDBOperation,
	}, {
		// Failure Case SaveDeviceAttr
		name:                   "FailureCaseSaveDeviceAttr",
		successInsertReturnErr: nil,
		successInsertTimes:     1,
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        1,
		wantErr:                errFailedDBOperation,
	}, {
		// Failure Case SaveDeviceTwin
		name:                   "FailureCaseSaveDeviceAttr",
		successInsertReturnErr: nil,
		successInsertTimes:     2,
		failInsertReturnErr:    errFailedDBOperation,
		failInsertTimes:        1,
		wantErr:                errFailedDBOperation,
	}, {
		// Success Case SaveDeviceTwin
		name:                   "SuccessCaseSaveDeviceAttr",
		successInsertReturnErr: nil,
		successInsertTimes:     3,
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
			// DoTx success insert
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.successInsertReturnErr).Times(test.successInsertTimes)
			// DoTx fail insert
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.failInsertReturnErr).Times(test.failInsertTimes)
			err := AddDeviceTrans(adds, addAttrs, addTwins)
			if test.wantErr != err {
				t.Errorf("AddDeviceTrans case failed: wanted error %v and got error %v", test.wantErr, err)
			}
		})
	}
}

// TestDeleteDeviceTrans is function to test DeleteDeviceTrans
func TestDeleteDeviceTrans(t *testing.T) {
	ormerMock, querySeterMock := testtools.InitOrmerMock(t)

	cases := []struct {
		// name is name of the testcase
		name string
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
	dbm.DefaultOrmFunc = func() orm.Ormer {
		return ormerMock
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			// success delete
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.successDeleteReturnErr).Times(test.successDeleteTimes)
			// fail delete
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.failDeleteReturnErr).Times(test.failDeleteTimes)

			err := DeleteDeviceTrans(deletes)
			if test.wantErr != err {
				t.Errorf("DeleteDeviceTrans Case failed : wanted %v and got %v", test.wantErr, err)
			}
		})
	}
}
