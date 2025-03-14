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
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/clients"
)

// Handler handler different messages
type Handler interface {
	Filter(message *model.Message) bool
	Process(message *model.Message, clientHub clients.Adapter) error
}

// SimpleHandler a simple handler implementation, just implement the function fields
type SimpleHandler struct {
	FilterFunc  func(message *model.Message) bool
	ProcessFunc func(message *model.Message, clientHub clients.Adapter) error
}

func (h *SimpleHandler) Filter(message *model.Message) bool {
	return h.FilterFunc(message)
}

func (h *SimpleHandler) Process(message *model.Message, client clients.Adapter) error {
	return h.ProcessFunc(message, client)
}

var handlers []Handler

// RegisterHandler registers message handlers of EdgeHub.
func RegisterHandlers() {
	handlers = []Handler{
		newMetaMessageHandler(),
		newTwinMessageHandler(),
		newBusMessageHandler(),
		newTaskMessageHandler(),
	}
}

// ProcessHandler return true if handler filtered
func ProcessHandler(message model.Message, client clients.Adapter) error {
	for _, handle := range handlers {
		klog.V(2).Infof("try to match the message, source: %s, group: %s, resource: %s",
			message.GetSource(), message.GetGroup(), message.GetResource())
		if handle.Filter(&message) {
			klog.V(2).Infof("matched the %T, start to process", handle)
			if err := handle.Process(&message, client); err != nil {
				return fmt.Errorf("failed to handle message, message group: %s, error: %+v", message.GetGroup(), err)
			}
			return nil
		}
	}
	return fmt.Errorf("failed to handle message, no handler found for the message, message group: %s", message.GetGroup())
}
