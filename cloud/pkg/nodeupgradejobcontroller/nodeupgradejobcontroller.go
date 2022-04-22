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

package nodeupgradejobcontroller

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// NodeUpgradeJobController is controller for processing upgrading edge node from cloud
type NodeUpgradeJobController struct {
	downstream *controller.DownstreamController
	upstream   *controller.UpstreamController
	enable     bool
}

var _ core.Module = (*NodeUpgradeJobController)(nil)

func newNodeUpgradeJobController(enable bool) *NodeUpgradeJobController {
	if !enable {
		return &NodeUpgradeJobController{enable: enable}
	}
	downstream, err := controller.NewDownstreamController(informers.GetInformersManager().GetCRDInformerFactory())
	if err != nil {
		klog.Exitf("New NodeUpgradeJob Controller downstream failed with error: %s", err)
	}
	upstream, err := controller.NewUpstreamController(downstream)
	if err != nil {
		klog.Exitf("New NodeUpgradeJob Controller upstream failed with error: %s", err)
	}
	return &NodeUpgradeJobController{
		downstream: downstream,
		upstream:   upstream,
		enable:     enable,
	}
}

func Register(dc *v1alpha1.NodeUpgradeJobController) {
	config.InitConfigure(dc)
	core.Register(newNodeUpgradeJobController(dc.Enable))
}

// Name of controller
func (uc *NodeUpgradeJobController) Name() string {
	return modules.NodeUpgradeJobControllerModuleName
}

// Group of controller
func (uc *NodeUpgradeJobController) Group() string {
	return modules.NodeUpgradeJobControllerModuleGroup
}

// Enable indicates whether enable this module
func (uc *NodeUpgradeJobController) Enable() bool {
	return uc.enable
}

// Start controller
func (uc *NodeUpgradeJobController) Start() {
	if err := uc.downstream.Start(); err != nil {
		klog.Exitf("start NodeUpgradeJob controller downstream failed with error: %s", err)
	}
	// wait for downstream controller to start and load NodeUpgradeJob
	// TODO think about sync
	time.Sleep(1 * time.Second)
	if err := uc.upstream.Start(); err != nil {
		klog.Exitf("start NodeUpgradeJob controller upstream failed with error: %s", err)
	}
}
