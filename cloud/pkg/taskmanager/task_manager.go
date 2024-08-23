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

package taskmanager

import (
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/imageprepullcontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/nodeupgradecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/controller"
)

type TaskManager struct {
	downstream      *manager.DownstreamController
	executorMachine *manager.ExecutorMachine
	upstream        *manager.UpstreamController
	enable          bool
}

var _ core.Module = (*TaskManager)(nil)

func newTaskManager(enable bool) *TaskManager {
	if !enable {
		return &TaskManager{enable: enable}
	}
	taskMessage := make(chan util.TaskMessage, 10)
	downStreamMessage := make(chan model.Message, 10)
	downstream, err := manager.NewDownstreamController(downStreamMessage)
	if err != nil {
		klog.Exitf("New task manager downstream failed with error: %s", err)
	}
	upstream, err := manager.NewUpstreamController(downstream)
	if err != nil {
		klog.Exitf("New task manager upstream failed with error: %s", err)
	}
	executorMachine, err := manager.NewExecutorMachine(taskMessage, downStreamMessage)
	if err != nil {
		klog.Exitf("New executor machine failed with error: %s", err)
	}

	upgradeNodeController, err := nodeupgradecontroller.NewNodeUpgradeController(taskMessage)
	if err != nil {
		klog.Exitf("New upgrade node controller failed with error: %s", err)
	}

	imagePrePullController, err := imageprepullcontroller.NewImagePrePullController(taskMessage)
	if err != nil {
		klog.Exitf("New upgrade node controller failed with error: %s", err)
	}
	controller.Register(util.TaskUpgrade, upgradeNodeController)
	controller.Register(util.TaskPrePull, imagePrePullController)

	return &TaskManager{
		downstream:      downstream,
		executorMachine: executorMachine,
		upstream:        upstream,
		enable:          enable,
	}
}

func Register(dc *v1alpha1.TaskManager) {
	config.InitConfigure(dc)
	core.Register(newTaskManager(dc.Enable))
	//core.Register(newNodeUpgradeJobController())
}

// Name of controller
func (uc *TaskManager) Name() string {
	return modules.TaskManagerModuleName
}

// Group of controller
func (uc *TaskManager) Group() string {
	return modules.TaskManagerModuleGroup
}

// Enable indicates whether enable this module
func (uc *TaskManager) Enable() bool {
	return uc.enable
}

// Start controller
func (uc *TaskManager) Start() {
	if err := uc.downstream.Start(); err != nil {
		klog.Exitf("start task manager downstream failed with error: %s", err)
	}
	// wait for downstream controller to start and load NodeUpgradeJob
	// TODO think about sync
	time.Sleep(1 * time.Second)
	if err := uc.upstream.Start(); err != nil {
		klog.Exitf("start task manager upstream failed with error: %s", err)
	}
	if err := uc.executorMachine.Start(); err != nil {
		klog.Exitf("start task manager executorMachine failed with error: %s", err)
	}

	if err := controller.StartAllController(); err != nil {
		klog.Exitf("start controller failed with error: %s", err)
	}
}
