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

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
)

// NotInitialized checks if the node upgrade job has not been initialized.
func (c *NodeUpgradeJobController) NotInitialized(job *operationsv1alpha2.NodeUpgradeJob) bool {
	return job.Status.State == "" ||
		// Abnormal error status
		len(job.Status.NodeStatus) == 0 && job.Status.Phase != operationsv1alpha2.JobPhaseFailure
}

// InitNodesStatus initializes the nodes status of the node upgrade job.
func (c *NodeUpgradeJobController) InitNodesStatus(ctx context.Context, job *operationsv1alpha2.NodeUpgradeJob) {
	verifyResult, err := VerifyNodeDefine(ctx, c.che, job.Spec.NodeNames, job.Spec.LabelSelector)
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
			BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
				NodeName: it.NodeName,
				Phase:    phase,
				Reason:   it.ErrorMessage,
			},
		})
	}
	job.Status.NodeStatus = nodeStatus
}

// CalculateStatus calculates the node upgrade job phase through the all node tasks phases.
func (c *NodeUpgradeJobController) CalculateStatus(ctx context.Context, job *operationsv1alpha2.NodeUpgradeJob) bool {
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
	if job.Status.Phase != phase {
		job.Status.Phase = phase
		return true
	}
	return false
}
