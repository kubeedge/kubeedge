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

package message

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
)

var (
	beehiveInitOnce  sync.Once
	edgeHubOnce      sync.Once
)

// initBeehive initializes the global beehive context exactly once per test binary.
func initBeehive() {
	beehiveInitOnce.Do(func() {
		beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	})
}

// registerEdgeHub registers EdgeHubModuleName in the context exactly once,
// so repeated test runs or parallel subtests do not panic on double registration.
func registerEdgeHub() {
	initBeehive()
	edgeHubOnce.Do(func() {
		module := &common.ModuleInfo{
			ModuleName: modules.EdgeHubModuleName,
			ModuleType: common.MsgCtxTypeChannel,
		}
		beehiveContext.AddModule(module)
		beehiveContext.AddModuleGroup(module.ModuleName, module.ModuleName)
	})
}

func TestReportTaskResult(t *testing.T) {
	assert := assert.New(t)
	registerEdgeHub()

	taskType := "upgrade"
	taskID := "task-123"
	resp := types.NodeTaskResponse{
		NodeName: "edge-node",
	}

	ReportTaskResult(taskType, taskID, resp)

	received, err := beehiveContext.Receive(modules.EdgeHubModuleName)
	assert.NoError(err)
	assert.NotNil(received)
	assert.Equal(modules.EdgeHubModuleName, received.GetSource())
	assert.Equal(modules.HubGroup, received.GetGroup())
	assert.Equal(fmt.Sprintf("task/%s/node/%s", taskID, resp.NodeName), received.GetResource())
	assert.Equal(taskType, received.GetOperation())
	assert.Equal(resp, received.GetContent())
}

func TestReportNodeTaskStatus(t *testing.T) {
	assert := assert.New(t)
	registerEdgeHub()

	res := taskmsg.Resource{
		JobName:  "my-job",
		NodeName: "my-node",
	}
	msgBody := taskmsg.UpstreamMessage{
		Action: "upgrade",
		Succ:   true,
	}

	ReportNodeTaskStatus(res, msgBody)

	received, err := beehiveContext.Receive(modules.EdgeHubModuleName)
	assert.NoError(err)
	assert.NotNil(received)
	assert.Equal(modules.EdgeHubModuleName, received.GetSource())
	assert.Equal(modules.HubGroup, received.GetGroup())
	assert.Equal(res.String(), received.GetResource())
	assert.Equal(taskmsg.OperationUpdateNodeActionStatus, received.GetOperation())
	assert.Equal(msgBody, received.GetContent())
}
