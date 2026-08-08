/*
Copyright 2026 The KubeEdge Authors.

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

package messagelayer

import (
	"testing"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	nodetaskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	"github.com/stretchr/testify/require"
)

func TestBuildNodeTaskRouter(t *testing.T) {
	resource := nodetaskmsg.Resource{
		APIVersion:   "operations.kubeedge.io/v1alpha2",
		ResourceType: "nodeupgradejobs",
		JobName:      "upgrade-job-1",
		NodeName:     "node-1",
	}
	opr := "upgrade"

	msg := BuildNodeTaskRouter(resource, opr)

	require.NotNil(t, msg)
	require.Equal(t, modules.TaskManagerModuleName, msg.GetSource())
	require.Equal(t, modules.TaskManagerModuleGroup, msg.GetGroup())
	require.Equal(t, resource.String(), msg.GetResource())
	require.Equal(t, opr, msg.GetOperation())
}
