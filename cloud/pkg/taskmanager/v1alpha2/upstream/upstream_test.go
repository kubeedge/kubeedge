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
	"reflect"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha2/wrap"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(updateNodeJobTaskStatus, func(res taskmsg.Resource,
		upmsg taskmsg.UpstreamMessage, handler UpstreamHandler,
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

func TestUpdateNodeJobTaskStatus(t *testing.T) {
	res := taskmsg.Resource{
		APIVersion:   "operations.kubeedge.io/v1alpha2",
		ResourceType: "imageprepulljob",
		JobName:      "test-job",
		NodeName:     "node1",
	}
	handler := &ImagePrePullJobHandler{
		logger: klog.Background(),
	}

	t.Run("not final action", func(t *testing.T) {
		var updated bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(handler), "UpdateNodeTaskStatus",
			func(jobname string, nodetask wrap.NodeJobTask) error {
				assert.Equal(t, jobname, "test-job")
				assert.Equal(t, operationsv1alpha2.NodeTaskPhaseInProgress, nodetask.Phase())
				act, err := nodetask.Action()
				assert.NoError(t, err)
				assert.NotNil(t, act)
				// check -> pull
				assert.Equal(t, string(operationsv1alpha2.ImagePrePullJobActionPull), act.Name)
				updated = true
				return nil
			})

		upmsg := taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionCheck),
			Succ:   true,
		}
		err := updateNodeJobTaskStatus(res, upmsg, handler)
		assert.NoError(t, err)
		assert.True(t, updated)
	})

	t.Run("final action", func(t *testing.T) {
		var updated, releaseExecutor bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(handler), "UpdateNodeTaskStatus",
			func(jobname string, nodetask wrap.NodeJobTask) error {
				assert.Equal(t, jobname, "test-job")
				assert.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, nodetask.Phase())
				updated = true
				return nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf(handler), "ReleaseExecutorConcurrent",
			func(res taskmsg.Resource) error {
				releaseExecutor = true
				return nil
			})

		upmsg := taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.ImagePrePullJobActionPull),
			Succ:   true,
		}
		err := updateNodeJobTaskStatus(res, upmsg, handler)
		assert.NoError(t, err)
		assert.True(t, updated)
		assert.True(t, releaseExecutor)
	})
}
