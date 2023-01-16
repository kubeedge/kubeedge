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
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type listenerManager struct {
	// Protects all internal maps.
	lock sync.RWMutex

	// first key: nodeID
	// second key: listenerID
	listenerByNodeID map[string]map[string]*SelectorListener

	// first key: GVR
	// second key: listenerID
	listenerByGVR map[schema.GroupVersionResource]map[string]*SelectorListener
}

func newListenerManager() *listenerManager {
	return &listenerManager{
		listenerByNodeID: make(map[string]map[string]*SelectorListener),
		listenerByGVR:    make(map[schema.GroupVersionResource]map[string]*SelectorListener),
	}
}

func (lm *listenerManager) AddListener(listener *SelectorListener) {
	lm.lock.Lock()
	defer lm.lock.Unlock()

	klog.Infof("add listener %s node %s", listener.id, listener.nodeName)

	_, exists := lm.listenerByNodeID[listener.nodeName]
	if !exists {
		lm.listenerByNodeID[listener.nodeName] = map[string]*SelectorListener{}
	}
	lm.listenerByNodeID[listener.nodeName][listener.id] = listener

	_, exists = lm.listenerByGVR[listener.gvr]
	if !exists {
		lm.listenerByGVR[listener.gvr] = map[string]*SelectorListener{}
	}
	lm.listenerByGVR[listener.gvr][listener.id] = listener
}

func (lm *listenerManager) DeleteListener(listener *SelectorListener) {
	lm.lock.Lock()
	defer lm.lock.Unlock()

	listeners, exists := lm.listenerByNodeID[listener.nodeName]
	if exists {
		delete(listeners, listener.id)
		if len(lm.listenerByNodeID[listener.nodeName]) == 0 {
			delete(lm.listenerByNodeID, listener.nodeName)
		}
	}

	listeners, exists = lm.listenerByGVR[listener.gvr]
	if exists {
		delete(listeners, listener.id)
		if len(lm.listenerByGVR[listener.gvr]) == 0 {
			delete(lm.listenerByGVR, listener.gvr)
		}
	}
}

func (lm *listenerManager) GetListenersForNode(nodeName string) map[string]*SelectorListener {
	lm.lock.RLock()
	defer lm.lock.RUnlock()

	listeners, exists := lm.listenerByNodeID[nodeName]
	if !exists {
		return nil
	}

	return listeners
}

func (lm *listenerManager) GetListenersForGVR(gvr schema.GroupVersionResource) map[string]*SelectorListener {
	lm.lock.RLock()
	defer lm.lock.RUnlock()

	listeners, exists := lm.listenerByGVR[gvr]
	if !exists {
		return nil
	}

	return listeners
}
