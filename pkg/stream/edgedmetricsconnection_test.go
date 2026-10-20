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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockResponseBody struct {
	Reader io.Reader
	Closed bool
}

func NewMockResponseBody(content string) *MockResponseBody {
	return &MockResponseBody{
		Reader: strings.NewReader(content),
	}
}

func (m *MockResponseBody) Read(p []byte) (n int, err error) {
	return m.Reader.Read(p)
}

func (m *MockResponseBody) Close() error {
	m.Closed = true
	return nil
}

type MockStreamTunneler struct {
	Messages     []*Message
	WriteErr     error
	DataWriteErr error
	ControlData  []byte
	ControlType  int
	ControlErr   error
	ReaderType   int
	ReaderData   []byte
	ReaderErr    error
	CloseErr     error
	Closed       bool
}

func (m *MockStreamTunneler) WriteMessage(msg *Message) error {
	if msg.MessageType == MessageTypeData && m.DataWriteErr != nil {
		return m.DataWriteErr
	}
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MockStreamTunneler) WriteControl(messageType int, data []byte, deadline time.Time) error {
	m.ControlType = messageType
	m.ControlData = data
	return m.ControlErr
}

func (m *MockStreamTunneler) NextReader() (messageType int, r io.Reader, err error) {
	if m.ReaderErr != nil {
		return 0, nil, m.ReaderErr
	}
	return m.ReaderType, strings.NewReader(string(m.ReaderData)), nil
}

func (m *MockStreamTunneler) Close() error {
	m.Closed = true
	return m.CloseErr
}

func TestMetricsConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)
	edgedMetricsConn := &EdgedMetricsConnection{
		MessID: 1,
	}

	msg, err := edgedMetricsConn.CreateConnectMessage()
	assert.NoError(err)

	expectedData, err := json.Marshal(edgedMetricsConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedMetricsConn.MessID, MessageTypeMetricConnect, expectedData)

	assert.Equal(expectedMessage, msg)
}

func TestMetricsConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)

	edgedMetricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	messID := edgedMetricsConn.GetMessageID()
	stdResult := uint64(100)

	assert.Equal(messID, stdResult)
}

func TestMetricsConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedMetricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	stdResult := "EDGE_METRICS_CONNECTOR Message MessageID 100"
	result := edgedMetricsConn.String()

	assert.Equal(result, stdResult)
}

func TestMetricsConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedMetricsConn := &EdgedMetricsConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedMetricsConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedMetricsConn.ReadChan)
}

func TestMetricsConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedMetricsConn := &EdgedMetricsConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedMetricsConn.CloseReadChannel()
	}()

	_, ok := <-edgedMetricsConn.ReadChan
	assert.False(ok)
}

func TestMetricsConnection_CleanChannel(t *testing.T) {
	assert := assert.New(t)

	metricsConn := &EdgedMetricsConnection{
		Stop: make(chan struct{}, 2),
	}

	metricsConn.Stop <- struct{}{}
	metricsConn.Stop <- struct{}{}

	metricsConn.CleanChannel()

	assert.Equal(0, len(metricsConn.Stop))
}

func TestMetricsConnection_receiveFromCloudStream(t *testing.T) {
	assert := assert.New(t)

	stop := make(chan struct{}, 1)

	metricsConn := &EdgedMetricsConnection{
		MessID:   uint64(100),
		ReadChan: make(chan *Message, 3),
	}

	removeConnMsg := NewMessage(metricsConn.MessID, MessageTypeRemoveConnect, nil)

	metricsConn.ReadChan <- removeConnMsg

	close(metricsConn.ReadChan)

	metricsConn.receiveFromCloudStream(stop)

	assert.Equal(1, len(stop))
}

func TestMetricsConnection_write2CloudStream(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockStreamTunneler{}

	responseBody := NewMockResponseBody("line1\nline2\nline3")
	mockResponse := &http.Response{
		Body: responseBody,
	}

	metricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	err := metricsConn.write2CloudStream(mockTunneler, mockResponse)

	assert.Equal(3, len(mockTunneler.Messages))
	assert.Equal(MessageTypeData, mockTunneler.Messages[0].MessageType)
	assert.Contains(string(mockTunneler.Messages[0].Data), "line1")
	assert.NoError(err)
}

func TestMetricsConnection_write2CloudStream_WriteError(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockStreamTunneler{
		WriteErr: errors.New("tunnel write error"),
	}

	responseBody := NewMockResponseBody("test data for tunnel")
	mockResponse := &http.Response{
		Body: responseBody,
	}

	metricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	err := metricsConn.write2CloudStream(mockTunneler, mockResponse)

	assert.ErrorContains(err, "tunnel write error")
}

func TestMetricsConnection_Serve(t *testing.T) {
	tests := []struct {
		name               string
		truncated, stopped bool
		writeError         error
		wantError          string
	}{
		{name: "success"},
		{name: "scanner failure", truncated: true, wantError: "unexpected EOF"},
		{name: "write failure", writeError: errors.New("tunnel write failed"), wantError: "tunnel write failed"},
		{name: "cloud cancellation", stopped: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.stopped {
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				if tt.truncated {
					w.Header().Set("Content-Length", "100")
				}
				_, _ = io.WriteString(w, "test metric data")
			}))
			defer server.Close()
			metricsURL, err := url.Parse(server.URL + "/metrics")
			require.NoError(t, err)
			metrics := &EdgedMetricsConnection{
				MessID: 100, URL: *metricsURL, Header: http.Header{},
				ReadChan: make(chan *Message), Stop: make(chan struct{}, 1),
			}
			defer metrics.CloseReadChannel()
			if tt.stopped {
				metrics.Stop <- struct{}{}
			}
			tunnel := &MockStreamTunneler{DataWriteErr: tt.writeError}

			err = metrics.Serve(tunnel)

			if tt.wantError != "" {
				assert.ErrorContains(t, err, tt.wantError)
			} else {
				assert.NoError(t, err)
			}
			require.NotEmpty(t, tunnel.Messages)
			completion := tunnel.Messages[len(tunnel.Messages)-1]
			assert.Equal(t, MessageTypeRemoveConnect, completion.MessageType)
			if tt.wantError != "" || tt.stopped {
				assert.Empty(t, completion.Data)
			} else {
				assert.Equal(t, "metrics-stream-success-v1", string(completion.Data))
			}
		})
	}
}
