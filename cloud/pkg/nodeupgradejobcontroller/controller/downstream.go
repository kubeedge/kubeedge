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

package controller

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/nodeupgradejobcontroller/manager"
)

var dc *DownstreamController

type DownstreamController struct {
	nodeUpgradeJobManager *manager.NodeUpgradeJobManager
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start NodeUpgradeJob Downstream Controller")

	dc.nodeUpgradeJobManager.Start()

	return nil
}

func NewDownstreamController() (*DownstreamController, error) {
	nodeUpgradeJobManager, err := manager.NewNodeUpgradeJobManager()
	if err != nil {
		klog.Warningf("Create NodeUpgradeJob manager failed with error: %s", err)
		return nil, err
	}
	dc := &DownstreamController{
		nodeUpgradeJobManager: nodeUpgradeJobManager,
	}
	return dc, nil
}
