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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/messagelayer"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var gvrs = []schema.GroupVersionResource{
	{"", "v1", "pod"},
	{"", "v1", "configmap"},
}

// DynamicController use dynamicSharedInformer to dispatch messages
type DynamicController struct {
	enable                       bool
	messageLayer                 messagelayer.MessageLayer
	dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	resourceToNode               map[schema.GroupVersionResource][]nodeFilter
	eventHandler                 map[schema.GroupVersionResource]CommonResourceEventHandler
}

func Register(dc *configv1alpha1.DynamicController) {
	config.InitConfigure(dc)
	core.Register(newDynamicController(dc.Enable))
}

// Name of controller
func (dctl *DynamicController) Name() string {
	return modules.DynamicControllerModuleName
}

// Group of controller
func (dctl *DynamicController) Group() string {
	return modules.DynamicControllerModuleGroup
}

// Group of controller
func (dctl *DynamicController) Enable() bool {
	return dctl.enable
}

// Start controller
func (dctl *DynamicController) Start() {
	for gvr, cacheSync := range dctl.dynamicSharedInformerFactory.WaitForCacheSync(beehiveContext.Done()) {
		if !cacheSync {
			klog.Fatalf("unable to sync caches for: %s", gvr.String())
		}
	}

	for _, gvr := range gvrs {
		go dctl.eventHandler[gvr].dispatchEvents()
	}

	go dctl.receiveMessage()
}

func newDynamicController(enable bool) *DynamicController {
	var dctl = &DynamicController{
		enable:                       enable,
		messageLayer:                 messagelayer.NewContextMessageLayer(),
		dynamicSharedInformerFactory: informers.GetInformersManager().GetDynamicSharedInformerFactory(),
		resourceToNode:               make(map[schema.GroupVersionResource][]nodeFilter),
		eventHandler:                 make(map[schema.GroupVersionResource]CommonResourceEventHandler),
	}

	for _, gvr := range gvrs {
		// Retrieve a "GroupVersionResource" type that we need when generating our informer from our dynamic factory
		//gvr, _ := schema.ParseResourceArg("deployments.v1.apps")
		// Finally, create our informer for deployments!
		dctl.dynamicSharedInformerFactory.ForResource(gvr)

		dctl.eventHandler[gvr] = CommonResourceEventHandler{
			events:       make(chan watch.Event, 100),
			listeners:    make(map[string]nodeFilter),
			messageLayer: dctl.messageLayer,
			resourceType: gvr.Resource,
		}

		dctl.dynamicSharedInformerFactory.ForResource(gvr).Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				dctl.eventHandler[gvr].objToEvent(watch.Added, obj)
			},
			UpdateFunc: func(oldObj, obj interface{}) {
				dctl.eventHandler[gvr].objToEvent(watch.Modified, obj)
			},
			DeleteFunc: func(obj interface{}) {
				dctl.eventHandler[gvr].objToEvent(watch.Deleted, obj)
			},
		})

		dctl.resourceToNode[gvr] = []nodeFilter{}
	}

	return dctl
}

func (dctl *DynamicController) receiveMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop dispatchMessage")
			return
		default:
		}
		_, err := dctl.messageLayer.Receive()
		if err != nil {
			klog.Warningf("receive message failed, %s", err)
			continue
		}

		// 动态调整map 添加
		gvr := schema.GroupVersionResource{}
		nodefilter := nodeFilter{}
		create := true
		if create {
			if !isNodeFilterExist(dctl.resourceToNode[gvr], nodefilter) {
				dctl.resourceToNode[gvr] = append(dctl.resourceToNode[gvr], nodefilter)

				dctl.eventHandler[gvr].addProcessListener(nodefilter)
				rets, err := dctl.dynamicSharedInformerFactory.ForResource(gvr).Lister().List(nodefilter.filter)
				if err != nil {

				}
				nodefilter.sendAllObjects(rets, gvr.Resource, dctl.eventHandler[gvr].messageLayer)
			}
		}

		// 动态调整 删除
		delete := true
		if delete {
			if isNodeFilterExist(dctl.resourceToNode[gvr], nodefilter) {
				dctl.eventHandler[gvr].removeProcessListener(nodefilter)
			}
		}

	}
}

func isNodeFilterExist(nodefilters []nodeFilter, nodefilter nodeFilter) bool {
	for _, nf := range nodefilters {
		if reflect.DeepEqual(nf, nodefilter) {
			return true
		}
	}
	return false
}
