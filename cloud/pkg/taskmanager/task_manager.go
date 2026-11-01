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
	"github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

// TaskManager is a module for node task management.
type TaskManager struct {
	enable   bool
	msglayer messagelayer.MessageLayer

	// fields of v1alpha2
	upstreamV1alpha2Chan chan model.Message
}

var _ core.Module = (*TaskManager)(nil)

// newTaskManager creates a new TaskManager instance.
func newTaskManager(enable bool) *TaskManager {
	return &TaskManager{
		enable:               enable,
		upstreamV1alpha2Chan: make(chan model.Message, 64),
		msglayer:             messagelayer.TaskManagerMessageLayer(),
	}
}

// Register initializes TaskManager module.
func Register(dc *v1alpha1.TaskManager) {
	tm := newTaskManager(dc.Enable)
	ctx := beehiveContext.GetContext()

	// The informer event handler registration needs to be done before calling the informer Start(..).
	// The Start() function of KubeEdge crds informer is called at the end of the CloudCore Run.
	// Refer to: cloud/cmd/cloudcore/app/server.go#L137
	status.Init(ctx)
	if err := v1alpha2downstream.Init(ctx); err != nil {
		panic(err)
	}
	v1alpha2upstream.Init(ctx)
	core.Register(tm)
}

// Name of controller.
func (tm *TaskManager) Name() string {
	return modules.TaskManagerModuleName
}

// Group of controller.
func (tm *TaskManager) Group() string {
	return modules.TaskManagerModuleGroup
}

// Enable indicates whether enable this module.
func (tm *TaskManager) Enable() bool {
	return tm.enable
}

func (tm *TaskManager) RestartPolicy() *core.ModuleRestartPolicy {
	return nil
}

// Start the task manager module.
func (tm *TaskManager) Start() {
	ctx := beehiveContext.GetContext()
	asyncCallFunc(ctx, tm.dispatchMessage)
	v1alpha2downstream.Start(ctx)
	v1alpha2upstream.Start(ctx, tm.upstreamV1alpha2Chan)
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
		}
	}
}
