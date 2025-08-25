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
package priority

import (
	"container/heap"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
)

// MessagePriorityQueue is a min-heap ordering of beehive model.Message by Priority (lower is higher).
// Same priority preserves FIFO by sequence number.
type MessagePriorityQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	heap    messageHeap
	stopped bool
	seq     int64
}

type messageItem struct {
	msg      model.Message
	priority int32
	seq      int64
	index    int
}

type messageHeap []*messageItem

func (h messageHeap) Len() int { return len(h) }
func (h messageHeap) Less(i, j int) bool {
	if h[i].priority == h[j].priority {
		return h[i].seq < h[j].seq
	}
	return h[i].priority < h[j].priority
}
func (h messageHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *messageHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*messageItem)
	item.index = n
	*h = append(*h, item)
}
func (h *messageHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}

func NewMessagePriorityQueue() *MessagePriorityQueue {
	q := &MessagePriorityQueue{}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.heap)
	return q
}

func (q *MessagePriorityQueue) Add(msg model.Message) {
	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		klog.Warning("add to stopped message priority queue")
		return
	}
	q.seq++
	heap.Push(&q.heap, &messageItem{msg: msg, priority: msg.GetPriority(), seq: q.seq})
	q.cond.Signal()
	q.mu.Unlock()
}

func (q *MessagePriorityQueue) Get() (model.Message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.heap.Len() == 0 && !q.stopped {
		q.cond.Wait()
	}
	if q.stopped {
		return model.Message{}, false
	}
	item := heap.Pop(&q.heap).(*messageItem)
	return item.msg, true
}

func (q *MessagePriorityQueue) Close() {
	q.mu.Lock()
	q.stopped = true
	q.mu.Unlock()
	q.cond.Broadcast()
}
