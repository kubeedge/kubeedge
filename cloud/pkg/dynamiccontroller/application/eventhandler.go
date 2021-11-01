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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/messagelayer"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// HandlerCenter is used to prepare corresponding CommonResourceEventHandler for listener
// CommonResourceEventHandler then will send to the listener the objs it is interested in, including subsequent changes (watch event)
type HandlerCenter interface {
	AddListener(s *SelectorListener) error
	DeleteListener(s *SelectorListener)
	ForResource(gvr schema.GroupVersionResource) *CommonResourceEventHandler
}

type handlerCenter struct {
	handlers                     map[schema.GroupVersionResource]*CommonResourceEventHandler
	dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	messageLayer                 messagelayer.MessageLayer
}

func NewHandlerCenter(informerFactory dynamicinformer.DynamicSharedInformerFactory) HandlerCenter {
	handlers := make(map[schema.GroupVersionResource]*CommonResourceEventHandler)

	c := handlerCenter{
		handlers:                     handlers,
		dynamicSharedInformerFactory: informerFactory,
		messageLayer:                 messagelayer.NewContextMessageLayer(),
	}
	return &c
}

func (c *handlerCenter) ForResource(gvr schema.GroupVersionResource) *CommonResourceEventHandler {
	var handler *CommonResourceEventHandler
	if store, ok := c.handlers[gvr]; ok {
		handler = store
	} else {
		klog.Infof("[metaserver/HandlerCenter] prepare a new resourceEventHandler(%v)", gvr)
		handler = NewCommonResourceEventHandler(gvr, c.dynamicSharedInformerFactory, c.messageLayer)
		c.handlers[gvr] = handler
	}
	return handler
}

// dispatch listeners to corresponding CommonResourceEventHandler according it's gvr
func (c *handlerCenter) AddListener(s *SelectorListener) error {
	return c.ForResource(s.gvr).AddListener(s)
}

func (c *handlerCenter) DeleteListener(s *SelectorListener) {
	c.handlers[s.gvr].DeleteListener(s)
}

// CommonResourceEventHandler can be used by configmapManager and podManager
type CommonResourceEventHandler struct {
	events chan watch.Event
	//TODO: num of listeners is proportional to the number of request, need reduce.
	listeners    map[string]*SelectorListener
	messageLayer messagelayer.MessageLayer
	gvr          schema.GroupVersionResource
	informer     informers.GenericInformer
}

func NewCommonResourceEventHandler(gvr schema.GroupVersionResource, informerFactory dynamicinformer.DynamicSharedInformerFactory, layer messagelayer.MessageLayer) *CommonResourceEventHandler {
	handler := &CommonResourceEventHandler{
		events:       make(chan watch.Event, 100),
		listeners:    make(map[string]*SelectorListener),
		messageLayer: layer,
		gvr:          gvr,
	}

	klog.Infof("[metaserver/resourceEventHandler] handler(%v) init, prepare informer...", gvr)
	informer := informerFactory.ForResource(gvr)
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	informerFactory.Start(beehiveContext.Done())
	klog.Infof("[metaserver/resourceEventHandler] handler(%v) init, wait for informer starting...", gvr)
	for gvr, cacheSync := range informerFactory.WaitForCacheSync(beehiveContext.Done()) {
		if !cacheSync {
			klog.Exitf("unable to sync caches for: %s", gvr.String())
		}
	}
	handler.informer = informer
	klog.Infof("[metaserver/resourceEventHandler] handler(%v) init successfully, start to dispatch events to it's listeners", gvr)
	go handler.dispatchEvents()
	return handler
}

func (c *CommonResourceEventHandler) objToEvent(t watch.EventType, obj interface{}) {
	eventObj, ok := obj.(runtime.Object)
	if !ok {
		klog.Warningf("unknown type: %T, ignore", obj)
		return
	}
	// All obj from client has been removed the information of apiversion/kind called MetaType,
	// which is fatal to decode the obj as unstructured.Unstructure or unstructured.UnstructureList at edge.
	err := util.SetMetaType(eventObj)
	if err != nil {
		klog.Warningf("failed to set metatype :%v", err)
	}
	c.events <- watch.Event{Type: t, Object: eventObj}
}

func (c *CommonResourceEventHandler) AddListener(s *SelectorListener) error {
	// filter s.selector.field when sendAllObjects
	ret, err := c.informer.Lister().List(s.selector.Label)
	if err != nil {
		return fmt.Errorf("failed to list: %v", err)
	}
	s.sendAllObjects(ret, c.messageLayer)
	c.listeners[s.id] = s
	return nil
}

func (c *CommonResourceEventHandler) DeleteListener(s *SelectorListener) {
	delete(c.listeners, s.id)
}

func (c *CommonResourceEventHandler) dispatchEvents() {
	for event := range c.events {
		klog.V(4).Infof("[metaserver/resourceEventHandler] handler(%v), send obj event{%v/%v} to listeners", c.gvr, event.Type, event.Object.GetObjectKind().GroupVersionKind().String())
		for _, listener := range c.listeners {
			listener.sendObj(event, c.messageLayer)
		}
	}
	klog.Warningf("[metaserver/resourceEventHandler] handler(%v) stopped!", c.gvr.String())
}
