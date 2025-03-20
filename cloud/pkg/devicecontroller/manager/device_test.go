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
package manager

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

const (
	numGoroutines  = 10
	numOperations  = 100
	testTimeout    = 5 * time.Second
	defaultTimeout = 1 * time.Second
)

func init() {
	dc := &v1alpha1.DeviceController{
		Buffer: &v1alpha1.DeviceControllerBuffer{
			DeviceEvent: 1024,
		},
	}
	config.InitConfigure(dc)
}

type mockSharedIndexInformer struct {
	cache.SharedIndexInformer
	handler   cache.ResourceEventHandler
	handlerFn func(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)
}

func (m *mockSharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	if m.handlerFn != nil {
		return m.handlerFn(handler)
	}
	m.handler = handler
	return &mockEventHandlerRegistration{}, nil
}

type mockEventHandlerRegistration struct{}

func (m *mockEventHandlerRegistration) HasSynced() bool                     { return true }
func (m *mockEventHandlerRegistration) Handler() cache.ResourceEventHandler { return nil }

func newMockInformer(returnError bool) *mockSharedIndexInformer {
	m := &mockSharedIndexInformer{}
	if returnError {
		m.handlerFn = func(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
			return nil, errors.New("mock error")
		}
	}
	return m
}

func TestNewDeviceManager(t *testing.T) {
	testCases := []struct {
		name         string
		mockInformer *mockSharedIndexInformer
		expectError  bool
	}{
		{
			name:         "Success case",
			mockInformer: newMockInformer(false),
			expectError:  false,
		},
		{
			name:         "Error case",
			mockInformer: newMockInformer(true),
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := NewDeviceManager(tc.mockInformer)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, dm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dm)
				assert.NotNil(t, dm.events)
			}
		})
	}
}

func TestDeviceManager_Events(t *testing.T) {
	eventChan := make(chan watch.Event, 10)
	dm := &DeviceManager{
		events: eventChan,
		Device: sync.Map{},
	}

	resultChan := dm.Events()
	assert.Equal(t, eventChan, resultChan)
}

func TestDeviceManagerWithRealEvents(t *testing.T) {
	mockInformer := newMockInformer(false)
	dm, err := NewDeviceManager(mockInformer)
	assert.NoError(t, err)
	assert.NotNil(t, dm)

	testDevice := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceSpec{
			NodeName: "test-node",
		},
	}

	eventReceived := make(chan bool)
	go func() {
		select {
		case event := <-dm.Events():
			assert.Equal(t, watch.Added, event.Type)
			assert.Equal(t, testDevice, event.Object)
			eventReceived <- true
		case <-time.After(time.Second):
			eventReceived <- false
		}
	}()

	dm.events <- watch.Event{
		Type:   watch.Added,
		Object: testDevice,
	}

	received := <-eventReceived
	assert.True(t, received)
}

func TestDeviceManagerDeviceMap(t *testing.T) {
	dm := &DeviceManager{
		Device: sync.Map{},
	}

	deviceID := "default/test-device"
	testDevice := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
	}

	dm.Device.Store(deviceID, testDevice)

	storedDevice, exists := dm.Device.Load(deviceID)
	assert.True(t, exists)
	assert.Equal(t, testDevice, storedDevice)

	dm.Device.Delete(deviceID)
	_, exists = dm.Device.Load(deviceID)
	assert.False(t, exists)
}

func TestDeviceManagerConcurrency(_ *testing.T) {
	dm := &DeviceManager{
		Device: sync.Map{},
		events: make(chan watch.Event, 100),
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				deviceID := fmt.Sprintf("device-%d-%d", routineID, j)
				device := &v1beta1.Device{
					ObjectMeta: metav1.ObjectMeta{
						Name: deviceID,
					},
				}

				dm.Device.Store(deviceID, device)
				_, _ = dm.Device.Load(deviceID)
				dm.Device.Delete(deviceID)
			}
		}(i)
	}

	wg.Wait()
}
