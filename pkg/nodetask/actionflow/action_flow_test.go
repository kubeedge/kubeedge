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
	confirm := waitConfirm.Next(true)
	require.Equal(t, string(v1alpha2.NodeUpgradeJobActionConfirm), confirm.Name)
	require.Nil(t, confirm.Next(false))
	backUp := confirm.Next(true)
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
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionConfirm)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionBackUp)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionUpgrade)))
	require.NotNil(t, FlowNodeUpgradeJob.Find(string(v1alpha2.NodeUpgradeJobActionRollBack)))
	require.Nil(t, FlowNodeUpgradeJob.Find("unknown"))
}
