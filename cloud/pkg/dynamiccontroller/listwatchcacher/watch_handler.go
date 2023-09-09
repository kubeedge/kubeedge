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

package listwatchcacher

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/etcd3"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	genericinformers "github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

const (
	// defaultCapacity is a default value for event cache capacity.
	defaultCapacity = 1024
)

// WatchHandler is responsible for serving WATCH requests for a given
// resource from its internal cache and updating its cache in the background
// based on the underlying informer contents.
type WatchHandler struct {
	// Incoming events that should be dispatched to watchers.
	incoming chan watchCacheEvent

	// TODO: num of watchers is proportional to the number of request, need reduce.
	// watcherManager stores and manages access to watchers
	watcherManager *watcherManager

	// messageLayer is used to send downstream event messages.
	messageLayer messagelayer.MessageLayer

	// gvr defines the resource that the current handler is dealing with.
	gvr schema.GroupVersionResource

	// the resource informer
	informer *genericinformers.InformerPair

	// "sliding window" of recent changes of objects and the current state.
	watchCache *watchCache

	// Versioner is used to handle resource versions.
	versioner storage.Versioner

	// stopCh is used to shut down the handler.
	stopCh chan struct{}
}

func newWatchHandler(gvr schema.GroupVersionResource, watcherManager *watcherManager) (*WatchHandler, error) {
	handler := &WatchHandler{
		gvr:            gvr,
		watcherManager: watcherManager,
		stopCh:         make(chan struct{}),
		versioner:      etcd3.APIObjectVersioner{},
		watchCache:     newWatchCache(defaultCapacity),
		incoming:       make(chan watchCacheEvent, defaultCapacity),
		messageLayer:   messagelayer.DynamicControllerMessageLayer(),
	}

	klog.Infof("[watchHandler] handler(%v) init, prepare informer...", gvr)
	informerPair, err := genericinformers.GetInformersManager().GetInformerPair(gvr)
	if err != nil {
		return nil, fmt.Errorf("get informer for %s err: %v", gvr.String(), err)
	}

	informerPair.Informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handler.objToEvent(watch.Added, obj)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			handler.objToEvent(watch.Modified, obj)
		},
		DeleteFunc: func(obj interface{}) {
			handler.objToEvent(watch.Deleted, obj)
		},
	})

	handler.informer = informerPair
	klog.Infof("[watchHandler] handler(%v) init successfully", gvr)

	go handler.dispatchEvents()

	return handler, nil
}

func (wh *WatchHandler) objToEvent(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		klog.Warningf("Unknown type: %T, ignore", obj)
		return
	}

	// All obj from client has been removed the information of apiversion/kind called MetaType,
	// which is fatal to decode the obj as unstructured.Unstructure or unstructured.UnstructureList at edge.
	err := util.SetMetaType(eventObj)
	if err != nil {
		klog.Warningf("Failed to set metatype :%v", err)
	}

	rv, err := wh.versioner.ObjectResourceVersion(eventObj)
	if err != nil {
		klog.Errorf("failed to get object resource version: %v", err)
		return
	}

	wce := watchCacheEvent{
		event:           watch.Event{Type: t, Object: eventObj},
		ResourceVersion: rv,
	}

	// add event to watchCache
	wh.watchCache.Add(wce)

	wh.processEvent(wce)
}

func (wh *WatchHandler) processEvent(event watchCacheEvent) {
	wh.incoming <- event
}

func (wh *WatchHandler) dispatchEvents() {
	klog.Infof("WatchHandler() start to dispatch events to watchers", wh.gvr.String())
	for {
		select {
		case event, ok := <-wh.incoming:
			if !ok {
				return
			}

			for _, watcher := range wh.watcherManager.GetWatchersForGVR(wh.gvr) {
				watcher.add(event)
			}

		case <-wh.stopCh:
			klog.Warningf("[WatchHandler] handler(%v) stopped!", wh.gvr.String())
			return
		}
	}
}

func (wh *WatchHandler) Watch(app *metaserver.Application) error {
	option := new(metav1.ListOptions)
	if err := app.OptionTo(option); err != nil {
		return err
	}

	gvr, namespace, _ := metaserver.ParseKey(app.Key)
	selector := NewSelector(option.LabelSelector, option.FieldSelector)
	if namespace != "" {
		selector.Field = fields.AndSelectors(selector.Field, fields.OneTermEqualSelector("metadata.namespace", namespace))
	}

	watchRV, err := wh.versioner.ParseResourceVersion(option.ResourceVersion)
	if err != nil {
		return err
	}

	initEvents, err := wh.getInitEvents(watchRV, namespace, selector)

	watcher := NewCacheWatcher(app.ID, app.Nodename, gvr, selector)
	go watcher.processEvents(initEvents, watchRV)

	wh.watcherManager.AddWatcher(watcher)

	return nil
}

func (wh *WatchHandler) getInitEvents(watchRV uint64, namespace string, selector LabelFieldSelector) ([]watchCacheEvent, error) {
	if watchRV != 0 {
		return wh.watchCache.GetAllEventsSince(watchRV)
	}

	var err error
	var ret []runtime.Object

	if len(namespace) != 0 {
		ret, err = wh.informer.Lister.ByNamespace(namespace).List(selector.Label)
	} else {
		ret, err = wh.informer.Lister.List(selector.Label)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list for %s: %v", selector.Label.String(), err)
	}

	initEvents := make([]watchCacheEvent, 0, len(ret))
	for _, obj := range ret {
		initEvents = append(initEvents, watchCacheEvent{
			event: watch.Event{Type: watch.Added, Object: obj},
		})
	}

	return initEvents, nil
}
