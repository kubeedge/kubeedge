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

func TestCallbackIgnoresInvalidCallbackType(t *testing.T) {
	MessageHandlerInstance.callbackHandlers.Store("invalid", "not a callback")

	if err := MessageHandlerInstance.HandleMessage(newReply("invalid")); err != nil {
		t.Fatalf("HandleMessage returned err: %v", err)
	}
	if _, exist := MessageHandlerInstance.callbackHandlers.Load("invalid"); exist {
		t.Fatal("invalid callback was left registered")
	}
}
