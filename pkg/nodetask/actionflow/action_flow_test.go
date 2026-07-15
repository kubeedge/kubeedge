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

package actionflow

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/operations/v1alpha2"
)

func TestNodeUpgradeActionFlow(t *testing.T) {
	check := FlowNodeUpgradeJob.First
	require.Equal(t, string(v1alpha2.NodeUpgradeJobActionCheck), check.Name)
	require.Nil(t, FlowNodeUpgradeJob.First.Next(false))
	waitConfirm := check.Next(true)
	require.Equal(t, string(v1alpha2.NodeUpgradeJobActionWaitingConfirmation), waitConfirm.Name)
	require.Nil(t, waitConfirm.Next(false))
	backUp := waitConfirm.Next(true)
	require.Equal(t, string(v1alpha2.NodeUpgradeJobActionBackUp), backUp.Name)
	require.Nil(t, backUp.Next(false))
	upgrade := backUp.Next(true)
	require.Equal(t, string(v1alpha2.NodeUpgradeJobActionUpgrade), upgrade.Name)
	require.Nil(t, upgrade.Next(true))
	rollback := upgrade.Next(false)
	require.Equal(t, string(v1alpha2.NodeUpgradeJobActionRollBack), rollback.Name)
}

func TestFound(t *testing.T) {
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionCheck)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionWaitingConfirmation)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionBackUp)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionUpgrade)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionRollBack)))
	require.Nil(t, FlowNodeUpgradeJob.Find("unknown"))
}

func TestImagePrePullActionFlow(t *testing.T) {
	check := FlowImagePrePullJob.First
	require.Equal(t, string(v1alpha2.ImagePrePullJobActionCheck), check.Name)
	require.Nil(t, check.Next(false))
	pulls := check.Next(true)
	require.Equal(t, string(v1alpha2.ImagePrePullJobActionPull), pulls.Name)
	require.Nil(t, pulls.Next(true))
	require.Nil(t, pulls.Next(false))
}

func TestConfigUpdateActionFlow(t *testing.T) {
	check := FlowConfigUpdateJob.First
	require.Equal(t, string(v1alpha2.ConfigUpdateJobActionCheck), check.Name)
	require.Nil(t, check.Next(false))
	backUp := check.Next(true)
	require.Equal(t, string(v1alpha2.ConfigUpdateJobActionBackUp), backUp.Name)
	require.Nil(t, backUp.Next(false))
	update := backUp.Next(true)
	require.Equal(t, string(v1alpha2.ConfigUpdateJobActionUpdate), update.Name)
	require.Nil(t, update.Next(true))
	rollback := update.Next(false)
	require.Equal(t, string(v1alpha2.ConfigUpdateJobActionRollBack), rollback.Name)
}

func TestFoundAllFlows(t *testing.T) {
	// ImagePrePullJob
	require.NotNil(t, FlowImagePrePullJob.Find(string(v1alpha2.ImagePrePullJobActionCheck)))
	require.NotNil(t, FlowImagePrePullJob.Find(string(v1alpha2.ImagePrePullJobActionPull)))
	require.Nil(t, FlowImagePrePullJob.Find("unknown"))

	// ConfigUpdateJob -- RollBack hangs off NextFailure; this was the broken case
	require.NotNil(t, FlowConfigUpdateJob.Find(string(v1alpha2.ConfigUpdateJobActionCheck)))
	require.NotNil(t, FlowConfigUpdateJob.Find(string(v1alpha2.ConfigUpdateJobActionBackUp)))
	require.NotNil(t, FlowConfigUpdateJob.Find(string(v1alpha2.ConfigUpdateJobActionUpdate)))
	require.NotNil(t, FlowConfigUpdateJob.Find(string(v1alpha2.ConfigUpdateJobActionRollBack)))
	require.Nil(t, FlowConfigUpdateJob.Find("unknown"))
}

func TestFindWithNilFlow(t *testing.T) {
	var f *Flow
	require.Nil(t, f.Find("anything"))
}

func TestFindWithEmptyFlow(t *testing.T) {
	f := &Flow{}
	require.Nil(t, f.Find("anything"))
}
