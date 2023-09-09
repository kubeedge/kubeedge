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
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type Manager interface {
	AddWatcher(watcher *CacheWatcher)
	DeleteWatcher(watcher *CacheWatcher)
	GetWatchersForNode(NodeName string) map[string]*CacheWatcher
	GetWatchersForGVR(gvr schema.GroupVersionResource) map[string]*CacheWatcher
}

// watcherManager stores and manages access to watchers, maintaining the mappings
// between nodes and watchers and also the mappings between gvr and watchers.
type watcherManager struct {
	// Protects all internal maps.
	lock sync.RWMutex

	// first key: nodeID
	// second key: listenerID
	watcherByNodeID map[string]map[string]*CacheWatcher

	// first key: GVR
	// second key: listenerID
	watcherByGVR map[schema.GroupVersionResource]map[string]*CacheWatcher
}

func newWatcherManager() *watcherManager {
	return &watcherManager{
		watcherByNodeID: make(map[string]map[string]*CacheWatcher),
		watcherByGVR:    make(map[schema.GroupVersionResource]map[string]*CacheWatcher),
	}
}

func (wm *watcherManager) AddWatcher(watcher *CacheWatcher) {
	wm.lock.Lock()
	defer wm.lock.Unlock()

	klog.Infof("add watcher %s node %s", watcher.WatcherID, watcher.NodeName)

	_, exists := wm.watcherByNodeID[watcher.NodeName]
	if !exists {
		wm.watcherByNodeID[watcher.NodeName] = map[string]*CacheWatcher{}
	}
	wm.watcherByNodeID[watcher.NodeName][watcher.WatcherID] = watcher

	_, exists = wm.watcherByGVR[watcher.gvr]
	if !exists {
		wm.watcherByGVR[watcher.gvr] = map[string]*CacheWatcher{}
	}
	wm.watcherByGVR[watcher.gvr][watcher.WatcherID] = watcher
}

func (wm *watcherManager) DeleteWatcher(watcher *CacheWatcher) {
	wm.lock.Lock()
	defer wm.lock.Unlock()

	watchers, exists := wm.watcherByNodeID[watcher.NodeName]
	if exists {
		delete(watchers, watcher.WatcherID)
		if len(wm.watcherByNodeID[watcher.NodeName]) == 0 {
			delete(wm.watcherByNodeID, watcher.NodeName)
		}
	}

	watchers, exists = wm.watcherByGVR[watcher.gvr]
	if exists {
		delete(watchers, watcher.WatcherID)
		if len(wm.watcherByGVR[watcher.gvr]) == 0 {
			delete(wm.watcherByGVR, watcher.gvr)
		}
	}

	watcher.stop()
}

func (wm *watcherManager) GetWatchersForNode(NodeName string) map[string]*CacheWatcher {
	wm.lock.RLock()
	defer wm.lock.RUnlock()

	watchers, exists := wm.watcherByNodeID[NodeName]
	if !exists {
		return nil
	}

	return watchers
}

func (wm *watcherManager) GetWatchersForGVR(gvr schema.GroupVersionResource) map[string]*CacheWatcher {
	wm.lock.RLock()
	defer wm.lock.RUnlock()

	watchers, exists := wm.watcherByGVR[gvr]
	if !exists {
		return nil
	}

	return watchers
}
