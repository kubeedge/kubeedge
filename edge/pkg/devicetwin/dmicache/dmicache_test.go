/*
Copyright 2025 The KubeEdge Authors.

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

package dmicache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
)

func TestNewDMICache(t *testing.T) {
	cache := NewDMICache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.mapperMu)
	assert.NotNil(t, cache.deviceMu)
	assert.NotNil(t, cache.deviceModelMu)
	assert.NotNil(t, cache.mapperList)
	assert.NotNil(t, cache.deviceModelList)
	assert.NotNil(t, cache.deviceList)
	assert.Equal(t, 0, len(cache.mapperList))
	assert.Equal(t, 0, len(cache.deviceModelList))
	assert.Equal(t, 0, len(cache.deviceList))
}

func TestDMICache_Mapper_Operations(t *testing.T) {
	t.Run("PutMapper and GetMapper", func(t *testing.T) {
		cache := NewDMICache()
		mapper1 := &pb.MapperInfo{
			Name:       "test-mapper-1",
			Version:    "v1.0.0",
			ApiVersion: "v1beta1",
			Protocol:   "modbus",
			Address:    []byte("tcp://localhost:1502"),
			State:      "MAPPER_CONNECTED",
		}

		mapper2 := &pb.MapperInfo{
			Name:       "test-mapper-2",
			Version:    "v1.1.0",
			ApiVersion: "v1beta1",
			Protocol:   "opcua",
			Address:    []byte("opc.tcp://localhost:4840"),
			State:      "MAPPER_DISCONNECTED",
		}

		// Put mapper1
		cache.PutMapper(mapper1)

		// Get mapper1
		retrieved, exists := cache.GetMapper("test-mapper-1")
		assert.True(t, exists)
		assert.Equal(t, mapper1, retrieved)

		// Get non-existent mapper
		retrieved, exists = cache.GetMapper("non-existent")
		assert.False(t, exists)
		assert.Nil(t, retrieved)

		// Put mapper2
		cache.PutMapper(mapper2)

		// Get mapper2
		retrieved, exists = cache.GetMapper("test-mapper-2")
		assert.True(t, exists)
		assert.Equal(t, mapper2, retrieved)

		// mapper1 should still exist
		retrieved, exists = cache.GetMapper("test-mapper-1")
		assert.True(t, exists)
		assert.Equal(t, mapper1, retrieved)
	})

	t.Run("RemoveMapper", func(t *testing.T) {
		cache := NewDMICache()
		mapper1 := &pb.MapperInfo{
			Name:       "test-mapper-1",
			Version:    "v1.0.0",
			ApiVersion: "v1beta1",
			Protocol:   "modbus",
			Address:    []byte("tcp://localhost:1502"),
			State:      "MAPPER_CONNECTED",
		}

		mapper2 := &pb.MapperInfo{
			Name:       "test-mapper-2",
			Version:    "v1.1.0",
			ApiVersion: "v1beta1",
			Protocol:   "opcua",
			Address:    []byte("opc.tcp://localhost:4840"),
			State:      "MAPPER_DISCONNECTED",
		}

		// Setup: add both mappers
		cache.PutMapper(mapper1)
		cache.PutMapper(mapper2)

		// Remove mapper1
		cache.RemoveMapper("test-mapper-1")

		// mapper1 should not exist
		retrieved, exists := cache.GetMapper("test-mapper-1")
		assert.False(t, exists)
		assert.Nil(t, retrieved)

		// mapper2 should still exist
		retrieved, exists = cache.GetMapper("test-mapper-2")
		assert.True(t, exists)
		assert.Equal(t, mapper2, retrieved)

		// Remove mapper2
		cache.RemoveMapper("test-mapper-2")
		retrieved, exists = cache.GetMapper("test-mapper-2")
		assert.False(t, exists)
		assert.Nil(t, retrieved)

		// Remove non-existent mapper (should not panic)
		cache.RemoveMapper("non-existent")
	})

	t.Run("PutMapper overwrites existing", func(t *testing.T) {
		cache := NewDMICache()
		mapper1 := &pb.MapperInfo{
			Name:       "test-mapper-1",
			Version:    "v1.0.0",
			ApiVersion: "v1beta1",
			Protocol:   "modbus",
			Address:    []byte("tcp://localhost:1502"),
			State:      "MAPPER_CONNECTED",
		}

		// Put original mapper
		cache.PutMapper(mapper1)

		// Create updated mapper with same name
		updatedMapper := &pb.MapperInfo{
			Name:       "test-mapper-1",
			Version:    "v2.0.0", // Different version
			ApiVersion: "v1beta1",
			Protocol:   "modbus",
			Address:    []byte("tcp://localhost:1503"), // Different address
			State:      "MAPPER_CONNECTED",
		}

		// Put updated mapper
		cache.PutMapper(updatedMapper)

		// Should get updated mapper
		retrieved, exists := cache.GetMapper("test-mapper-1")
		assert.True(t, exists)
		assert.Equal(t, updatedMapper, retrieved)
		assert.Equal(t, "v2.0.0", retrieved.Version)
		assert.Equal(t, []byte("tcp://localhost:1503"), retrieved.Address)
	})
}

func TestDMICache_DeviceModel_Operations(t *testing.T) {
	t.Run("PutDeviceModel and GetDeviceModel", func(t *testing.T) {
		cache := NewDMICache()
		deviceModel1 := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "temperature-sensor",
				Namespace: "default",
			},
			Spec: v1beta1.DeviceModelSpec{
				Protocol: "modbus",
				Properties: []v1beta1.ModelProperty{
					{
						Name:       "temperature",
						Type:       v1beta1.FLOAT,
						AccessMode: v1beta1.ReadOnly,
					},
				},
			},
		}

		deviceModel2 := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "humidity-sensor",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceModelSpec{
				Protocol: "opcua",
				Properties: []v1beta1.ModelProperty{
					{
						Name:       "humidity",
						Type:       v1beta1.FLOAT,
						AccessMode: v1beta1.ReadOnly,
					},
				},
			},
		}

		// Put deviceModel1
		cache.PutDeviceModel(deviceModel1)

		// Get deviceModel1
		retrieved, exists := cache.GetDeviceModel("default", "temperature-sensor")
		assert.True(t, exists)
		assert.Equal(t, deviceModel1, retrieved)

		// Get non-existent device model
		retrieved, exists = cache.GetDeviceModel("default", "non-existent")
		assert.False(t, exists)
		assert.Nil(t, retrieved)

		// Put deviceModel2
		cache.PutDeviceModel(deviceModel2)

		// Get deviceModel2
		retrieved, exists = cache.GetDeviceModel("test-namespace", "humidity-sensor")
		assert.True(t, exists)
		assert.Equal(t, deviceModel2, retrieved)

		// deviceModel1 should still exist
		retrieved, exists = cache.GetDeviceModel("default", "temperature-sensor")
		assert.True(t, exists)
		assert.Equal(t, deviceModel1, retrieved)
	})

	t.Run("RemoveDeviceModel", func(t *testing.T) {
		cache := NewDMICache()
		deviceModel1 := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "temperature-sensor",
				Namespace: "default",
			},
			Spec: v1beta1.DeviceModelSpec{
				Protocol: "modbus",
				Properties: []v1beta1.ModelProperty{
					{
						Name:       "temperature",
						Type:       v1beta1.FLOAT,
						AccessMode: v1beta1.ReadOnly,
					},
				},
			},
		}

		deviceModel2 := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "humidity-sensor",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceModelSpec{
				Protocol: "opcua",
				Properties: []v1beta1.ModelProperty{
					{
						Name:       "humidity",
						Type:       v1beta1.FLOAT,
						AccessMode: v1beta1.ReadOnly,
					},
				},
			},
		}

		// Setup: add both device models
		cache.PutDeviceModel(deviceModel1)
		cache.PutDeviceModel(deviceModel2)

		// Remove deviceModel1
		cache.RemoveDeviceModel("default", "temperature-sensor")

		// deviceModel1 should not exist
		retrieved, exists := cache.GetDeviceModel("default", "temperature-sensor")
		assert.False(t, exists)
		assert.Nil(t, retrieved)

		// deviceModel2 should still exist
		retrieved, exists = cache.GetDeviceModel("test-namespace", "humidity-sensor")
		assert.True(t, exists)
		assert.Equal(t, deviceModel2, retrieved)

		// Remove deviceModel2
		cache.RemoveDeviceModel("test-namespace", "humidity-sensor")
		retrieved, exists = cache.GetDeviceModel("test-namespace", "humidity-sensor")
		assert.False(t, exists)
		assert.Nil(t, retrieved)

		// Remove non-existent device model (should not panic)
		cache.RemoveDeviceModel("default", "non-existent")
	})
}

func TestDMICache_Device_Operations(t *testing.T) {
	t.Run("PutDevice and GetOverriddenDevice", func(t *testing.T) {
		cache := NewDMICache()
		deviceModel := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "temp-sensor-model",
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
							ProtocolName: "modbus",
							ConfigData: &v1beta1.CustomizedValue{
								Data: map[string]interface{}{
									"register": "HoldingRegister",
									"address":  40001,
								},
							},
						},
					},
				},
				ProtocolConfigData: &v1beta1.CustomizedValue{
					Data: map[string]interface{}{
						"slaveID": 1,
						"timeout": 5000,
					},
				},
			},
		}

		device1 := &v1beta1.Device{
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
				},
				Properties: []v1beta1.DeviceProperty{
					{
						Name: "temperature",
						Visitors: v1beta1.VisitorConfig{
							ProtocolName: "modbus",
							ConfigData: &v1beta1.CustomizedValue{
								Data: map[string]interface{}{
									"register": "HoldingRegister",
									"address":  40002,
								},
							},
						},
					},
				},
			},
		}

		overridenDevice1 := &v1beta1.Device{
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
							"slaveID": 1,
							"timeout": 5000,
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
									"register": "HoldingRegister",
									"address":  40002,
								},
							},
						},
					},
				},
			},
		}

		// Device model for device2 (same content, different namespace)
		deviceModel2 := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "temp-sensor-model",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceModelSpec{
				Protocol: "modbus",
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
								},
							},
						},
					},
				},
				ProtocolConfigData: &v1beta1.CustomizedValue{
					Data: map[string]interface{}{
						"slaveID": 1,
						"timeout": 5000,
					},
				},
			},
		}

		device2 := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-002",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceSpec{
				DeviceModelRef: &v1.LocalObjectReference{
					Name: "temp-sensor-model",
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
		}

		overridenDevice2 := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-002",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceSpec{
				DeviceModelRef: &v1.LocalObjectReference{
					Name: "temp-sensor-model",
				},
				Protocol: v1beta1.ProtocolConfig{
					ProtocolName: "modbus",
					ConfigData: &v1beta1.CustomizedValue{
						Data: map[string]interface{}{
							"slaveID": 1,
							"timeout": 5000,
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
									"register": "HoldingRegister",
									"address":  40001,
								},
							},
						},
					},
				},
			},
		}

		// Put device model first
		cache.PutDeviceModel(deviceModel)

		// Put device1
		cache.PutDevice(device1)

		// Get overridden device1
		retrieved, model, err := cache.GetOverriddenDevice("default", "sensor-001")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, overridenDevice1, retrieved)
		assert.NotNil(t, model)
		assert.Equal(t, deviceModel, model)

		// Get non-existent device
		retrieved, model, err = cache.GetOverriddenDevice("default", "non-existent")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Nil(t, model)
		assert.Contains(t, err.Error(), "not found in cache")

		// Put device model for device2
		cache.PutDeviceModel(deviceModel2)

		// Put device2
		cache.PutDevice(device2)

		// Get overridden device2
		retrieved, model, err = cache.GetOverriddenDevice("test-namespace", "sensor-002")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, overridenDevice2, retrieved)
		assert.NotNil(t, model)
		assert.Equal(t, deviceModel2, model)
	})

	t.Run("GetOverriddenDevice without DeviceModel", func(t *testing.T) {
		cache := NewDMICache()
		// Create device without corresponding device model in cache
		deviceWithoutModel := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "orphan-device",
				Namespace: "default",
			},
			Spec: v1beta1.DeviceSpec{
				DeviceModelRef: &v1.LocalObjectReference{
					Name: "non-existent-model",
				},
			},
		}

		cache.PutDevice(deviceWithoutModel)

		// Should get error when device model not found
		retrieved, model, err := cache.GetOverriddenDevice("default", "orphan-device")
		assert.Nil(t, model)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "not found in cache")
	})

	t.Run("GetOverriddenDevice without DeviceModelRef", func(t *testing.T) {
		cache := NewDMICache()
		// Create device without DeviceModelRef
		deviceWithoutRef := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-ref-device",
				Namespace: "default",
			},
			Spec: v1beta1.DeviceSpec{
				// No DeviceModelRef
			},
		}

		cache.PutDevice(deviceWithoutRef)

		// Should get error when DeviceModelRef is nil
		retrieved, model, err := cache.GetOverriddenDevice("default", "no-ref-device")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Nil(t, model)
		assert.Contains(t, err.Error(), "has no device model reference")
	})

	t.Run("DeviceIds", func(t *testing.T) {
		cache := NewDMICache()

		device1 := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-001",
				Namespace: "default",
			},
		}

		device2 := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-002",
				Namespace: "test-namespace",
			},
		}

		// Initially should be empty
		ids := cache.DeviceIds()
		assert.Equal(t, 0, len(ids))

		// Add devices
		cache.PutDevice(device1)
		cache.PutDevice(device2)

		// Should have 2 device IDs
		ids = cache.DeviceIds()
		assert.Equal(t, 2, len(ids))

		// Check that IDs contain expected values
		expectedIds := []string{
			"default/sensor-001",
			"test-namespace/sensor-002",
		}
		for _, expectedId := range expectedIds {
			assert.Contains(t, ids, expectedId)
		}
	})

	t.Run("RemoveDevice", func(t *testing.T) {
		cache := NewDMICache()

		deviceModel := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "temp-sensor-model",
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
							ProtocolName: "modbus",
							ConfigData: &v1beta1.CustomizedValue{
								Data: map[string]interface{}{
									"register": "HoldingRegister",
									"address":  40001,
								},
							},
						},
					},
				},
			},
		}

		deviceModel2 := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "temp-sensor-model",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceModelSpec{
				Protocol: "modbus",
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
								},
							},
						},
					},
				},
			},
		}

		device1 := &v1beta1.Device{
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
		}

		device2 := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-002",
				Namespace: "test-namespace",
			},
			Spec: v1beta1.DeviceSpec{
				DeviceModelRef: &v1.LocalObjectReference{
					Name: "temp-sensor-model",
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
		}

		// Setup: add device models and devices
		cache.PutDeviceModel(deviceModel)
		cache.PutDeviceModel(deviceModel2)
		cache.PutDevice(device1)
		cache.PutDevice(device2)

		// Remove device1
		cache.RemoveDevice("default", "sensor-001")

		// device1 should not exist
		retrieved, model, err := cache.GetOverriddenDevice("default", "sensor-001")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Nil(t, model)

		// device2 should still exist
		retrieved, model, err = cache.GetOverriddenDevice("test-namespace", "sensor-002")
		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.NotNil(t, model)

		// DeviceIds should only contain device2
		ids := cache.DeviceIds()
		assert.Equal(t, 1, len(ids))
		assert.Contains(t, ids, "test-namespace/sensor-002")

		// Remove device2
		cache.RemoveDevice("test-namespace", "sensor-002")
		retrieved, _, err = cache.GetOverriddenDevice("test-namespace", "sensor-002")
		assert.Error(t, err)
		assert.Nil(t, retrieved)

		// DeviceIds should be empty
		ids = cache.DeviceIds()
		assert.Equal(t, 0, len(ids))

		// Remove non-existent device (should not panic)
		cache.RemoveDevice("default", "non-existent")
	})
}

func TestDMICache_DeepCopyCustomizedValue(t *testing.T) {
	tests := []struct {
		name     string
		src      *v1beta1.CustomizedValue
		expected *v1beta1.CustomizedValue
	}{
		{
			name:     "nil input",
			src:      nil,
			expected: nil,
		},
		{
			name:     "empty CustomizedValue",
			src:      &v1beta1.CustomizedValue{},
			expected: &v1beta1.CustomizedValue{},
		},
		{
			name: "nil Data",
			src: &v1beta1.CustomizedValue{
				Data: nil,
			},
			expected: &v1beta1.CustomizedValue{},
		},
		{
			name: "simple values",
			src: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"string": "test",
					"int":    42,
					"float":  3.14,
					"bool":   true,
				},
			},
			expected: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"string": "test",
					"int":    42,
					"float":  3.14,
					"bool":   true,
				},
			},
		},
		{
			name: "nested map",
			src: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"nested": map[string]interface{}{
						"key1": "value1",
						"key2": 123,
					},
					"simple": "value",
				},
			},
			expected: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"nested": map[string]interface{}{
						"key1": "value1",
						"key2": 123,
					},
					"simple": "value",
				},
			},
		},
		{
			name: "slice values",
			src: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"items":   []interface{}{"item1", "item2", 42},
					"numbers": []interface{}{1, 2, 3},
				},
			},
			expected: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"items":   []interface{}{"item1", "item2", 42},
					"numbers": []interface{}{1, 2, 3},
				},
			},
		},
		{
			name: "complex nested structure",
			src: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"config": map[string]interface{}{
						"servers": []interface{}{
							map[string]interface{}{
								"name": "server1",
								"port": 8080,
							},
							map[string]interface{}{
								"name": "server2",
								"port": 8081,
							},
						},
						"timeout": 30,
					},
				},
			},
			expected: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"config": map[string]interface{}{
						"servers": []interface{}{
							map[string]interface{}{
								"name": "server1",
								"port": 8080,
							},
							map[string]interface{}{
								"name": "server2",
								"port": 8081,
							},
						},
						"timeout": 30,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepCopyCustomizedValue(tt.src)

			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, result)

			// Verify it's a deep copy by modifying original and checking copy is unchanged
			if tt.src != nil && tt.src.Data != nil {
				// Modify original
				if nestedMap, ok := tt.src.Data["config"]; ok {
					if configMap, ok := nestedMap.(map[string]interface{}); ok {
						configMap["modified"] = true
					}
				} else if _, exists := tt.src.Data["simple"]; exists {
					tt.src.Data["modified"] = "changed"
				}

				// Original expected should not be affected since result is a deep copy
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDMICache_OverrideDeviceInstanceConfig(t *testing.T) {
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
							"items":      []interface{}{"item1", "item2"},
							"settings": map[string]interface{}{
								"parity":   "none",
								"stopBits": 1,
							},
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
								"timeout":    3000,                            // instance overrides
								"slaveID":    2,                               // instance specific
								"retryTimes": 3,                               // from model
								"baudRate":   9600,                            // from model
								"items":      []interface{}{"item1", "item2"}, // from model
								"settings": map[string]interface{}{
									"parity":   "none",
									"stopBits": 1,
								}, // from model
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
			// Setup DMICache
			dmiCache := NewDMICache()

			// Add device model to cache if provided
			if tt.deviceModel != nil {
				dmiCache.PutDeviceModel(tt.deviceModel)
			}

			// Execute the function
			_, _, err := dmiCache.overrideDeviceInstanceConfig(tt.device)
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

func TestDMICache_CompareDeviceSpecHasChanged(t *testing.T) {
	tests := []struct {
		name                  string
		oldDevice             *v1beta1.Device
		newDevice             *v1beta1.Device
		expectedCompareResult bool
	}{
		// 1 property、1 method, properties and methods are the same.
		{
			name: "1 property、1 method, properties and methods are the same",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			expectedCompareResult: false,
		},

		// 1 property、1 method, properties and methods are different.
		{
			name: "1 property、1 method, properties and methods are different",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getTemperature",
							Description: "getTemperature description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},

		// 1 property、1 method, properties are different, methods are the same.
		{
			name: "1 property、1 method, properties are different, methods are the same",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},
		// 1 property、1 method, properties are the same, methods are different.
		{
			name: "1 property、1 method, properties are the same, methods are different",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getTemperature",
							Description: "getTemperature description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},
		// 2 properties、2 methods, properties and methods are the same.
		{
			name: "2 properties、2 methods, properties and methods are the same",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: false,
		},
		// 2 properties、2 methods, properties and methods are different.
		{
			name: "2 properties、2 methods, properties and methods are different",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "getTemperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
						{
							Name: "getHumidity",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"getTemperature",
								"getHumidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"getTemperature",
								"getHumidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "setTemperature",
							Description: "setTemperature description",
							PropertyNames: []string{
								"temperature",
							},
						},
						{
							Name:        "setHumidity",
							Description: "setHumidity description",
							PropertyNames: []string{
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},
		// 2 properties、2 methods, methods are the same, the only difference is the order of the properties.
		{
			name: "2 properties、2 methods, methods are the same, the only difference is the order of the properties",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "humidity",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: false,
		},
		// 2 properties、2 methods, properties are the same, the only difference is the order of the properties methods.
		{
			name: "2 properties、2 methods, properties are the same, the only difference is the order of the properties methods",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: false,
		},
		// 2 properties、2 methods, methods are the same, properties are different.
		{
			name: "2 properties、2 methods, methods are the same, properties are different",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40003,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},
		// 2 properties、2 methods, properties are the same, methods are different.
		{
			name: "2 properties、2 methods, properties are the same, methods are different",
			oldDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
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
					},
					Properties: []v1beta1.DeviceProperty{
						{
							Name: "temperature",
							Visitors: v1beta1.VisitorConfig{
								ProtocolName: "modbus",
								ConfigData: &v1beta1.CustomizedValue{
									Data: map[string]interface{}{
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "getStatus",
							Description: "getStatus description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},
		// Different DeviceModelRef
		{
			name: "different device models",
			oldDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sensor-001",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "temp-sensor-model01",
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
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			newDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sensor-001",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "temp-sensor-model02",
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
										"register": "HoldingRegister",
										"address":  40001,
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
										"register": "HoldingRegister",
										"address":  40002,
									},
								},
							},
						},
					},
					Methods: []v1beta1.DeviceMethod{
						{
							Name:        "turnOn",
							Description: "turn on description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
						{
							Name:        "turnOff",
							Description: "turn off description",
							PropertyNames: []string{
								"temperature",
								"humidity",
							},
						},
					},
				},
			},
			expectedCompareResult: true,
		},
	}

	// Setup DMICache
	dmiCache := NewDMICache()

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test %d: %s", i, test.name), func(t *testing.T) {
			dmiCache.PutDevice(test.oldDevice)
			changed := dmiCache.CompareDeviceSpecHasChanged(test.newDevice)
			assert.Equal(t, test.expectedCompareResult, changed)
		})
	}
}
