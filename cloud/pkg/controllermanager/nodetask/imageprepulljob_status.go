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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

// NotInitialized checks if the image prepull job has not been initialized.
func (c *ImagePrePullJobController) NotInitialized(job *operationsv1alpha2.ImagePrePullJob) bool {
	return job.Status.State == "" ||
		// Abnormal error status
		len(job.Status.NodeStatus) == 0 && job.Status.State != operationsv1alpha2.JobStateFailure
}

// InitNodesStatus initializes the nodes status of the image prepull job.
func (c *ImagePrePullJobController) InitNodesStatus(ctx context.Context, job *operationsv1alpha2.ImagePrePullJob) {
	res, err := VerifyNodeDefine(ctx, c.che, job.Spec.ImagePrePullTemplate.CheckItems,
		job.Spec.ImagePrePullTemplate.LabelSelector)
	if err != nil {
		job.Status.State = operationsv1alpha2.JobStateFailure
		job.Status.Reason = err.Error()
		return
	}
	job.Status.State = operationsv1alpha2.JobStateInit
	nodeStatus := make([]operationsv1alpha2.ImagePrePullNodeTaskStatus, 0, len(res))
	for _, node := range res {
		var status metav1.ConditionStatus
		if node.ErrorMessage == "" {
			status = metav1.ConditionTrue
		} else {
			status = metav1.ConditionFalse
		}
		nodeStatus = append(nodeStatus, operationsv1alpha2.ImagePrePullNodeTaskStatus{
			Action: operationsv1alpha2.ImagePrePullJobActionInit,
			BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
				NodeName: node.NodeName,
				Status:   status,
				Reason:   node.ErrorMessage,
			},
		})
	}
	job.Status.NodeStatus = nodeStatus
}

// CalculateStatus calculates the node task status through the task processing status on each node.
func (c *ImagePrePullJobController) CalculateStatus(job *operationsv1alpha2.ImagePrePullJob) bool {
	var processingCount, failedCount int64
	// Statistics node execution task status.
	for _, it := range job.Status.NodeStatus {
		if it.Status == metav1.ConditionFalse || it.Status == metav1.ConditionUnknown {
			failedCount++
			continue
		}
		act := actionflow.FlowNodeUpgradeJob.Find(string(it.Action))
		if !act.Final() { // Node action is the final step.
			processingCount++
		}
	}

	jobState := CalculateStatusWithCounts(int64(len(job.Status.NodeStatus)),
		processingCount, failedCount, job.Spec.ImagePrePullTemplate.FailureTolerate)
	if job.Status.State != jobState {
		job.Status.State = jobState
		return true
	}
	return false
}
