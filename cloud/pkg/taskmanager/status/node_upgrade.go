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

package status

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdcliset "github.com/kubeedge/api/client/clientset/versioned"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func tryUpdateNodeUpgradeJobStatus(ctx context.Context, cli crdcliset.Interface, opts TryUpdateStatusOptions) error {
	// Get the node upgrade job.
	job, err := cli.OperationsV1alpha2().NodeUpgradeJobs().Get(ctx, opts.JobName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node upgrade job %s, err: %v", opts.JobName, err)
	}

	// Find the node task status by node name. If not found, return error.
	var nodeStatus *operationsv1alpha2.NodeUpgradeJobNodeTaskStatus
	for i := range job.Status.NodeStatus {
		it := &job.Status.NodeStatus[i]
		if it.NodeName == opts.NodeName {
			nodeStatus = it
			break
		}
	}
	if nodeStatus == nil {
		return fmt.Errorf("unable to match node task, invalid node name '%s'", opts.NodeName)
	}

	// Set the node task status fields and update the node upgrade job.
	if opts.ActionStatus != nil {
		actionStatus, ok := opts.ActionStatus.(*operationsv1alpha2.NodeUpgradeJobActionStatus)
		if !ok {
			return fmt.Errorf("invalid node upgrade action status type %T", opts.ActionStatus)
		}
		if nodeStatus.ActionFlow == nil {
			nodeStatus.ActionFlow = make([]operationsv1alpha2.NodeUpgradeJobActionStatus, 0)
		}
		nodeStatus.ActionFlow = append(nodeStatus.ActionFlow, *actionStatus)
	}

	nodeStatus.Phase = opts.Phase
	if opts.ExtendInfo != "" {
		fromVer, toVer, err := taskmsg.ParseNodeUpgradeJobExtend(opts.ExtendInfo)
		if err != nil {
			return fmt.Errorf("failed to parse node upgrade job extend, err: %v", err)
		}
		nodeStatus.HistoricVersion = fromVer
		nodeStatus.CurrentVersion = toVer
	}

	_, err = cli.OperationsV1alpha2().NodeUpgradeJobs().UpdateStatus(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update node upgrade job %s status, err: %v", opts.NodeName, err)
	}
	return nil
}
