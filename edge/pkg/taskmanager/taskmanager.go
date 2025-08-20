/*
Copyright 2025 The KubeEdge Authors.

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

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	v1alpha2cfg "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	taskmgrv1alpha1 "github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/features"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

type TaskManager struct {
	cfg    *v1alpha2cfg.TaskManager
	logger logr.Logger
}

// Check whether TaskManager implements core.Module interface
var _ core.Module = (*TaskManager)(nil)

// Register registers the taskmanager module
func Register(cfg *v1alpha2cfg.TaskManager) {
	taskmgrv1alpha1.Init()
	InitRunner()

	core.Register(&TaskManager{
		cfg:    cfg,
		logger: klog.Background().WithName(modules.TaskManagerModuleName),
	})
}

func (TaskManager) Name() string {
	return modules.TaskManagerModuleName
}

func (TaskManager) Group() string {
	return modules.TaskManagerGroup
}

func (t *TaskManager) Enable() bool {
	return t.cfg != nil && t.cfg.Enable
}

func (t *TaskManager) RestartPolicy() *core.ModuleRestartPolicy {
	if !features.DefaultFeatureGate.Enabled(features.ModuleRestart) {
		return nil
	}
	return &core.ModuleRestartPolicy{
		RestartType:            core.RestartTypeOnFailure,
		IntervalTimeGrowthRate: 2.0,
	}
}

func (t TaskManager) Start() {
	ctx := beehiveContext.GetContext()
	for {
		select {
		case <-ctx.Done():
			t.logger.Info("module stopped")
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.TaskManagerModuleName)
		if err != nil {
			t.logger.Error(err, "failed to receive message, wait 10s")
			time.Sleep(10 * time.Second) // Prevent logs from flooding the screen when the chennel is closed.
			continue
		}
		t.logger.V(2).Info("receive the message", "source", msg.GetSource(), "resource", msg.GetResource(),
			"operation", msg.GetOperation())

		if msg.GetSource() == message.SourceNodeConnection {
			if err = ReportUpgradeStatus(ctx); err != nil {
				t.logger.Error(err, "failed to report upgrade status and run next action")
			}
			continue
		}

		var runFunc func(*model.Message) error
		if nodetaskmsg.IsNodeJobResource(msg.GetResource()) {
			runFunc = RunTask
		} else {
			runFunc = taskmgrv1alpha1.RunTask
		}
		go func() {
			if err := runFunc(&msg); err != nil {
				t.logger.Error(err, "failed to run node task")
			}
		}()
	}
}
