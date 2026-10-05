/*
Copyright 2024 The KubeEdge Authors.

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
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/fifo"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/keeper"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsTestPair creates an in-process WebSocket client/server pair using httptest.
// It returns the server-side and client-side *websocket.Conn.
// A buffered channel is used to pass the server connection from the HTTP handler
// goroutine to the caller, eliminating the data race and the need for time.Sleep.
func wsTestPair(t *testing.T) (server, client *websocket.Conn) {
	t.Helper()
	serverChan := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		serverChan <- c
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	select {
	case s := <-serverChan:
		t.Cleanup(func() {
			_ = s.Close()
		})
		return s, c
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server WebSocket connection")
		return nil, nil
	}
}

// TestHandleRawDataDoesNotPanic is a regression test for the bug where
// handleRawData mistakenly used api.ProtocolTypeQuic instead of
// api.ProtocolTypeWS. NewLane(ProtocolTypeQuic, wsConn) returned nil because
// *websocket.Conn does not satisfy the quic.Stream interface, causing
// io.Copy to panic with a nil reader.
//
// The test verifies that:
//  1. handleRawData does not panic when called on a live WSConnection
//  2. data written by the remote peer is forwarded to the consumer
func TestHandleRawDataDoesNotPanic(t *testing.T) {
	serverConn, clientConn := wsTestPair(t)
	if serverConn == nil {
		t.Fatal("server websocket conn is nil")
	}

	payload := []byte("hello-raw-data")
	consumer := &bytes.Buffer{}

	wsConn := &WSConnection{
		wsConn:    serverConn,
		consumer:  consumer,
		autoRoute: true,
		state:     &ConnectionState{State: api.StatConnected},
	}

	// Run handleRawData in a goroutine — it blocks on io.Copy.
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The test passes if this does not panic.
		wsConn.handleRawData()
	}()

	// Send a binary frame from the client side; the server's WSLane.Read
	// will receive it and io.Copy forwards it to consumer.
	if err := clientConn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		t.Fatalf("client write failed: %v", err)
	}

	// Close the client so io.Copy on the server side gets EOF and returns.
	clientConn.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handleRawData did not return within timeout")
	}

	if !bytes.Equal(consumer.Bytes(), payload) {
		t.Errorf("unexpected payload: got %q, want %q", consumer.Bytes(), payload)
	}
}

// newTestWSConn spins up a server-side websocket and connects a client to
// it. With serverReads=true the server runs a normal read loop, which makes
// gorilla answer the client's pings with pongs automatically (an idle but
// healthy peer). With serverReads=false the server swallows bytes at the TCP
// level without WebSocket-level processing, so pings are never answered —
// emulating a stalled/half-open peer while keeping the TCP connection open.
// The returned WSConnection is configured with the supplied read deadline
// interval; the caller must close srv to release the goroutine.
func newTestWSConn(t *testing.T, readDeadlineInterval time.Duration, serverReads bool) (*WSConnection, *httptest.Server) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer c.Close()
		if serverReads {
			// Hold the connection without sending anything so the client
			// side blocks on Read; pings are auto-answered with pongs.
			for {
				if _, _, err := c.ReadMessage(); err != nil {
					return
				}
			}
		}
		// Stalled peer: consume raw bytes so the TCP connection stays
		// open but no pong (or any frame) is ever sent back.
		buf := make([]byte, 1024)
		for {
			if _, err := c.UnderlyingConn().Read(buf); err != nil {
				return
			}
		}
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}

	conn := &WSConnection{
		wsConn:               wsConn,
		state:                &ConnectionState{State: api.StatConnected, Headers: http.Header{}},
		connUse:              api.UseTypeMessage,
		messageFifo:          fifo.NewMessageFifo(),
		syncKeeper:           keeper.NewSyncKeeper(),
		readDeadlineInterval: readDeadlineInterval,
	}
	return conn, srv
}

// TestHandleMessageReadDeadlineFiresWithinInterval verifies the half-open
// detection: when readDeadlineInterval is set and the peer stops answering
// entirely (not even pongs), handleMessage exits within roughly that
// interval. Without the fix the goroutine would block until kernel TCP
// retransmission timeout (~15min).
func TestHandleMessageReadDeadlineFiresWithinInterval(t *testing.T) {
	conn, srv := newTestWSConn(t, 100*time.Millisecond, false)
	defer srv.Close()

	done := make(chan struct{})
	go func() {
		conn.handleMessage()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleMessage did not return within 2s; deadline not applied")
	}

	// messageFifo must be closed so callers blocked on Get() observe the
	// error immediately rather than waiting for the next keepalive failure.
	msg := &model.Message{}
	if err := conn.ReadMessage(msg); err == nil {
		t.Fatal("expected ReadMessage to return error after handleMessage timeout")
	}
}

// TestHandleMessageZeroReadDeadlineKeepsLegacyBehavior verifies that the
// existing zero-value behavior (no deadline = block forever) is preserved.
func TestHandleMessageZeroReadDeadlineKeepsLegacyBehavior(t *testing.T) {
	conn, srv := newTestWSConn(t, 0, true)
	defer srv.Close()

	done := make(chan struct{})
	go func() {
		conn.handleMessage()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("handleMessage returned unexpectedly with no read deadline")
	case <-time.After(300 * time.Millisecond):
		// expected: blocks indefinitely
	}

	// Cleanup: close the underlying conn to unblock the goroutine.
	_ = conn.wsConn.Close()
	<-done
}

// TestHandleMessagePingKeepsIdleConnectionAlive verifies that an idle but
// healthy connection is NOT torn down by the read deadline: pingLoop keeps
// sending pings, the peer answers with pongs, and each pong extends the
// deadline. Only after the peer stops answering does the deadline fire.
func TestHandleMessagePingKeepsIdleConnectionAlive(t *testing.T) {
	// Generous interval so that a scheduling hiccup on a loaded CI runner
	// (ping every interval/2, pong must arrive within interval) does not
	// fail the test spuriously.
	interval := 800 * time.Millisecond
	conn, srv := newTestWSConn(t, interval, true)
	defer srv.Close()

	done := make(chan struct{})
	go func() {
		conn.handleMessage()
		close(done)
	}()

	// Idle for several intervals: without pong-driven deadline extension
	// handleMessage would exit within ~interval.
	select {
	case <-done:
		t.Fatal("handleMessage exited on an idle but healthy connection; pings/pongs did not extend the deadline")
	case <-time.After(3 * interval):
		// expected: still alive
	}

	// Cleanup: close the conn to unblock handleMessage. (Deadline expiry on
	// a stalled peer is covered by
	// TestHandleMessageReadDeadlineFiresWithinInterval.)
	_ = conn.wsConn.Close()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleMessage did not return after the connection was closed")
	}
}

// TestSetReadDeadlinePropagatesToWSConn verifies the SetReadDeadline bug
// fix: the call must reach gorilla/websocket's underlying conn. We trigger
// this by setting a past deadline and observing that the next read errors
// out immediately.
func TestSetReadDeadlinePropagatesToWSConn(t *testing.T) {
	conn, srv := newTestWSConn(t, 0, true)
	defer srv.Close()
	// Close the client conn explicitly so the server-side handler goroutine
	// (looping on ReadMessage) exits even if srv.Close alone does not
	// terminate the hijacked websocket connection.
	defer conn.wsConn.Close()

	if err := conn.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}

	if _, _, err := conn.wsConn.ReadMessage(); err == nil {
		t.Fatal("expected immediate read error after past deadline; SetReadDeadline did not propagate")
	}
}

// TestNewWSConnWiresReadDeadlineInterval verifies the constructor path used in
// production (as opposed to the struct literals in the tests above): the
// interval configured on ConnectionOptions must reach the connection.
func TestNewWSConnWiresReadDeadlineInterval(t *testing.T) {
	_, client := wsTestPair(t)
	called := false
	conn := NewWSConn(&ConnectionOptions{
		Base:                 client,
		State:                &ConnectionState{State: api.StatConnected},
		ConnUse:              api.UseTypeMessage,
		AutoRoute:            true,
		OnReadTransportErr:   func(string, string) { called = true },
		ReadDeadlineInterval: 5 * time.Second,
	})

	if conn.readDeadlineInterval != 5*time.Second {
		t.Fatalf("readDeadlineInterval = %v, want 5s", conn.readDeadlineInterval)
	}
	if conn.wsConn != client || !conn.autoRoute || conn.connUse != api.UseTypeMessage {
		t.Fatal("NewWSConn did not carry over the basic options")
	}
	if conn.messageFifo == nil || conn.syncKeeper == nil {
		t.Fatal("NewWSConn must initialize messageFifo and syncKeeper")
	}
	conn.OnReadTransportErr("", "")
	if !called {
		t.Fatal("OnReadTransportErr was not wired")
	}
}

// TestPingLoopClosesConnOnTerminalWriteError verifies the terminal-error path:
// when the connection is already broken and the read loop has not stopped the
// ping loop, the ping loop closes the connection itself so the read side
// unblocks immediately. A 1ns interval also exercises the sub-2ns period guard.
func TestPingLoopClosesConnOnTerminalWriteError(t *testing.T) {
	_, client := wsTestPair(t)
	// Break the connection before the first ping so WriteControl fails with a
	// non-timeout error.
	_ = client.Close()

	conn := &WSConnection{wsConn: client, readDeadlineInterval: time.Nanosecond}
	stop := make(chan struct{})
	defer close(stop)

	done := make(chan struct{})
	go func() {
		conn.pingLoop(stop)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("pingLoop did not stop after a terminal write error")
	}
}

// wsTestPairSmallBuffers is wsTestPair with tiny kernel socket buffers and a
// server that never reads, so a large client write blocks inside gorilla's
// frame writer while holding its write lock.
func wsTestPairSmallBuffers(t *testing.T) (server, client *websocket.Conn) {
	t.Helper()
	serverChan := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		if tcp, ok := c.UnderlyingConn().(*net.TCPConn); ok {
			_ = tcp.SetReadBuffer(4096)
		}
		serverChan <- c
	}))
	t.Cleanup(srv.Close)

	dialer := websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			c, err := (&net.Dialer{}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if tcp, ok := c.(*net.TCPConn); ok {
				_ = tcp.SetWriteBuffer(4096)
			}
			return c, nil
		},
	}
	c, _, err := dialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	select {
	case s := <-serverChan:
		t.Cleanup(func() { _ = s.Close() })
		return s, c
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server WebSocket connection")
		return nil, nil
	}
}

// TestPingLoopKeepsRunningWhileWriteBlocked verifies the transient-timeout
// path: while a large data frame is blocked on the socket (peer not reading),
// gorilla holds its write lock, so WriteControl fails with a timeout. That must
// not stop the ping loop; liveness is judged by the read deadline, not by the
// ping writer.
func TestPingLoopKeepsRunningWhileWriteBlocked(t *testing.T) {
	server, client := wsTestPairSmallBuffers(t)

	// A frame far larger than the socket buffers blocks in the frame writer
	// until the peer reads or the connection is torn down.
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- client.WriteMessage(websocket.BinaryMessage, make([]byte, 8<<20))
	}()

	conn := &WSConnection{wsConn: client, readDeadlineInterval: 40 * time.Millisecond}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		conn.pingLoop(stop)
		close(done)
	}()

	// Several ping periods elapse with the write lock held; each WriteControl
	// times out and the loop must keep going.
	select {
	case <-done:
		t.Fatal("pingLoop stopped while the write lock was held; write timeouts must be tolerated")
	case <-time.After(200 * time.Millisecond):
	}

	// Tear down: closing the server unblocks the pending write with an error
	// and lets the ping loop exit through either the stop or the error path.
	_ = server.Close()
	close(stop)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("pingLoop did not exit after teardown")
	}
	select {
	case <-writeDone:
	case <-time.After(3 * time.Second):
		t.Fatal("blocked write did not return after the peer closed")
	}
}

// TestSetWriteDeadlinePropagatesToWSConn mirrors the SetReadDeadline test for
// the write side: a past deadline must make the next write fail immediately.
func TestSetWriteDeadlinePropagatesToWSConn(t *testing.T) {
	conn, srv := newTestWSConn(t, 0, true)
	defer srv.Close()
	defer conn.wsConn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("SetWriteDeadline: %v", err)
	}
	if err := conn.wsConn.WriteMessage(websocket.BinaryMessage, []byte("x")); err == nil {
		t.Fatal("expected immediate write error after past deadline; SetWriteDeadline did not propagate")
	}
}
