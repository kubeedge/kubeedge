/*
Copyright 2019 The KubeEdge Authors.

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

package devicetwin

import (
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtmodule"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/testutil"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/mocks"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

type CasesDevice []struct {
	name     string
	context  *dtcontext.DTContext
	deviceID string
	wantErr  error
}

// createFakeDevice() is function to create fake device.
func createFakeDevice() *[]models.Device {
	fakeDevice := new([]models.Device)
	fakeDeviceArray := make([]models.Device, 1)
	fakeDeviceArray[0] = models.Device{ID: "Test"}
	fakeDevice = &fakeDeviceArray
	return fakeDevice
}

// createFakeAttribute() is function to create fake device attribute.
func createFakeDeviceAttribute() *[]models.DeviceAttr {
	fakeDeviceAttr := new([]models.DeviceAttr)
	fakeDeviceAttrArray := make([]models.DeviceAttr, 1)
	fakeDeviceAttrArray[0] = models.DeviceAttr{DeviceID: "Test"}
	fakeDeviceAttr = &fakeDeviceAttrArray
	return fakeDeviceAttr
}

// createFakeDeviceTwin() is function to create fake devicetwin.
func createFakeDeviceTwin() *[]models.DeviceTwin {
	fakeDeviceTwin := new([]models.DeviceTwin)
	fakeDeviceTwinArray := make([]models.DeviceTwin, 1)
	fakeDeviceTwinArray[0] = models.DeviceTwin{DeviceID: "Test"}
	fakeDeviceTwin = &fakeDeviceTwinArray
	return fakeDeviceTwin
}

var (
	originalDeviceServiceFactory = DeviceServiceFactory
)

// TestRegisterDTModule is function to test RegisterDTmodule().
func TestRegisterDTModule(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	dtContexts, _ := dtcontext.InitDTContext()
	var moduleRegistered bool
	dtc := &DeviceTwin{
		HeartBeatToModule: make(map[string]chan interface{}),
		DTContexts:        dtContexts,
		DTModules:         make(map[string]dtmodule.DTModule),
	}
	tests := []struct {
		name       string
		moduleName string
	}{
		{
			name:       "MemModule",
			moduleName: "MemModule",
		},
		{
			name:       "TwinModule",
			moduleName: "TwinModule",
		},
		{
			name:       "CommModule",
			moduleName: "CommModule",
		},
		{
			name:       "DeviceModule",
			moduleName: "DeviceModule",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtc.RegisterDTModule(tt.moduleName)
			moduleRegistered = false
			for _, name := range dtc.DTModules {
				if name.Name == tt.moduleName {
					moduleRegistered = true
					break
				}
			}
			if !moduleRegistered {
				t.Errorf("RegisterDTModule failed to register the module %v", tt.moduleName)
			}
		})
	}
}

// TestDTController_distributeMsg is function to test distributeMsg().
func TestDTController_distributeMsg(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	dtContexts, _ := dtcontext.InitDTContext()
	dtc := &DeviceTwin{
		HeartBeatToModule: make(map[string]chan interface{}),
		DTModules:         make(map[string]dtmodule.DTModule),
		DTContexts:        dtContexts,
	}

	content := testutil.GenerateAddDevicePalyloadMsg(t)

	var msg = &model.Message{
		Header: model.MessageHeader{
			ParentID: DeviceTwinModuleName,
		},
		Content: string(content),
		Router: model.MessageRoute{
			Source:   "edgemgr",
			Resource: "membership/detail",
		},
	}
	tests := []struct {
		name    string
		message interface{}
		wantErr error
	}{
		{
			//Failure Case
			name:    "distributeMsgTest-NilMessage",
			message: "",
			wantErr: errors.New("distribute message, msg is nil"),
		},
		{
			//Failure Case
			name: "distributeMsgTest-ClassifyMsgFail",
			message: model.Message{
				Router: model.MessageRoute{
					Source:   "bus",
					Resource: "membership/detail",
				},
			},
			wantErr: errors.New("not found action"),
		},
		{
			//Failure Case
			name:    "distributeMsgTest-ActualMessage-NoChanel",
			message: *msg,
			wantErr: errors.New("Not found chan to communicate"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := dtc.distributeMsg(tt.message); !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("DTController.distributeMsg() error = %v, wantError %v", err, tt.wantErr)
			}
		})
	}

	//Successful Case
	dh := make(chan interface{}, 1)
	ch := make(chan interface{}, 1)
	mh := make(chan interface{}, 1)
	deh := make(chan interface{}, 1)
	th := make(chan interface{}, 1)
	dtc.DTContexts.CommChan["DeviceStateUpdate"] = dh
	dtc.DTContexts.CommChan["CommModule"] = ch
	dtc.DTContexts.CommChan["MemModule"] = mh
	dtc.DTContexts.CommChan["DeviceModule"] = deh
	dtc.DTContexts.CommChan["TwinModule"] = th
	name := "distributeMsgTest-ActualMessage-Success"
	t.Run(name, func(t *testing.T) {
		if err := dtc.distributeMsg(*msg); !reflect.DeepEqual(err, nil) {
			t.Errorf("DTController.distributeMsg() error = %v, wantError %v", err, nil)
		}
	})
}

// TestSyncSqlite is function to test SyncSqlite().
func TestSyncSqlite(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	dtContexts, _ := dtcontext.InitDTContext()

	tests := []struct {
		name            string
		context         *dtcontext.DTContext
		setupMock       func(*mocks.MockDeviceService)
		wantErr         bool
		wantErrContains string
	}{
		{
			// Failure Case: Query failed
			name:    "SyncSqliteTest-QuerySqliteFailed",
			context: dtContexts,
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceAllFunc = func() ([]models.Device, error) {
					return nil, errors.New("Query sqlite failed while syncing sqlite")
				}
			},
			wantErr:         true,
			wantErrContains: "Query sqlite failed while syncing sqlite",
		},
		{
			// Success Case: Query returns nil
			name:    "SyncSqliteTest-QuerySqliteNil",
			context: dtContexts,
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceAllFunc = func() ([]models.Device, error) {
					return nil, nil
				}
			},
			wantErr: false,
		},
		{
			// Success Case: Query returns empty list
			name:    "SyncSqliteTest-QuerySqliteEmpty",
			context: dtContexts,
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceAllFunc = func() ([]models.Device, error) {
					return []models.Device{}, nil
				}
			},
			wantErr: false,
		},
		{
			// Success Case: Query returns devices
			name:    "SyncSqliteTest-QuerySqliteSuccess",
			context: dtContexts,
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceAllFunc = func() ([]models.Device, error) {
					return []models.Device{
						{ID: "device1", Name: "Device1"},
						{ID: "device2", Name: "Device2"},
					}, nil
				}
				m.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
					return []models.Device{}, nil
				}
				m.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
					return &[]models.DeviceAttr{}, nil
				}
				m.QueryDeviceTwinFunc = func(key, condition string) (*[]models.DeviceTwin, error) {
					return &[]models.DeviceTwin{}, nil
				}
			},
			wantErr: false,
		},
	}

	// Save original and defer restore
	defer func() {
		DeviceServiceFactory = originalDeviceServiceFactory
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh mock for each test case
			mockService := mocks.NewMockDeviceService()
			tt.setupMock(mockService)

			// Replace factory for this test
			DeviceServiceFactory = func() interface {
				QueryDeviceAll() ([]models.Device, error)
				QueryDevice(key string, condition string) ([]models.Device, error)
				QueryDeviceAttr(key, condition string) (*[]models.DeviceAttr, error)
				QueryDeviceTwin(key, condition string) (*[]models.DeviceTwin, error)
			} {
				return mockService
			}

			err := SyncSqlite(tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("SyncSqlite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.wantErrContains != "" {
				if !reflect.DeepEqual(err.Error(), tt.wantErrContains) {
					t.Errorf("SyncSqlite() error = %v, wantErrContains %v", err.Error(), tt.wantErrContains)
				}
			}
		})
	}
}

// TestSyncDeviceFromSqlite is function to test SyncDeviceFromSqlite().
func TestSyncDeviceFromSqlite(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	dtContext, _ := dtcontext.InitDTContext()

	tests := []struct {
		name      string
		context   *dtcontext.DTContext
		deviceID  string
		setupMock func(*mocks.MockDeviceService)
		wantErr   bool
	}{
		{
			// Failure Case: Query device failed
			name:     "TestSyncDeviceFromSqlite-QueryDeviceFailure",
			context:  dtContext,
			deviceID: "DeviceA",
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
					return nil, errors.New("Query Device Failed")
				}
			},
			wantErr: true,
		},
		{
			// Failure Case: Query device attribute failed
			name:     "TestSyncDeviceFromSqlite-QueryDeviceAttributeFailed",
			context:  dtContext,
			deviceID: "DeviceB",
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
					return []models.Device{{ID: "DeviceB"}}, nil
				}
				m.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
					return nil, errors.New("query device attr failed")
				}
			},
			wantErr: true,
		},
		{
			// Failure Case: Query device twin failed
			name:     "TestSyncDeviceFromSqlite-QueryDeviceTwinFailed",
			context:  dtContext,
			deviceID: "DeviceC",
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
					return []models.Device{{ID: "DeviceC"}}, nil
				}
				m.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
					return &[]models.DeviceAttr{}, nil
				}
				m.QueryDeviceTwinFunc = func(key, condition string) (*[]models.DeviceTwin, error) {
					return nil, errors.New("query device twin failed")
				}
			},
			wantErr: true,
		},
		{
			// Success Case
			name:     "TestSyncDeviceFromSqlite-SuccessCase",
			context:  dtContext,
			deviceID: "DeviceD",
			setupMock: func(m *mocks.MockDeviceService) {
				m.QueryDeviceFunc = func(key, condition string) ([]models.Device, error) {
					return []models.Device{{
						ID:    "DeviceD",
						Name:  "Device D",
						State: "online",
					}}, nil
				}
				m.QueryDeviceAttrFunc = func(key, condition string) (*[]models.DeviceAttr, error) {
					return &[]models.DeviceAttr{}, nil
				}
				m.QueryDeviceTwinFunc = func(key, condition string) (*[]models.DeviceTwin, error) {
					return &[]models.DeviceTwin{}, nil
				}
			},
			wantErr: false,
		},
	}

	// Save original and defer restore
	defer func() {
		DeviceServiceFactory = originalDeviceServiceFactory
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create fresh mock for each test case
			mockService := mocks.NewMockDeviceService()
			test.setupMock(mockService)

			// Replace factory for this test
			DeviceServiceFactory = func() interface {
				QueryDeviceAll() ([]models.Device, error)
				QueryDevice(key string, condition string) ([]models.Device, error)
				QueryDeviceAttr(key, condition string) (*[]models.DeviceAttr, error)
				QueryDeviceTwin(key, condition string) (*[]models.DeviceTwin, error)
			} {
				return mockService
			}

			err := SyncDeviceFromSqlite(test.context, test.deviceID)
			if (err != nil) != test.wantErr {
				t.Errorf("SyncDeviceFromSqlite() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

// Test_classifyMsg is function to test classifyMsg().
func Test_classifyMsg(t *testing.T) {
	//Encoded resource with LifeCycleConnectETPrefix
	connectTopic := dtcommon.LifeCycleConnectETPrefix + "testtopic"
	encodedConnectTopicResource := base64.URLEncoding.EncodeToString([]byte(connectTopic))
	//Encoded resource with LifeCycleDisconnectETPrefix
	disconnectTopic := dtcommon.LifeCycleDisconnectETPrefix + "testtopic"
	encodedDisconnectResource := base64.URLEncoding.EncodeToString([]byte(disconnectTopic))
	//Encoded resource with other Prefix
	otherTopic := "/membership/detail/result"
	otherEncodedTopic := base64.URLEncoding.EncodeToString([]byte(otherTopic))
	//Encoded eventbus resource
	eventbusTopic := "$hw/events/device/+/+/state/update"
	eventbusResource := base64.URLEncoding.EncodeToString([]byte(eventbusTopic))

	content := testutil.GenerateAddDevicePalyloadMsg(t)
	tests := []struct {
		name     string
		message  *dttype.DTMessage
		wantBool bool
	}{
		{
			//Failure Case
			name: "classifyMsgTest-UnencodedMessageResource",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "bus",
						Resource: "membership/detail",
					},
				},
			},
			wantBool: false,
		},
		{
			//Success Case
			name: "classifyMsgTest-Source:bus-Prefix:LifeCycleConnectETPrefix",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "bus",
						Resource: encodedConnectTopicResource,
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Success Case
			name: "classifyMsgTest-Source:bus-Prefix:LifeCycleDisconnectETPrefix",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "bus",
						Resource: encodedDisconnectResource,
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Failure Case
			name: "classifyMessageTest-Source:bus-Prefix:OtherPrefix",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "bus",
						Resource: otherEncodedTopic,
					},
					Content: string(content),
				},
			},
			wantBool: false,
		},
		{
			//Success Case
			name: "classifyMessageTest-Source:bus-Resource:eventbus",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "bus",
						Resource: eventbusResource,
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Success Case
			name: "classifyMessageTest-Source:edgemgr-Resource:membership/detail",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "edgemgr",
						Resource: "membership/detail",
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Success Case
			name: "classifyMessageTest-Source:edgemgr-Resource:membership",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "edgemgr",
						Resource: "membership",
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Success Case
			name: "classifyMessageTest-Source:edgemgr-Resourcetwin:cloud_updated",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "edgemgr",
						Resource: "twin/cloud_updated",
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Success Case
			name: "classifyMessageTest-Source:edgemgr-Resource:device/updated-Operation:updated",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:    "edgemgr",
						Resource:  "device/updated",
						Operation: "updated",
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Failure Case
			name: "calssifyMessageTest-Source:edgemgr-no resource and operation",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source: "edgemgr",
					},
					Content: string(content),
				},
			},
			wantBool: false,
		},
		{
			//Success Case
			name: "classifyMessageTest-Source:edgehub-Resource:node/connection",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "edgehub",
						Resource: "node/connection",
					},
					Content: string(content),
				},
			},
			wantBool: true,
		},
		{
			//Failure Case
			name: "classifyMessageTest-Source:edgehub-Resource:node",
			message: &dttype.DTMessage{
				Msg: &model.Message{
					Router: model.MessageRoute{
						Source:   "edgehub",
						Resource: "node",
					},
					Content: string(content),
				},
			},
			wantBool: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyMsg(tt.message); got != tt.wantBool {
				t.Errorf("classifyMsg() = %v, wantError %v", got, tt.wantBool)
			}
		})
	}
}
