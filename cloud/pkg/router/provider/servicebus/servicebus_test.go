package servicebus

import (
	"net/http"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/listener"
)

func newReply(parentID string) *model.Message {
	msg := model.NewMessage(parentID)
	msg.Header.ParentID = parentID
	return msg
}

func TestNewResponseCallbackDeliversReply(t *testing.T) {
	respCh, callback := newResponseCallback("deliver")
	reply := newReply("deliver")

	callback(reply)

	select {
	case got := <-respCh:
		if got != reply {
			t.Fatalf("delivered %v, want %v", got, reply)
		}
	default:
		t.Fatal("no response delivered")
	}
}

func TestNewResponseCallbackDoesNotBlockWithoutReceiver(t *testing.T) {
	_, callback := newResponseCallback("abandoned")

	done := make(chan struct{})
	go func() {
		defer close(done)
		callback(newReply("abandoned"))
		callback(newReply("abandoned"))
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("callback blocked with no receiver on the response channel")
	}
}

func TestNewResponseCallbackDoesNotBlockDispatch(t *testing.T) {
	messageID := "dispatch"
	_, callback := newResponseCallback(messageID)
	listener.MessageHandlerInstance.SetCallback(messageID, callback)
	defer listener.MessageHandlerInstance.DelCallback(messageID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 2; i++ {
			if err := listener.MessageHandlerInstance.HandleMessage(newReply(messageID)); err != nil {
				t.Errorf("HandleMessage returned err: %v", err)
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("HandleMessage blocked, router dispatch loop would stall")
	}
}

func patchSession(t *testing.T, nodeName string) *gomonkey.Patches {
	t.Helper()
	mgr := session.NewSessionManager(1)
	mgr.NodeSessions.Store(nodeName, &session.NodeSession{})
	patches := gomonkey.ApplyFunc(cloudhub.GetSessionManager, func() (*session.Manager, error) {
		return mgr, nil
	})
	patches.ApplyFunc(beehiveContext.Send, func(string, model.Message) {})
	return patches
}

func targetData(messageID, nodeName string) map[string]interface{} {
	return map[string]interface{}{
		"messageID": messageID,
		"param":     "",
		"nodeName":  nodeName,
		"method":    http.MethodGet,
		"header":    http.Header{},
		"data":      []byte("{}"),
	}
}

func TestGoToTargetReturnsReply(t *testing.T) {
	patches := patchSession(t, "node1")
	defer patches.Reset()

	messageID := "gototarget-reply"
	sb := &ServiceBus{targetPath: "/api", servicePort: "8080"}
	stop := make(chan struct{})

	type result struct {
		resp interface{}
		err  error
	}
	results := make(chan result, 1)
	go func() {
		resp, err := sb.GoToTarget(targetData(messageID, "node1"), stop)
		results <- result{resp, err}
	}()

	deliver := time.NewTicker(10 * time.Millisecond)
	defer deliver.Stop()
	for {
		select {
		case got := <-results:
			if got.err != nil {
				t.Fatalf("GoToTarget returned err: %v", got.err)
			}
			if _, ok := got.resp.(*model.Message); !ok {
				t.Fatalf("GoToTarget returned %T, want *model.Message", got.resp)
			}
			return
		case <-deliver.C:
			if err := listener.MessageHandlerInstance.HandleMessage(newReply(messageID)); err != nil {
				t.Errorf("HandleMessage returned err: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("GoToTarget did not return after the reply was dispatched")
		}
	}
}

func TestGoToTargetReturnsOnStop(t *testing.T) {
	patches := patchSession(t, "node1")
	defer patches.Reset()

	sb := &ServiceBus{targetPath: "/api", servicePort: "8080"}
	stop := make(chan struct{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		resp, err := sb.GoToTarget(targetData("gototarget-stop", "node1"), stop)
		if err != nil {
			t.Errorf("GoToTarget returned err: %v", err)
		}
		if msg, ok := resp.(*model.Message); !ok || msg != nil {
			t.Errorf("GoToTarget returned %v, want a nil message", resp)
		}
	}()

	stop <- struct{}{}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("GoToTarget did not return after stop was signalled")
	}
}
