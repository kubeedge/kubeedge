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

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/taskmanager/actions"
)

func TestRunTask(t *testing.T) {
	globalPatches := gomonkey.NewPatches()
	defer globalPatches.Reset()

	globalPatches.ApplyFunc(options.GetEdgeCoreConfig, func() *v1alpha2.EdgeCoreConfig {
		return v1alpha2.NewDefaultEdgeCoreConfig()
	})

	t.Run("runner is nil", func(t *testing.T) {
		msg := model.NewMessage("").
			SetRoute(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup).
			SetResourceOperation("operations.kubeedge.io/v1alpha2/unknown/taskname/nodes/node1", "")
		err := RunTask(msg)
		require.ErrorContains(t, err, "invalid resource type unknown")
	})

	t.Run("process successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(actions.GetRunner, func(_ string) *actions.ActionRunner {
			return &actions.ActionRunner{}
		})
		patches.ApplyMethodFunc(&actions.ActionRunner{}, "RunAction",
			func(_ctx context.Context, _jobname, _nodename, _action string, _specData []byte) {})

		msg := model.NewMessage("").
			SetRoute(modules.TaskManagerModuleName, modules.TaskManagerModuleGroup).
			SetResourceOperation("operations.kubeedge.io/v1alpha2/fakeRunner/taskname/nodes/node1", "")
		err := RunTask(msg)
		require.NoError(t, err)
	})
}
