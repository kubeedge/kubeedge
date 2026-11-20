/*
Copyright 2026 The KubeEdge Authors.

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

func TestTryUpdateConfigUpdateJobStatus(t *testing.T) {
	var (
		ctx      = context.TODO()
		jobName  = "test-config-job"
		nodeName = "edge1"
	)
	cli := crdfake.NewSimpleClientset(&operationsv1alpha2.ConfigUpdateJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Status: operationsv1alpha2.ConfigUpdateJobStatus{
			NodeStatus: []operationsv1alpha2.ConfigUpdateJobNodeTaskStatus{
				{
					NodeName: nodeName,
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
			},
		},
	})

	t.Run("record the failure reason of the node task", func(t *testing.T) {
		err := tryUpdateConfigUpdateJobStatus(ctx, cli, TryUpdateStatusOptions{
			JobName:  jobName,
			NodeName: nodeName,
			Phase:    operationsv1alpha2.NodeTaskPhaseFailure,
			Reason:   "the node edge1 is not connected to the current cloudcore instance",
		})
		require.NoError(t, err)
		job, err := cli.OperationsV1alpha2().ConfigUpdateJobs().
			Get(ctx, jobName, metav1.GetOptions{})
		require.NoError(t, err)
		nodeStatus := job.Status.NodeStatus[0]
		require.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, nodeStatus.Phase)
		require.Equal(t, "the node edge1 is not connected to the current cloudcore instance", nodeStatus.Reason)
	})
}
