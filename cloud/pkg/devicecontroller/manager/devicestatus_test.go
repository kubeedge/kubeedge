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
	"reflect"
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

// mockDeviceStatusInformer implements SharedIndexInformer for testing
type mockDeviceStatusInformer struct {
	cache.SharedIndexInformer
	handler   cache.ResourceEventHandler
	handlerFn func(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)
}

func (m *mockDeviceStatusInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	if m.handlerFn != nil {
		return m.handlerFn(handler)
	}
	m.handler = handler
	return &mockEventHandlerRegistration{}, nil
}

func newMockDeviceStatusInformer(returnError bool) *mockDeviceStatusInformer {
	m := &mockDeviceStatusInformer{}
	if returnError {
		m.handlerFn = func(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
			return nil, errors.New("mock error")
		}
	}
	return m
}

func TestDeviceStatusManager_Events(t *testing.T) {
	e := make(chan watch.Event, 1)
	e <- watch.Event{Type: watch.Added}
	tests := []struct {
		name   string
		events chan watch.Event
		want   chan watch.Event
	}{
		{
			name:   "base",
			events: e,
			want:   e,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsm := &DeviceStatusManager{
				events: tt.events,
			}
			if got := dsm.Events(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeviceStatusManager.Events() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDeviceStatusManager(t *testing.T) {
	dt := int32(1)
	config.Config = config.Configure{
		DeviceController: v1alpha1.DeviceController{
			Buffer: &v1alpha1.DeviceControllerBuffer{
				DeviceEvent: dt,
			},
		},
	}
	e := make(chan watch.Event, dt)
	tests := []struct {
		name    string
		si      cache.SharedIndexInformer
		want    *DeviceStatusManager
		wantErr bool
	}{
		{
			name: "base",
			si:   newMockDeviceStatusInformer(false),
			want: &DeviceStatusManager{
				events: e,
			},
		},
		{
			name:    "error case",
			si:      newMockDeviceStatusInformer(true),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDeviceStatusManager(tt.si)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDeviceStatusManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if got != nil {
					t.Errorf("NewDeviceStatusManager() = %+v, want nil", got)
				}
			} else {
				if !reflect.DeepEqual(len(got.events), len(tt.want.events)) {
					t.Errorf("NewDeviceStatusManager() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestNewDeviceStatusManagerWithRealConfig(t *testing.T) {
	testCases := []struct {
		name         string
		mockInformer *mockDeviceStatusInformer
		expectError  bool
	}{
		{
			name:         "Success case",
			mockInformer: newMockDeviceStatusInformer(false),
			expectError:  false,
		},
		{
			name:         "Error case",
			mockInformer: newMockDeviceStatusInformer(true),
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dsm, err := NewDeviceStatusManager(tc.mockInformer)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, dsm)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dsm)
				assert.NotNil(t, dsm.events)
			}
		})
	}
}

func TestDeviceStatusManagerWithRealEvents(t *testing.T) {
	mockInformer := newMockDeviceStatusInformer(false)
	dsm, err := NewDeviceStatusManager(mockInformer)
	assert.NoError(t, err)
	assert.NotNil(t, dsm)

	testDeviceStatus := &v1beta1.DeviceStatus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device-status",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceStatusSpec{},
	}

	eventReceived := make(chan bool)
	go func() {
		select {
		case event := <-dsm.Events():
			assert.Equal(t, watch.Added, event.Type)
			assert.Equal(t, testDeviceStatus, event.Object)
			eventReceived <- true
		case <-time.After(time.Second):
			eventReceived <- false
		}
	}()

	dsm.events <- watch.Event{
		Type:   watch.Added,
		Object: testDeviceStatus,
	}

	received := <-eventReceived
	assert.True(t, received)
}

func TestDeviceStatusManagerDeviceStatusMap(t *testing.T) {
	dsm := &DeviceStatusManager{
		DeviceStatus: sync.Map{},
	}

	deviceStatusID := "default/test-device-status"
	testDeviceStatus := &v1beta1.DeviceStatus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device-status",
			Namespace: "default",
		},
	}

	dsm.DeviceStatus.Store(deviceStatusID, testDeviceStatus)

	storedDeviceStatus, exists := dsm.DeviceStatus.Load(deviceStatusID)
	assert.True(t, exists)
	assert.Equal(t, testDeviceStatus, storedDeviceStatus)

	dsm.DeviceStatus.Delete(deviceStatusID)
	_, exists = dsm.DeviceStatus.Load(deviceStatusID)
	assert.False(t, exists)
}

func TestDeviceStatusManagerConcurrency(_ *testing.T) {
	dsm := &DeviceStatusManager{
		DeviceStatus: sync.Map{},
		events:       make(chan watch.Event, 100),
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				deviceStatusID := fmt.Sprintf("device-status-%d-%d", routineID, j)
				deviceStatus := &v1beta1.DeviceStatus{
					ObjectMeta: metav1.ObjectMeta{
						Name: deviceStatusID,
					},
				}

				dsm.DeviceStatus.Store(deviceStatusID, deviceStatus)
				_, _ = dsm.DeviceStatus.Load(deviceStatusID)
				dsm.DeviceStatus.Delete(deviceStatusID)
			}
		}(i)
	}

	wg.Wait()
}

func TestDeviceStatusManagerMultipleEvents(t *testing.T) {
	dt := int32(10)
	config.Config = config.Configure{
		DeviceController: v1alpha1.DeviceController{
			Buffer: &v1alpha1.DeviceControllerBuffer{
				DeviceEvent: dt,
			},
		},
	}

	mockInformer := newMockDeviceStatusInformer(false)
	dsm, err := NewDeviceStatusManager(mockInformer)
	assert.NoError(t, err)
	assert.NotNil(t, dsm)

	testEvents := []watch.Event{
		{
			Type: watch.Added,
			Object: &v1beta1.DeviceStatus{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "device-status-1",
					Namespace: "default",
				},
			},
		},
		{
			Type: watch.Modified,
			Object: &v1beta1.DeviceStatus{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "device-status-2",
					Namespace: "default",
				},
			},
		},
		{
			Type: watch.Deleted,
			Object: &v1beta1.DeviceStatus{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "device-status-3",
					Namespace: "default",
				},
			},
		},
	}

	// Send events in a separate goroutine to avoid blocking
	go func() {
		for _, event := range testEvents {
			dsm.events <- event
		}
	}()

	// Receive and verify events
	for i, expectedEvent := range testEvents {
		select {
		case receivedEvent := <-dsm.Events():
			assert.Equal(t, expectedEvent.Type, receivedEvent.Type, "Event %d type mismatch", i)
			assert.Equal(t, expectedEvent.Object, receivedEvent.Object, "Event %d object mismatch", i)
		case <-time.After(time.Second):
			t.Errorf("Timeout waiting for event %d", i)
			return
		}
	}
}
