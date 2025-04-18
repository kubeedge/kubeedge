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

package upstream

import (
	"encoding/json"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/executor"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestConvToNodeTask(t *testing.T) {
	t.Run("upstream action run successful", func(t *testing.T) {
		upmsg := taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionCheck),
			Succ:   true,
		}
		handler := ImagePrePullJobHandler{logger: klog.Background()}
		task, err := handler.ConvToNodeTask("node1", &upmsg)
		assert.NoError(t, err)
		assert.NotNil(t, task)

		taskStatus, ok := task.GetObject().(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
		assert.True(t, ok)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, taskStatus.Phase)
		assert.Equal(t, operationsv1alpha2.ImagePrePullJobActionCheck, taskStatus.Action)
	})

	t.Run("upstream action run failed", func(t *testing.T) {
		imgStatus := []operationsv1alpha2.ImageStatus{
			{Image: "image1", Status: metav1.ConditionTrue},
			{Image: "image2", Status: metav1.ConditionFalse, Reason: "pull failed"},
		}
		bff, err := json.Marshal(imgStatus)
		assert.NoError(t, err)
		upmsg := taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			Succ:   false,
			Extend: string(bff),
		}
		handler := ImagePrePullJobHandler{logger: klog.Background()}
		task, err := handler.ConvToNodeTask("node1", &upmsg)
		assert.NoError(t, err)
		assert.NotNil(t, task)

		taskStatus, ok := task.GetObject().(*operationsv1alpha2.ImagePrePullNodeTaskStatus)
		assert.True(t, ok)
		assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, taskStatus.Phase)
		assert.Equal(t, operationsv1alpha2.ImagePrePullJobActionPull, taskStatus.Action)
		assert.Len(t, taskStatus.ImageStatus, 2)
	})
}

func TestGetCurrentAction(t *testing.T) {
	t.Run("get action successful", func(t *testing.T) {
		nodetask := &operationsv1alpha2.ImagePrePullNodeTaskStatus{
			Action: operationsv1alpha2.ImagePrePullJobActionCheck,
		}
		handler := ImagePrePullJobHandler{logger: klog.Background()}
		act, err := handler.GetCurrentAction(nodetask)
		assert.NoError(t, err)
		assert.NotNil(t, act)
		assert.Equal(t, string(operationsv1alpha2.ImagePrePullJobActionCheck), act.Name)
	})

	t.Run("invalid action", func(t *testing.T) {
		nodetask := &operationsv1alpha2.ImagePrePullNodeTaskStatus{
			Action: operationsv1alpha2.ImagePrePullJobAction("xxx"),
		}
		handler := ImagePrePullJobHandler{logger: klog.Background()}
		act, err := handler.GetCurrentAction(nodetask)
		assert.ErrorContains(t, err, "invalid action")
		assert.Nil(t, act)
	})
}

func TestReleaseExecutorConcurrent(t *testing.T) {
	var finishTask bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(executor.GetExecutor, func(_resourceType, _jobname string,
	) (*executor.NodeTaskExecutor, error) {
		return &executor.NodeTaskExecutor{}, nil
	})
	patches.ApplyMethodFunc(&executor.NodeTaskExecutor{}, "FinishTask",
		func() {
			finishTask = true
		})

	handler := ImagePrePullJobHandler{logger: klog.Background()}
	err := handler.ReleaseExecutorConcurrent(taskmsg.Resource{})
	assert.NoError(t, err)
	assert.True(t, finishTask)
}
