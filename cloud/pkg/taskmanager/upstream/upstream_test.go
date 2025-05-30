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
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/executor"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(handleUpstreamMessage, func(
		handler UpstreamHandler,
		upmsg taskmsg.UpstreamMessage,
		res taskmsg.Resource,
	) error {
		assert.Equal(t, res.ResourceType, "imageprepulljob")
		assert.Equal(t, res.JobName, "test-job")
		assert.Equal(t, res.NodeName, "node1")
		assert.Equal(t, upmsg.Action, string(operationsv1alpha2.ImagePrePullJobActionPull))
		assert.True(t, upmsg.Succ)
		assert.NotNil(t, handler)
		wg.Done()
		return nil
	})

	Init(ctx)
	statusChan := make(chan model.Message)
	Start(ctx, statusChan)
	statusChan <- model.Message{
		Router: model.MessageRoute{
			Resource: "operations.kubeedge.io/v1alpha2/imageprepulljob/test-job/node/node1",
		},
		Content: taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			Succ:   true,
		},
	}
	wg.Wait()
}

func TestHandleUpstreamMessage(t *testing.T) {
	res := taskmsg.Resource{
		APIVersion:   "operations.kubeedge.io/v1alpha2",
		ResourceType: "imageprepulljob",
		JobName:      "test-job",
		NodeName:     "node1",
	}
	handler := &ImagePrePullJobHandler{
		logger: klog.Background(),
	}

	var releaseExecutorCalled bool
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(releaseExecutorConcurrent, func(_res taskmsg.Resource) error {
		releaseExecutorCalled = true
		return nil
	})
	patches.ApplyMethodFunc(reflect.TypeOf(handler), "UpdateNodeTaskStatus",
		func(_jobName, _nodeName string, _isFinalAction bool, _upmsg taskmsg.UpstreamMessage) error {
			return nil
		})

	t.Run("invalid action", func(t *testing.T) {
		upmsg := taskmsg.UpstreamMessage{
			Action: "invalid",
			Succ:   true,
		}
		err := handleUpstreamMessage(handler, upmsg, res)
		require.ErrorContains(t, err, "invalid imageprepulljob action invalid")
	})

	t.Run("not final action", func(t *testing.T) {
		releaseExecutorCalled = false
		upmsg := taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionCheck),
			Succ:   true,
		}
		err := handleUpstreamMessage(handler, upmsg, res)
		require.NoError(t, err)
		assert.False(t, releaseExecutorCalled)
	})

	t.Run("is final action", func(t *testing.T) {
		releaseExecutorCalled = false
		upmsg := taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			Succ:   true,
		}
		err := handleUpstreamMessage(handler, upmsg, res)
		require.NoError(t, err)
		assert.True(t, releaseExecutorCalled)
	})
}

func TestReleaseExecutorConcurrent(t *testing.T) {
	var finishTaskCalled bool

	globpatches := gomonkey.NewPatches()
	defer globpatches.Reset()

	globpatches.ApplyFunc(executor.GetExecutor, func(resourceType, _jobname string,
	) (*executor.NodeTaskExecutor, error) {
		switch resourceType {
		case "fake":
			return &executor.NodeTaskExecutor{}, nil
		case "wantError":
			return nil, errors.New("test error")
		default:
			return nil, executor.ErrExecutorNotExists
		}
	})

	globpatches.ApplyMethodFunc(reflect.TypeOf((*executor.NodeTaskExecutor)(nil)),
		"FinishTask", func() {
			finishTaskCalled = true
		})

	t.Run("get executor failed", func(t *testing.T) {
		finishTaskCalled = false
		res := taskmsg.Resource{
			APIVersion:   "operations.kubeedge.io/v1alpha2",
			ResourceType: "wantError",
			JobName:      "test-job",
			NodeName:     "node1",
		}
		err := releaseExecutorConcurrent(res)
		require.ErrorContains(t, err, "failed to get executor")
	})

	t.Run("executor not exists", func(t *testing.T) {
		res := taskmsg.Resource{
			APIVersion:   "operations.kubeedge.io/v1alpha2",
			ResourceType: "notExists",
			JobName:      "test-job",
			NodeName:     "node1",
		}
		err := releaseExecutorConcurrent(res)
		require.NoError(t, err)
		assert.False(t, finishTaskCalled)
	})

	t.Run("finish task", func(t *testing.T) {
		res := taskmsg.Resource{
			APIVersion:   "operations.kubeedge.io/v1alpha2",
			ResourceType: "fake",
			JobName:      "test-job",
			NodeName:     "node1",
		}
		err := releaseExecutorConcurrent(res)
		require.NoError(t, err)
		assert.True(t, finishTaskCalled)
	})
}
