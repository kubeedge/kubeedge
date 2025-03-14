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
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	v1alpha2cfg "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	taskmgrv1alpha1 "github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha1"
	taskmgrv1alpha2 "github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha2/actions"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	upgradeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
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
	taskmgrv1alpha2.Init()

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

		if msg.GetSource() == message.SourceNodeConnection {
			if err = postHubConnected(ctx); err != nil {
				t.logger.Error(err, "failed to report upgrade status and run next action")
			}
			continue
		}

		var runFunc func(*model.Message) error
		if nodetaskmsg.IsNodeJobResource(msg.GetResource()) {
			runFunc = taskmgrv1alpha2.RunTask
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

func postHubConnected(ctx context.Context) error {
	info, err := upgradeedge.ParseJSONReporterInfo()
	if err != nil {
		return fmt.Errorf("failed to parse json reporter info, err: %v", err)
	}

	// TODO: get jobname and nodename and specData
	var jobname, nodename, action string
	var specData []byte

	switch info.EventType {
	case upgradeedge.EventTypeUpgrade:
		action = string(operationsv1alpha2.NodeUpgradeJobActionUpgrade)
	case upgradeedge.EventTypeRollback:
		action = string(operationsv1alpha2.NodeUpgradeJobActionRollBack)
	default:
		return fmt.Errorf("unsupported event type %s", info.EventType)
	}

	res := taskmsg.Resource{
		APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
		ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
		JobName:      jobname,
		NodeName:     nodename,
	}
	body := taskmsg.UpstreamMessage{
		Action: action,
		Extend: taskmsg.FormatNodeUpgradeJobExtend(info.FromVersion, info.ToVersion),
	}
	if info.ErrorMessage != "" {
		body.Succ = false
		body.Reason = info.ErrorMessage
	} else {
		body.Succ = true
	}
	message.ReportNodeTaskStatus(res, body)

	actions.GetRunner(operationsv1alpha2.ResourceNodeUpgradeJob).
		RunAction(ctx, jobname, nodename, action, specData)
	return nil
}
