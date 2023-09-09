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
	"fmt"
	"sort"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"
)

// watchCacheEvent is a single "watch event" that is send to users of
// watchCache. Additionally to a typical "watch.Event" it contains
// the previous value of the object to enable proper filtering in the
// upper layers.
type watchCacheEvent struct {
	event           watch.Event
	ResourceVersion uint64
}

func newWatchCache(capacity int) *watchCache {
	return &watchCache{
		startIndex: 0,
		endIndex:   0,
		capacity:   capacity,
		cache:      make([]watchCacheEvent, capacity),
	}
}

// watchCache is a "sliding window" (with a limited capacity) of objects
// observed from a watch.
type watchCache struct {
	sync.RWMutex
	// Maximum size of history window.
	capacity int
	// cache is used a cyclic buffer - its first element (with the smallest
	// resourceVersion) is defined by startIndex, its last element is defined
	// by endIndex (if cache is full it will be startIndex + capacity).
	// Both startIndex and endIndex can be greater than buffer capacity -
	// you should always apply modulo capacity to get an index in cache array.
	cache      []watchCacheEvent
	startIndex int
	endIndex   int
}

// isCacheFullLocked used to judge whether watchCacheEvent is full.
// Assumes that lock is already held for write.
func (w *watchCache) isCacheFullLocked() bool {
	return w.endIndex == w.startIndex+w.capacity
}

func (w *watchCache) Add(wce watchCacheEvent) {
	w.Lock()
	defer w.Unlock()

	if w.isCacheFullLocked() {
		// Cache is full - remove the oldest element.
		w.startIndex++
	}

	w.cache[w.endIndex%w.capacity] = wce
	w.endIndex++
}

func (w *watchCache) GetAllEventsSince(resourceVersion uint64) ([]watchCacheEvent, error) {
	w.Lock()
	defer w.Unlock()

	size := w.endIndex - w.startIndex
	if size <= 0 {
		return nil, nil
	}

	// the oldest watch event we can deliver is the first one in the buffer.
	oldest := w.cache[w.startIndex%w.capacity].ResourceVersion

	if resourceVersion < oldest-1 {
		return nil, errors.NewResourceExpired(fmt.Sprintf("too old resource version: %d (%d)", resourceVersion, oldest-1))
	}

	// Binary search the smallest index at which resourceVersion is greater than the given one.
	f := func(i int) bool {
		return w.cache[(w.startIndex+i)%w.capacity].ResourceVersion > resourceVersion
	}
	first := sort.Search(size, f)
	result := make([]watchCacheEvent, size-first)
	for i := 0; i < size-first; i++ {
		result[i] = w.cache[(w.startIndex+first+i)%w.capacity]
	}

	return result, nil
}
