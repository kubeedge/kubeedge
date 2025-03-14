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
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
)

func newMetaMessageHandler() *SimpleHandler {
	return &SimpleHandler{
		FilterFunc: func(msg *model.Message) bool {
			group := msg.GetGroup()
			return group == message.ResourceGroupName ||
				group == message.FuncGroupName
		},
		ProcessFunc: func(msg *model.Message, _clientHub clients.Adapter) error {
			if msg.GetParentID() != "" {
				beehiveContext.SendResp(*msg)
			} else {
				beehiveContext.SendToGroup(modules.MetaGroup, *msg)
			}
			return nil
		},
	}
}
