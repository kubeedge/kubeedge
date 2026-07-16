package conn

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSConnection_handleRawData_CopiesData(t *testing.T) {
	const payload = "hello from peer"

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server failed to upgrade: %v", err)
			return
		}
		defer peerConn.Close()

		if err := peerConn.WriteMessage(websocket.BinaryMessage, []byte(payload)); err != nil {
			t.Errorf("server failed to write message: %v", err)
			return
		}
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}
	defer clientConn.Close()

	var consumer bytes.Buffer
	c := &WSConnection{
		wsConn:    clientConn,
		state:     &ConnectionState{},
		consumer:  &consumer,
		autoRoute: true,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		c.handleRawData()
	}()

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		t.Fatal("handleRawData did not return; it may be blocked or the fix regressed")
	}

	if got := consumer.String(); got != payload {
		t.Fatalf("consumer got %q, want %q", got, payload)
	}
}
