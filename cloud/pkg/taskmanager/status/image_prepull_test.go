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
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	crdfake "github.com/kubeedge/api/client/clientset/versioned/fake"
)

func TestTryUpdateImagePrePullJobStatus(t *testing.T) {
	ctx := context.TODO()
	cli := crdfake.NewSimpleClientset(&operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job1",
		},
		Status: operationsv1alpha2.ImagePrePullJobStatus{
			NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
				{
					NodeName: "node1",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
			},
		},
	})

	t.Run("failed to get not exists job", func(t *testing.T) {
		err := tryUpdateImagePrePullJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  "not-found-job",
			NodeName: "node1",
			Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
		})
		require.ErrorContains(t, err, "failed to get image prepull job not-found-job")
	})

	t.Run("unable to match node task", func(t *testing.T) {
		err := tryUpdateImagePrePullJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  "test-job1",
			NodeName: "node2",
			Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
		})
		require.ErrorContains(t, err, "unable to match node task, invalid node name 'node2'")
	})

	t.Run("invalid action status type", func(t *testing.T) {
		err := tryUpdateImagePrePullJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:      "test-job1",
			NodeName:     "node1",
			Phase:        operationsv1alpha2.NodeTaskPhaseInProgress,
			ActionStatus: operationsv1alpha2.ImagePrePullJobActionStatus{}, // Want pointer
		})
		require.ErrorContains(t, err, "invalid image pre-pull action status type v1alpha2.ImagePrePullJobActionStatus")
	})

	t.Run("update job status successfully", func(t *testing.T) {
		err := tryUpdateImagePrePullJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  "test-job1",
			NodeName: "node1",
			Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
			ActionStatus: &operationsv1alpha2.ImagePrePullJobActionStatus{
				Action: operationsv1alpha2.ImagePrePullJobActionPull,
				Status: metav1.ConditionTrue,
				Time:   "2025-01-01T00:00:00Z",
			},
			ExtendInfo: `[{"image":"nginx:latest","status":"True"}]`,
		})
		require.NoError(t, err)
		job, err := cli.OperationsV1alpha2().ImagePrePullJobs().
			Get(ctx, "test-job1", metav1.GetOptions{})
		require.NoError(t, err)
		require.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, job.Status.NodeStatus[0].Phase)

		require.Len(t, job.Status.NodeStatus[0].ImageStatus, 1)
		imageStatus := job.Status.NodeStatus[0].ImageStatus[0]
		require.Equal(t, "nginx:latest", imageStatus.Image)
		require.Equal(t, metav1.ConditionTrue, imageStatus.Status)

		require.Len(t, job.Status.NodeStatus[0].ActionFlow, 1)
		actionStatus := job.Status.NodeStatus[0].ActionFlow[0]
		require.Equal(t, operationsv1alpha2.ImagePrePullJobActionPull, actionStatus.Action)
		require.Equal(t, metav1.ConditionTrue, actionStatus.Status)
	})
}
