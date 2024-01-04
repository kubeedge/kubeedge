/*
Copyright 2023 The KubeEdge Authors.

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

package util

import (
	"fmt"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func ReportTaskResult(taskType, taskID string, resp types.NodeTaskResponse) {
	msg := model.NewMessage("").SetRoute(modules.EdgeHubModuleName, modules.HubGroup).
		SetResourceOperation(fmt.Sprintf("task/%s/node/%s", taskID, resp.NodeName), taskType).FillBody(resp)
	beehiveContext.Send(modules.EdgeHubModuleName, *msg)
}
