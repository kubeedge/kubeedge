package listener

import (
	"sync"
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
	var mu sync.Mutex
	var calls int
	MessageHandlerInstance.SetCallback("once", func(*model.Message) {
		mu.Lock()
		calls++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := MessageHandlerInstance.HandleMessage(newReply("once")); err != nil {
				t.Errorf("HandleMessage returned err: %v", err)
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("callback invoked %d times, want 1", calls)
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
