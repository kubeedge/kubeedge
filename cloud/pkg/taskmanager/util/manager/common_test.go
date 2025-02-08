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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// TestObject implements runtime.Object interface for testing
type TestObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (t *TestObject) DeepCopyObject() runtime.Object {
	return &TestObject{
		TypeMeta:   t.TypeMeta,
		ObjectMeta: t.ObjectMeta,
	}
}

func TestNewCommonResourceEventHandler(t *testing.T) {
	events := make(chan watch.Event)
	handler := NewCommonResourceEventHandler(events)

	assert.NotNil(t, handler, "Handler should not be nil")
	assert.Equal(t, events, handler.events, "Events channel should be properly set")
}

func TestCommonResourceEventHandler_OnAdd(t *testing.T) {
	tests := []struct {
		name        string
		obj         interface{}
		isValid     bool
		expectedObj runtime.Object
	}{
		{
			name: "valid runtime object",
			obj: &TestObject{
				TypeMeta:   metav1.TypeMeta{Kind: "Test"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-obj"},
			},
			isValid:     true,
			expectedObj: &TestObject{TypeMeta: metav1.TypeMeta{Kind: "Test"}, ObjectMeta: metav1.ObjectMeta{Name: "test-obj"}},
		},
		{
			name:    "invalid object type",
			obj:     "invalid",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan watch.Event)
			handler := NewCommonResourceEventHandler(events)

			// Start goroutine to receive events
			received := make(chan watch.Event)
			go func() {
				select {
				case event := <-events:
					received <- event
				case <-time.After(time.Second):
					close(received)
				}
			}()

			handler.OnAdd(tt.obj, false)

			if tt.isValid {
				event := <-received
				assert.Equal(t, watch.Added, event.Type)
				assert.Equal(t, tt.expectedObj, event.Object)
			} else {
				event, ok := <-received
				assert.False(t, ok, "Should not receive event for invalid object, got %v", event)
			}
		})
	}
}

func TestCommonResourceEventHandler_OnUpdate(t *testing.T) {
	tests := []struct {
		name        string
		oldObj      interface{}
		newObj      interface{}
		isValid     bool
		expectedObj runtime.Object
	}{
		{
			name: "valid runtime object update",
			oldObj: &TestObject{
				TypeMeta:   metav1.TypeMeta{Kind: "Test"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-obj-old"},
			},
			newObj: &TestObject{
				TypeMeta:   metav1.TypeMeta{Kind: "Test"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-obj-new"},
			},
			isValid:     true,
			expectedObj: &TestObject{TypeMeta: metav1.TypeMeta{Kind: "Test"}, ObjectMeta: metav1.ObjectMeta{Name: "test-obj-new"}},
		},
		{
			name:    "invalid object type",
			oldObj:  "old-invalid",
			newObj:  "new-invalid",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan watch.Event)
			handler := NewCommonResourceEventHandler(events)

			received := make(chan watch.Event)
			go func() {
				select {
				case event := <-events:
					received <- event
				case <-time.After(time.Second):
					close(received)
				}
			}()

			handler.OnUpdate(tt.oldObj, tt.newObj)

			if tt.isValid {
				event := <-received
				assert.Equal(t, watch.Modified, event.Type)
				assert.Equal(t, tt.expectedObj, event.Object)
			} else {
				event, ok := <-received
				assert.False(t, ok, "Should not receive event for invalid object, got %v", event)
			}
		})
	}
}

func TestCommonResourceEventHandler_OnDelete(t *testing.T) {
	tests := []struct {
		name        string
		obj         interface{}
		isValid     bool
		expectedObj runtime.Object
	}{
		{
			name: "valid runtime object",
			obj: &TestObject{
				TypeMeta:   metav1.TypeMeta{Kind: "Test"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-obj"},
			},
			isValid:     true,
			expectedObj: &TestObject{TypeMeta: metav1.TypeMeta{Kind: "Test"}, ObjectMeta: metav1.ObjectMeta{Name: "test-obj"}},
		},
		{
			name:    "invalid object type",
			obj:     "invalid",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan watch.Event)
			handler := NewCommonResourceEventHandler(events)

			received := make(chan watch.Event)
			go func() {
				select {
				case event := <-events:
					received <- event
				case <-time.After(time.Second):
					close(received)
				}
			}()

			handler.OnDelete(tt.obj)

			if tt.isValid {
				event := <-received
				assert.Equal(t, watch.Deleted, event.Type)
				assert.Equal(t, tt.expectedObj, event.Object)
			} else {
				event, ok := <-received
				assert.False(t, ok, "Should not receive event for invalid object, got %v", event)
			}
		})
	}
}
