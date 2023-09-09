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
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
	"github.com/kubeedge/kubeedge/cloud/pkg/edgecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

const initProcessThreshold = 500 * time.Millisecond

// CacheWatcher has a one-to-one relationship with watch requests. For each
// watch request from the edge side, a CacheWatcher will be created in memory
type CacheWatcher struct {
	input chan watchCacheEvent

	// gvr unambiguously identifies the resource for the watcher
	gvr schema.GroupVersionResource

	// The unique WatcherID of the watcher, which is the same as the watch on the edge
	// side, it is used to synchronize watch request between the cloud and edge
	WatcherID string

	// NodeName indicates which edge node the watcher request comes from
	NodeName string

	// e.g. labels and fields(metadata.namespace metadata.name spec.nodename)
	selector LabelFieldSelector

	// messageLayer is used to send watch events
	messageLayer messagelayer.MessageLayer

	// stopCh is used to shut down the watcher.
	stopCh chan struct{}

	// The stopOnce is used to ensure that close is only executed once
	stopOnce sync.Once
}

func NewCacheWatcher(watcherID, nodeName string, gvr schema.GroupVersionResource, selector LabelFieldSelector) *CacheWatcher {
	listener := &CacheWatcher{
		gvr:          gvr,
		stopCh:       make(chan struct{}),
		WatcherID:    watcherID,
		NodeName:     nodeName,
		selector:     selector,
		input:        make(chan watchCacheEvent, 1024),
		messageLayer: messagelayer.DynamicControllerMessageLayer(),
	}

	return listener
}

func (c *CacheWatcher) add(event watchCacheEvent) {
	c.input <- event
}

func (c *CacheWatcher) stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

func (c *CacheWatcher) processEvents(initEvents []watchCacheEvent, resourceVersion uint64) {
	startTime := time.Now()
	for _, event := range initEvents {
		c.sendWatchCacheEvent(event)
	}

	if len(initEvents) > 0 {
		// With some events already sent, update resourceVersion
		// so that events that were buffered and not yet processed
		// won't be delivered to this watcher second time causing
		// going back in time.
		resourceVersion = initEvents[len(initEvents)-1].ResourceVersion
	}
	processingTime := time.Since(startTime)
	if processingTime > initProcessThreshold {
		klog.V(2).Infof("processing %d initEvents of %s (%s) took %v", len(initEvents), c.gvr.String(), c.WatcherID, processingTime)
	}

	c.process(resourceVersion)
}

func (c *CacheWatcher) process(resourceVersion uint64) {
	for {
		select {
		case event, ok := <-c.input:
			if !ok {
				return
			}

			// only send events newer than resourceVersion
			if event.ResourceVersion > resourceVersion {
				c.sendWatchCacheEvent(event)
			}

		case <-c.stopCh:
			klog.Infof("the watcher(ID=%s, gvr=%s) is closed", c.WatcherID, c.gvr.String())
			return
		}
	}
}

func (c *CacheWatcher) sendWatchCacheEvent(wce watchCacheEvent) {
	event := wce.event
	accessor, err := meta.Accessor(event.Object)
	if err != nil {
		klog.Errorf("Accessor object %v err: %v", event.Object, err)
		return
	}

	klog.V(4).Infof("[CacheWatcher] watcher(%v) is sending obj %v", c.WatcherID, accessor.GetName())
	// do not send obj if obj does not match listener's selector
	if !c.selector.MatchObj(event.Object) {
		return
	}

	// filter message
	filterEvent := *(event.DeepCopy())
	filter.MessageFilter(filterEvent.Object, c.NodeName)

	namespace := accessor.GetNamespace()
	if namespace == "" {
		namespace = v2.NullNamespace
	}
	kind := util.UnsafeResourceToKind(c.gvr.Resource)
	resourceType := strings.ToLower(kind)
	resource, err := messagelayer.BuildResource(c.NodeName, namespace, resourceType, accessor.GetName())
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return
	}

	var operation string
	switch filterEvent.Type {
	case watch.Added:
		operation = model.InsertOperation
	case watch.Modified:
		operation = model.UpdateOperation
	case watch.Deleted:
		operation = model.DeleteOperation
	default:
		klog.Warningf("event type: %s unsupported", filterEvent.Type)
		return
	}

	msg := model.NewMessage("").
		SetResourceVersion(accessor.GetResourceVersion()).
		BuildRouter(modules.DynamicControllerModuleName, constants.GroupResource, resource, operation).
		FillBody(filterEvent.Object)

	if err := c.messageLayer.Send(*msg); err != nil {
		klog.Warningf("send message failed with error: %s, operation: %s, resource: %s", err, msg.GetOperation(), msg.GetResource())
	} else {
		klog.V(4).Infof("send message successfully, operation: %s, resource: %s", msg.GetOperation(), msg.GetResource())
	}
}
