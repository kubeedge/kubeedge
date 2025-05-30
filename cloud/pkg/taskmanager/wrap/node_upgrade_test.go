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

package wrap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/pkg/nodetask/actionflow"
)

func TestNodeUpgradeJob(t *testing.T) {
	obj := &operationsv1alpha2.NodeUpgradeJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
		Spec: operationsv1alpha2.NodeUpgradeJobSpec{
			Concurrency: 10,
		},
		Status: operationsv1alpha2.NodeUpgradeJobStatus{
			Phase: operationsv1alpha2.JobPhaseInProgress,
			NodeStatus: []operationsv1alpha2.NodeUpgradeJobNodeTaskStatus{
				{
					NodeName: "node1",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
				{
					NodeName: "node2",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
				{
					NodeName: "node3",
					Phase:    operationsv1alpha2.NodeTaskPhaseSuccessful,
					ActionFlow: []operationsv1alpha2.NodeUpgradeJobActionStatus{
						{
							Action: operationsv1alpha2.NodeUpgradeJobActionUpgrade,
							Status: metav1.ConditionTrue,
							Time:   time.Now().Format(time.RFC3339),
						},
					},
				},
			},
		},
	}

	job := NodeUpgradeJob{Obj: obj}

	assert.Equal(t, obj.Name, job.Name())
	assert.Equal(t, operationsv1alpha2.ResourceNodeUpgradeJob, job.ResourceType())
	assert.Equal(t, int(obj.Spec.Concurrency), job.Concurrency())
	assert.Equal(t, obj.Spec, job.Spec())
	assert.Equal(t, obj, job.GetObject())

	tasks := job.Tasks()
	assert.Len(t, tasks, 3)
	assert.True(t, tasks[0].CanExecute())
	act, err := tasks[0].Action()
	assert.NoError(t, err)
	assert.Equal(t, actionflow.FlowNodeUpgradeJob.First, act)
	tasks[0].SetPhase(operationsv1alpha2.NodeTaskPhaseInProgress)
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, tasks[0].Phase())
	tasks[0].SetPhase(operationsv1alpha2.NodeTaskPhaseFailure, "test error")
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, tasks[0].Phase())
	assert.Equal(t, "test error", obj.Status.NodeStatus[0].Reason)

	tasks[1].SetPhase(operationsv1alpha2.NodeTaskPhaseSuccessful)
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, tasks[1].Phase())
}
