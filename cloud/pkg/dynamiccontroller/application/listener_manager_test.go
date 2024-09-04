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

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var testGVR = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

var selector1 = NewSelector("key1=value1", "")
var selector2 = NewSelector("key2=value2", "")

func TestNewListenerManager(t *testing.T) {
	assert := assert.New(t)

	lm := newListenerManager()

	assert.NotNil(lm)
	assert.NotNil(lm.listenerByNodeID)
	assert.Empty(lm.listenerByNodeID)

	assert.NotNil(lm.listenerByGVR)
	assert.Empty(lm.listenerByGVR)
}

func TestAddListener(t *testing.T) {
	assert := assert.New(t)

	listener1 := NewSelectorListener("testID1", "node1", testGVR, selector1)
	listener2 := NewSelectorListener("testID2", "node2", testGVR, selector2)

	lm := newListenerManager()

	lm.AddListener(listener1)
	lm.AddListener(listener2)

	listenerByNodeID := lm.GetListenersForNode("node1")
	assert.Len(listenerByNodeID, 1)

	for _, v := range listenerByNodeID {
		assert.Equal(listener1, v)
	}

	listenerByGVR := lm.GetListenersForGVR(testGVR)
	assert.Len(listenerByGVR, 2)

	expected := map[string]*SelectorListener{
		listener1.id: listener1,
		listener2.id: listener2,
	}

	assert.Equal(expected, listenerByGVR)
}

func TestDeleteListener(t *testing.T) {
	assert := assert.New(t)

	listener1 := NewSelectorListener("testID1", "node1", testGVR, selector1)
	listener2 := NewSelectorListener("testID2", "node2", testGVR, selector2)

	lm := newListenerManager()

	lm.AddListener(listener1)
	lm.AddListener(listener2)

	lm.DeleteListener(listener1)
	listenerByNodeID := lm.GetListenersForNode("node1")
	assert.Len(listenerByNodeID, 0)

	lm.DeleteListener(listener2)
	listenerByGVR := lm.GetListenersForGVR(testGVR)
	assert.Len(listenerByGVR, 0)
}

func TestGetListenersForNode(t *testing.T) {
	assert := assert.New(t)

	lm := newListenerManager()

	listener1 := NewSelectorListener("testID1", "node1", testGVR, selector1)
	listener2 := NewSelectorListener("testID2", "node1", testGVR, selector2)
	listener3 := NewSelectorListener("testID3", "node2", testGVR, selector1)

	lm.AddListener(listener1)
	lm.AddListener(listener2)
	lm.AddListener(listener3)

	node1Listeners := lm.GetListenersForNode("node1")
	assert.Len(node1Listeners, 2)
	assert.Contains(node1Listeners, listener1.id)
	assert.Contains(node1Listeners, listener2.id)
	assert.Equal(listener1, node1Listeners[listener1.id])
	assert.Equal(listener2, node1Listeners[listener2.id])

	node2Listeners := lm.GetListenersForNode("node2")
	assert.Len(node2Listeners, 1)
	assert.Contains(node2Listeners, listener3.id)
	assert.Equal(listener3, node2Listeners[listener3.id])

	// Get listeners for non existent node
	nonExistentNodeListeners := lm.GetListenersForNode("node3")
	assert.Nil(nonExistentNodeListeners)
}

func TestGetListenersForGVR(t *testing.T) {
	assert := assert.New(t)

	lm := newListenerManager()

	listener1 := NewSelectorListener("testID1", "node1", testGVR, selector1)
	listener2 := NewSelectorListener("testID2", "node2", testGVR, selector2)
	differentGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	listener3 := NewSelectorListener("testID3", "node1", differentGVR, selector1)

	lm.AddListener(listener1)
	lm.AddListener(listener2)
	lm.AddListener(listener3)

	gvrListeners := lm.GetListenersForGVR(testGVR)
	assert.Len(gvrListeners, 2)
	assert.Contains(gvrListeners, listener1.id)
	assert.Contains(gvrListeners, listener2.id)
	assert.Equal(listener1, gvrListeners[listener1.id])
	assert.Equal(listener2, gvrListeners[listener2.id])

	differentGVRListeners := lm.GetListenersForGVR(differentGVR)
	assert.Len(differentGVRListeners, 1)
	assert.Contains(differentGVRListeners, listener3.id)
	assert.Equal(listener3, differentGVRListeners[listener3.id])

	// Get listeners for non existent GVR
	nonExistentGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nonexistent"}
	nonExistentGVRListeners := lm.GetListenersForGVR(nonExistentGVR)
	assert.Nil(nonExistentGVRListeners)
}
