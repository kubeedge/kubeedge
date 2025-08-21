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

package messagehandler

import (
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	cloudmodules "github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
)

// newTaskMessageHandler returns a SimpleHandler for task messages (taskmanager group).
// It filters messages by group and dispatches them to the TaskManager module.
func newTaskMessageHandler() *SimpleHandler {
	return &SimpleHandler{
		// FilterFunc determines whether the message belongs to the taskmanager group.
		FilterFunc: func(msg *model.Message) bool {
			return msg.GetGroup() == cloudmodules.TaskManagerModuleName
		},
		// ProcessFunc processes the matched message and dispatches it to TaskManager.
		ProcessFunc: func(msg *model.Message, _clientHub clients.Adapter) error {
			beehiveContext.Send(modules.TaskManagerModuleName, *msg)
			return nil
		},
	}
}
