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
	"os"
	"strconv"
	"time"

	"k8s.io/klog/v2"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/priority"
	model "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"

	"github.com/kubeedge/kubeedge/pkg/features"
)

// runDownstreamPrioritySenderForNode pops from per-node queue and enqueues into that node's ack/noack
func (md *messageDispatcher) runDownstreamPrioritySenderForNode(nodeID string, pq *prioritySendQueueCloud, stop <-chan struct{}) {
	for {
		select {
		case <-beehivecontext.Done():
			klog.Warningf("CloudHub downstream sender stopped for node %s", nodeID)
			return
		case <-stop:
			klog.V(4).Infof("CloudHub downstream sender stop signal for node %s", nodeID)
			return
		default:
		}
		msg, ok := pq.Get()
		if !ok {
			return
		}
		// safety: double-check routing and nodeID
		mid, err := GetNodeID(&msg)
		if mid == "" || err != nil {
			klog.Warningf("node id not found in message (expected %s): %+v", nodeID, msg)
			continue
		}
		if mid != nodeID {
			// put back to the right node queue if mismatch
			other := md.getOrCreateNodePQ(mid)
			other.Add(msg)
			continue
		}
		if !model.IsToEdge(&msg) {
			klog.V(4).Infof("skip message not to edge node %s: %+v", nodeID, msg)
			continue
		}

		// backpressure: keep ACK/NO-ACK queues shallow so priority ordering is preserved at the feeder
		nodeMessagePool := md.GetNodeMessagePool(nodeID)
		if noAckRequired(&msg) {
			if nodeMessagePool.NoAckMessageQueue.Len() >= noAckQueueHighWatermark {
				pq.Add(msg)
				time.Sleep(10 * time.Millisecond)
				continue
			}
			md.enqueueNoAckMessage(nodeID, &msg)
			continue
		}
		if nodeMessagePool.AckMessageQueue.Len() >= ackQueueHighWatermark {
			pq.Add(msg)
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
	pq := &prioritySendQueueCloud{MessagePriorityQueue: priority.NewMessagePriorityQueue()}
	if features.DefaultFeatureGate.Enabled(features.PriorityQueueAging) {
		interval := 2 * time.Second
		if s := os.Getenv("CLOUDHUB_PRIORITY_AGING_INTERVAL_SEC"); s != "" {
			if v, err := strconv.Atoi(s); err == nil && v > 0 {
				interval = time.Duration(v) * time.Second
			}
		}
		pq.EnableAging(interval)
	}
	return pq
}
