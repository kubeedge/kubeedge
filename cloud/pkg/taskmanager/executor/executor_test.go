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

package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/wrap"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestExecutorOperation(t *testing.T) {
	job, err := wrap.WithEventObj(&operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
		Spec: operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				Concurrency: 2,
			},
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
	assert.NoError(t, err)

	updateFun := func(_ctx context.Context, _job wrap.NodeJob, _errTask wrap.NodeJobTask) {}
	ctx := context.TODO()
	exec, loaded, err := NewNodeTaskExecutor(ctx, job, updateFun)
	assert.NoError(t, err)
	assert.False(t, loaded)
	assert.NotNil(t, exec)

	exec, loaded, err = NewNodeTaskExecutor(ctx, job, updateFun)
	assert.NoError(t, err)
	assert.True(t, loaded)
	assert.NotNil(t, exec)

	exec, err = GetExecutor(job.ResourceType(), job.Name())
	assert.NoError(t, err)
	assert.NotNil(t, exec)

	RemoveExecutor(job.ResourceType(), job.Name())

	exec, err = GetExecutor(job.ResourceType(), job.Name())
	assert.Equal(t, ErrExecutorNotExists, err)
	assert.Nil(t, exec)
}

func TestExecute(t *testing.T) {
	obj := &operationsv1alpha2.ImagePrePullJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-job",
		},
		Spec: operationsv1alpha2.ImagePrePullJobSpec{
			ImagePrePullTemplate: operationsv1alpha2.ImagePrePullTemplate{
				Concurrency: 2,
			},
		},
		Status: operationsv1alpha2.ImagePrePullJobStatus{
			NodeStatus: []operationsv1alpha2.ImagePrePullNodeTaskStatus{
				{ // phase is in progress, ignore it.
					NodeName: "node1",
					Phase:    operationsv1alpha2.NodeTaskPhaseInProgress,
				},
				{
					NodeName: "node2",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
				{
					NodeName: "node3",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
				{ // not in connectedNodes, failed
					NodeName: "node4",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
				{ // send message failed
					NodeName: "node5",
					Phase:    operationsv1alpha2.NodeTaskPhasePending,
				},
			},
		},
	}
	job, err := wrap.WithEventObj(obj)
	assert.NoError(t, err)

	updateFun := func(_ctx context.Context, _job wrap.NodeJob, _task wrap.NodeJobTask) {}

	ctx := context.TODO()
	exec, _, err := NewNodeTaskExecutor(ctx, job, updateFun)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethodFunc(&messagelayer.ContextMessageLayer{}, "Send",
		func(message model.Message) error {
			res := taskmsg.ParseResource(message.GetResource())
			if res.NodeName == "node5" {
				return errors.New("test error")
			}
			exec.FinishTask()
			return nil
		})

	exec.Execute(ctx, []string{"node1", "node2", "node3", "node5"})
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, obj.Status.NodeStatus[0].Phase)
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, obj.Status.NodeStatus[1].Phase)
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, obj.Status.NodeStatus[2].Phase)
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, obj.Status.NodeStatus[3].Phase)
	assert.Contains(t, obj.Status.NodeStatus[3].Reason, "the node node4 is not connected to the current cloudcore instance")
	assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, obj.Status.NodeStatus[4].Phase)
	assert.Contains(t, obj.Status.NodeStatus[4].Reason, "failed to send message to edge")
}
