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

package stream

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	defaultTestTimeout = 100 * time.Millisecond
	defaultWaitTime    = 1 * time.Second
)

type mockSafeWriteTunneler struct {
	written  []*Message
	writeErr error
	closeErr error
	nextErr  error
	nextType int
	nextData []byte
}

func (m *mockSafeWriteTunneler) WriteMessage(msg *Message) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.written = append(m.written, msg)
	return nil
}

func (m *mockSafeWriteTunneler) Close() error {
	return m.closeErr
}

func (m *mockSafeWriteTunneler) WriteControl(_ int, _ []byte, _ time.Time) error {
	return nil
}

func (m *mockSafeWriteTunneler) NextReader() (messageType int, r io.Reader, err error) {
	if m.nextErr != nil {
		return 0, nil, m.nextErr
	}
	return m.nextType, bytes.NewReader(m.nextData), nil
}

type mockReader struct {
	buf     *bytes.Buffer
	readErr error
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return m.buf.Read(p)
}

func TestLogConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)

	edgedLogsConn := &EdgedLogsConnection{
		MessID: 1,
	}
	msg, err := edgedLogsConn.CreateConnectMessage()
	assert.NoError(err)

	exceptedData, err := json.Marshal(edgedLogsConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedLogsConn.MessID, MessageTypeLogsConnect, exceptedData)

	assert.Equal(expectedMessage, msg)
}

func TestLogConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)
	edgedLogsConn := &EdgedLogsConnection{
		MessID: uint64(100),
	}

	stdResult := uint64(100)
	assert.Equal(stdResult, edgedLogsConn.MessID)
}

func TestLogConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedLogsConn := &EdgedLogsConnection{
		MessID: uint64(100),
	}

	result := edgedLogsConn.String()
	stdResult := "EDGE_LOGS_CONNECTOR Message MessageID 100"
	assert.Equal(stdResult, result)
}

func TestLogConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedLogsConn := &EdgedLogsConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedLogsConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedLogsConn.ReadChan)
}

func TestLogConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedLogsConn := &EdgedLogsConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedLogsConn.CloseReadChannel()
	}()

	_, ok := <-edgedLogsConn.ReadChan
	assert.False(ok)
}

func TestLogConnection_receiveFromCloudStream(t *testing.T) {
	assert := assert.New(t)

	stop := make(chan struct{}, 1)
	readChan := make(chan *Message, 2)

	logs := &EdgedLogsConnection{
		ReadChan: readChan,
		MessID:   1,
	}

	defer close(logs.ReadChan)

	go logs.receiveFromCloudStream(stop)

	dataMsg := NewMessage(1, MessageTypeData, []byte("test data"))
	logs.ReadChan <- dataMsg

	select {
	case <-stop:
		assert.Fail("Should not receive stop signal for data message")
	case <-time.After(defaultTestTimeout):
	}

	removeMsg := NewMessage(1, MessageTypeRemoveConnect, nil)
	logs.ReadChan <- removeMsg

	select {
	case <-stop:
	case <-time.After(defaultWaitTime):
		assert.Fail("Should have received stop signal")
	}
}

func TestEdgedLogsConnection_write2CloudStream(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name        string
		input       string
		readErr     error
		writeErr    error
		expectStop  bool
		expectWrite bool
	}{
		{
			name:        "successful write",
			input:       "test log data",
			expectStop:  true,
			expectWrite: true,
		},
		{
			name:        "EOF error",
			input:       "",
			readErr:     io.EOF,
			expectStop:  true,
			expectWrite: false,
		},
		{
			name:        "read error",
			input:       "test data",
			readErr:     errors.New("read error"),
			expectStop:  true,
			expectWrite: false,
		},
		{
			name:        "write error",
			input:       "test data",
			writeErr:    errors.New("write error"),
			expectStop:  true,
			expectWrite: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stop := make(chan struct{}, 1)

			mockTunnel := &mockSafeWriteTunneler{
				writeErr: tc.writeErr,
			}

			var buf bytes.Buffer
			buf.WriteString(tc.input)
			reader := bufio.NewReader(&mockReader{
				buf:     &buf,
				readErr: tc.readErr,
			})

			logs := &EdgedLogsConnection{
				MessID: 1,
			}

			go logs.write2CloudStream(mockTunnel, *reader, stop)

			select {
			case <-stop:
				if !tc.expectStop {
					assert.Fail("Unexpected stop signal")
				}
			case <-time.After(100 * time.Millisecond):
				if tc.expectStop {
					assert.Fail("Should have received stop signal")
				}
			}

			if tc.expectWrite {
				assert.NotEmpty(mockTunnel.written, "Expected messages to be written")
				if len(mockTunnel.written) > 0 {
					assert.Equal(tc.input, string(mockTunnel.written[0].Data))
				}
			} else {
				assert.Empty(mockTunnel.written, "Expected no messages to be written")
			}
		})
	}
}

func TestEdgedLogsConnection_Serve(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		expectError   bool
	}{
		{
			name: "successful connection",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test log data"))
			},
			expectError: false,
		},
		{
			name: "server error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serverHandler))
			defer server.Close()

			serverURL, _ := url.Parse(server.URL)

			logs := &EdgedLogsConnection{
				MessID:   1,
				URL:      *serverURL,
				Header:   http.Header{},
				ReadChan: make(chan *Message),
				Stop:     make(chan struct{}),
			}

			mockTunnel := &mockSafeWriteTunneler{}

			errChan := make(chan error)
			go func() {
				errChan <- logs.Serve(mockTunnel)
			}()

			go func() {
				time.Sleep(defaultTestTimeout)
				logs.Stop <- struct{}{}
			}()

			err := <-errChan
			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
