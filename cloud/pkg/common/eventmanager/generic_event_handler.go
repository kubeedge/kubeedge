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

package eventmanager

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// genericResourceEventHandler is a generic Event Handler for kubernetes resource
type genericResourceEventHandler struct {
	events chan watch.Event
}

func (c *genericResourceEventHandler) obj2Event(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", obj)
		return
	}
	// All obj from client has been removed the information of apiversion/kind called MetaType,
	// it is fatal to decode the obj as unstructured.Unstructure or unstructured.UnstructureList at edge.
	err := util.SetMetaType(eventObj)
	if err != nil {
		klog.Warningf("failed to set meta type :%v", err)
	}

	c.events <- watch.Event{Type: t, Object: eventObj}
}

// OnAdd handle Add event
func (c *genericResourceEventHandler) OnAdd(obj interface{}) {
	c.obj2Event(watch.Added, obj)
}

// OnUpdate handle Update event
func (c *genericResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	c.obj2Event(watch.Modified, newObj)
}

// OnDelete handle Delete event
func (c *genericResourceEventHandler) OnDelete(obj interface{}) {
	c.obj2Event(watch.Deleted, obj)
}

// NewGenericResourceEventHandler return genericResourceEventHandler
func NewGenericResourceEventHandler(events chan watch.Event) cache.ResourceEventHandler {
	return &genericResourceEventHandler{events: events}
}
