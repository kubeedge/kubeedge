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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
)

const (
	numGoroutines = 10
	numOperations = 100
	testTimeout   = 5 * time.Second
)

func init() {
	// Initialize global configuration for tests
	config.InitConfigure(&v1alpha1.DeviceController{
		Buffer: &v1alpha1.DeviceControllerBuffer{
			DeviceEvent: 1024,
		},
	})
}

// ---------------------- Mock utilities ----------------------

// mockSharedIndexInformer mocks the AddEventHandler behavior of a SharedIndexInformer.
type mockSharedIndexInformer struct {
	cache.SharedIndexInformer
	handlerFn func(cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)
}

func (m *mockSharedIndexInformer) AddEventHandler(h cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	if m.handlerFn != nil {
		return m.handlerFn(h)
	}
	return &mockEventHandlerRegistration{}, nil
}

// mockEventHandlerRegistration is a stub for ResourceEventHandlerRegistration.
type mockEventHandlerRegistration struct{}

func (*mockEventHandlerRegistration) HasSynced() bool                     { return true }
func (*mockEventHandlerRegistration) Handler() cache.ResourceEventHandler { return nil }

// newMockInformer returns a mockSharedIndexInformer which optionally returns an error
// when AddEventHandler is called.
func newMockInformer(shouldError bool) *mockSharedIndexInformer {
	if shouldError {
		return &mockSharedIndexInformer{
			handlerFn: func(cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
				return nil, errors.New("mock error")
			},
		}
	}
	return &mockSharedIndexInformer{}
}

// ---------------------- Unit tests ----------------------

func TestNewDeviceManager(t *testing.T) {
	cases := []struct {
		name        string
		informerErr bool
		wantErr     bool
	}{
		{"success", false, false},
		{"informer returns error", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := NewDeviceManager(newMockInformer(tc.informerErr))
			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, dm)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, dm)
				assert.NotNil(t, dm.events)
			}
		})
	}
}

func TestDeviceManager_Events(t *testing.T) {
	eventChan := make(chan watch.Event, 1)
	dm := &DeviceManager{events: eventChan}

	assert.Equal(t, eventChan, dm.Events(), "Events() should return the internal events channel")
}

func TestDeviceManagerWithRealEvents(t *testing.T) {
	dm, err := NewDeviceManager(newMockInformer(false))
	require.NoError(t, err)

	testDevice := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
		Spec: v1beta1.DeviceSpec{NodeName: "test-node"},
	}

	done := make(chan bool)
	go func() {
		select {
		case e := <-dm.Events():
			assert.Equal(t, watch.Added, e.Type)
			assert.Equal(t, testDevice, e.Object)
			done <- true
		case <-time.After(time.Second):
			done <- false
		}
	}()

	dm.events <- watch.Event{Type: watch.Added, Object: testDevice}
	assert.True(t, <-done, "Event was not received before timeout")
}

func TestDeviceManagerDeviceMap(t *testing.T) {
	dm := &DeviceManager{Device: sync.Map{}}

	id := "default/test-device"
	dev := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
	}

	// Store device in the map
	dm.Device.Store(id, dev)
	v, ok := dm.Device.Load(id)
	assert.True(t, ok)
	assert.Equal(t, dev, v)

	// Delete device and ensure it is removed
	dm.Device.Delete(id)
	_, ok = dm.Device.Load(id)
	assert.False(t, ok)
}

func TestDeviceManagerConcurrency(t *testing.T) {
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
				id := fmt.Sprintf("device-%d-%d", routineID, j)
				d := &v1beta1.Device{ObjectMeta: metav1.ObjectMeta{Name: id}}

				dm.Device.Store(id, d)
				_, _ = dm.Device.Load(id)
				dm.Device.Delete(id)
			}
		}(i)
	}

	wg.Wait()
}
