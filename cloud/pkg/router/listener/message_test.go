package listener

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func newReply(parentID string) *model.Message {
	msg := model.NewMessage(parentID)
	msg.Header.ParentID = parentID
	return msg
}

func TestCallbackDoesNotBlockOnAbandonedReceiver(t *testing.T) {
	respCh := make(chan *model.Message, 1)
	MessageHandlerInstance.SetCallback("abandoned", func(message *model.Message) {
		select {
		case respCh <- message:
		default:
		}
	})
	respCh <- newReply("filler")

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := MessageHandlerInstance.HandleMessage(newReply("abandoned")); err != nil {
			t.Errorf("HandleMessage returned err: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("HandleMessage blocked, router dispatch loop would stall")
	}
}

func TestCallbackIsInvokedOnlyOnce(t *testing.T) {
	var calls int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	MessageHandlerInstance.SetCallback("once", func(*model.Message) {
		atomic.AddInt32(&calls, 1)
		// Hold the first delivery inside the callback so the second reply is
		// dispatched while the registration would still be present if lookup
		// and removal were not atomic.
		select {
		case entered <- struct{}{}:
			<-release
		default:
		}
	})

	var wg sync.WaitGroup
	deliver := func() {
		defer wg.Done()
		if err := MessageHandlerInstance.HandleMessage(newReply("once")); err != nil {
			t.Errorf("HandleMessage returned err: %v", err)
		}
	}
	wg.Add(1)
	go deliver()
	<-entered

	wg.Add(1)
	go deliver()
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("callback invoked %d times, want 1", got)
	}
}

func TestDelCallbackPreventsInvocation(t *testing.T) {
	called := make(chan struct{}, 1)
	MessageHandlerInstance.SetCallback("deleted", func(*model.Message) {
		called <- struct{}{}
	})
	MessageHandlerInstance.DelCallback("deleted")

	if err := MessageHandlerInstance.HandleMessage(newReply("deleted")); err != nil {
		t.Fatalf("HandleMessage returned err: %v", err)
	}
	select {
	case <-called:
		t.Fatal("callback ran after DelCallback")
	default:
	}
}
