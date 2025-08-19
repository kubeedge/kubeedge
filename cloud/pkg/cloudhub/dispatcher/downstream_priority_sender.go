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
package dispatcher

import (
	"time"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/priority"
	chmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"k8s.io/klog/v2"
)

// runDownstreamPrioritySender pops from queue and processes like original DispatchDownstream body
func (md *messageDispatcher) runDownstreamPrioritySender() {
	for {
		select {
		case <-beehivecontext.Done():
			klog.Warning("CloudHub downstream sender stopped")
			return
		default:
		}
		msg, ok := md.downstreamPQ.Get()
		if !ok {
			return
		}

		nodeID, err := GetNodeID(&msg)
		if nodeID == "" || err != nil {
			klog.Warningf("node id is not found in the message: %+v", msg)
			continue
		}
		if !chmodel.IsToEdge(&msg) {
			klog.Warningf("skip message not to edge node %s: %+v", nodeID, msg)
			continue
		}

		// backpressure: keep ACK/NO-ACK queues shallow so priority ordering is preserved at the feeder
		nodeMessagePool := md.GetNodeMessagePool(nodeID)
		if noAckRequired(&msg) {
			if nodeMessagePool.NoAckMessageQueue.Len() >= noAckQueueHighWatermark {
				// requeue and wait briefly to let downstream drain
				md.downstreamPQ.Add(msg)
				time.Sleep(10 * time.Millisecond)
				continue
			}
			md.enqueueNoAckMessage(nodeID, &msg)
			continue
		}
		if nodeMessagePool.AckMessageQueue.Len() >= ackQueueHighWatermark {
			// requeue and wait briefly to let downstream drain
			md.downstreamPQ.Add(msg)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		md.enqueueAckMessage(nodeID, &msg)
	}
}

// queue depth high watermarks for backpressure control
const (
	ackQueueHighWatermark   = 20
	noAckQueueHighWatermark = 10
)

// priority queue wrapper
type prioritySendQueueCloud struct{ *priority.MessagePriorityQueue }

func newPrioritySendQueueCloud() *prioritySendQueueCloud {
	return &prioritySendQueueCloud{MessagePriorityQueue: priority.NewMessagePriorityQueue()}
}
