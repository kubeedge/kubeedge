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

package imitator

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestEventListFormats(t *testing.T) {
	const first = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"first"}}`
	const second = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"second"}}`
	const bare = `{"metadata":{"name":"bare"}}`
	tests := []struct {
		name     string
		content  interface{}
		expected []string
	}{
		{"object", []byte(first), []string{"first"}},
		{"object list", []byte(`{"apiVersion":"v1","kind":"PodList","items":[` + first + `,` + second + `]}`), []string{"first", "second"}},
		{"object list body", map[string]interface{}{
			"apiVersion": "v1", "kind": "PodList",
			"items": []map[string]interface{}{
				{"kind": "Pod", "metadata": map[string]interface{}{"name": "first"}},
				{"kind": "Pod", "metadata": map[string]interface{}{"name": "second"}},
			},
		}, []string{"first", "second"}},
		{"object list item without kind", []byte(`{"apiVersion":"v1","kind":"PodList","items":[` + bare + `]}`), []string{"bare"}},
		{"array bytes", []byte(" \n[" + first + "," + second + "]"), []string{"first", "second"}},
		{"array item without kind", []byte(`[` + bare + `]`), []string{"bare"}},
		{"array body", []map[string]interface{}{
			{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "first"}},
			{"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "second"}},
		}, []string{"first", "second"}},
		{"empty array", []byte(`[]`), nil},
		{"invalid array item", []byte(`[` + first + `,42]`), nil},
		{"null array item", []byte(`[` + first + `,null]`), nil},
	}

	s := newV2Client().(*imitator)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &model.Message{
				Router:  model.MessageRoute{Operation: model.InsertOperation, Resource: "default/pod"},
				Content: tt.content,
			}
			var names []string
			for _, event := range s.Event(msg) {
				if event.Type != watch.Added {
					t.Errorf("event type = %q, want %q", event.Type, watch.Added)
				}
				obj, ok := event.Object.(*unstructured.Unstructured)
				if !ok {
					t.Fatalf("event object type = %T, want *unstructured.Unstructured", event.Object)
				}
				names = append(names, obj.GetName())
			}
			if !reflect.DeepEqual(names, tt.expected) {
				t.Errorf("event names = %v, expected %v", names, tt.expected)
			}
		})
	}
}

func TestEventPreservesLargeInteger(t *testing.T) {
	const value int64 = 9007199254740993 //2^53 + 1
	const item = `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"first"},"spec":{"large":9007199254740993}}`
	tests := []struct {
		name    string
		content interface{}
	}{
		{"object list", []byte(`{"apiVersion":"v1","kind":"PodList","items":[` + item + `]}`)},
		{"array bytes", []byte(`[` + item + `]`)},
		{"array body", []map[string]interface{}{{
			"apiVersion": "v1", "kind": "Pod", "metadata": map[string]interface{}{"name": "first"},
			"spec": map[string]interface{}{"large": value},
		}}},
	}

	s := newV2Client().(*imitator)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &model.Message{
				Router:  model.MessageRoute{Operation: model.InsertOperation, Resource: "default/pod"},
				Content: tt.content,
			}
			events := s.Event(msg)
			if len(events) != 1 {
				t.Fatalf("got %d events, want 1", len(events))
			}
			obj, ok := events[0].Object.(*unstructured.Unstructured)
			if !ok {
				t.Fatalf("event object type = %T, want *unstructured.Unstructured", events[0].Object)
			}
			got, found, err := unstructured.NestedInt64(obj.Object, "spec", "large")
			if err != nil || !found || got != value {
				t.Errorf("spec.large = %d (found %t, error %v), want int64 %d", got, found, err, value)
			}
		})
	}
}
