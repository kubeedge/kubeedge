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
)

var (
	imagePrePullJobStatusUpdater *StatusUpdater[operationsv1alpha2.ImagePrePullNodeTaskStatus]

	nodeUpgradeJobStatusUpdater *StatusUpdater[operationsv1alpha2.NodeUpgradeJobNodeTaskStatus]
)

func Init(ctx context.Context) {
	imagePrePullJobStatusUpdater = NewStatusUpdater(ctx, tryUpdateImagePrePullJobStatus)
	go imagePrePullJobStatusUpdater.WatchUpdateChannel()

	nodeUpgradeJobStatusUpdater = NewStatusUpdater(ctx, tryUpdateNodeUpgradeJobStatus)
	go nodeUpgradeJobStatusUpdater.WatchUpdateChannel()
}

func GetImagePrePullJobStatusUpdater() *StatusUpdater[operationsv1alpha2.ImagePrePullNodeTaskStatus] {
	return imagePrePullJobStatusUpdater
}

func GetNodeUpgradeJobStatusUpdater() *StatusUpdater[operationsv1alpha2.NodeUpgradeJobNodeTaskStatus] {
	return nodeUpgradeJobStatusUpdater
}

func tryUpdateImagePrePullJobStatus(
	ctx context.Context,
	cli crdcliset.Interface,
	jobName string,
	nodeTaskStatus operationsv1alpha2.ImagePrePullNodeTaskStatus,
) error {
	job, err := cli.OperationsV1alpha2().ImagePrePullJobs().
		Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("faield to get image prepull job %s, err: %v", jobName, err)
	}
	for i := range job.Status.NodeStatus {
		status := &job.Status.NodeStatus[i]
		if status.NodeName == nodeTaskStatus.NodeName {
			if nodeTaskStatus.Time == "" {
				nodeTaskStatus.Time = job.Status.NodeStatus[i].Time
			}
			job.Status.NodeStatus[i] = nodeTaskStatus
			break
		}
	}
	_, err = cli.OperationsV1alpha2().ImagePrePullJobs().
		UpdateStatus(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update image prepull job %s status, err: %v", jobName, err)
	}
	return nil
}

func tryUpdateNodeUpgradeJobStatus(
	ctx context.Context,
	cli crdcliset.Interface,
	jobName string,
	nodeTaskStatus operationsv1alpha2.NodeUpgradeJobNodeTaskStatus,
) error {
	job, err := cli.OperationsV1alpha2().NodeUpgradeJobs().
		Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("faield to get node upgrade job %s, err: %v", jobName, err)
	}
	for i := range job.Status.NodeStatus {
		status := &job.Status.NodeStatus[i]
		if status.NodeName == nodeTaskStatus.NodeName {
			if nodeTaskStatus.Time == "" {
				nodeTaskStatus.Time = job.Status.NodeStatus[i].Time
			}
			job.Status.NodeStatus[i] = nodeTaskStatus
			break
		}
	}
	_, err = cli.OperationsV1alpha2().NodeUpgradeJobs().
		UpdateStatus(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update node upgrade job %s status, err: %v", jobName, err)
	}
	return nil
}
