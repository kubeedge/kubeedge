/*
Copyright 2025 The KubeEdge Authors.

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

package stream

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// mockWebsocketServer creates a test websocket server for testing
func mockWebsocketServer(t *testing.T) (*httptest.Server, *websocket.Conn) {
	var upgrader = websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo back any messages received
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			err = conn.WriteMessage(mt, message)
			if err != nil {
				break
			}
		}
	}))

	// Convert http://... to ws://...
	wsURL := "ws" + server.URL[4:]

	// Connect to the server
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to websocket server: %v", err)
	}

	return server, conn
}

func TestNewDefaultTunnel(t *testing.T) {
	server, conn := mockWebsocketServer(t)
	defer server.Close()
	defer conn.Close()

	tunnel := NewDefaultTunnel(conn)
	assert.NotNil(t, tunnel)
	assert.NotNil(t, tunnel.lock)
	assert.NotNil(t, tunnel.con)
}

func TestDefaultTunnel_WriteMessage(t *testing.T) {
	server, conn := mockWebsocketServer(t)
	defer server.Close()
	defer conn.Close()

	tunnel := NewDefaultTunnel(conn)

	// Test valid message cases
	testCases := []struct {
		name    string
		message *Message
	}{
		{
			name:    "valid data message",
			message: NewMessage(1, MessageTypeData, []byte("test message")),
		},
		{
			name:    "empty data message",
			message: NewMessage(2, MessageTypeData, []byte{}),
		},
		{
			name:    "logs connect message",
			message: NewMessage(3, MessageTypeLogsConnect, []byte("logs connection")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tunnel.WriteMessage(tc.message)
			assert.NoError(t, err)
		})
	}

	// Test nil message case in a separate sub-test to handle the panic
	t.Run("nil message", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "Expected panic when writing nil message")
		}()

		_ = tunnel.WriteMessage(nil)
		t.Error("Expected panic did not occur")
	})
}

func TestDefaultTunnel_WriteControl(t *testing.T) {
	server, conn := mockWebsocketServer(t)
	defer server.Close()
	defer conn.Close()

	tunnel := NewDefaultTunnel(conn)

	testCases := []struct {
		name        string
		messageType int
		data        []byte
		deadline    time.Time
		wantErr     bool
	}{
		{
			name:        "ping message",
			messageType: websocket.PingMessage,
			data:        []byte("ping"),
			deadline:    time.Now().Add(time.Second),
			wantErr:     false,
		},
		{
			name:        "pong message",
			messageType: websocket.PongMessage,
			data:        []byte("pong"),
			deadline:    time.Now().Add(time.Second),
			wantErr:     false,
		},
		{
			name:        "close message",
			messageType: websocket.CloseMessage,
			data:        websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			deadline:    time.Now().Add(time.Second),
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tunnel.WriteControl(tc.messageType, tc.data, tc.deadline)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultTunnel_NextReader(t *testing.T) {
	server, conn := mockWebsocketServer(t)
	defer server.Close()
	defer conn.Close()

	tunnel := NewDefaultTunnel(conn)

	// Create and write a test message
	testMessage := NewMessage(1, MessageTypeData, []byte("test message"))
	err := conn.WriteMessage(websocket.TextMessage, testMessage.Bytes())
	assert.NoError(t, err)

	// Test NextReader
	messageType, reader, err := tunnel.NextReader()
	assert.NoError(t, err)
	assert.Equal(t, websocket.TextMessage, messageType)
	assert.NotNil(t, reader)

	// Read and verify the message content
	receivedMessage, err := ReadMessageFromTunnel(reader)
	assert.NoError(t, err)
	assert.Equal(t, testMessage.ConnectID, receivedMessage.ConnectID)
	assert.Equal(t, testMessage.MessageType, receivedMessage.MessageType)
	assert.Equal(t, testMessage.Data, receivedMessage.Data)
}

func TestDefaultTunnel_Close(t *testing.T) {
	server, conn := mockWebsocketServer(t)
	defer server.Close()

	tunnel := NewDefaultTunnel(conn)

	err := tunnel.Close()
	assert.NoError(t, err)

	// Verify the connection is closed by attempting to write
	message := NewMessage(1, MessageTypeData, []byte("test"))
	err = tunnel.WriteMessage(message)
	assert.Error(t, err)
}

func TestSafeWriteTunneler_Interface(_ *testing.T) {
	var _ SafeWriteTunneler = &DefaultTunnel{}
}
