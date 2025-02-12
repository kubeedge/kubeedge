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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestCommonResourceEventHandler(t *testing.T) {
	testCases := []struct {
		name         string
		input        interface{}
		expectedType watch.EventType
		methodToCall string
	}{
		{
			name:         "OnAdd Event",
			input:        &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap"}},
			expectedType: watch.Added,
			methodToCall: "OnAdd",
		},
		{
			name:         "OnUpdate Event",
			input:        &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}},
			expectedType: watch.Modified,
			methodToCall: "OnUpdate",
		},
		{
			name:         "OnDelete Event",
			input:        &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service"}},
			expectedType: watch.Deleted,
			methodToCall: "OnDelete",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eventsChan := make(chan watch.Event, 1)
			handler := NewCommonResourceEventHandler(eventsChan)

			switch tc.methodToCall {
			case "OnAdd":
				handler.OnAdd(tc.input, false)
			case "OnUpdate":
				handler.OnUpdate(nil, tc.input)
			case "OnDelete":
				handler.OnDelete(tc.input)
			}

			select {
			case event := <-eventsChan:
				if event.Type != tc.expectedType {
					t.Errorf("Expected event type %v, got %v", tc.expectedType, event.Type)
				}
			default:
				t.Errorf("No event was sent to the channel for %s", tc.name)
			}
		})
	}

	t.Run("Unsupported Object Type", func(t *testing.T) {
		eventsChan := make(chan watch.Event, 1)
		handler := NewCommonResourceEventHandler(eventsChan)

		unsupportedObj := "not a runtime object"
		handler.OnAdd(unsupportedObj, false)

		select {
		case <-eventsChan:
			t.Errorf("Event should not be sent for unsupported type")
		default:
			// Expected behavior - no event sent
		}
	})
}
