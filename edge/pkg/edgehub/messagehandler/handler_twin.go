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
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
)

// newTwinMessageHandler returns a SimpleHandler for twin messages (TwinGroup).
// It filters messages by group and dispatches them to the TwinGroup module.
func newTwinMessageHandler() *SimpleHandler {
	return &SimpleHandler{
		// FilterFunc determines whether the message belongs to the twinmanager group.
		FilterFunc: func(msg *model.Message) bool {
			return msg.GetGroup() == modules.TwinGroup
		},
		// ProcessFunc processes the matched message and dispatches it to TwinGroup.
		ProcessFunc: func(message *model.Message, _clientHub clients.Adapter) error {
			beehiveContext.SendToGroup(modules.TwinGroup, *message)
			return nil
		},
	}
}
