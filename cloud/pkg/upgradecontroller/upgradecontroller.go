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

package upgradecontroller

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/upgradecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/upgradecontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// UpgradeController is controller for processing Upgrade Edge Nodes from cloud side
type UpgradeController struct {
	downstream *controller.DownstreamController
	upstream   *controller.UpstreamController
	enable     bool
}

var _ core.Module = (*UpgradeController)(nil)

func newUpgradeController(enable bool) *UpgradeController {
	if !enable {
		return &UpgradeController{enable: enable}
	}
	downstream, err := controller.NewDownstreamController(informers.GetInformersManager().GetCRDInformerFactory())
	if err != nil {
		klog.Exitf("New downstream controller failed with error: %s", err)
	}
	upstream, err := controller.NewUpstreamController(downstream)
	if err != nil {
		klog.Exitf("new upstream controller failed with error: %s", err)
	}
	return &UpgradeController{
		downstream: downstream,
		upstream:   upstream,
		enable:     enable,
	}
}

func Register(dc *v1alpha1.UpgradeController) {
	config.InitConfigure(dc)
	core.Register(newUpgradeController(dc.Enable))
}

// Name of controller
func (uc *UpgradeController) Name() string {
	return modules.UpgradeControllerModuleName
}

// Group of controller
func (uc *UpgradeController) Group() string {
	return modules.UpgradeControllerModuleGroup
}

// Enable indicates whether enable this module
func (uc *UpgradeController) Enable() bool {
	return uc.enable
}

// Start controller
func (uc *UpgradeController) Start() {
	if err := uc.downstream.Start(); err != nil {
		klog.Exitf("start upgrade controller downstream failed with error: %s", err)
	}
	// wait for downstream controller to start and load Upgrades
	// TODO think about sync
	time.Sleep(1 * time.Second)
	if err := uc.upstream.Start(); err != nil {
		klog.Exitf("start upgrade controller upstream failed with error: %s", err)
	}
}
