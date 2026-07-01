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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// wsUpgrader is a permissive upgrader used by the in-process test HTTP server.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newTestWSPair spins up an httptest server and returns a pair of *websocket.Conn:
// the server-side conn and the client-side conn.  Both are closed when t finishes.
func newTestWSPair(t *testing.T) (serverConn, clientConn *websocket.Conn) {
	t.Helper()

	var sConn *websocket.Conn
	ready := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade connection: %v", err)
			return
		}
		sConn = c
		close(ready)
		// Keep the handler alive until the connection is closed by the test.
		<-r.Context().Done()
	}))

	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	cConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)

	<-ready // wait until server has upgraded

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
// lane and reads it back through the server lane, verifying the payload is
// preserved end-to-end.
func TestWSLaneWithoutPack_RoundTrip(t *testing.T) {
	serverConn, clientConn := newTestWSPair(t)

	clientLane := NewWSLaneWithoutPack(clientConn)
	serverLane := NewWSLaneWithoutPack(serverConn)

	want := &model.Message{
		Header: model.MessageHeader{
			ID:        "test-id-123",
			ParentID:  "parent-456",
			Timestamp: time.Now().UnixNano() / 1e6,
		},
		Router: model.MessageRoute{
			Source:    "edge",
			Group:     "resource",
			Resource:  "node/status",
			Operation: "update",
		},
	}

	// Write from client, read on server.
	require.NoError(t, clientLane.WriteMessage(want))

	got := &model.Message{}
	require.NoError(t, serverLane.ReadMessage(got))

	assert.Equal(t, want.GetID(), got.GetID())
	assert.Equal(t, want.GetParentID(), got.GetParentID())
	assert.Equal(t, want.GetSource(), got.GetSource())
	assert.Equal(t, want.GetOperation(), got.GetOperation())
	assert.Equal(t, want.GetResource(), got.GetResource())
}

// ---------------------------------------------------------------------------
// WSLane – WriteMessage / ReadMessage round trip (packed, protobuf)
// ---------------------------------------------------------------------------

// TestWSLane_RoundTrip writes a model.Message through WSLane (packer+protobuf)
// from the client and reads it back on the server, verifying the payload is
// preserved end-to-end.
func TestWSLane_RoundTrip(t *testing.T) {
	serverConn, clientConn := newTestWSPair(t)

	clientLane := NewWSLane(clientConn)
	serverLane := NewWSLane(serverConn)

	want := &model.Message{
		Header: model.MessageHeader{
			ID:        "ws-lane-id",
			ParentID:  "ws-parent",
			Timestamp: time.Now().UnixNano() / 1e6,
		},
		Router: model.MessageRoute{
			Source:    "cloud",
			Group:     "device",
			Resource:  "device/twin",
			Operation: "patch",
		},
	}

	require.NoError(t, clientLane.WriteMessage(want))

	got := &model.Message{}
	require.NoError(t, serverLane.ReadMessage(got))

	assert.Equal(t, want.GetID(), got.GetID())
	assert.Equal(t, want.GetParentID(), got.GetParentID())
	assert.Equal(t, want.GetSource(), got.GetSource())
	assert.Equal(t, want.GetOperation(), got.GetOperation())
	assert.Equal(t, want.GetResource(), got.GetResource())
}

// ---------------------------------------------------------------------------
// SetReadDeadline / SetWriteDeadline – propagation to the underlying conn
// ---------------------------------------------------------------------------

// TestWSLaneWithoutPack_SetReadDeadline verifies that SetReadDeadline stores the
// deadline on the lane and propagates it to the underlying websocket.Conn
// without returning an error.
func TestWSLaneWithoutPack_SetReadDeadline(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)
	deadline := time.Now().Add(5 * time.Second)

	err := l.SetReadDeadline(deadline)

	assert.NoError(t, err)
	assert.Equal(t, deadline, l.readDeadline)
}

// TestWSLaneWithoutPack_SetWriteDeadline verifies that SetWriteDeadline stores
// the deadline on the lane and propagates it to the underlying websocket.Conn
// without returning an error.
func TestWSLaneWithoutPack_SetWriteDeadline(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)
	deadline := time.Now().Add(10 * time.Second)

	err := l.SetWriteDeadline(deadline)

	assert.NoError(t, err)
	assert.Equal(t, deadline, l.writeDeadline)
}

// TestWSLane_SetReadDeadline verifies that WSLane.SetReadDeadline propagates
// the deadline to the underlying websocket.Conn without returning an error.
func TestWSLane_SetReadDeadline(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLane(clientConn)
	deadline := time.Now().Add(5 * time.Second)

	err := l.SetReadDeadline(deadline)

	assert.NoError(t, err)
	assert.Equal(t, deadline, l.readDeadline)
}

// TestWSLane_SetWriteDeadline verifies that WSLane.SetWriteDeadline propagates
// the deadline to the underlying websocket.Conn without returning an error.
func TestWSLane_SetWriteDeadline(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLane(clientConn)
	deadline := time.Now().Add(10 * time.Second)

	err := l.SetWriteDeadline(deadline)

	assert.NoError(t, err)
	assert.Equal(t, deadline, l.writeDeadline)
}

// ---------------------------------------------------------------------------
// SetReadDeadline with zero time – reset deadline
// ---------------------------------------------------------------------------

// TestWSLaneWithoutPack_SetReadDeadline_Zero verifies that setting a zero
// time.Time clears the read deadline on the underlying connection.
func TestWSLaneWithoutPack_SetReadDeadline_Zero(t *testing.T) {
	_, clientConn := newTestWSPair(t)

	l := NewWSLaneWithoutPack(clientConn)

	err := l.SetReadDeadline(time.Time{})

	assert.NoError(t, err)
	assert.Equal(t, time.Time{}, l.readDeadline)
}
