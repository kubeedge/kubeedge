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

func TestTryUpdateNodeUpgradeJobStatus(t *testing.T) {
	ctx := context.TODO()
	cli := crdfake.NewSimpleClientset(&operationsv1alpha2.NodeUpgradeJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job1",
		},
		Status: operationsv1alpha2.NodeUpgradeJobStatus{
			NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
				{
					NodeName: "node1",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
			},
		},
	})

	t.Run("failed to get not exists job", func(t *testing.T) {
		err := tryUpdateNodeUpgradeJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  "not-found-job",
			NodeName: "node1",
			Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
		})
		require.ErrorContains(t, err, "failed to get node upgrade job not-found-job")
	})

	t.Run("unable to match node task", func(t *testing.T) {
		err := tryUpdateNodeUpgradeJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  "test-job1",
			NodeName: "node2",
			Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
		})
		require.ErrorContains(t, err, "unable to match node task, invalid node name 'node2'")
	})

	t.Run("invalid action status type", func(t *testing.T) {
		err := tryUpdateNodeUpgradeJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:      "test-job1",
			NodeName:     "node1",
			Phase:        operationsv1alpha2.NodeTaskPhaseInProgress,
			ActionStatus: operationsv1alpha2.NodeUpgradeJobActionStatus{}, // Want pointer
		})
		require.ErrorContains(t, err, "invalid node upgrade action status type v1alpha2.NodeUpgradeJobActionStatus")
	})

	t.Run("update job status successfully", func(t *testing.T) {
		err := tryUpdateNodeUpgradeJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  "test-job1",
			NodeName: "node1",
			Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
			ActionStatus: &operationsv1alpha2.NodeUpgradeJobActionStatus{
				Action: operationsv1alpha2.NodeUpgradeJobActionUpgrade,
				Status: metav1.ConditionTrue,
				Time:   "2025-01-01T00:00:00Z",
			},
			ExtendInfo: "v1.20.0,v1.21.0",
		})
		require.NoError(t, err)
		job, err := cli.OperationsV1alpha2().NodeUpgradeJobs().
			Get(ctx, "test-job1", metav1.GetOptions{})
		require.NoError(t, err)
		nodeStatus := job.Status.NodeStatus[0]
		require.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, nodeStatus.Phase)
		require.Equal(t, "v1.20.0", nodeStatus.HistoricVersion)
		require.Equal(t, "v1.21.0", nodeStatus.CurrentVersion)

		require.Len(t, job.Status.NodeStatus[0].ActionFlow, 1)
		actionStatus := job.Status.NodeStatus[0].ActionFlow[0]
		require.Equal(t, operationsv1alpha2.NodeUpgradeJobActionUpgrade, actionStatus.Action)
		require.Equal(t, metav1.ConditionTrue, actionStatus.Status)
	})
}
