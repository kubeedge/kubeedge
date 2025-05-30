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
	v1alpha2downstream "github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/downstream"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/status"
	v1alpha2upstream "github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/upstream"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/imageprepullcontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/nodeupgradecontroller"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/util"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/util/controller"
	"github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

// TaskManager is a module for node task management.
type TaskManager struct {
	enable   bool
	msglayer messagelayer.MessageLayer

	// fields of v1alpah1
	upstreamV1alpha1Chan chan model.Message
	downstream           *manager.DownstreamController
	executorMachine      *manager.ExecutorMachine
	upstream             *manager.UpstreamController

	// fields of v1alpha2
	upstreamV1alpha2Chan chan model.Message
}

var _ core.Module = (*TaskManager)(nil)

// newTaskManager creates a new TaskManager instance.
func newTaskManager(enable bool) *TaskManager {
	return &TaskManager{
		enable:               enable,
		upstreamV1alpha1Chan: make(chan model.Message, 64),
		upstreamV1alpha2Chan: make(chan model.Message, 64),
		msglayer:             messagelayer.TaskManagerMessageLayer(),
	}
}

// Register initializes TaskManager module.
func Register(dc *v1alpha1.TaskManager) {
	config.InitConfigure(dc)
	tm := newTaskManager(dc.Enable)
	ctx := beehiveContext.GetContext()

	// The informer event handler registration needs to be done before calling the informer Start(..).
	// The Start() function of KubeEdge crds informer is called at the end of the CloudCore Run.
	// Refer to: cloud/cmd/cloudcore/app/server.go#L137
	if !features.DefaultFeatureGate.Enabled(features.DisableNodeTaskV1alpha2) {
		status.Init(ctx)
		if err := v1alpha2downstream.Init(ctx); err != nil {
			panic(err)
		}
		v1alpha2upstream.Init(ctx)
	} else {
		if err := tm.initV1alpha1(); err != nil {
			panic(fmt.Errorf("failed to init node task v1alpha1 controller, err: %v", err))
		}
	}
	core.Register(tm)
}

// Name of controller.
func (TaskManager) Name() string {
	return modules.TaskManagerModuleName
}

// Group of controller.
func (TaskManager) Group() string {
	return modules.TaskManagerModuleGroup
}

// Enable indicates whether enable this module.
func (tm TaskManager) Enable() bool {
	return tm.enable
}

// Start the task manager module.
func (tm *TaskManager) Start() {
	ctx := beehiveContext.GetContext()
	asyncCallFunc(ctx, tm.dispatchMessage)
	if !features.DefaultFeatureGate.Enabled(features.DisableNodeTaskV1alpha2) {
		v1alpha2downstream.Start(ctx)
		v1alpha2upstream.Start(ctx, tm.upstreamV1alpha2Chan)
	} else {
		asyncCallFunc(ctx, tm.startV1alpha1)
	}
	<-ctx.Done()
}

// asyncCallFunc calls the function using the gocoroutine and panic the error if the function returns an error
func asyncCallFunc(ctx context.Context, fn func(context.Context) error) {
	go func() {
		if err := fn(ctx); err != nil {
			panic(err)
		}
	}()
}

// dispatchMessage dispatches the node task upstream message to the corresponding handler.
func (tm *TaskManager) dispatchMessage(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			klog.Info("stop dispatch task upstream message")
			return nil
		default:
		}
		msg, err := tm.msglayer.Receive()
		if err != nil {
			klog.Warningf("failed to receive node task upstream message, err: %v", err)
			continue
		}
		if message.IsNodeJobResource(msg.GetResource()) {
			tm.upstreamV1alpha2Chan <- msg
		} else {
			tm.upstreamV1alpha1Chan <- msg
		}
	}
}

// initV1alpha1 initializes the v1alpha1 version of node task upstream/downstream.
func (tm *TaskManager) initV1alpha1() error {
	var err error
	taskMessage := make(chan util.TaskMessage, 10)
	downStreamMessage := make(chan model.Message, 10)
	tm.downstream, err = manager.NewDownstreamController(downStreamMessage)
	if err != nil {
		return fmt.Errorf("new task manager downstream failed with error: %s", err)
	}
	tm.upstream, err = manager.NewUpstreamController(tm.downstream, tm.upstreamV1alpha1Chan)
	if err != nil {
		return fmt.Errorf("new task manager upstream failed with error: %s", err)
	}
	tm.executorMachine, err = manager.NewExecutorMachine(taskMessage, downStreamMessage)
	if err != nil {
		return fmt.Errorf("new executor machine failed with error: %s", err)
	}

	upgradeNodeController, err := nodeupgradecontroller.NewNodeUpgradeController(taskMessage)
	if err != nil {
		return fmt.Errorf("new upgrade node controller failed with error: %s", err)
	}

	imagePrePullController, err := imageprepullcontroller.NewImagePrePullController(taskMessage)
	if err != nil {
		return fmt.Errorf("new upgrade node controller failed with error: %s", err)
	}
	controller.Register(util.TaskUpgrade, upgradeNodeController)
	controller.Register(util.TaskPrePull, imagePrePullController)
	return nil
}

// startV1alpha1 starts the v1alpha1 version of node task upstream/downstream.
func (tm *TaskManager) startV1alpha1(_ context.Context) error {
	if err := tm.downstream.Start(); err != nil {
		return fmt.Errorf("start task manager downstream failed with error: %s", err)
	}
	// wait for downstream to start and load NodeUpgradeJob
	// TODO: think about sync
	time.Sleep(1 * time.Second)
	if err := tm.upstream.Start(); err != nil {
		return fmt.Errorf("start task manager upstream failed with error: %s", err)
	}
	if err := tm.executorMachine.Start(); err != nil {
		return fmt.Errorf("start task manager executorMachine failed with error: %s", err)
	}
	if err := controller.StartAllController(); err != nil {
		return fmt.Errorf("start controller failed with error: %s", err)
	}
	return nil
}
