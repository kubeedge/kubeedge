/*
Copyright 2019 The KubeEdge Authors.

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
package manager

import (
	"github.com/kubeedge/beehive/pkg/common/log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// Manager define the interface of a Manager, configmapManager and podManager implement it
type Manager interface {
	Events() chan watch.Event
}

// CommonResourceEventHandler can be used by configmapManager and podManager
type CommonResourceEventHandler struct {
	events chan watch.Event
}

func (c *CommonResourceEventHandler) obj2Event(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		log.LOGGER.Warnf("unknown type: %T, ignore", obj)
		return
	}
	c.events <- watch.Event{Type: t, Object: eventObj}
}

// OnAdd handle Add event
func (c *CommonResourceEventHandler) OnAdd(obj interface{}) {
	c.obj2Event(watch.Added, obj)
}

// OnUpdate handle Update event
func (c *CommonResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	c.obj2Event(watch.Modified, newObj)
}

// OnDelete handle Delete event
func (c *CommonResourceEventHandler) OnDelete(obj interface{}) {
	c.obj2Event(watch.Deleted, obj)
}

// NewCommonResourceEventHandler create CommonResourceEventHandler used by configmapManager and podManager
func NewCommonResourceEventHandler(events chan watch.Event) *CommonResourceEventHandler {
	return &CommonResourceEventHandler{events: events}
}
