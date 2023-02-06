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
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/application"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter/defaultmaster"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter/endpointresource"
	configv1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// DynamicController use dynamicSharedInformer to dispatch messages
type DynamicController struct {
	enable                       bool
	messageLayer                 messagelayer.MessageLayer
	dynamicSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	applicationCenter            *application.Center
	informersSyncedFuncs         []cache.InformerSynced
}

var (
	_                 core.Module = (*DynamicController)(nil)
	dynamicController *DynamicController
)

func Register(dc *configv1alpha1.DynamicController) {
	config.InitConfigure(dc)
	dynamicController = newDynamicController(dc.Enable)
	core.Register(dynamicController)
}

// Name of controller
func (dctl *DynamicController) Name() string {
	return modules.DynamicControllerModuleName
}

// Group of controller
func (dctl *DynamicController) Group() string {
	return modules.DynamicControllerModuleGroup
}

// Enable of controller
func (dctl *DynamicController) Enable() bool {
	return dctl.enable
}

// Start controller
func (dctl *DynamicController) Start() {
	endpointresource.Register()
	defaultmaster.Register()
	dctl.dynamicSharedInformerFactory.Start(beehiveContext.Done())
	for gvr, cacheSync := range dctl.dynamicSharedInformerFactory.WaitForCacheSync(beehiveContext.Done()) {
		if !cacheSync {
			klog.Exitf("Unable to sync caches for: %s", gvr.String())
		}
	}

	if !cache.WaitForCacheSync(beehiveContext.Done(), dctl.informersSyncedFuncs...) {
		klog.Errorf("unable to sync caches for dynamic controller")
		return
	}

	go dctl.receiveMessage()
}

func newDynamicController(enable bool) *DynamicController {
	var dctl = &DynamicController{
		enable:                       enable,
		messageLayer:                 messagelayer.DynamicControllerMessageLayer(),
		dynamicSharedInformerFactory: informers.GetInformersManager().GetDynamicInformerFactory(),
	}

	dctl.applicationCenter = application.NewApplicationCenter(dctl.dynamicSharedInformerFactory)
	dctl.applicationCenter.ForResource(v1.SchemeGroupVersion.WithResource("nodes"))
	dctl.applicationCenter.ForResource(v1.SchemeGroupVersion.WithResource("services"))

	k8sInformerFactory := informers.GetInformersManager().GetKubeInformerFactory()
	nodesInformer := k8sInformerFactory.Core().V1().Nodes()
	nodesInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			node, isNode := obj.(*v1.Node)
			if !isNode {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					klog.Errorf("Received unexpected object: %v", obj)
					return
				}
				node, ok = deletedState.Obj.(*v1.Node)
				if !ok {
					klog.Errorf("DeletedFinalStateUnknown contained non-Node object: %v", deletedState.Obj)
					return
				}
			}

			dctl.applicationCenter.DeleteListenersForNode(node.Name)
		},
	})

	dctl.informersSyncedFuncs = append(dctl.informersSyncedFuncs, nodesInformer.Informer().HasSynced)

	return dctl
}

func (dctl *DynamicController) receiveMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop dispatchMessage")
			return
		default:
		}
		msg, err := dctl.messageLayer.Receive()
		if err != nil {
			klog.Warningf("Receive message failed, %s", err)
			continue
		}

		klog.V(4).Infof("[DynamicController] receive, msg: %+v", msg)
		dctl.applicationCenter.Process(msg)
	}
}
