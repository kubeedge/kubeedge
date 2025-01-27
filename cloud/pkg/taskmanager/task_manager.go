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
	"context"
	"fmt"
	"time"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/imageprepullcontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/nodeupgradecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/util/controller"
	v1alpha2ctl "github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/controller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/controller/handlers"
	"github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type TaskManager struct {
	enable               bool
	upstreamV1alpha1Chan chan model.Message
	upstreamV1alpha2Chan chan model.Message
	msglayer             messagelayer.MessageLayer
}

var _ core.Module = (*TaskManager)(nil)

func newTaskManager(enable bool) *TaskManager {
	return &TaskManager{
		enable:               enable,
		upstreamV1alpha1Chan: make(chan model.Message, 64),
		upstreamV1alpha2Chan: make(chan model.Message, 64),
		msglayer:             messagelayer.TaskManagerMessageLayer(),
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
	ctx := beehiveContext.GetContext()
	asyncCallFunc(ctx, uc.dispatchMessage)
	asyncCallFunc(ctx, uc.initAndStartV1alpha1Controller)
	asyncCallFunc(ctx, uc.initAndStartV1alpha2Controller)
	<-ctx.Done()
}

func asyncCallFunc(ctx context.Context, fn func(context.Context) error) {
	go func() {
		if err := fn(ctx); err != nil {
			panic(err)
		}
	}()
}

func (uc *TaskManager) dispatchMessage(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			klog.Info("stop dispatch task upstream message")
			return nil
		default:
		}
		msg, err := uc.msglayer.Receive()
		if err != nil {
			klog.Warningf("failed to receive node task upstream message, err: %v", err)
			continue
		}
		if message.IsNodeTaskResource(msg.GetResource()) {
			uc.upstreamV1alpha2Chan <- msg
		} else {
			uc.upstreamV1alpha1Chan <- msg
		}
	}
}

func (uc *TaskManager) initAndStartV1alpha1Controller(_ context.Context) error {
	taskMessage := make(chan util.TaskMessage, 10)
	downStreamMessage := make(chan model.Message, 10)
	downstream, err := manager.NewDownstreamController(downStreamMessage)
	if err != nil {
		return fmt.Errorf("New task manager downstream failed with error: %s", err)
	}
	upstream, err := manager.NewUpstreamController(downstream, uc.upstreamV1alpha1Chan)
	if err != nil {
		return fmt.Errorf("New task manager upstream failed with error: %s", err)
	}
	executorMachine, err := manager.NewExecutorMachine(taskMessage, downStreamMessage)
	if err != nil {
		return fmt.Errorf("New executor machine failed with error: %s", err)
	}

	upgradeNodeController, err := nodeupgradecontroller.NewNodeUpgradeController(taskMessage)
	if err != nil {
		return fmt.Errorf("New upgrade node controller failed with error: %s", err)
	}

	imagePrePullController, err := imageprepullcontroller.NewImagePrePullController(taskMessage)
	if err != nil {
		return fmt.Errorf("New upgrade node controller failed with error: %s", err)
	}
	controller.Register(util.TaskUpgrade, upgradeNodeController)
	controller.Register(util.TaskPrePull, imagePrePullController)

	if err := downstream.Start(); err != nil {
		return fmt.Errorf("start task manager downstream failed with error: %s", err)
	}
	// wait for downstream controller to start and load NodeUpgradeJob
	// TODO think about sync
	time.Sleep(1 * time.Second)
	if err := upstream.Start(); err != nil {
		return fmt.Errorf("start task manager upstream failed with error: %s", err)
	}
	if err := executorMachine.Start(); err != nil {
		return fmt.Errorf("start task manager executorMachine failed with error: %s", err)
	}
	if err := controller.StartAllController(); err != nil {
		return fmt.Errorf("start controller failed with error: %s", err)
	}
	return nil
}

func (uc *TaskManager) initAndStartV1alpha2Controller(ctx context.Context) error {
	mgr := v1alpha2ctl.NewManager(uc.upstreamV1alpha2Chan)
	mgr.Registry(handlers.NewNodeUpgradeJobHandler()).
		Registry(handlers.NewImagePrePullJobHandler())

	if err := mgr.DoDownstream(ctx); err != nil {
		return fmt.Errorf("failed to do node task downstream, err: %v", err)
	}
	go mgr.DoUpstream(ctx)
	return nil
}
