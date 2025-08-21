package edgehub

import (
	"container/heap"
	"sync"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"k8s.io/klog/v2"
)

// Priority levels for messages
const (
	PriorityEmergency = 0 // Emergency message: heartbeat, health check
	PriorityImportant = 1 // Important message: event notification, task dispatch
	PriorityNormal    = 2 // Normal message: data synchronization, resource request
	PriorityLow       = 3 // Low message: log, status report
)

// PriorityMessage wraps a message with priority information
type PriorityMessage struct {
	Message  *model.Message
	Priority int
	Index    int // used by heap.Interface
	Created  time.Time
}

// PriorityQueue implements heap.Interface
type PriorityQueue []*PriorityMessage

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	if pq[i].Priority != pq[j].Priority {
		return pq[i].Priority < pq[j].Priority
	}
	// If priorities are equal, older messages come first
	return pq[i].Created.Before(pq[j].Created)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PriorityMessage)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// MessagePriorityQueue manages messages with priority
type MessagePriorityQueue struct {
	queue *PriorityQueue
	mutex sync.RWMutex
}

// NewMessagePriorityQueue creates a new priority queue
func NewMessagePriorityQueue() *MessagePriorityQueue {
	pq := &PriorityQueue{}
	heap.Init(pq)
	return &MessagePriorityQueue{
		queue: pq,
	}
}

// Push adds a message to the priority queue
func (mpq *MessagePriorityQueue) Push(message *model.Message, priority int) {
	mpq.mutex.Lock()
	defer mpq.mutex.Unlock()

	priorityMsg := &PriorityMessage{
		Message:  message,
		Priority: priority,
		Created:  time.Now(),
	}

	heap.Push(mpq.queue, priorityMsg)
	klog.V(4).Infof("Added message to priority queue with priority %d, queue size: %d", priority, mpq.queue.Len())
}

// Pop removes and returns the highest priority message
func (mpq *MessagePriorityQueue) Pop() *model.Message {
	mpq.mutex.Lock()
	defer mpq.mutex.Unlock()

	if mpq.queue.Len() == 0 {
		return nil
	}

	priorityMsg := heap.Pop(mpq.queue).(*PriorityMessage)
	klog.V(4).Infof("Popped message with priority %d, queue size: %d", priorityMsg.Priority, mpq.queue.Len())
	return priorityMsg.Message
}

// Size returns the current size of the queue
func (mpq *MessagePriorityQueue) Size() int {
	mpq.mutex.RLock()
	defer mpq.mutex.RUnlock()
	return mpq.queue.Len()
}

// IsEmpty checks if the queue is empty
func (mpq *MessagePriorityQueue) IsEmpty() bool {
	return mpq.Size() == 0
}

// GetPriorityForMessage determines the priority for a given message
func GetPriorityForMessage(message *model.Message) int {
	// Check if it's a heartbeat message
	if message.GetOperation() == "keepalive" {
		return PriorityEmergency
	}

	// Check if it's an ACK response message
	if message.GetOperation() == "response" {
		return PriorityImportant // ACK response is important
	}

	// Check message source and operation for other priorities
	switch message.GetSource() {
	case "eventbus":
		return PriorityImportant
	case "metamanager":
		return PriorityNormal
	default:
		return PriorityLow
	}
}
