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
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lucas-clemente/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// wsUpgrader is a permissive upgrader used by the in-process test HTTP server.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newTestWSPair spins up an httptest server and returns a pair of
// *websocket.Conn: the server-side conn and the client-side conn.
// Both are closed when t finishes.
//
// The handler returns immediately after handing off the upgraded connection
// via a channel; the hijacked WebSocket remains valid after ServeHTTP returns.
// This avoids the goroutine leak caused by blocking on <-r.Context().Done()
// after the WebSocket connection has already been closed by the test.
// A separate error channel ensures an upgrade failure surfaces immediately
// rather than leaving the test waiting on a channel that is never closed.
func newTestWSPair(t *testing.T) (serverConn, clientConn *websocket.Conn) {
	t.Helper()

	connCh := make(chan *websocket.Conn, 1) // buffered: handler must not block
	errCh := make(chan error, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			errCh <- err
			return
		}
		// Hand off the connection and return immediately.
		// The hijacked WebSocket is still valid after ServeHTTP returns.
		connCh <- c
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	cConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err, "WebSocket client dial failed")

	// Wait for either a successful server-side upgrade or an error.
	var sConn *websocket.Conn
	select {
	case sConn = <-connCh:
	case upgradeErr := <-errCh:
		cConn.Close()
		t.Fatalf("server-side WebSocket upgrade failed: %v", upgradeErr)
	case <-time.After(5 * time.Second):
		cConn.Close()
		t.Fatal("timed out waiting for server-side WebSocket upgrade")
	}

	t.Cleanup(func() {
		if cConn != nil {
			_ = cConn.Close()
		}
		if sConn != nil {
			_ = sConn.Close()
		}
	})

	return sConn, cConn
}

// ---------------------------------------------------------------------------
// fakeQuicStream — minimal quic.Stream implementation for constructor tests.
// A real QUIC network is not required; we only need to pass the type check
// inside NewQuicLane / NewLane.
// ---------------------------------------------------------------------------

type fakeQuicStream struct{}

func (fakeQuicStream) StreamID() quic.StreamID           { return 0 }
func (fakeQuicStream) Read(_ []byte) (int, error)        { return 0, io.EOF }
func (fakeQuicStream) Write(p []byte) (int, error)       { return len(p), nil }
func (fakeQuicStream) Close() error                      { return nil }
func (fakeQuicStream) CancelWrite(_ quic.ErrorCode) error { return nil }
func (fakeQuicStream) CancelRead(_ quic.ErrorCode) error  { return nil }
func (fakeQuicStream) Context() context.Context           { return context.Background() }
func (fakeQuicStream) SetReadDeadline(_ time.Time) error  { return nil }
func (fakeQuicStream) SetWriteDeadline(_ time.Time) error { return nil }
func (fakeQuicStream) SetDeadline(_ time.Time) error      { return nil }

// ---------------------------------------------------------------------------
// TestNewLane
// ---------------------------------------------------------------------------

// TestNewLane_WebSocket verifies that NewLane returns a *WSLaneWithoutPack for
// the WebSocket protocol constant.
func TestNewLane_WebSocket(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewLane(api.ProtocolTypeWS, clientConn)

	assert.NotNil(t, l)
	_, ok := l.(*WSLaneWithoutPack)
	assert.True(t, ok, "expected *WSLaneWithoutPack for ProtocolTypeWS")
}

// TestNewLane_Quic verifies that NewLane returns a *QuicLane for the QUIC
// protocol constant.  A minimal fakeQuicStream is used so that no real QUIC
// network is required — the constructor only performs a type assertion.
func TestNewLane_Quic(t *testing.T) {
	l := NewLane(api.ProtocolTypeQuic, fakeQuicStream{})

	assert.NotNil(t, l)
	_, ok := l.(*QuicLane)
	assert.True(t, ok, "expected *QuicLane for ProtocolTypeQuic")
}

// TestNewLane_UnknownProtocol verifies that NewLane returns nil and does not
// panic for an unrecognised protocol string.
func TestNewLane_UnknownProtocol(t *testing.T) {
	l := NewLane("bogus-protocol", nil)
	assert.Nil(t, l)
}

// ---------------------------------------------------------------------------
// WSLaneWithoutPack – constructor
// ---------------------------------------------------------------------------

// TestNewWSLaneWithoutPack_Success verifies that a valid *websocket.Conn yields
// a non-nil *WSLaneWithoutPack with the connection stored on it.
func TestNewWSLaneWithoutPack_Success(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)

	require.NotNil(t, l)
	assert.Equal(t, clientConn, l.conn)
}

// TestNewWSLaneWithoutPack_BadType verifies that passing a non-*websocket.Conn
// value returns nil.
func TestNewWSLaneWithoutPack_BadType(t *testing.T) {
	l := NewWSLaneWithoutPack("not-a-conn")
	assert.Nil(t, l)
}

// ---------------------------------------------------------------------------
// WSLane – constructor
// ---------------------------------------------------------------------------

// TestNewWSLane_Success verifies that a valid *websocket.Conn yields a non-nil
// *WSLane.
func TestNewWSLane_Success(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLane(clientConn)

	require.NotNil(t, l)
	assert.Equal(t, clientConn, l.conn)
}

// TestNewWSLane_BadType verifies that passing a non-*websocket.Conn value
// returns nil.
func TestNewWSLane_BadType(t *testing.T) {
	l := NewWSLane(42)
	assert.Nil(t, l)
}

// ---------------------------------------------------------------------------
// WSLaneWithoutPack – ReadMessage / WriteMessage round trip
// ---------------------------------------------------------------------------

// TestWSLaneWithoutPack_RoundTrip writes a model.Message through the client
// lane and reads it back through the server lane, verifying that all header,
// routing, and content fields are preserved end-to-end.
func TestWSLaneWithoutPack_RoundTrip(t *testing.T) {
	serverConn, clientConn := newTestWSPair(t)

	clientLane := NewWSLaneWithoutPack(clientConn)
	serverLane := NewWSLaneWithoutPack(serverConn)

	want := &model.Message{
		Header: model.MessageHeader{
			ID:        "test-id-123",
			ParentID:  "parent-456",
			Timestamp: time.Now().UnixNano() / 1e6,
			Sync:      true,
		},
		Router: model.MessageRoute{
			Source:    "edge",
			Group:     "resource",
			Resource:  "node/status",
			Operation: "update",
		},
		Content: "hello-wsnopack-content",
	}

	// Write from client, read on server.
	require.NoError(t, clientLane.WriteMessage(want))

	got := &model.Message{}
	require.NoError(t, serverLane.ReadMessage(got))

	assert.Equal(t, want.GetID(), got.GetID())
	assert.Equal(t, want.GetParentID(), got.GetParentID())
	assert.Equal(t, want.Header.Timestamp, got.Header.Timestamp, "Timestamp must be preserved")
	assert.Equal(t, want.GetSource(), got.GetSource())
	assert.Equal(t, want.GetGroup(), got.GetGroup())
	assert.Equal(t, want.GetOperation(), got.GetOperation())
	assert.Equal(t, want.GetResource(), got.GetResource())
	assert.Equal(t, want.Header.Sync, got.Header.Sync, "Sync flag must be preserved")
	// Content is marshalled as JSON; compare the serialised form.
	wantContent, _ := want.GetContentData()
	gotContent, _ := got.GetContentData()
	assert.True(t, bytes.Equal(wantContent, gotContent), "Content mismatch: want %q, got %q", wantContent, gotContent)
}

// ---------------------------------------------------------------------------
// WSLane – WriteMessage / ReadMessage round trip (packed, protobuf)
// ---------------------------------------------------------------------------

// TestWSLane_RoundTrip writes a model.Message through WSLane (packer+protobuf)
// from the client and reads it back on the server, verifying that all header,
// routing, and content fields are preserved end-to-end.
func TestWSLane_RoundTrip(t *testing.T) {
	serverConn, clientConn := newTestWSPair(t)

	clientLane := NewWSLane(clientConn)
	serverLane := NewWSLane(serverConn)

	want := &model.Message{
		Header: model.MessageHeader{
			ID:        "ws-lane-id",
			ParentID:  "ws-parent",
			Timestamp: time.Now().UnixNano() / 1e6,
			Sync:      true,
		},
		Router: model.MessageRoute{
			Source:    "cloud",
			Group:     "device",
			Resource:  "device/twin",
			Operation: "patch",
		},
		Content: "hello-wslane-content",
	}

	require.NoError(t, clientLane.WriteMessage(want))

	got := &model.Message{}
	require.NoError(t, serverLane.ReadMessage(got))

	assert.Equal(t, want.GetID(), got.GetID())
	assert.Equal(t, want.GetParentID(), got.GetParentID())
	assert.Equal(t, want.Header.Timestamp, got.Header.Timestamp, "Timestamp must be preserved")
	assert.Equal(t, want.GetSource(), got.GetSource())
	assert.Equal(t, want.GetGroup(), got.GetGroup())
	assert.Equal(t, want.GetOperation(), got.GetOperation())
	assert.Equal(t, want.GetResource(), got.GetResource())
	assert.Equal(t, want.Header.Sync, got.Header.Sync, "Sync flag must be preserved")
	wantContent, _ := want.GetContentData()
	gotContent, _ := got.GetContentData()
	assert.True(t, bytes.Equal(wantContent, gotContent), "Content mismatch: want %q, got %q", wantContent, gotContent)
}

// ---------------------------------------------------------------------------
// SetReadDeadline / SetWriteDeadline – behavioral verification
// ---------------------------------------------------------------------------

// TestWSLaneWithoutPack_ReadDeadlineExpired verifies that setting an already-
// expired read deadline causes the next ReadMessage call to return a timeout
// error promptly.  ReadMessage is executed in a goroutine so that if deadline
// propagation regresses the test fails within ~1 s rather than blocking until
// the package-level go test timeout.
func TestWSLaneWithoutPack_ReadDeadlineExpired(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)
	// Set a deadline in the past — the connection should time out immediately.
	require.NoError(t, l.SetReadDeadline(time.Now().Add(-time.Second)))

	errCh := make(chan error, 1)
	go func() {
		errCh <- l.ReadMessage(&model.Message{})
	}()

	select {
	case err := <-errCh:
		require.Error(t, err, "ReadMessage must fail after an expired read deadline")
		var netErr interface{ Timeout() bool }
		require.True(t, errors.As(err, &netErr) && netErr.Timeout(),
			"error must satisfy Timeout()==true, got: %v", err)
	case <-time.After(time.Second):
		clientConn.Close()
		t.Fatal("ReadMessage did not return within 1 s after expired read deadline; deadline propagation may be broken")
	}
}

// TestWSLane_ReadDeadlineExpired verifies the same bounded behavior for the
// packed WSLane variant.
func TestWSLane_ReadDeadlineExpired(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLane(clientConn)
	require.NoError(t, l.SetReadDeadline(time.Now().Add(-time.Second)))

	errCh := make(chan error, 1)
	go func() {
		errCh <- l.ReadMessage(&model.Message{})
	}()

	select {
	case err := <-errCh:
		require.Error(t, err, "ReadMessage must fail after an expired read deadline")
		var netErr interface{ Timeout() bool }
		require.True(t, errors.As(err, &netErr) && netErr.Timeout(),
			"error must satisfy Timeout()==true, got: %v", err)
	case <-time.After(time.Second):
		clientConn.Close()
		t.Fatal("ReadMessage did not return within 1 s after expired read deadline; deadline propagation may be broken")
	}
}

// TestWSLaneWithoutPack_WriteDeadlineExpired verifies that setting an already-
// expired write deadline causes the next WriteMessage call to return a timeout
// error, proving the deadline was propagated to the underlying connection.
func TestWSLaneWithoutPack_WriteDeadlineExpired(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)
	require.NoError(t, l.SetWriteDeadline(time.Now().Add(-time.Second)))

	msg := &model.Message{Header: model.MessageHeader{ID: "deadline-test"}}
	err := l.WriteMessage(msg)
	require.Error(t, err, "WriteMessage must fail after an expired write deadline")
}

// TestWSLane_WriteDeadlineExpired verifies that setting an already-expired write
// deadline causes the next WriteMessage call to return a timeout error for the
// packed WSLane variant, proving deadline propagation to the underlying conn.
func TestWSLane_WriteDeadlineExpired(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLane(clientConn)
	require.NoError(t, l.SetWriteDeadline(time.Now().Add(-time.Second)))

	msg := &model.Message{Header: model.MessageHeader{ID: "wslane-deadline-test"}}
	err := l.WriteMessage(msg)
	require.Error(t, err, "WriteMessage must fail after an expired write deadline")
}

// TestWSLaneWithoutPack_SetReadDeadline_Zero verifies that setting a zero
// time.Time clears the read deadline on the underlying connection without error.
func TestWSLaneWithoutPack_SetReadDeadline_Zero(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)

	err := l.SetReadDeadline(time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, time.Time{}, l.readDeadline)
}

// ---------------------------------------------------------------------------
// Error-path coverage
// ---------------------------------------------------------------------------

// TestWSLaneWithoutPack_ReadAfterClose verifies that ReadMessage returns an
// error when the underlying WebSocket connection has been closed by the peer.
// A short read deadline bounds the test so it cannot block indefinitely if
// the close frame is delayed.
func TestWSLaneWithoutPack_ReadAfterClose(t *testing.T) {
	serverConn, clientConn := newTestWSPair(t)

	serverLane := NewWSLaneWithoutPack(serverConn)

	// Close the client side — the server should see an error on its next read.
	require.NoError(t, clientConn.Close())

	// Set a short read deadline so the test is deterministic: if the close
	// frame is delayed the read will time out rather than blocking forever.
	require.NoError(t, serverLane.SetReadDeadline(time.Now().Add(200*time.Millisecond)))

	msg := &model.Message{}
	err := serverLane.ReadMessage(msg)
	assert.Error(t, err, "ReadMessage must return an error after peer closes the connection")
}

// TestWSLaneWithoutPack_WriteAfterClose verifies that WriteMessage returns an
// error when the underlying WebSocket connection has been closed.
func TestWSLaneWithoutPack_WriteAfterClose(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)

	// Close the connection before writing.
	require.NoError(t, clientConn.Close())

	msg := &model.Message{Header: model.MessageHeader{ID: "closed-conn"}}
	err := l.WriteMessage(msg)
	assert.Error(t, err, "WriteMessage must return an error after the connection is closed")
}

// TestWSLane_ReadMalformedData verifies that WSLane.ReadMessage returns an
// error when the peer sends a binary frame that fails the packer's length-
// header protocol.  We set a short read deadline to bound the test duration
// in case the packer blocks waiting for further frames.
func TestWSLane_ReadMalformedData(t *testing.T) {
	serverConn, clientConn := newTestWSPair(t)

	serverLane := NewWSLane(serverConn)

	// Apply a short read deadline so the test cannot block indefinitely if
	// the packer happens to expect more data after the short frame.
	require.NoError(t, serverLane.SetReadDeadline(time.Now().Add(200*time.Millisecond)))

	// Send a binary frame with fewer bytes than the packer's 4-byte length
	// header.  The packer will return an error (short read or timeout).
	malformed := []byte{0x00, 0x00} // only 2 bytes – packer expects 4-byte header
	require.NoError(t, clientConn.WriteMessage(websocket.BinaryMessage, malformed))

	msg := &model.Message{}
	err := serverLane.ReadMessage(msg)
	assert.Error(t, err, "ReadMessage must return an error for malformed/short packed data")
}
