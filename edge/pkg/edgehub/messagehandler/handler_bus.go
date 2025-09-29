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
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
)

// newBusMessageHandler returns a SimpleHandler for bus messages (eventbus/servicebus).
// It filters messages by group and dispatches them to the appropriate module based on the source.
func newBusMessageHandler() *SimpleHandler {
	return &SimpleHandler{
		//FilterFunc determines whether the message belongs to the user group (bus).
		FilterFunc: func(msg *model.Message) bool {
			return msg.GetGroup() == message.UserGroupName
		},
		// ProcessFunc processes the matched message and dispatches it to EventBus or ServiceBus.
		ProcessFunc: func(msg *model.Message, _clientHub clients.Adapter) error {
			if msg.GetParentID() != "" {
				beehiveContext.SendResp(*msg)
				return nil
			}
			switch msg.GetSource() {
			case "router_eventbus":
				beehiveContext.Send(modules.EventBusModuleName, *msg)
			case "router_servicebus":
				beehiveContext.Send(modules.ServiceBusModuleName, *msg)
			case "streamrule_endpoint":
				beehiveContext.Send(modules.StreamRuleEndpointModuleName, *msg)
			default:
				klog.Warningf("unsupported bus message source '%s', message lost", msg.GetSource())
			}
			return nil
		},
	}
}
