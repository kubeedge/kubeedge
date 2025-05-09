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
	const (
		jobName  = "test-job"
		nodeName = "node1"
	)
	ctx := context.TODO()
	cli := crdfake.NewSimpleClientset(&operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Status: operationsv1alpha2.ImagePrePullJobStatus{
			NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
				{
					BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
						NodeName: nodeName,
						Phase:    operationsv1alpha2.NodeTaskPhasePending,
					},
				},
			},
		},
	})
	err := tryUpdateImagePrePullJobStatus(ctx, cli, jobName, operationsv1alpha2.ImagePrePullNodeTaskStatus{
		BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
			NodeName: nodeName,
			Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
		},
	})
	require.NoError(t, err)
	job, err := cli.OperationsV1alpha2().ImagePrePullJobs().
		Get(ctx, jobName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, job.Status.NodeStatus[0].Phase)
}

func TestTryUpdateNodeUpgradeJobStatus(t *testing.T) {
	const (
		jobName  = "test-job"
		nodeName = "node1"
	)
	ctx := context.TODO()
	cli := crdfake.NewSimpleClientset(&operationsv1alpha2.NodeUpgradeJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Status: operationsv1alpha2.NodeUpgradeJobStatus{
			NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
				{
					BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
						NodeName: nodeName,
						Phase:    operationsv1alpha2.NodeTaskPhasePending,
					},
				},
			},
		},
	})
	err := tryUpdateNodeUpgradeJobStatus(ctx, cli, jobName, operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
		BasicNodeTaskStatus: operationsv1alpha2.BasicNodeTaskStatus{
			NodeName: nodeName,
			Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
		},
	})
	require.NoError(t, err)
	job, err := cli.OperationsV1alpha2().NodeUpgradeJobs().
		Get(ctx, jobName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, job.Status.NodeStatus[0].Phase)
}
