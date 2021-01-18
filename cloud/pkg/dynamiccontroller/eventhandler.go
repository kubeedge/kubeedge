/*
Copyright 2021 The KubeEdge Authors.

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

package dynamiccontroller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
)

// CommonResourceEventHandler can be used by configmapManager and podManager
type CommonResourceEventHandler struct {
	events       chan watch.Event
	listeners    map[string]nodeFilter
	messageLayer messagelayer.MessageLayer
	resourceType string
}

type nodeFilter struct {
	nodeName string
	filter   func(l labels.Set, f fields.Set) bool
}

func (c *CommonResourceEventHandler) objToEvent(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", obj)
		return
	}
	c.events <- watch.Event{Type: t, Object: eventObj}
}

func (c *CommonResourceEventHandler) addProcessListener(nodefilter nodeFilter) {
	c.listeners[nodefilter.nodeName] = nodefilter
}

func (c *CommonResourceEventHandler) removeProcessListener(nodefilter nodeFilter) {
	delete(c.listeners, nodefilter.nodeName)
}

func (c *CommonResourceEventHandler) dispatchEvents() {
	for {
		select {
		case event, ok := <-c.events:
			if !ok {
				return
			}

			for _, listener := range c.listeners {
				// todo: add filter here
				listener.sendObj(event, c.resourceType, c.messageLayer)
			}
		}
	}
}

func (nf *nodeFilter) sendAllObjects(rets []runtime.Object, resourceType string, messageLayer messagelayer.MessageLayer) {
	for _, ret := range rets {
		event := watch.Event{
			Type:   watch.Added,
			Object: ret,
		}
		nf.sendObj(event, resourceType, messageLayer)
	}
}

func (nf *nodeFilter) sendObj(event watch.Event, resourceType string, messageLayer messagelayer.MessageLayer) {
	msg := model.NewMessage("")
	object, err := meta.Accessor(event.Object.(runtime.Object))
	if err != nil {
		klog.Error()
		return
	}

	msg.SetResourceVersion(object.GetResourceVersion())
	resource, err := messagelayer.BuildResource(nf.nodeName, object.GetNamespace(), resourceType, object.GetName())
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return
	}
	msg.Content = event.Object
	switch event.Type {
	case watch.Added:
		msg.BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
	case watch.Modified:
		msg.BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
	case watch.Deleted:
		msg.BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
	default:
		klog.Warningf("event type: %s unsupported", event.Type)
	}
	if err := messageLayer.Send(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
	} else {
		klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
	}
}
