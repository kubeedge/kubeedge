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

package application

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	genericinformers "github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// HandlerCenter is used to prepare corresponding CommonResourceEventHandler for listener
// CommonResourceEventHandler then will send to the listener the objs it is interested in,
// including subsequent changes (watch event)
type HandlerCenter interface {
	AddListener(s *SelectorListener) error
	DeleteListener(s *SelectorListener)
	ForResource(gvr schema.GroupVersionResource) *CommonResourceEventHandler
	GetListenersForNode(nodeName string) map[string]*SelectorListener
}

type handlerCenter struct {
	// handlerLock used to protect the handlers map
	handlerLock                  sync.Mutex
	listenerManager              *listenerManager
	handlers                     map[schema.GroupVersionResource]*CommonResourceEventHandler
	dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	messageLayer                 messagelayer.MessageLayer
}

func NewHandlerCenter(informerFactory dynamicinformer.DynamicSharedInformerFactory) HandlerCenter {
	handlers := make(map[schema.GroupVersionResource]*CommonResourceEventHandler)

	c := handlerCenter{
		listenerManager:              newListenerManager(),
		handlers:                     handlers,
		dynamicSharedInformerFactory: informerFactory,
		messageLayer:                 messagelayer.DynamicControllerMessageLayer(),
	}
	return &c
}

func (c *handlerCenter) ForResource(gvr schema.GroupVersionResource) *CommonResourceEventHandler {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()

	if handler, ok := c.handlers[gvr]; ok {
		return handler
	}

	klog.Infof("[metaserver/HandlerCenter] prepare a new resourceEventHandler(%v)", gvr)

	handler := NewCommonResourceEventHandler(gvr, c.listenerManager, c.messageLayer)
	c.handlers[gvr] = handler

	return handler
}

// AddListener dispatch listeners to corresponding CommonResourceEventHandler according it's gvr
func (c *handlerCenter) AddListener(s *SelectorListener) error {
	return c.ForResource(s.gvr).AddListener(s)
}

func (c *handlerCenter) DeleteListener(s *SelectorListener) {
	c.handlerLock.Lock()
	c.handlers[s.gvr].DeleteListener(s)
	c.handlerLock.Unlock()
}

func (c *handlerCenter) GetListenersForNode(nodeName string) map[string]*SelectorListener {
	return c.listenerManager.GetListenersForNode(nodeName)
}

// CommonResourceEventHandler can be used by configmapManager and podManager
type CommonResourceEventHandler struct {
	events chan watch.Event

	//TODO: num of listeners is proportional to the number of request, need reduce.
	listenerManager *listenerManager
	messageLayer    messagelayer.MessageLayer
	gvr             schema.GroupVersionResource
	informer        *genericinformers.InformerPair
}

func NewCommonResourceEventHandler(
	gvr schema.GroupVersionResource,
	listenerManager *listenerManager,
	layer messagelayer.MessageLayer) *CommonResourceEventHandler {
	handler := &CommonResourceEventHandler{
		listenerManager: listenerManager,
		events:          make(chan watch.Event, 100),
		messageLayer:    layer,
		gvr:             gvr,
	}

	klog.Infof("[metaserver/resourceEventHandler] handler(%v) init, prepare informer...", gvr)
	informerPair, err := genericinformers.GetInformersManager().GetInformerPair(gvr)
	if err != nil {
		klog.Exitf("get informer for %s err: %v", gvr.String(), err)
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
	klog.Infof("[metaserver/resourceEventHandler] handler(%v) init successfully, start to dispatch events to it's listeners", gvr)
	go handler.dispatchEvents()
	return handler
}

func (c *CommonResourceEventHandler) objToEvent(t watch.EventType, obj interface{}) {
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
	c.events <- watch.Event{Type: t, Object: eventObj}
}

func (c *CommonResourceEventHandler) AddListener(s *SelectorListener) error {
	// filter s.selector.field when sendAllObjects
	ret, err := c.informer.Lister.List(s.selector.Label)
	if err != nil {
		return fmt.Errorf("Failed to list: %v", err)
	}
	s.sendAllObjects(ret, c)

	c.listenerManager.AddListener(s)

	return nil
}

func (c *CommonResourceEventHandler) DeleteListener(s *SelectorListener) {
	c.listenerManager.DeleteListener(s)
}

func (c *CommonResourceEventHandler) dispatchEvents() {
	for event := range c.events {
		klog.V(4).Infof("[metaserver/resourceEventHandler] handler(%v), send obj event{%v/%v} to listeners", c.gvr, event.Type, event.Object.GetObjectKind().GroupVersionKind().String())
		for _, listener := range c.listenerManager.GetListenersForGVR(c.gvr) {
			listener.sendObj(event, c.messageLayer)
		}
	}
	klog.Warningf("[metaserver/resourceEventHandler] handler(%v) stopped!", c.gvr.String())
}
