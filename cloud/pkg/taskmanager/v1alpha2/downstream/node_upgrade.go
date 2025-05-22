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

package downstream

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/status"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/wrap"
)

type NodeUpgradeJobHandler struct {
	logger         logr.Logger
	crdcli         crdcliset.Interface
	downstreamChan chan wrap.NodeJob
}

func newNodeUpgradeJobHandler(ctx context.Context) (*NodeUpgradeJobHandler, error) {
	logger := klog.FromContext(ctx).
		WithName(fmt.Sprintf("downstream-%s", operationsv1alpha2.ResourceNodeUpgradeJob))
	handler := &NodeUpgradeJobHandler{
		logger:         logger,
		crdcli:         client.GetCRDClient(),
		downstreamChan: make(chan wrap.NodeJob, downstreamChanSize),
	}
	informer := informers.GetInformersManager().
		GetKubeEdgeInformerFactory().
		Operations().
		V1alpha2().
		NodeUpgradeJobs().
		Informer()
	_, err := informer.AddEventHandler(NewNodeJobEventHandler(handler.logger, handler.downstreamChan))
	if err != nil {
		return nil, fmt.Errorf("failed to add NodeUpgradeJob event handler, err: %v", err)
	}
	return handler, nil
}

func (h *NodeUpgradeJobHandler) Logger() logr.Logger {
	return h.logger
}

func (h *NodeUpgradeJobHandler) CanDownstreamPhase(obj any) bool {
	job, ok := obj.(*operationsv1alpha2.NodeUpgradeJob)
	if !ok {
		h.logger.Error(nil, "failed to convert obj to NodeUpgradeJob", "invalid type", reflect.TypeOf(obj))
		return false
	}
	// TODO: retry execution is not supported due to some reasons.
	// To support retry execution, the cloud edge needs to consider many situations.
	return job.Status.Phase == operationsv1alpha2.JobPhaseInit
}

func (h *NodeUpgradeJobHandler) ExecutorChan() chan wrap.NodeJob {
	return h.downstreamChan
}

func (h *NodeUpgradeJobHandler) InterruptExecutor(obj any) {
	job, ok := obj.(*operationsv1alpha2.NodeUpgradeJob)
	if !ok {
		h.logger.Error(nil, "failed to convert obj to NodeUpgradeJob", "invalid type", reflect.TypeOf(obj))
		return
	}
	exec, err := executor.GetExecutor(operationsv1alpha2.ResourceNodeUpgradeJob, job.Name)
	if err != nil && !errors.Is(err, executor.ErrExecutorNotExists) {
		h.logger.Error(err, "failed to get executor", "job name", job.Name)
		return
	}
	if exec != nil {
		exec.Interrupt()
		executor.RemoveExecutor(operationsv1alpha2.ResourceNodeUpgradeJob, job.Name)
	}
}

func (h *NodeUpgradeJobHandler) UpdateNodeTaskStatus(
	ctx context.Context,
	job wrap.NodeJob,
	task wrap.NodeJobTask,
) {
	nodeUpgradeJob, ok := job.GetObject().(*operationsv1alpha2.NodeUpgradeJob)
	if !ok {
		h.logger.Error(nil, "failed to convert job to NodeUpgradeJob",
			"invalid type", reflect.TypeOf(job.GetObject()))
		return
	}
	taskStatus, ok := task.GetObject().(*operationsv1alpha2.NodeUpgradeJobNodeTaskStatus)
	if !ok {
		h.logger.Error(nil, "failed to convert task to NodeUpgradeJobNodeTaskStatus",
			"invalid type", reflect.TypeOf(task.GetObject()))
		return
	}

	opts := status.UpdateStatusOptions{
		TryUpdateStatusOptions: status.TryUpdateStatusOptions{
			JobName:  nodeUpgradeJob.Name,
			NodeName: taskStatus.NodeName,
			Phase:    taskStatus.Phase,
			Reason:   taskStatus.Reason,
		},
		Callback: func(err error) {
			if err != nil {
				h.logger.Error(err, "failed to update NodeUpgradeJob node task status",
					"job name", nodeUpgradeJob.Name, "node name", taskStatus.NodeName)
			}
		},
	}
	status.GetNodeUpgradeJobStatusUpdater().UpdateStatus(opts)
}
