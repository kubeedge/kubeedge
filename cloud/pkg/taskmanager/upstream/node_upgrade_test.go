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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/status"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

func TestNodeUpgradeJobUpdateNodeTaskStatus(t *testing.T) {
	var (
		jobName  = "test-job"
		nodeNmae = "node1"
	)
	t.Run("final action successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		gomonkey.ApplyFunc(status.GetNodeUpgradeJobStatusUpdater, func() *status.StatusUpdater {
			return &status.StatusUpdater{}
		})
		gomonkey.ApplyMethodFunc(reflect.TypeOf(&status.StatusUpdater{}), "UpdateStatus",
			func(opts status.UpdateStatusOptions) {
				act, ok := opts.ActionStatus.(*operationsv1alpha2.NodeUpgradeJobActionStatus)
				require.True(t, ok)
				assert.Equal(t, operationsv1alpha2.NodeUpgradeJobActionUpgrade, act.Action)
				assert.Equal(t, operationsv1alpha2.NodeTaskPhaseSuccessful, opts.Phase)
				assert.Equal(t, jobName, opts.JobName)
				assert.Equal(t, nodeNmae, opts.NodeName)
				require.NotNil(t, opts.Callback)
				opts.Callback(nil)
			})

		handler := &NodeUpgradeJobHandler{}
		err := handler.UpdateNodeTaskStatus(jobName, nodeNmae, true, taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.NodeUpgradeJobActionUpgrade),
			Succ:   true,
		})
		require.NoError(t, err)
	})

	t.Run("final action failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		gomonkey.ApplyFunc(status.GetNodeUpgradeJobStatusUpdater, func() *status.StatusUpdater {
			return &status.StatusUpdater{}
		})
		gomonkey.ApplyMethodFunc(reflect.TypeOf(&status.StatusUpdater{}), "UpdateStatus",
			func(opts status.UpdateStatusOptions) {
				act, ok := opts.ActionStatus.(*operationsv1alpha2.NodeUpgradeJobActionStatus)
				require.True(t, ok)
				assert.Equal(t, operationsv1alpha2.NodeUpgradeJobActionRollBack, act.Action)
				assert.Equal(t, operationsv1alpha2.NodeTaskPhaseFailure, opts.Phase)
				assert.Equal(t, jobName, opts.JobName)
				assert.Equal(t, nodeNmae, opts.NodeName)
				require.NotNil(t, opts.Callback)
				opts.Callback(nil)
			})

		handler := &NodeUpgradeJobHandler{}
		err := handler.UpdateNodeTaskStatus(jobName, nodeNmae, true, taskmsg.UpstreamMessage{
			Action: string(operationsv1alpha2.NodeUpgradeJobActionRollBack),
			Succ:   false,
		})
		require.NoError(t, err)
	})
}
