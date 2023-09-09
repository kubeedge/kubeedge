/*
Copyright 2023 The KubeEdge Authors.

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

package listwatchcacher

import (
	"strconv"
	"testing"

	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func newTestWatchCache(capacity int) *watchCache {
	return &watchCache{
		capacity:   capacity,
		startIndex: 0,
		endIndex:   0,
		cache:      make([]watchCacheEvent, capacity),
	}
}

func makeTestEvent(name string, resourceVersion uint64, eventType watch.EventType) watchCacheEvent {
	return watchCacheEvent{
		event: watch.Event{
			Object: makeTestPod(name, resourceVersion),
			Type:   eventType,
		},
		ResourceVersion: resourceVersion,
	}
}

func makeTestPod(name string, resourceVersion uint64) *v1.Pod {
	return makeTestPodDetails(name, resourceVersion, "some-node", map[string]string{"k8s-app": "my-app"})
}

func makeTestPodDetails(name string, resourceVersion uint64, nodeName string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            name,
			ResourceVersion: strconv.FormatUint(resourceVersion, 10),
			Labels:          labels,
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func TestWatchCache(t *testing.T) {
	watchCache := newTestWatchCache(5)

	watchCache.Add(makeTestEvent("pod", 3, watch.Added))

	// Test for Added event.
	{
		_, err := watchCache.GetAllEventsSince(1)
		if err == nil {
			t.Errorf("expected error too old")
		}
		if _, ok := err.(*errors.StatusError); !ok {
			t.Errorf("expected error to be of type StatusError")
		}
	}
	{
		result, err := watchCache.GetAllEventsSince(2)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("unexpected events: %v", result)
		}
		if result[0].event.Type != watch.Added {
			t.Errorf("unexpected event type: %v", result[0].event.Type)
		}
		pod := makeTestPod("pod", uint64(3))
		if !apiequality.Semantic.DeepEqual(pod, result[0].event.Object) {
			t.Errorf("unexpected item: %v, expected: %v", result[0].event.Object, pod)
		}
	}

	watchCache.Add(makeTestEvent("pod", 4, watch.Modified))
	watchCache.Add(makeTestEvent("pod", 5, watch.Modified))

	// Test with not full cache.
	{
		_, err := watchCache.GetAllEventsSince(1)
		if err == nil {
			t.Errorf("expected error too old")
		}
	}
	{
		result, err := watchCache.GetAllEventsSince(3)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("unexpected events: %v", result)
		}
		for i := 0; i < 2; i++ {
			if result[i].event.Type != watch.Modified {
				t.Errorf("unexpected event type: %v", result[i].event.Type)
			}
			pod := makeTestPod("pod", uint64(i+4))
			if !apiequality.Semantic.DeepEqual(pod, result[i].event.Object) {
				t.Errorf("unexpected item: %v, expected: %v", result[i].event.Object, pod)
			}
		}
	}

	for i := 6; i < 10; i++ {
		watchCache.Add(makeTestEvent("pod", uint64(i), watch.Modified))
	}

	// Test with full cache - there should be elements from 5 to 9.
	{
		_, err := watchCache.GetAllEventsSince(3)
		if err == nil {
			t.Errorf("expected error too old")
		}
	}
	{
		result, err := watchCache.GetAllEventsSince(4)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 5 {
			t.Fatalf("unexpected events: %v", result)
		}
		for i := 0; i < 5; i++ {
			pod := makeTestPod("pod", uint64(i+5))
			if !apiequality.Semantic.DeepEqual(pod, result[i].event.Object) {
				t.Errorf("unexpected item: %v, expected: %v", result[i].event.Object, pod)
			}
		}
	}

	// Test for delete event.
	watchCache.Add(makeTestEvent("pod", uint64(10), watch.Deleted))

	{
		result, err := watchCache.GetAllEventsSince(9)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("unexpected events: %v", result)
		}
		if result[0].event.Type != watch.Deleted {
			t.Errorf("unexpected event type: %v", result[0].event.Type)
		}
		pod := makeTestPod("pod", uint64(10))
		if !apiequality.Semantic.DeepEqual(pod, result[0].event.Object) {
			t.Errorf("unexpected item: %v, expected: %v", result[0].event.Object, pod)
		}
	}
}
