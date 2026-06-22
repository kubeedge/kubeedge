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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsTestPair creates an in-process WebSocket client/server pair using httptest.
// It returns the server-side and client-side *websocket.Conn.
func wsTestPair(t *testing.T) (server, client *websocket.Conn) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		server = c
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	// Give the server handler time to assign server conn
	time.Sleep(50 * time.Millisecond)
	return server, c
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
		wsConn:   serverConn,
		consumer: consumer,
		autoRoute: true,
		state:    &ConnectionState{State: api.StatConnected},
	}

	// Run handleRawData in a goroutine — it blocks on io.Copy.
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The test passes if this does not panic.
		wsConn.handleRawData()
	}()

	// Give the goroutine time to enter io.Copy before we send data.
	time.Sleep(50 * time.Millisecond)

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

	if consumer.Len() == 0 {
		t.Error("consumer received no data; expected the payload to be forwarded")
	}
}
