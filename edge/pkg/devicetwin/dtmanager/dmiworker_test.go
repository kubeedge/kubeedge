/*
Copyright 2022 The KubeEdge Authors.

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

package dtmanager

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiserver"
	"github.com/kubeedge/kubeedge/pkg/util"
)

func TestDMIWorker_overrideDeviceInstanceConfig(t *testing.T) {
	tests := []struct {
		name           string
		deviceModel    *v1beta1.DeviceModel
		device         *v1beta1.Device
		expectedDevice *v1beta1.Device
		expectError    bool
	}{
		{
			name: "complete merge - model and instance both have config",
			deviceModel: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "temp-sensor-model",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceModelSpec{
					Protocol: "modbus",
					ProtocolConfigData: &v1beta1.CustomizedValue{
						Data: map[string]interface{}{
							"timeout":    5000,
							"retryTimes": 3,
							"baudRate":   9600,
						},
					},
					Properties: []v1beta1.ModelProperty{
						{
							Name:       "temperature",
							Type:       v1beta1.FLOAT,
							AccessMode: v1beta1.ReadOnly,
							Visitors: &v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
										"quantity": 1,
										"scale":    0.1,
									},
								},
							},
						},
						{
							Name:       "humidity",
							Type:       v1beta1.FLOAT,
							AccessMode: v1beta1.ReadOnly,
							Visitors: &v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40002,
										"quantity": 1,
										"scale":    0.01,
									},
								},
							},
						},
					},
				},
			},
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sensor-001",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "temp-sensor-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"slaveID": 2,
								"timeout": 3000,
							},
						},
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 30001,
										"scale":   0.2,
									},
								},
							},
						},
						{
							Name: "humidity",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 30002,
									},
								},
							},
						},
					},
				},
			},
			expectedDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sensor-001",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "temp-sensor-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"timeout":    3000, // instance overrides
								"slaveID":    2,    // instance specific
								"retryTimes": 3,    // from model
								"baudRate":   9600, // from model
							},
						},
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus", // from model
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister", // from model
										"address":  30001,             // instance overrides
										"quantity": 1,                 // from model
										"scale":    0.2,               // instance overrides
									},
								},
							},
						},
						{
							Name: "humidity",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus", // from model
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister", // from model
										"address":  30002,             // instance overrides
										"quantity": 1,                 // from model
										"scale":    0.01,              // from model
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "model has config, instance has no config",
			deviceModel: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple-model",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceModelSpec{
					Protocol: "modbus",
					ProtocolConfigData: &v1beta1.CustomizedValue{
						Data: map[string]interface{}{
							"timeout": 5000,
						},
					},
					Properties: []v1beta1.ModelProperty{
						{
							Name: "temperature",
							Visitors: &v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 40001,
									},
								},
							},
						},
					},
				},
			},
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "simple-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
							},
						},
					},
				},
			},
			expectedDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "simple-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "simple-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"timeout": 5000, // from model
							},
						},
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus", // from model
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 40001, // from model
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "model has no config, instance has config",
			deviceModel: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-model",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceModelSpec{
					Protocol: "modbus",
					Properties: []v1beta1.ModelProperty{
						{
							Name: "temperature",
							Visitors: &v1beta1.VisitorConfig{
								ProtocolName: "modbus",
							},
						},
					},
				},
			},
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "configured-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "empty-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"slaveID": 1,
							},
						},
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 30001,
									},
								},
							},
						},
					},
				},
			},
			expectedDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "configured-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "empty-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
						ConfigData: &v1beta1.CustomizedValue{
							Data: map[string]interface{}{
								"slaveID": 1, // instance config preserved
							},
						},
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus", // from model
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 30001, // instance config preserved
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "device has properties not in model",
			deviceModel: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "partial-model",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceModelSpec{
					Protocol: "modbus",
					Properties: []v1beta1.ModelProperty{
						{
							Name: "temperature",
							Visitors: &v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 40001,
										"scale":   0.1,
									},
								},
							},
						},
					},
				},
			},
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extended-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "partial-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 30001, // override model
									},
								},
							},
						},
						{
							// This property doesn't exist in model
							Name: "pressure",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 50001,
										"unit":    "Pa",
									},
								},
							},
						},
					},
				},
			},
			expectedDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "extended-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "partial-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus", // from model
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 30001, // instance overrides
										"scale":   0.1,   // from model
									},
								},
							},
						},
						{
							// This property should remain unchanged
							Name: "pressure",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus", // original instance value
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 50001, // original instance value
										"unit":    "Pa",  // original instance value
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "model has ProtocolName but no ConfigData, instance has ConfigData",
			deviceModel: &v1beta1.DeviceModel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "protocol-only-model",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceModelSpec{
					Protocol: "modbus",
					Properties: []v1beta1.ModelProperty{
						{
							Name:       "temperature",
							Type:       v1beta1.FLOAT,
							AccessMode: v1beta1.ReadOnly,
							Visitors: &v1beta1.VisitorConfig{
								ProtocolName: "modbus-tcp", // Model provides ProtocolName
								// No ConfigData in model
							},
						},
					},
				},
			},
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "instance-with-config",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "protocol-only-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								// Instance provides ConfigData but no ProtocolName
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 40001,
										"scale":   0.1,
									},
								},
							},
						},
					},
				},
			},
			expectedDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "instance-with-config",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "protocol-only-model",
					},
					Protocol: v1beta1.ProtocolConfig{
						ProtocolName: "modbus",
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus-tcp", // Should preserve model's ProtocolName
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"address": 40001, // Should use instance's ConfigData
										"scale":   0.1,
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup DMIWorker with cache
			dw := &DMIWorker{
				dmiCache: &dmiserver.DMICache{
					MapperMu:        &sync.Mutex{},
					DeviceMu:        &sync.Mutex{},
					DeviceModelMu:   &sync.Mutex{},
					MapperList:      make(map[string]*pb.MapperInfo),
					DeviceList:      make(map[string]*v1beta1.Device),
					DeviceModelList: make(map[string]*v1beta1.DeviceModel),
				},
			}

			// Add device model to cache if provided
			if tt.deviceModel != nil {
				deviceModelID := util.GetResourceID(tt.deviceModel.Namespace, tt.deviceModel.Name)
				dw.dmiCache.DeviceModelList[deviceModelID] = tt.deviceModel
			}

			// Execute the function
			err := dw.overrideDeviceInstanceConfig(tt.device)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Check the result
			if tt.expectedDevice != nil {
				// Check protocol config
				if tt.expectedDevice.Spec.Protocol.ConfigData != nil {
					assert.NotNil(t, tt.device.Spec.Protocol.ConfigData)
					assert.Equal(t, tt.expectedDevice.Spec.Protocol.ConfigData.Data, tt.device.Spec.Protocol.ConfigData.Data)
				}

				// Check properties
				assert.Equal(t, len(tt.expectedDevice.Spec.Properties), len(tt.device.Spec.Properties))
				for i, expectedProp := range tt.expectedDevice.Spec.Properties {
					actualProp := tt.device.Spec.Properties[i]
					assert.Equal(t, expectedProp.Name, actualProp.Name)
					assert.Equal(t, expectedProp.Visitors.ProtocolName, actualProp.Visitors.ProtocolName)

					if expectedProp.Visitors.ConfigData != nil {
						assert.NotNil(t, actualProp.Visitors.ConfigData)
						assert.Equal(t, expectedProp.Visitors.ConfigData.Data, actualProp.Visitors.ConfigData.Data)
					}
				}
			}
		})
	}
}

func TestFindModelProperty(t *testing.T) {
	properties := []v1beta1.ModelProperty{
		{
			Name: "temperature",
			Type: v1beta1.FLOAT,
		},
		{
			Name: "humidity",
			Type: v1beta1.FLOAT,
		},
	}

	tests := []struct {
		name         string
		propertyName string
		expected     *v1beta1.ModelProperty
	}{
		{
			name:         "find existing property",
			propertyName: "temperature",
			expected:     &properties[0],
		},
		{
			name:         "find another existing property",
			propertyName: "humidity",
			expected:     &properties[1],
		},
		{
			name:         "property not found",
			propertyName: "pressure",
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findModelProperty(properties, tt.propertyName)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.Type, result.Type)
			}
		})
	}
}
