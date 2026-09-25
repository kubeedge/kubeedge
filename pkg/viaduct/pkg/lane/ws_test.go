/*
Copyright 2026 The KubeEdge Authors.

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

package lane

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newWSPair returns a connected server/client websocket pair backed by an
// httptest server; both are closed on cleanup.
func newWSPair(t *testing.T) (server, client *websocket.Conn) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverChan := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverChan <- c
	}))
	t.Cleanup(srv.Close)

	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	select {
	case s := <-serverChan:
		t.Cleanup(func() { _ = s.Close() })
		return s, c
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for the server side of the websocket")
		return nil, nil
	}
}

// TestWSLaneReadReturnsDeadlineTimeout verifies that an expired read deadline
// surfaces as a net.Error timeout from Read (the designed half-open detection
// path, reported by the caller rather than logged here).
func TestWSLaneReadReturnsDeadlineTimeout(t *testing.T) {
	_, client := newWSPair(t)
	if err := client.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}

	_, err := NewWSLane(client).Read(make([]byte, 16))
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected a timeout net.Error, got %v", err)
	}
}

// TestWSLaneReadPropagatesPeerClose verifies that a close initiated by the
// peer is returned to the caller as a non-timeout error.
func TestWSLaneReadPropagatesPeerClose(t *testing.T) {
	server, client := newWSPair(t)
	if err := server.Close(); err != nil {
		t.Fatalf("server close: %v", err)
	}

	_, err := NewWSLane(client).Read(make([]byte, 16))
	if err == nil {
		t.Fatal("expected an error after the peer closed the connection")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		t.Fatalf("peer close must not be reported as a timeout: %v", err)
	}
}

// TestWSLaneReadDelivers verifies the happy path used by the tests above:
// a frame written by the peer is returned from Read.
func TestWSLaneReadDelivers(t *testing.T) {
	server, client := newWSPair(t)
	if err := server.WriteMessage(websocket.BinaryMessage, []byte("hello")); err != nil {
		t.Fatalf("server write: %v", err)
	}

	buf := make([]byte, 16)
	n, err := NewWSLane(client).Read(buf)
	if err != nil || n != len("hello") {
		t.Fatalf("Read = (%d, %v), want (%d, nil)", n, err, len("hello"))
	}
}
