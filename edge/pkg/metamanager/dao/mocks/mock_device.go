/*
Copyright 2026 The KubeEdge Authors.

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

package mocks

import (
	"sync"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

// MockDeviceService provides a mock implementation of DeviceService for testing
type MockDeviceService struct {
	mu sync.RWMutex

	// QueryDeviceAllFunc can be overridden for testing
	QueryDeviceAllFunc func() ([]models.Device, error)

	// QueryDeviceFunc can be overridden for testing
	QueryDeviceFunc func(key, condition string) ([]models.Device, error)

	// QueryDeviceAttrFunc can be overridden for testing
	QueryDeviceAttrFunc func(key, condition string) (*[]models.DeviceAttr, error)

	// QueryDeviceTwinFunc can be overridden for testing
	QueryDeviceTwinFunc func(key, condition string) (*[]models.DeviceTwin, error)

	// UpdateDeviceFieldsFunc can be overridden for testing
	UpdateDeviceFieldsFunc func(deviceID string, cols map[string]interface{}) error

	// DeviceAttrTransFunc can be overridden for testing
	DeviceAttrTransFunc func(adds []models.DeviceAttr, deletes []models.DeviceDelete, updates []models.DeviceAttrUpdate) error

	// DeviceTwinTransFunc can be overridden for testing
	DeviceTwinTransFunc func(adds []models.DeviceTwin, deletes []models.DeviceDelete, updates []models.DeviceTwinUpdate) error

	// AddDeviceTransFunc can be overridden for testing
	AddDeviceTransFunc func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error

	// DeleteDeviceTransFunc can be overridden for testing
	DeleteDeviceTransFunc func(deletes []string) error
}

// NewMockDeviceService creates a new mock device service with default implementations
func NewMockDeviceService() *MockDeviceService {
	return &MockDeviceService{
		QueryDeviceAllFunc: func() ([]models.Device, error) {
			return nil, nil
		},
		QueryDeviceFunc: func(key, condition string) ([]models.Device, error) {
			return nil, nil
		},
		QueryDeviceAttrFunc: func(key, condition string) (*[]models.DeviceAttr, error) {
			return nil, nil
		},
		QueryDeviceTwinFunc: func(key, condition string) (*[]models.DeviceTwin, error) {
			return nil, nil
		},
		UpdateDeviceFieldsFunc: func(deviceID string, cols map[string]interface{}) error {
			return nil
		},
		DeviceAttrTransFunc: func(adds []models.DeviceAttr, deletes []models.DeviceDelete, updates []models.DeviceAttrUpdate) error {
			return nil
		},
		DeviceTwinTransFunc: func(adds []models.DeviceTwin, deletes []models.DeviceDelete, updates []models.DeviceTwinUpdate) error {
			return nil
		},
		AddDeviceTransFunc: func(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
			return nil
		},
		DeleteDeviceTransFunc: func(deletes []string) error {
			return nil
		},
	}
}

// QueryDeviceAll mocks the QueryDeviceAll method
func (m *MockDeviceService) QueryDeviceAll() ([]models.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryDeviceAllFunc()
}

// QueryDevice mocks the QueryDevice method
func (m *MockDeviceService) QueryDevice(key, condition string) ([]models.Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryDeviceFunc(key, condition)
}

// QueryDeviceAttr mocks the QueryDeviceAttr method
func (m *MockDeviceService) QueryDeviceAttr(key, condition string) (*[]models.DeviceAttr, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryDeviceAttrFunc(key, condition)
}

// QueryDeviceTwin mocks the QueryDeviceTwin method
func (m *MockDeviceService) QueryDeviceTwin(key, condition string) (*[]models.DeviceTwin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.QueryDeviceTwinFunc(key, condition)
}

// UpdateDeviceFields mocks the UpdateDeviceFields method
func (m *MockDeviceService) UpdateDeviceFields(deviceID string, cols map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.UpdateDeviceFieldsFunc(deviceID, cols)
}

// DeviceAttrTrans mocks the DeviceAttrTrans method
func (m *MockDeviceService) DeviceAttrTrans(adds []models.DeviceAttr, deletes []models.DeviceDelete, updates []models.DeviceAttrUpdate) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.DeviceAttrTransFunc(adds, deletes, updates)
}

// DeviceTwinTrans mocks the DeviceTwinTrans method
func (m *MockDeviceService) DeviceTwinTrans(adds []models.DeviceTwin, deletes []models.DeviceDelete, updates []models.DeviceTwinUpdate) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.DeviceTwinTransFunc(adds, deletes, updates)
}

func (m *MockDeviceService) SaveDevice(doc *models.Device) error {
	return nil
}

func (m *MockDeviceService) DeleteDeviceByID(id string) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceField(deviceID string, col string, value interface{}) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceMulti(updates []models.DeviceUpdate) error {
	return nil
}

func (m *MockDeviceService) AddDeviceTrans(adds []models.Device, addAttrs []models.DeviceAttr, addTwins []models.DeviceTwin) error {
	return nil
}

func (m *MockDeviceService) DeleteDeviceTrans(deletes []string) error {
	return nil
}

func (m *MockDeviceService) SaveDeviceAttr(doc *models.DeviceAttr) error {
	return nil
}

func (m *MockDeviceService) DeleteDeviceAttr(deviceID string, name string) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceAttrField(deviceID, name, col string, value interface{}) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceAttrFields(deviceID, name string, cols map[string]interface{}) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceAttrMulti(updates []models.DeviceAttrUpdate) error {
	return nil
}

func (m *MockDeviceService) SaveDeviceTwin(doc *models.DeviceTwin) error {
	return nil
}

func (m *MockDeviceService) DeleteDeviceTwin(deviceID, name string) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceTwinField(deviceID, name, col string, value interface{}) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceTwinFields(deviceID, name string, cols map[string]interface{}) error {
	return nil
}

func (m *MockDeviceService) UpdateDeviceTwinMulti(updates []models.DeviceTwinUpdate) error {
	return nil
}
