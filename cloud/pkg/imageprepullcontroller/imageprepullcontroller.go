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

package imageprepullcontroller

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/imageprepullcontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/imageprepullcontroller/controller"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

// ImagePrePullController is controller for processing prepull images on edge nodes
type ImagePrePullController struct {
	downstream *controller.DownstreamController
	upstream   *controller.UpstreamController
	enable     bool
}

var _ core.Module = (*ImagePrePullController)(nil)

func newImagePrePullController(enable bool) *ImagePrePullController {
	if !enable {
		return &ImagePrePullController{enable: enable}
	}
	downstream, err := controller.NewDownstreamController(informers.GetInformersManager().GetKubeEdgeInformerFactory())
	if err != nil {
		klog.Exitf("New ImagePrePull Controller downstream failed with error: %s", err)
	}
	upstream, err := controller.NewUpstreamController(downstream)
	if err != nil {
		klog.Exitf("New ImagePrePull Controller upstream failed with error: %s", err)
	}
	return &ImagePrePullController{
		downstream: downstream,
		upstream:   upstream,
		enable:     enable,
	}
}

func Register(dc *v1alpha1.ImagePrePullController) {
	config.InitConfigure(dc)
	core.Register(newImagePrePullController(dc.Enable))
}

// Name of controller
func (uc *ImagePrePullController) Name() string {
	return modules.ImagePrePullControllerModuleName
}

// Group of controller
func (uc *ImagePrePullController) Group() string {
	return modules.ImagePrePullControllerModuleGroup
}

// Enable indicates whether enable this module
func (uc *ImagePrePullController) Enable() bool {
	return uc.enable
}

// Start controller
func (uc *ImagePrePullController) Start() {
	if err := uc.downstream.Start(); err != nil {
		klog.Exitf("start ImagePrePullJob controller downstream failed with error: %s", err)
	}
	// wait for downstream controller to start and load ImagePrePullJob
	// TODO think about sync
	time.Sleep(1 * time.Second)
	if err := uc.upstream.Start(); err != nil {
		klog.Exitf("start ImagePrePullJob controller upstream failed with error: %s", err)
	}
}
