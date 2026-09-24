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
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

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

// TestMessageFifo_Close_ReleasesBlockedGet covers the reason Close exists: a
// caller already parked in Get has to be released, otherwise the reader of a
// torn-down connection never returns.
func TestMessageFifo_Close_ReleasesBlockedGet(t *testing.T) {
	f := NewMessageFifo()

	released := make(chan error, 1)
	go func() {
		var msg model.Message
		released <- f.Get(&msg)
	}()

	select {
	case <-released:
		t.Fatal("Get returned before Close")
	case <-time.After(50 * time.Millisecond):
	}

	f.Close()

	select {
	case err := <-released:
		assert.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("Get is still blocked after Close")
	}
}

// TestMessageFifo_Close_ConcurrentWithPut guards against closing the message
// channel: Put runs on a connection read loop while Close runs on the
// reconnect path, so a Close that closed the channel would panic with "send on
// closed channel" when the two overlap.
func TestMessageFifo_Close_ConcurrentWithPut(t *testing.T) {
	assert.NotPanics(t, func() {
		for i := 0; i < 100; i++ {
			f := NewMessageFifo()
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					f.Put(&model.Message{Header: model.MessageHeader{ID: "concurrent"}})
				}
			}()
			go func() {
				defer wg.Done()
				f.Close()
			}()
			wg.Wait()
		}
	})
}

// TestMessageFifo_Close_DrainsMessageDeliveredAtClose covers the interleaving
// where a message lands while the close happens and a Get is already parked:
// both the message and the close are then ready, so Get has to prefer the
// message rather than letting the select pick at random.
func TestMessageFifo_Close_DrainsMessageDeliveredAtClose(t *testing.T) {
	// The window is between the reader's drain attempt and its blocking
	// select: when Put and Close both complete inside it, the select sees
	// both cases ready. The reader is started without a delay so it is often
	// still in that window, and the interleaving is hunted by repetition
	// rather than by sleeping; against a Get without the recheck this fails
	// on every run within a few thousand iterations.
	for i := 0; i < 100000; i++ {
		f := NewMessageFifo()

		got := make(chan error, 1)
		var received model.Message
		go func() {
			got <- f.Get(&received)
		}()

		f.Put(&model.Message{Header: model.MessageHeader{ID: "at-close"}})
		f.Close()

		if err := <-got; err != nil {
			t.Fatalf("iteration %d: Get reported the close and dropped a message that arrived before it: %v", i, err)
		}
		assert.Equal(t, "at-close", received.Header.ID)
	}
}

// TestMessageFifo_Put_DropsAfterClose covers the fast path: once the fifo is
// closed, a Put must not enqueue even if there is capacity, otherwise a Get
// can deliver a post-close message instead of the close error.
func TestMessageFifo_Put_DropsAfterClose(t *testing.T) {
	f := NewMessageFifo()
	f.Close()

	for i := 0; i < 100; i++ {
		f.Put(&model.Message{Header: model.MessageHeader{ID: "late"}})
	}

	var msg model.Message
	assert.Error(t, f.Get(&msg))
	assert.Equal(t, 0, len(f.fifo))
}

// TestMessageFifo_Put_ReturnsWhenClosedDuringOverflow guards the overflow path
// against a close that lands while producers are already inside it. After the
// close nothing drains the buffer, and a producer that dropped the oldest
// message can lose the freed slot to another producer (one per QUIC stream);
// its second send would then block forever and leak the stream's read
// goroutine. Producers are churning through the overflow path when Close
// happens, and a refiller keeps the buffer full from then on so the lost-slot
// interleaving is reliable.
func TestMessageFifo_Put_ReturnsWhenClosedDuringOverflow(t *testing.T) {
	// Every overflow logs a warning and this test overflows tens of
	// thousands of times, which would bury the rest of the -v output.
	klog.LogToStderr(false)
	klog.SetOutput(io.Discard)
	t.Cleanup(func() {
		klog.Flush()
		klog.SetOutput(os.Stderr)
		klog.LogToStderr(true)
	})

	for round := 0; round < 50; round++ {
		f := NewMessageFifo()
		for i := 0; i < comm.MessageFiFoSizeMax; i++ {
			f.Put(&model.Message{Header: model.MessageHeader{ID: "fill"}})
		}

		var wg sync.WaitGroup
		for p := 0; p < 4; p++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 500; j++ {
					f.Put(&model.Message{Header: model.MessageHeader{ID: "late"}})
				}
			}()
		}
		time.Sleep(time.Millisecond) // let the producers reach the overflow path

		stop := make(chan struct{})
		go func() { // steals every slot the producers free from now on
			filler := model.Message{Header: model.MessageHeader{ID: "stolen"}}
			for {
				select {
				case <-stop:
					return
				case f.fifo <- filler:
				default:
				}
			}
		}()
		f.Close()

		finished := make(chan struct{})
		go func() {
			wg.Wait()
			close(finished)
		}()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Fatalf("round %d: a producer is still blocked in Put after Close", round)
		}
		close(stop)
	}
}

// TestMessageFifo_Close_DrainsBufferedMessages documents that a close does not
// discard messages that were already delivered into the fifo.
func TestMessageFifo_Close_DrainsBufferedMessages(t *testing.T) {
	f := NewMessageFifo()
	f.Put(&model.Message{Header: model.MessageHeader{ID: "buffered"}})
	f.Close()

	var msg model.Message
	assert.NoError(t, f.Get(&msg))
	assert.Equal(t, "buffered", msg.Header.ID)

	assert.Error(t, f.Get(&msg))
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
