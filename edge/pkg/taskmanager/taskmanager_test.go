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

package taskmanager

import (
	"context"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	taskmgrv1alpha1 "github.com/kubeedge/kubeedge/edge/pkg/taskmanager/v1alpha1"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestStart(t *testing.T) {
	var postHubConnectedCalled, v1alpha1RunTaskCalled, v1alpha2RunTaskCalled bool

	globpatches := gomonkey.NewPatches()
	defer globpatches.Reset()

	globpatches.ApplyFunc(ReportUpgradeStatus, func(_ctx context.Context) error {
		postHubConnectedCalled = true
		return nil
	})
	globpatches.ApplyFunc(taskmgrv1alpha1.RunTask, func(_msg *model.Message) error {
		v1alpha1RunTaskCalled = true
		return nil
	})
	globpatches.ApplyFunc(RunTask, func(_msg *model.Message) error {
		v1alpha2RunTaskCalled = true
		return nil
	})

	t.Run("hook post connected when edgehub connected", func(t *testing.T) {
		postHubConnectedCalled, v1alpha1RunTaskCalled, v1alpha2RunTaskCalled = false, false, false
		ctx, cancel := context.WithCancel(context.TODO())
		msgchan := make(chan model.Message, 1)
		defer close(msgchan)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(beehiveContext.GetContext, func() context.Context {
			return ctx
		})
		patches.ApplyFunc(beehiveContext.Receive, func(_module string) (model.Message, error) {
			defer cancel()
			msg := model.NewMessage("").
				SetRoute(message.SourceNodeConnection, modules.TaskManagerGroup)
			return *msg, nil
		})

		tm := &TaskManager{}
		tm.Start()

		assert.True(t, postHubConnectedCalled)
		assert.False(t, v1alpha1RunTaskCalled)
		assert.False(t, v1alpha2RunTaskCalled)
	})

	t.Run("call the RunTask of v1alpha2", func(t *testing.T) {
		postHubConnectedCalled, v1alpha1RunTaskCalled, v1alpha2RunTaskCalled = false, false, false
		ctx, cancel := context.WithCancel(context.TODO())
		msgchan := make(chan model.Message, 1)
		defer close(msgchan)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(beehiveContext.GetContext, func() context.Context {
			return ctx
		})
		patches.ApplyFunc(beehiveContext.Receive, func(_module string) (model.Message, error) {
			defer cancel()

			msgres := taskmsg.Resource{
				APIVersion:   operationsv1alpha2.SchemeGroupVersion.String(),
				ResourceType: operationsv1alpha2.ResourceNodeUpgradeJob,
				JobName:      "test-job",
				NodeName:     "test-node",
			}
			msg := model.NewMessage("").
				SetRoute(modules.TaskManagerGroup, modules.TaskManagerGroup).
				SetResourceOperation(msgres.String(), "upgrade")
			return *msg, nil
		})

		tm := &TaskManager{}
		tm.Start()
		// Wait for the RunTask to be called.
		time.Sleep(100 * time.Millisecond)

		assert.False(t, postHubConnectedCalled)
		assert.False(t, v1alpha1RunTaskCalled)
		assert.True(t, v1alpha2RunTaskCalled)
	})

	t.Run("call the RunTask of v1alpha1", func(t *testing.T) {
		postHubConnectedCalled, v1alpha1RunTaskCalled, v1alpha2RunTaskCalled = false, false, false
		ctx, cancel := context.WithCancel(context.TODO())
		msgchan := make(chan model.Message, 1)
		defer close(msgchan)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(beehiveContext.GetContext, func() context.Context {
			return ctx
		})
		patches.ApplyFunc(beehiveContext.Receive, func(_module string) (model.Message, error) {
			defer cancel()

			msg := model.NewMessage("").
				SetRoute(modules.TaskManagerGroup, modules.TaskManagerGroup).
				SetResourceOperation("/nodeupgrade/xxx/node/xxx", "upgrade")
			return *msg, nil
		})

		tm := &TaskManager{}
		tm.Start()
		// Wait for the RunTask to be called.
		time.Sleep(100 * time.Millisecond)

		assert.False(t, postHubConnectedCalled)
		assert.True(t, v1alpha1RunTaskCalled)
		assert.False(t, v1alpha2RunTaskCalled)
	})
}
