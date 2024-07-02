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
)

// TestSaveDeviceTwin is function to test SaveDeviceTwin
func TestSaveDeviceTwin(t *testing.T) {
	ormerMock, cases := GetCasesSave(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.doTXReturnErr).Times(1)
			err := SaveDeviceTwin(dbm.DBAccess, &DeviceTwin{})
			if test.doTXReturnErr != err {
				t.Errorf("Save Device Twin Case failed: wanted error %v and got error %v", test.doTXReturnErr, err)
			}
		})
	}
}

// TestDeleteDeviceTwinByDeviceID is function to test DeleteDeviceTwinByDeviceID
func TestDeleteDeviceTwinByDeviceID(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesDelete(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(1)
			err := DeleteDeviceTwinByDeviceID(dbm.DBAccess, "test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceTwinByDeviceID Case failed: wanted error %v and got error %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestDeleteDeviceTwin is function to test DeleteDeviceTwin
func TestDeleteDeviceTwin(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesDelete(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(1)
			err := DeleteDeviceTwin(dbm.DBAccess, "test", "test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceTwin Case failed: wanted error %v and got error %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceTwinField is function to test UpdateDeviceTwinField
func TestUpdateDeviceTwinField(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

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
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceTwinFields(dbm.DBAccess, "test", "test", make(map[string]interface{}))
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceTwinFields Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestQueryDeviceTwin is function to test QueryDeviceTwin
func TestQueryDeviceTwin(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesQuery(t)

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
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

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
	ormerMock, querySeterMock, cases := GetCasesTrans("DeviceTwin", t)

	// adds is fake DeviceTwin used as argument
	adds := make([]DeviceTwin, 0)
	// deletes is fake DeviceDelete used as argument
	deletes := make([]DeviceDelete, 0)
	// updates is fake DeviceTwinUpdate used as argument
	updates := make([]DeviceTwinUpdate, 0)
	adds = append(adds, DeviceTwin{DeviceID: "Test"})
	deletes = append(deletes, DeviceDelete{DeviceID: "test", Name: "test"})
	updates = append(updates, DeviceTwinUpdate{DeviceID: "test", Name: "test", Cols: make(map[string]interface{})})

	dbm.DefaultOrmFunc = func() orm.Ormer {
		return ormerMock
	}

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.insertReturnErr).Times(test.insertTimes)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(test.deleteTimes)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.filterTimes)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(test.updateTimes)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.queryTableTimes)
			err := DeviceTwinTrans(adds, deletes, updates)
			if test.wantErr != err {
				t.Errorf("TestDeviceTwinTrans Case failed: wanted error %v and got error %v", test.wantErr, err)
			}
		})
	}
}
