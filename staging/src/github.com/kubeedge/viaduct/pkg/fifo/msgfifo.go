/*
Copyright 2019 The KubeEdge Authors.

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

package fifo

import (
	"fmt"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/comm"
)

type MessageFifo struct {
	fifo chan model.Message
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
