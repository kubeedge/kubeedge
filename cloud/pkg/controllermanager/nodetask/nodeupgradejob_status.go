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
)

func (c *NodeUpgradeJobController) HasInitialized(job *operationsv1alpha2.NodeUpgradeJob) bool {
	return job.Status.State == ""
}

func (c *NodeUpgradeJobController) InitNodesStatus(ctx context.Context, job *operationsv1alpha2.NodeUpgradeJob) {
	res, err := VerifyNodeDefine(ctx, c.che, job.Spec.NodeNames, job.Spec.LabelSelector)
	if err != nil {
		job.Status.State = operationsv1alpha2.JobStateFailure
		job.Status.Reason = err.Error()
		return
	}
	job.Status.State = operationsv1alpha2.JobStateInit
	nodeStatus := make([]operationsv1alpha2.NodeUpgradeJobNodeTaskStatus, 0, len(res))
	for _, node := range res {
		var status metav1.ConditionStatus
		if node.ErrorMessage == "" {
			status = metav1.ConditionTrue
		} else {
			status = metav1.ConditionFalse
		}
		nodeStatus = append(nodeStatus, operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
			Action: operationsv1alpha2.NodeUpgradeJobActionInit,
			BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
				NodeName: node.NodeName,
				Status:   status,
				Reason:   node.ErrorMessage,
			},
		})
	}
	job.Status.NodeStatus = nodeStatus
}

func (c *NodeUpgradeJobController) CalculateStatus(job *operationsv1alpha2.NodeUpgradeJob) bool {
	// TODO: ...
	return false
}
