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

package conn

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// dialTestConn starts a websocket server with the given handler and returns a
// client connection to it together with the server for cleanup.
func dialTestConn(t *testing.T, handler http.HandlerFunc) (*websocket.Conn, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		server.Close()
		t.Fatalf("failed to dial test server: %v", err)
	}
	return wsConn, server
}

func newTestWSConn(base *websocket.Conn, readDeadline time.Duration, onErr func(string, string)) *WSConnection {
	return NewWSConn(&ConnectionOptions{
		ConnType: api.ProtocolTypeWS,
		ConnUse:  api.UseTypeMessage,
		Base:     base,
		State: &ConnectionState{
			State:   api.StatConnected,
			Headers: http.Header{},
		},
		ReadDeadline:       readDeadline,
		OnReadTransportErr: onErr,
	})
}

// TestWSConnectionReadDeadlineHalfOpen verifies that when the peer stops
// responding (no data and no pong), the read deadline fires and the read loop
// tears the connection down promptly instead of blocking until the kernel TCP
// timeout.
func TestWSConnectionReadDeadlineHalfOpen(t *testing.T) {
	upgrader := websocket.Upgrader{}
	// The server upgrades and then goes silent: it never reads, so gorilla never
	// answers the client's pings with a pong, emulating a half-open connection.
	handler := func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		time.Sleep(5 * time.Second)
	}
	wsConn, server := dialTestConn(t, handler)
	defer server.Close()

	var once sync.Once
	fired := make(chan struct{})
	conn := newTestWSConn(wsConn, time.Second, func(string, string) {
		once.Do(func() { close(fired) })
	})
	conn.ServeConn()

	select {
	case <-fired:
		// detected within the read deadline, as expected
	case <-time.After(4 * time.Second):
		t.Fatal("read deadline did not fire; half-open connection was not detected")
	}
	if conn.state.State != api.StatDisconnected {
		t.Errorf("connection state = %q, want %q", conn.state.State, api.StatDisconnected)
	}
}

// TestWSConnectionReadDeadlineHealthy verifies that a responsive peer keeps the
// connection alive across several read-deadline windows via ping/pong, so the
// connection is not torn down while it is healthy.
func TestWSConnectionReadDeadlineHealthy(t *testing.T) {
	upgrader := websocket.Upgrader{}
	// The server keeps reading, so gorilla answers the client's pings with pongs.
	handler := func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}
	wsConn, server := dialTestConn(t, handler)
	defer server.Close()

	var once sync.Once
	fired := make(chan struct{})
	// Use a generous read deadline so the ping/pong round-trip and goroutine
	// scheduling have enough slack to keep this test stable under CI load.
	conn := newTestWSConn(wsConn, 3*time.Second, func(string, string) {
		once.Do(func() { close(fired) })
	})
	conn.ServeConn()

	select {
	case <-fired:
		t.Fatal("connection was torn down while the peer was responsive")
	case <-time.After(5 * time.Second):
		// stayed alive past the read-deadline window via ping/pong, as expected
	}
	_ = conn.Close()
}
