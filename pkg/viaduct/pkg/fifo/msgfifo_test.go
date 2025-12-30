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

package fifo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/comm"
)

func TestNewMessageFifo(t *testing.T) {
	f := NewMessageFifo()
	assert.NotNil(t, f)
	assert.NotNil(t, f.fifo)
	assert.Equal(t, comm.MessageFiFoSizeMax, cap(f.fifo))
}

func TestMessageFifo_Put_Get(t *testing.T) {
	f := NewMessageFifo()
	msg := &model.Message{
		Header: model.MessageHeader{
			ID: "test-msg",
		},
	}

	// Test Put
	f.Put(msg)
	assert.Equal(t, 1, len(f.fifo))

	// Test Get
	var receivedMsg model.Message
	err := f.Get(&receivedMsg)
	assert.NoError(t, err)
	assert.Equal(t, msg.Header.ID, receivedMsg.Header.ID)
	assert.Equal(t, 0, len(f.fifo))
}

func TestMessageFifo_Overflow(t *testing.T) {
	// Since MessageFiFoSizeMax might be large (100), we don't want to fill it all in a simple test if it's too large,
	// but here it's 100, which is manageable.
	// However, to strictly test the "discard old message" logic, we need to fill it up.

	f := NewMessageFifo()

	// Fill the fifo
	for i := 0; i < comm.MessageFiFoSizeMax; i++ {
		f.Put(&model.Message{Header: model.MessageHeader{ID: "old"}})
	}
	assert.Equal(t, comm.MessageFiFoSizeMax, len(f.fifo))

	// Put one more, should trigger discard old
	newMsg := &model.Message{Header: model.MessageHeader{ID: "new"}}
	f.Put(newMsg)

	assert.Equal(t, comm.MessageFiFoSizeMax, len(f.fifo))

	// The first message we get should be "old" (since we discarded one "old" but there are still SizeMax-1 "old" ones before the "new" one?)
	// Wait, the logic is:
	// select {
	// case f.fifo <- *msg:
	// default:
	//    <-f.fifo (removes oldest)
	//    f.fifo <- *msg (adds new)
	// }
	// So if capacity is 100, we put 100.
	// Put 101th: removes 1st, adds 101th.
	// So the queue should now contain: 2nd, 3rd ... 100th, 101th.

	// Let's drain the first SizeMax-1 messages
	var receivedMsg model.Message
	for i := 0; i < comm.MessageFiFoSizeMax-1; i++ {
		err := f.Get(&receivedMsg)
		assert.NoError(t, err)
		assert.Equal(t, "old", receivedMsg.Header.ID)
	}

	// The last one should be "new"
	err := f.Get(&receivedMsg)
	assert.NoError(t, err)
	assert.Equal(t, "new", receivedMsg.Header.ID)
}

func TestMessageFifo_Close(t *testing.T) {
	f := NewMessageFifo()
	f.Close()

	// Verify channel is closed
	_, ok := <-f.fifo
	assert.False(t, ok)

	// Verify Get returns error on closed fifo
	var msg model.Message
	err := f.Get(&msg)
	assert.Error(t, err)
	assert.Equal(t, "the fifo is broken", err.Error())

	// Verify Close is idempotent (safe to call multiple times)
	assert.NotPanics(t, func() {
		f.Close()
	})
}

func TestMessageFifo_Get_Blocking(t *testing.T) {
	f := NewMessageFifo()
	msg := &model.Message{Header: model.MessageHeader{ID: "async"}}

	done := make(chan struct{})
	go func() {
		defer close(done)
		var receivedMsg model.Message
		err := f.Get(&receivedMsg)
		assert.NoError(t, err)
		assert.Equal(t, "async", receivedMsg.Header.ID)
	}()

	// Ensure Get is blocked by checking that done is not closed quickly
	select {
	case <-done:
		t.Fatal("Get returned before Put")
	case <-time.After(50 * time.Millisecond):
		// This is expected: Get should be blocking
	}

	f.Put(msg)

	// Now it should unblock
	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Get did not return after Put")
	}
}
