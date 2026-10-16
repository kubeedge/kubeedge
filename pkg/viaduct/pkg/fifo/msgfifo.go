package fifo

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/comm"
)

type MessageFifo struct {
	fifo   chan model.Message
	mu     sync.Mutex
	closed bool
}

// set the fifo capacity to MessageFiFoSizeMax
func NewMessageFifo() *MessageFifo {
	return &MessageFifo{
		fifo: make(chan model.Message, comm.MessageFiFoSizeMax),
	}
}

// Put put the message into fifo
//
// Put runs on the connection's read loop while Close runs on the teardown
// path, so the two are serialized by the lock: a Put that starts after Close
// is dropped instead of sending on a closed channel.
func (f *MessageFifo) Put(msg *model.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return
	}
	// Only Put sends, and it holds the lock, so a free slot seen here stays
	// free until the send below. The drop is non-blocking because Get may
	// empty the fifo meanwhile.
	if len(f.fifo) == cap(f.fifo) {
		select {
		case <-f.fifo:
			klog.Warning("too many message received, fifo overflow")
		default:
		}
	}
	f.fifo <- *msg
}

// Get get message from fifo
// this api is blocked when the fifo is empty
//
// Get takes no lock on purpose: a channel receive is safe against concurrent
// Put and Close, and blocking under f.mu would stall both.
func (f *MessageFifo) Get(msg *model.Message) error {
	var ok bool
	*msg, ok = <-f.fifo
	if !ok {
		return fmt.Errorf("the fifo is broken")
	}
	return nil
}

// Close releases the callers blocked in Get. Messages already buffered are
// still delivered by Get before it reports the close.
func (f *MessageFifo) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return
	}
	f.closed = true
	close(f.fifo)
}
