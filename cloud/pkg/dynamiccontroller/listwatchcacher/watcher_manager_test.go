/*
Copyright 2022 The KubeEdge Authors.

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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

var testGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

var selector1 = NewSelector("key1=value1", "")
var selector2 = NewSelector("key2=value2", "")

func TestAddGetListener(t *testing.T) {
	listener1 := NewCacheWatcher("testID1", "node1", testGVR, selector1)
	listener2 := NewCacheWatcher("testID2", "node2", testGVR, selector2)

	lm := newWatcherManager()

	lm.AddWatcher(listener1)
	lm.AddWatcher(listener2)

	listenerByNodeID := lm.GetWatchersForNode("node1")
	if len(listenerByNodeID) != 1 {
		t.Errorf("watcherByNodeID expected length 1. but got %v", len(listenerByNodeID))
	}

	for _, v := range listenerByNodeID {
		if !reflect.DeepEqual(v, listener1) {
			t.Errorf("expected %v. but got %v", listener1, v)
		}
	}

	listenerByGVR := lm.GetWatchersForGVR(testGVR)
	if len(listenerByGVR) != 2 {
		t.Errorf("watcherByGVR expected length 2. but got %v", len(listenerByNodeID))
	}

	expected := map[string]*CacheWatcher{
		listener1.WatcherID: listener1,
		listener2.WatcherID: listener2,
	}

	if !reflect.DeepEqual(expected, listenerByGVR) {
		t.Errorf("expected %v. but got %v", expected, listenerByGVR)
	}
}

func TestDeleteListener(t *testing.T) {
	listener1 := NewCacheWatcher("testID1", "node1", testGVR, selector1)
	listener2 := NewCacheWatcher("testID2", "node2", testGVR, selector2)

	lm := newWatcherManager()

	lm.AddWatcher(listener1)
	lm.AddWatcher(listener2)

	lm.DeleteWatcher(listener1)
	listenerByNodeID := lm.GetWatchersForNode("node1")
	if len(listenerByNodeID) != 0 {
		t.Errorf("watcherByNodeID expected length 0. but got %v", len(listenerByNodeID))
	}

	lm.DeleteWatcher(listener2)
	listenerByGVR := lm.GetWatchersForGVR(testGVR)
	if len(listenerByGVR) != 0 {
		t.Errorf("watcherByGVR expected length 0. but got %v", len(listenerByNodeID))
	}
}
