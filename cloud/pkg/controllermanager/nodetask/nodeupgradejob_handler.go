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

package nodetask

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

type NodeUpgradeJobReconcileHandler struct {
	cli client.Client
	che cache.Cache
}

var _ ReconcileHandler[operationsv1alpha2.NodeUpgradeJob] = (*NodeUpgradeJobReconcileHandler)(nil)

func NewNodeUpgradeJobReconcileHandler(cli client.Client, che cache.Cache) *NodeUpgradeJobReconcileHandler {
	return &NodeUpgradeJobReconcileHandler{
		cli: cli,
		che: che,
	}
}

func (h *NodeUpgradeJobReconcileHandler) GetJob(ctx context.Context, req controllerruntime.Request,
) (*operationsv1alpha2.NodeUpgradeJob, error) {
	var job operationsv1alpha2.NodeUpgradeJob
	if err := h.cli.Get(ctx, req.NamespacedName, &job); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node upgrade job %s, err: %v",
			req.NamespacedName, err)
	}
	return &job, nil
}

func (NodeUpgradeJobReconcileHandler) NotInitialized(job *operationsv1alpha2.NodeUpgradeJob) bool {
	return job.Status.Phase == ""
}

func (NodeUpgradeJobReconcileHandler) IsFinalPhase(job *operationsv1alpha2.NodeUpgradeJob) bool {
	// Node upgrade job has fail action path can be completed.
	return job.Status.Phase == operationsv1alpha2.JobPhaseCompleted
}

func (h *NodeUpgradeJobReconcileHandler) InitNodesStatus(ctx context.Context, job *operationsv1alpha2.NodeUpgradeJob) {
	verifyResult, err := VerifyNodeDefine(ctx, h.che, job.Spec.NodeNames, job.Spec.LabelSelector)
	if err != nil {
		job.Status.Phase = operationsv1alpha2.JobPhaseFailure
		job.Status.Reason = err.Error()
		return
	}
	job.Status.Phase = operationsv1alpha2.JobPhaseInit
	nodeStatus := make([]operationsv1alpha2.NodeUpgradeJobNodeTaskStatus, 0, len(verifyResult))
	for _, it := range verifyResult {
		var phase operationsv1alpha2.NodeTaskPhase
		if it.ErrorMessage == "" {
			phase = operationsv1alpha2.NodeTaskPhasePending
		} else {
			phase = operationsv1alpha2.NodeTaskPhaseFailure
		}
		nodeStatus = append(nodeStatus, operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
			NodeName: it.NodeName,
			Phase:    phase,
			Reason:   it.ErrorMessage,
		})
	}
	job.Status.NodeStatus = nodeStatus
}

func (NodeUpgradeJobReconcileHandler) CalculateStatus(ctx context.Context, job *operationsv1alpha2.NodeUpgradeJob) bool {
	var processingCount, failedCount int64
	for _, it := range job.Status.NodeStatus {
		if it.Phase == operationsv1alpha2.NodeTaskPhaseFailure ||
			it.Phase == operationsv1alpha2.NodeTaskPhaseUnknown {
			failedCount++
			continue
		}
		if it.Phase != operationsv1alpha2.NodeTaskPhaseSuccessful {
			processingCount++
			continue
		}
	}

	phase := CalculatePhaseWithCounts(int64(len(job.Status.NodeStatus)),
		processingCount, failedCount, job.Spec.FailureTolerate)
	var reason string
	if phase == operationsv1alpha2.JobPhaseFailure {
		reason = fmt.Sprintf("the number of failed nodes is %d/%d, which exceeds the failure tolerance threshold",
			failedCount, len(job.Status.NodeStatus))
	}
	var changed bool
	if job.Status.Phase != phase {
		job.Status.Phase = phase
		changed = true
	}
	if job.Status.Reason != reason {
		job.Status.Reason = reason
		changed = true
	}
	return changed
}

func (h *NodeUpgradeJobReconcileHandler) UpdateJobStatus(ctx context.Context, job *operationsv1alpha2.NodeUpgradeJob) error {
	if err := h.cli.Status().Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update node upgrade job %s status, err: %v",
			job.Name, err)
	}
	return nil
}
