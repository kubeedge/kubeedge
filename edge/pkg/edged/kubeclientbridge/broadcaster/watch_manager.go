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

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// EventBroadcaster allows registering watchers and broadcasting events to them.
type EventBroadcaster interface {
	// Subscribe returns a watch.Interface and immediately sends all currently buffered events.
	Subscribe() watch.Interface
	// Broadcast sends an event to all subscribers and updates the event buffer.
	Broadcast(event watch.Event)
}

type watchManager struct {
	mu          sync.RWMutex
	subscribers map[*watcher]struct{}
	// eventBuffer tracks the latest event for each object key (e.g. name)
	// to prevent List-Watch race conditions.
	eventBuffer map[string]watch.Event
}

// NewEventBroadcaster creates a new EventBroadcaster.
func NewEventBroadcaster() EventBroadcaster {
	return &watchManager{
		subscribers: make(map[*watcher]struct{}),
		eventBuffer: make(map[string]watch.Event),
	}
}

func (wm *watchManager) Subscribe() watch.Interface {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 100 is standard for kubernetes watch channels
	ch := make(chan watch.Event, 100)
	w := &watcher{
		ch:      ch,
		manager: wm,
	}

	wm.subscribers[w] = struct{}{}
	klog.Infof("watchManager Subscribe called, total subscribers: %d", len(wm.subscribers))

	// Flush buffered events to the new watcher immediately to solve the List-Watch race.
	for _, event := range wm.eventBuffer {
		select {
		case w.ch <- event:
		default:
			klog.Warning("watchManager channel full while subscribing, dropping initial event")
		}
	}

	return w
}

func (wm *watchManager) Broadcast(event watch.Event) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	klog.Infof("watchManager Broadcast called with event Type %v, subscribers count: %d", event.Type, len(wm.subscribers))

	// Update the event buffer
	if event.Object != nil {
		if key, err := cache.MetaNamespaceKeyFunc(event.Object); err == nil {
			if event.Type == watch.Deleted {
				delete(wm.eventBuffer, key)
			} else {
				wm.eventBuffer[key] = event
			}
		} else {
			klog.Errorf("watchManager failed to get key for object: %v", err)
		}
	}

	// Broadcast to all subscribers non-blockingly
	for w := range wm.subscribers {
		select {
		case w.ch <- event:
			klog.Infof("Successfully sent event to a subscriber channel")
		default:
			klog.Warning("watchManager subscriber channel full, dropping event and closing watcher")
			// Remove the watcher so it doesn't receive further events, and close its channel.
			// This signals the Kubernetes Reflector to reconnect (or Relist) to recover state.
			// Because we are already holding wm.mu.Lock(), we can directly delete and close.
			delete(wm.subscribers, w)
			close(w.ch)
		}
	}
}

func (wm *watchManager) removeWatcher(w *watcher) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.subscribers[w]; exists {
		delete(wm.subscribers, w)
		close(w.ch)
	}
}

type watcher struct {
	ch      chan watch.Event
	manager *watchManager
}

func (w *watcher) Stop() {
	w.manager.removeWatcher(w)
}

func (w *watcher) ResultChan() <-chan watch.Event {
	return w.ch
}
