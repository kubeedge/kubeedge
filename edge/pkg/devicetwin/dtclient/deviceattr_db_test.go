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

// TestSaveDeviceAttr is function to test SaveDeviceAttr
func TestSaveDeviceAttr(t *testing.T) {
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

// TestDeleteDeviceAttrByDeviceID is function to test DeleteDeviceAttrByDeviceID
func TestDeleteDeviceAttrByDeviceID(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesDelete(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(1)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(1)
			err := DeleteDeviceAttrByDeviceID(dbm.DBAccess, "test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceByID case failed: wanted %v and got %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestDeleteDeviceAttr is function to test DeleteDeviceAttr
func TestDeleteDeviceAttr(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesDelete(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			querySeterMock.EXPECT().Delete().Return(test.deleteReturnInt, test.deleteReturnErr).Times(1)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(1)
			err := DeleteDeviceAttr(dbm.DBAccess, "test", "test")
			if test.deleteReturnErr != err {
				t.Errorf("DeleteDeviceAttr Case failed: wanted error %v and got error %v", test.deleteReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceAttrField is function to test UpdateDeviceAttrField
func TestUpdateDeviceAttrField(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceAttrField("test", "test", "test", "test")
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceAttrField Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestUpdateDeviceAttrFields is function to test UpdateDeviceAttrFields
func TestUpdateDeviceAttrFields(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceAttrFields(dbm.DBAccess, "test", "test", make(map[string]interface{}))
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceAttrFields Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestQueryDeviceAttr is function to test QueryDeviceAttr
func TestQueryDeviceAttr(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesQuery(t)

	// fakeDeviceAttr is used to set the argument of All function
	fakeDeviceAttr := new([]DeviceAttr)
	fakeDeviceAttrArray := make([]DeviceAttr, 1)
	fakeDeviceAttrArray[0] = DeviceAttr{DeviceID: "Test"}
	fakeDeviceAttr = &fakeDeviceAttrArray

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().All(gomock.Any()).SetArg(0, *fakeDeviceAttr).Return(test.allReturnInt, test.allReturnErr).Times(1)
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			device, err := QueryDeviceAttr("test", "test")
			if test.allReturnErr != err {
				t.Errorf("QueryDeviceAttr Case failed: wanted error %v and got error %v", test.allReturnErr, err)
			}

			if err == nil {
				if len(*device) != 1 {
					t.Errorf("QueryDeviceAttr Case failed: wanted length 1 and got length %v", len(*device))
				}
			}
		})
	}
}

// TestUpdateDeviceAttrMulti is function to test UpdateDeviceAttrMulti
func TestUpdateDeviceAttrMulti(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesUpdate(t)

	// updateDevice is argument to UpdateDeviceAttrMulti function
	updateDevice := make([]DeviceAttrUpdate, 0)
	updateDevice = append(updateDevice, DeviceAttrUpdate{DeviceID: "test"})

	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(2)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(1)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(1)
			err := UpdateDeviceAttrMulti(updateDevice)
			if test.updateReturnErr != err {
				t.Errorf("UpdateDeviceAttrMulti Case failed: wanted error %v and got error %v", test.updateReturnErr, err)
			}
		})
	}
}

// TestDeviceAttrTrans is function to test DeviceAttrTrans
func TestDeviceAttrTrans(t *testing.T) {
	ormerMock, querySeterMock, cases := GetCasesTrans("DeviceAttr", t)

	// adds is fake DeviceAttr used as argument
	adds := make([]DeviceAttr, 0)
	// deletes is fake DeviceDelete used as argument
	deletes := make([]DeviceDelete, 0)
	// updates is fake DeviceAttrUpdate used as argument
	updates := make([]DeviceAttrUpdate, 0)
	adds = append(adds, DeviceAttr{DeviceID: "Test"})
	deletes = append(deletes, DeviceDelete{DeviceID: "test", Name: "test"})
	updates = append(updates, DeviceAttrUpdate{DeviceID: "test", Name: "test", Cols: make(map[string]interface{})})

	dbm.DefaultOrmFunc = func() orm.Ormer {
		return ormerMock
	}
	// run the test cases
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.insertReturnErr).Times(test.insertTimes)
			ormerMock.EXPECT().DoTx(gomock.Any()).Return(test.deleteReturnErr).Times(test.deleteTimes)
			ormerMock.EXPECT().QueryTable(gomock.Any()).Return(test.queryTableReturn).Times(test.queryTableTimes)

			querySeterMock.EXPECT().Filter(gomock.Any(), gomock.Any()).Return(test.filterReturn).Times(test.filterTimes)
			querySeterMock.EXPECT().Update(gomock.Any()).Return(test.updateReturnInt, test.updateReturnErr).Times(test.updateTimes)
			err := DeviceAttrTrans(adds, deletes, updates)
			if test.wantErr != err {
				t.Errorf("TestDeviceAttrTrans Case failed: wanted error %v and got error %v", test.wantErr, err)
			}
		})
	}
}
