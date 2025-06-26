/*
Copyright 2024 The KubeEdge Authors.

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

package controller

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/pkg/util"
)

type MockMessageLayer struct {
	sendFunc     func(message model.Message) error
	receiveFunc  func() (model.Message, error)
	responseFunc func(message model.Message) error
}

func (m *MockMessageLayer) Send(message model.Message) error {
	if m.sendFunc != nil {
		return m.sendFunc(message)
	}
	return nil
}

func (m *MockMessageLayer) Receive() (model.Message, error) {
	if m.receiveFunc != nil {
		return m.receiveFunc()
	}
	return model.Message{}, nil
}

func (m *MockMessageLayer) Response(message model.Message) error {
	if m.responseFunc != nil {
		return m.responseFunc(message)
	}
	return nil
}

func TestDeviceLifecycleFunctions(t *testing.T) {
	msgLayer := &MockMessageLayer{}

	deviceModelManager := &manager.DeviceModelManager{
		DeviceModel: sync.Map{},
	}

	deviceManager := &manager.DeviceManager{
		Device: sync.Map{},
	}

	deviceModel := &v1beta1.DeviceModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-model",
			Namespace: "default",
		},
	}
	deviceModelID := util.GetResourceID(deviceModel.Namespace, deviceModel.Name)
	deviceModelManager.DeviceModel.Store(deviceModelID, deviceModel)

	dc := &DownstreamController{
		deviceManager:      deviceManager,
		deviceModelManager: deviceModelManager,
		messageLayer:       msgLayer,
	}

	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "edge-node-1",
			DeviceModelRef: &v1.LocalObjectReference{
				Name: "test-model",
			},
		},
	}

	t.Run("deviceAdded", func(t *testing.T) {
		dc.deviceAdded(device)

		deviceID := util.GetResourceID(device.Namespace, device.Name)
		val, exists := deviceManager.Device.Load(deviceID)
		if !exists {
			t.Error("Device was not added to the device manager")
		}
		if val != device {
			t.Error("Added device does not match the input device")
		}
	})

	t.Run("deviceUpdated with node change", func(t *testing.T) {
		updatedDevice := device.DeepCopy()
		updatedDevice.Spec.NodeName = "edge-node-2"

		dc.deviceUpdated(updatedDevice)

		deviceID := util.GetResourceID(updatedDevice.Namespace, updatedDevice.Name)
		val, exists := deviceManager.Device.Load(deviceID)
		if !exists {
			t.Error("Device was not updated in the device manager")
		}
		storedDevice, ok := val.(*v1beta1.Device)
		if !ok {
			t.Error("Stored value is not a Device")
		} else if storedDevice.Spec.NodeName != "edge-node-2" {
			t.Errorf("Device node name was not updated, expected: edge-node-2, got: %s", storedDevice.Spec.NodeName)
		}
	})

	t.Run("deviceDeleted", func(t *testing.T) {
		deviceID := util.GetResourceID(device.Namespace, device.Name)
		deviceManager.Device.Store(deviceID, device)

		dc.deviceDeleted(device)

		_, exists := deviceManager.Device.Load(deviceID)
		if exists {
			t.Error("Device was not deleted from the device manager")
		}
	})

	t.Run("deviceModel functions", func(t *testing.T) {
		newModel := &v1beta1.DeviceModel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "new-model",
				Namespace: "default",
			},
		}

		dc.deviceModelAdded(newModel)
		modelID := util.GetResourceID(newModel.Namespace, newModel.Name)
		val, exists := deviceModelManager.DeviceModel.Load(modelID)
		if !exists {
			t.Error("DeviceModel was not added to the device model manager")
		}
		if val != newModel {
			t.Error("Added device model does not match the input model")
		}

		updatedModel := newModel.DeepCopy()
		updatedModel.Labels = map[string]string{"updated": "true"}

		dc.deviceModelUpdated(updatedModel)
		val, exists = deviceModelManager.DeviceModel.Load(modelID)
		if !exists {
			t.Error("DeviceModel was not updated in the device model manager")
		}
		if val != updatedModel {
			t.Error("Updated device model does not match the input model")
		}

		dc.deviceModelDeleted(updatedModel)
		_, exists = deviceModelManager.DeviceModel.Load(modelID)
		if exists {
			t.Error("DeviceModel was not deleted from the device model manager")
		}
	})
}

func TestRemoveTwinWithNameChanged(t *testing.T) {
	device := &v1beta1.Device{
		Spec: v1beta1.DeviceSpec{
			Properties: []v1beta1.DeviceProperty{
				{Name: "temp"},
				{Name: "humidity"},
			},
		},
		Status: v1beta1.DeviceStatus{
			Twins: []v1beta1.Twin{
				{
					PropertyName: "temp",
					Reported:     v1beta1.TwinProperty{Value: "25"},
				},
				{
					PropertyName: "pressure",
					Reported:     v1beta1.TwinProperty{Value: "1000"},
				},
				{
					PropertyName: "humidity",
					Reported:     v1beta1.TwinProperty{Value: "60"},
				},
			},
		},
	}

	expected := []v1beta1.Twin{
		{
			PropertyName: "temp",
			Reported:     v1beta1.TwinProperty{Value: "25"},
		},
		{
			PropertyName: "humidity",
			Reported:     v1beta1.TwinProperty{Value: "60"},
		},
	}

	removeTwinWithNameChanged(device)

	if len(device.Status.Twins) != len(expected) {
		t.Errorf("Expected %d twins, got %d", len(expected), len(device.Status.Twins))
	}

	for i, twin := range device.Status.Twins {
		if i >= len(expected) {
			break
		}
		if twin.PropertyName != expected[i].PropertyName {
			t.Errorf("Twin %d: expected PropertyName %s, got %s",
				i, expected[i].PropertyName, twin.PropertyName)
		}
		if twin.Reported.Value != expected[i].Reported.Value {
			t.Errorf("Twin %d: expected Value %s, got %s",
				i, expected[i].Reported.Value, twin.Reported.Value)
		}
	}
}

func TestIsDeviceUpdated(t *testing.T) {
	testCases := []struct {
		name     string
		old      *v1beta1.Device
		new      *v1beta1.Device
		expected bool
	}{
		{
			name: "No changes",
			old: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{Name: "dev1"},
				Spec:       v1beta1.DeviceSpec{NodeName: "node1"},
			},
			new: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{Name: "dev1"},
				Spec:       v1beta1.DeviceSpec{NodeName: "node1"},
			},
			expected: false,
		},
		{
			name: "ResourceVersion changed only",
			old: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "dev1",
					ResourceVersion: "1",
				},
				Spec: v1beta1.DeviceSpec{NodeName: "node1"},
			},
			new: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "dev1",
					ResourceVersion: "2",
				},
				Spec: v1beta1.DeviceSpec{NodeName: "node1"},
			},
			expected: false,
		},
		{
			name: "ObjectMeta changed",
			old: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{Name: "dev1"},
				Spec:       v1beta1.DeviceSpec{NodeName: "node1"},
			},
			new: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "dev1",
					Labels: map[string]string{"updated": "true"},
				},
				Spec: v1beta1.DeviceSpec{NodeName: "node1"},
			},
			expected: true,
		},
		{
			name: "Spec changed",
			old: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{Name: "dev1"},
				Spec:       v1beta1.DeviceSpec{NodeName: "node1"},
			},
			new: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{Name: "dev1"},
				Spec:       v1beta1.DeviceSpec{NodeName: "node2"},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isDeviceUpdated(tc.old, tc.new)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestCreateDevice(t *testing.T) {
	tests := []struct {
		name           string
		device         *v1beta1.Device
		expectedID     string
		expectedName   string
		expectedDesc   string
		hasDescription bool
	}{
		{
			name: "Device without description",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "temp-sensor",
					Namespace: "default",
				},
			},
			expectedID:     "default/temp-sensor",
			expectedName:   "temp-sensor",
			hasDescription: false,
		},
		{
			name: "Device with description",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "temp-sensor",
					Namespace: "default",
					Labels: map[string]string{
						"description": "Temperature sensor in room 101",
					},
				},
			},
			expectedID:     "default/temp-sensor",
			expectedName:   "temp-sensor",
			expectedDesc:   "Temperature sensor in room 101",
			hasDescription: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edgeDevice := createDevice(tt.device)
			if edgeDevice.ID != tt.expectedID {
				t.Errorf("Expected ID %s, got %s", tt.expectedID, edgeDevice.ID)
			}
			if edgeDevice.Name != tt.expectedName {
				t.Errorf("Expected Name %s, got %s", tt.expectedName, edgeDevice.Name)
			}

			if tt.hasDescription {
				if edgeDevice.Description != tt.expectedDesc {
					t.Errorf("Expected Description %s, got %s",
						tt.expectedDesc, edgeDevice.Description)
				}
			} else {
				if edgeDevice.Description != "" {
					t.Errorf("Expected empty Description, got %s", edgeDevice.Description)
				}
			}
		})
	}
}

func TestIsExistModel(t *testing.T) {
	deviceMap := &sync.Map{}

	device1 := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "device1",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "edge-node-1",
			DeviceModelRef: &v1.LocalObjectReference{
				Name: "model1",
			},
		},
	}

	device2 := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "device2",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "edge-node-1",
			DeviceModelRef: &v1.LocalObjectReference{
				Name: "model2",
			},
		},
	}

	device3 := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "device3",
			Namespace: "production",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "edge-node-1",
			DeviceModelRef: &v1.LocalObjectReference{
				Name: "model1",
			},
		},
	}

	device4 := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "device4",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "",
			DeviceModelRef: &v1.LocalObjectReference{
				Name: "model1",
			},
		},
	}

	deviceMap.Store("default/device1", device1)
	deviceMap.Store("default/device2", device2)
	deviceMap.Store("production/device3", device3)
	deviceMap.Store("default/device4", device4)

	tests := []struct {
		name     string
		device   *v1beta1.Device
		expected bool
	}{
		{
			name: "Model exists in same node and namespace",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-1",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expected: true,
		},
		{
			name: "Model doesn't exist in same node and namespace",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-2",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expected: false,
		},
		{
			name: "Check the same device (should be excluded)",
			device: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "device1",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-1",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExistModel(deviceMap, tt.device)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

type MockDeviceManager struct {
	Device *sync.Map
	events chan watch.Event
}

func (m *MockDeviceManager) Events() chan watch.Event {
	return m.events
}

type MockDeviceModelManager struct {
	DeviceModel *sync.Map
	events      chan watch.Event
}

func (m *MockDeviceModelManager) Events() chan watch.Event {
	return m.events
}

type MockDownstreamController struct {
	DeviceMap      *sync.Map
	DeletedDevices []*v1beta1.Device
	AddedDevices   []*v1beta1.Device
	SentModelMsgs  map[string][]string
	SentDeviceMsgs map[string][]string
}

func NewMockDownstreamController() *MockDownstreamController {
	return &MockDownstreamController{
		DeviceMap:      &sync.Map{},
		DeletedDevices: []*v1beta1.Device{},
		AddedDevices:   []*v1beta1.Device{},
		SentModelMsgs:  make(map[string][]string),
		SentDeviceMsgs: make(map[string][]string),
	}
}

func (m *MockDownstreamController) deviceDeleted(device *v1beta1.Device) {
	m.DeletedDevices = append(m.DeletedDevices, device)

	deviceID := util.GetResourceID(device.Namespace, device.Name)
	m.DeviceMap.Delete(deviceID)
}

func (m *MockDownstreamController) deviceAdded(device *v1beta1.Device) {
	m.AddedDevices = append(m.AddedDevices, device)

	deviceID := util.GetResourceID(device.Namespace, device.Name)
	m.DeviceMap.Store(deviceID, device)
}

func (m *MockDownstreamController) sendDeviceModelMsg(device *v1beta1.Device, operation string) {
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	m.SentModelMsgs[deviceID] = append(m.SentModelMsgs[deviceID], operation)
}

func (m *MockDownstreamController) sendDeviceMsg(device *v1beta1.Device, operation string) {
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	m.SentDeviceMsgs[deviceID] = append(m.SentDeviceMsgs[deviceID], operation)
}

func testDeviceUpdated(dc *MockDownstreamController, device *v1beta1.Device) {
	if len(device.Status.Twins) > 0 {
		removeTwinWithNameChanged(device)
	}

	deviceID := util.GetResourceID(device.Namespace, device.Name)
	value, ok := dc.DeviceMap.Load(deviceID)
	dc.DeviceMap.Store(deviceID, device)

	if ok {
		cachedDevice := value.(*v1beta1.Device)
		if isDeviceUpdated(cachedDevice, device) {
			if cachedDevice.Spec.NodeName != device.Spec.NodeName {
				deletedDevice := &v1beta1.Device{
					ObjectMeta: cachedDevice.ObjectMeta,
					Spec:       cachedDevice.Spec,
					Status:     cachedDevice.Status,
					TypeMeta:   device.TypeMeta,
				}
				dc.deviceDeleted(deletedDevice)
				dc.deviceAdded(device)
			} else {
				dc.sendDeviceModelMsg(device, model.UpdateOperation)
				dc.sendDeviceMsg(device, model.UpdateOperation)
			}
		}
	} else {
		dc.deviceAdded(device)
	}
}

func TestDeviceUpdated(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(*MockDownstreamController)
		updateDevice    *v1beta1.Device
		expectDeleted   bool
		expectAdded     bool
		expectModelMsg  bool
		expectDeviceMsg bool
	}{
		{
			name: "Device not in cache - should add",
			setupFunc: func(dc *MockDownstreamController) {
			},
			updateDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-1",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expectDeleted:   false,
			expectAdded:     true,
			expectModelMsg:  false,
			expectDeviceMsg: false,
		},
		{
			name: "Device in cache but not updated - no changes",
			setupFunc: func(dc *MockDownstreamController) {
				device := &v1beta1.Device{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-device",
						Namespace: "default",
					},
					Spec: v1beta1.DeviceSpec{
						NodeName: "edge-node-1",
						DeviceModelRef: &v1.LocalObjectReference{
							Name: "model1",
						},
					},
				}
				deviceID := util.GetResourceID(device.Namespace, device.Name)
				dc.DeviceMap.Store(deviceID, device)
			},
			updateDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-1",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expectDeleted:   false,
			expectAdded:     false,
			expectModelMsg:  false,
			expectDeviceMsg: false,
		},
		{
			name: "Device in cache and updated with same NodeName - send update msgs",
			setupFunc: func(dc *MockDownstreamController) {
				device := &v1beta1.Device{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "updated-device",
						Namespace: "default",
					},
					Spec: v1beta1.DeviceSpec{
						NodeName: "edge-node-1",
						DeviceModelRef: &v1.LocalObjectReference{
							Name: "model1",
						},
					},
				}
				deviceID := util.GetResourceID(device.Namespace, device.Name)
				dc.DeviceMap.Store(deviceID, device)
			},
			updateDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "updated-device",
					Namespace: "default",
					Labels:    map[string]string{"updated": "true"},
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-1",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expectDeleted:   false,
			expectAdded:     false,
			expectModelMsg:  true,
			expectDeviceMsg: true,
		},
		{
			name: "Device in cache with NodeName changed - delete and add",
			setupFunc: func(dc *MockDownstreamController) {
				device := &v1beta1.Device{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "node-change-device",
						Namespace: "default",
					},
					Spec: v1beta1.DeviceSpec{
						NodeName: "edge-node-1",
						DeviceModelRef: &v1.LocalObjectReference{
							Name: "model1",
						},
					},
				}
				deviceID := util.GetResourceID(device.Namespace, device.Name)
				dc.DeviceMap.Store(deviceID, device)
			},
			updateDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "node-change-device",
					Namespace: "default",
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-2",
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
			},
			expectDeleted:   true,
			expectAdded:     true,
			expectModelMsg:  false,
			expectDeviceMsg: false,
		},
		{
			name: "Device with twins - should clean up twins",
			setupFunc: func(dc *MockDownstreamController) {
				device := &v1beta1.Device{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "device-with-twins",
						Namespace: "default",
					},
					Spec: v1beta1.DeviceSpec{
						NodeName: "edge-node-1",
						Properties: []v1beta1.DeviceProperty{
							{Name: "temp"},
						},
					},
				}
				deviceID := util.GetResourceID(device.Namespace, device.Name)
				dc.DeviceMap.Store(deviceID, device)
			},
			updateDevice: &v1beta1.Device{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "device-with-twins",
					Namespace: "default",
					Labels:    map[string]string{"updated": "true"},
				},
				Spec: v1beta1.DeviceSpec{
					NodeName: "edge-node-1",
					Properties: []v1beta1.DeviceProperty{
						{Name: "temp"},
					},
					DeviceModelRef: &v1.LocalObjectReference{
						Name: "model1",
					},
				},
				Status: v1beta1.DeviceStatus{
					Twins: []v1beta1.Twin{
						{
							PropertyName: "temp",
							Reported: v1beta1.TwinProperty{
								Value: "25",
							},
						},
						{
							PropertyName: "pressure",
							Reported: v1beta1.TwinProperty{
								Value: "1000",
							},
						},
					},
				},
			},
			expectDeleted:   false,
			expectAdded:     false,
			expectModelMsg:  true,
			expectDeviceMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := NewMockDownstreamController()

			tt.setupFunc(dc)

			testDeviceUpdated(dc, tt.updateDevice)

			deviceID := util.GetResourceID(tt.updateDevice.Namespace, tt.updateDevice.Name)

			wasDeleted := false
			for _, device := range dc.DeletedDevices {
				if device.Name == tt.updateDevice.Name && device.Namespace == tt.updateDevice.Namespace {
					wasDeleted = true
					break
				}
			}
			assert.Equal(t, tt.expectDeleted, wasDeleted, "Expected device deleted: %v, got: %v", tt.expectDeleted, wasDeleted)

			wasAdded := false
			for _, device := range dc.AddedDevices {
				if device.Name == tt.updateDevice.Name && device.Namespace == tt.updateDevice.Namespace {
					wasAdded = true
					break
				}
			}
			assert.Equal(t, tt.expectAdded, wasAdded, "Expected device added: %v, got: %v", tt.expectAdded, wasAdded)

			modelMsgSent := len(dc.SentModelMsgs[deviceID]) > 0
			assert.Equal(t, tt.expectModelMsg, modelMsgSent, "Expected model msg sent: %v, got: %v", tt.expectModelMsg, modelMsgSent)

			deviceMsgSent := len(dc.SentDeviceMsgs[deviceID]) > 0
			assert.Equal(t, tt.expectDeviceMsg, deviceMsgSent, "Expected device msg sent: %v, got: %v", tt.expectDeviceMsg, deviceMsgSent)

			if len(tt.updateDevice.Status.Twins) > 0 {
				for _, twin := range tt.updateDevice.Status.Twins {
					propertyExists := false
					for _, prop := range tt.updateDevice.Spec.Properties {
						if prop.Name == twin.PropertyName {
							propertyExists = true
							break
						}
					}
					assert.True(t, propertyExists, "Twin with PropertyName %s should have been removed", twin.PropertyName)
				}
			}
		})
	}
}
