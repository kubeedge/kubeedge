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

package taskv1alpha2

import (
	"context"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/taskv1alpha2/actions"
)

func TestFilter(t *testing.T) {
	cases := []struct {
		name string
		msg  *model.Message
		exp  bool
	}{
		{
			name: "different group",
			msg: model.NewMessage("").
				SetRoute(modules.RouterModuleName, modules.RouterGroupName),
			exp: false,
		},
		{
			name: "different resource",
			msg: model.NewMessage("").
				SetRoute(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup),
			exp: false,
		},
		{
			name: "match message",
			msg: model.NewMessage("").
				SetRoute(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup).
				SetResourceOperation("operations.kubeedge.io/v1alpha2", ""),
			exp: true,
		},
	}
	msghandler := NewMessageHandler()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := msghandler.Filter(c.msg)
			if got != c.exp {
				require.Equal(t, c.exp, got)
			}
		})
	}
}

func TestProcess(t *testing.T) {
	t.Run("runner is nil", func(t *testing.T) {
		msghandler := NewMessageHandler()
		msg := model.NewMessage("").
			SetRoute(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup).
			SetResourceOperation("operations.kubeedge.io/v1alpha2/unknow/taskname/nodes/node1", "")
		err := msghandler.Process(msg, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid resource type unknow")
	})

	t.Run("process successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(actions.GetRunner, func(_ string) *actions.ActionRunner {
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(&actions.ActionRunner{}, "RunAction",
			func(_ctx context.Context, _jobname, _nodename, _action string, _specData []byte) {})

		msghandler := NewMessageHandler()
		msg := model.NewMessage("").
			SetRoute(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup).
			SetResourceOperation("operations.kubeedge.io/v1alpha2/fakeRunner/taskname/nodes/node1", "")
		err := msghandler.Process(msg, nil)
		require.NoError(t, err)
	})
}
