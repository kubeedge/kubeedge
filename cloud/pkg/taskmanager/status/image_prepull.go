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

func tryUpdateImagePrePullJobStatus(ctx context.Context, cli crdcliset.Interface, opts TryUpdateStatusOptions) error {
	// Get the image pre-pull job.
	job, err := cli.OperationsV1alpha2().ImagePrePullJobs().Get(ctx, opts.JobName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get image prepull job %s, err: %v", opts.JobName, err)
	}

	// Find the node task status by node name. If not found, return error.
	var nodeStatus *operationsv1alpha2.ImagePrePullNodeTaskStatus
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

	// Set the node task status fields and update the image pre-pull job.
	if opts.ActionStatus != nil {
		actionStatus, ok := opts.ActionStatus.(*operationsv1alpha2.ImagePrePullJobActionStatus)
		if !ok {
			return fmt.Errorf("invalid image pre-pull action status type %T", opts.ActionStatus)
		}
		if nodeStatus.ActionFlow == nil {
			nodeStatus.ActionFlow = make([]operationsv1alpha2.ImagePrePullJobActionStatus, 0)
		}
		nodeStatus.ActionFlow = append(nodeStatus.ActionFlow, *actionStatus)
	}

	nodeStatus.Phase = opts.Phase
	if opts.ExtendInfo != "" {
		imageStatus, err := taskmsg.ParseImagePrePullJobExtend(opts.ExtendInfo)
		if err != nil {
			return fmt.Errorf("failed to parse image prepull job extend, err: %v", err)
		}
		nodeStatus.ImageStatus = imageStatus
	}

	_, err = cli.OperationsV1alpha2().ImagePrePullJobs().UpdateStatus(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update image prepull job %s status, err: %v", opts.JobName, err)
	}
	return nil
}
