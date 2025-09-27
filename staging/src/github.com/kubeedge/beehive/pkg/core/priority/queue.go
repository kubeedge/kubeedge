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
	"time"

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

	// aging config
	agingEnabled  bool
	agingInterval time.Duration
	nextAgingAt   time.Time

	// now is for testability
	now func() time.Time
}

type messageItem struct {
	msg          model.Message
	priority     int32
	basePriority int32
	seq          int64
	index        int
	enqueuedAt   time.Time
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
	q.now = time.Now
	return q
}

// EnableAging turns on anti-starvation aging with the given interval.
// Each full interval waited promotes a message by one priority level, capped at highest.
func (q *MessagePriorityQueue) EnableAging(interval time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.agingEnabled = interval > 0
	q.agingInterval = interval
	if q.agingEnabled && q.nextAgingAt.IsZero() {
		q.nextAgingAt = q.now().Add(q.agingInterval)
	}
}

// setNowFunc is only for tests in this package.
func (q *MessagePriorityQueue) setNowFunc(now func() time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.now = now
}

func (q *MessagePriorityQueue) Add(msg model.Message) {
	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		klog.Warning("add to stopped message priority queue")
		return
	}
	q.seq++
	now := q.now()
	item := &messageItem{
		msg:          msg,
		priority:     msg.GetPriority(),
		basePriority: msg.GetPriority(),
		seq:          q.seq,
		enqueuedAt:   now,
	}
	heap.Push(&q.heap, item)
	if q.agingEnabled && (q.nextAgingAt.IsZero() || now.Add(q.agingInterval).Before(q.nextAgingAt)) {
		q.nextAgingAt = now.Add(q.agingInterval)
	}
	q.cond.Signal()
	q.mu.Unlock()
}

func (q *MessagePriorityQueue) applyAgingLocked(now time.Time) {
	if !q.agingEnabled || q.heap.Len() == 0 {
		return
	}
	if now.Before(q.nextAgingAt) {
		return
	}
	// Promote items by the number of full intervals they have waited.
	updated := false
	for i, it := range q.heap {
		waited := now.Sub(it.enqueuedAt)
		if waited <= 0 {
			continue
		}
		steps := int(waited / q.agingInterval)
		if steps <= 0 {
			continue
		}
		newPriority := it.basePriority - int32(steps)
		if newPriority < model.PriorityUrgent {
			newPriority = model.PriorityUrgent
		}
		if newPriority != it.priority {
			it.priority = newPriority
			heap.Fix(&q.heap, i)
			updated = true
		}
	}
	if updated {
		q.nextAgingAt = now.Add(q.agingInterval)
	} else {
		// No updates; schedule next check soon to avoid tight loops.
		q.nextAgingAt = now.Add(q.agingInterval)
	}
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
	if q.agingEnabled {
		q.applyAgingLocked(q.now())
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
