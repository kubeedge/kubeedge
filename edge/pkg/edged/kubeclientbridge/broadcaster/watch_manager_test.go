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

package broadcaster

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestWatchManager_SubscribeAndBroadcast(t *testing.T) {
	assert := assert.New(t)
	wm := NewEventBroadcaster()

	// 1. Subscribe watcher A
	watcherA := wm.Subscribe()
	defer watcherA.Stop()

	// 2. Broadcast an event
	rc := &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "kata"}}
	event := watch.Event{Type: watch.Added, Object: rc}
	wm.Broadcast(event)

	// 3. Verify watcher A receives the event
	select {
	case received := <-watcherA.ResultChan():
		assert.Equal(event, received)
	case <-time.After(1 * time.Second):
		t.Fatal("watcher A did not receive event")
	}

	// 4. Subscribe watcher B AFTER the broadcast
	watcherB := wm.Subscribe()
	defer watcherB.Stop()

	// 5. Verify watcher B receives the buffered event immediately
	select {
	case received := <-watcherB.ResultChan():
		assert.Equal(event, received)
	case <-time.After(1 * time.Second):
		t.Fatal("watcher B did not receive buffered event")
	}
}

func TestWatchManager_EventBuffer(t *testing.T) {
	assert := assert.New(t)
	wm := NewEventBroadcaster()

	rc1 := &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "kata1"}}
	rc2 := &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "kata2"}}

	// Broadcast Added
	wm.Broadcast(watch.Event{Type: watch.Added, Object: rc1})
	wm.Broadcast(watch.Event{Type: watch.Added, Object: rc2})

	// Broadcast Modified for kata1
	rc1Mod := &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "kata1"}, Handler: "kata"}
	wm.Broadcast(watch.Event{Type: watch.Modified, Object: rc1Mod})

	// Broadcast Deleted for kata2
	wm.Broadcast(watch.Event{Type: watch.Deleted, Object: rc2})

	// Subscribe
	watcher := wm.Subscribe()
	defer watcher.Stop()

	// Watcher should only receive Modified for kata1 (since Added was overwritten, and kata2 was deleted)
	var received []watch.Event
loop:
	for {
		select {
		case ev := <-watcher.ResultChan():
			received = append(received, ev)
		case <-time.After(100 * time.Millisecond):
			break loop
		}
	}

	assert.Len(received, 1)
	assert.Equal(watch.Modified, received[0].Type)
	assert.Equal(rc1Mod, received[0].Object)
}

func TestWatchManager_Concurrency(t *testing.T) {
	wm := NewEventBroadcaster()
	var wg sync.WaitGroup

	// Start 10 watchers
	watchers := make([]watch.Interface, 10)
	for i := 0; i < 10; i++ {
		watchers[i] = wm.Subscribe()
	}

	// Concurrently broadcast and stop watchers
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			wm.Broadcast(watch.Event{Type: watch.Added, Object: &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "test"}}})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			watchers[i].Stop()
		}
	}()

	wg.Wait()
}

func TestWatchManager_SlowWatcher(t *testing.T) {
	assert := assert.New(t)
	wm := NewEventBroadcaster().(*watchManager)

	watcher := wm.Subscribe().(*watcher)
	defer watcher.Stop()

	rc := &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	event := watch.Event{Type: watch.Added, Object: rc}

	// Fill the watcher channel (capacity is 100)
	for i := 0; i < 100; i++ {
		wm.Broadcast(event)
	}

	// Broadcast one more event, which should trigger the slow watcher removal
	wm.Broadcast(event)

	// Verify watcher is removed from subscribers
	wm.mu.RLock()
	_, exists := wm.subscribers[watcher]
	wm.mu.RUnlock()
	assert.False(exists, "Watcher should be removed from subscribers")

	// Verify channel is closed
	// Drain the channel first to reach the closed state
	for i := 0; i < 100; i++ {
		<-watcher.ResultChan()
	}

	// Next read should yield closed channel
	_, ok := <-watcher.ResultChan()
	assert.False(ok, "Watcher channel should be closed")
}

func TestWatchManager_HealthyWatcherIsolation(t *testing.T) {
	assert := assert.New(t)
	wm := NewEventBroadcaster().(*watchManager)

	watcherA := wm.Subscribe().(*watcher) // Slow watcher
	watcherB := wm.Subscribe().(*watcher) // Healthy watcher
	defer watcherA.Stop()
	defer watcherB.Stop()

	rc := &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	event := watch.Event{Type: watch.Added, Object: rc}

	// Drain B in the background to keep it healthy
	go func() {
		for range watcherB.ResultChan() {
		}
	}()

	// Fill A
	for i := 0; i < 100; i++ {
		wm.Broadcast(event)
	}

	// Broadcast one more event to overflow A
	wm.Broadcast(event)

	// Verify A is removed
	wm.mu.RLock()
	_, existsA := wm.subscribers[watcherA]
	_, existsB := wm.subscribers[watcherB]
	wm.mu.RUnlock()

	assert.False(existsA, "Slow watcher A should be removed")
	assert.True(existsB, "Healthy watcher B should remain active")

	// Verify A's channel is closed (after draining)
	for i := 0; i < 100; i++ {
		<-watcherA.ResultChan()
	}
	_, ok := <-watcherA.ResultChan()
	assert.False(ok, "Slow watcher A channel should be closed")
}

func TestWatchManager_ConcurrentSubscribeBroadcast(t *testing.T) {
	wm := NewEventBroadcaster()
	var wg sync.WaitGroup

	// Concurrently broadcast and subscribe
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			wm.Broadcast(watch.Event{Type: watch.Added, Object: &nodev1.RuntimeClass{ObjectMeta: metav1.ObjectMeta{Name: "test"}}})
		}
	}()

	go func() {
		defer wg.Done()
		watchers := make([]watch.Interface, 10)
		for i := 0; i < 10; i++ {
			watchers[i] = wm.Subscribe()
		}
		for i := 0; i < 10; i++ {
			watchers[i].Stop()
		}
	}()

	wg.Wait()
}

