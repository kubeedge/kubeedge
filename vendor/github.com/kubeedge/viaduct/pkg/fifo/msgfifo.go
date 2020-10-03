package fifo

import (
	"fmt"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/comm"
)

type MessageFifo struct {
	fifo      chan model.Message
	closeOnce sync.Once
}

// set the fifo capacity to MessageFiFoSizeMax
func NewMessageFifo() *MessageFifo {
	return &MessageFifo{
		fifo: make(chan model.Message, comm.MessageFiFoSizeMax),
	}
}

// Put put the message into fifo
func (f *MessageFifo) Put(msg *model.Message) {
	select {
	case f.fifo <- *msg:
	default:
		// discard the old message
		<-f.fifo
		// push into fifo
		f.fifo <- *msg
		klog.Warning("too many message received, fifo overflow")
	}
}

// Get get message from fifo
// this api is blocked when the fifo is empty
func (f *MessageFifo) Get(msg *model.Message) error {
	var ok bool
	*msg, ok = <-f.fifo
	if !ok {
		return fmt.Errorf("the fifo is broken")
	}
	return nil
}

func (f *MessageFifo) Close() {
	f.closeOnce.Do(func() {
		close(f.fifo)
	})
}
