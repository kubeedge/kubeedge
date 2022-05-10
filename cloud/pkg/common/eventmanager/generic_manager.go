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

package eventmanager

import (
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// Manager is an interface for managing kubernetes event
type Manager interface {
	Events() chan watch.Event
}

// genericManager is the default implement for Manager.
type genericManager struct {
	events chan watch.Event
}

// Events return the events channel from which the subscriber
// can receive kubernetes resource events.
func (gm *genericManager) Events() chan watch.Event {
	return gm.events
}

// NewGenericManager instantiates a new genericManager object and return it.
func NewGenericManager(eventBuffer int32, si cache.SharedIndexInformer) Manager {
	events := make(chan watch.Event, eventBuffer)
	rh := NewGenericResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &genericManager{events: events}
}
