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
package edgehub

import (
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/priority"
)

// runPrioritySender drains sendPQ and writes to cloud
func (eh *EdgeHub) runPrioritySender() {
	for {
		select {
		case <-eh.sendPQStop:
			return
		default:
		}
		msg, ok := eh.sendPQ.Get()
		if !ok {
			return
		}
		// throttle per message id
		_ = eh.tryThrottle(msg.GetID())
		if err := eh.sendToCloud(msg); err != nil {
			klog.Errorf("failed to send message to cloud: %v", err)
			// let reconnect handle errors
			return
		}
	}
}

// priority queue wrapper
type prioritySendQueue struct{ *priority.MessagePriorityQueue }

func newPrioritySendQueue() *prioritySendQueue {
	return &prioritySendQueue{MessagePriorityQueue: priority.NewMessagePriorityQueue()}
}
