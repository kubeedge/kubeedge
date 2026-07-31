/*
Copyright 2026 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0
*/

package conn

import (
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

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
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
