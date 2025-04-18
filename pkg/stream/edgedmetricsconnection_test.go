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
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
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
	Messages    []*Message
	WriteErr    error
	ControlData []byte
	ControlType int
	ControlErr  error
	ReaderType  int
	ReaderData  []byte
	ReaderErr   error
	CloseErr    error
	Closed      bool
}

func (m *MockStreamTunneler) WriteMessage(msg *Message) error {
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
	stop := make(chan struct{}, 1)

	responseBody := NewMockResponseBody("line1\nline2\nline3")
	mockResponse := &http.Response{
		Body: responseBody,
	}

	metricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	go metricsConn.write2CloudStream(mockTunneler, mockResponse, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(3, len(mockTunneler.Messages))
	assert.Equal(MessageTypeData, mockTunneler.Messages[0].MessageType)
	assert.Contains(string(mockTunneler.Messages[0].Data), "line1")

	assert.Equal(1, len(stop))
}

func TestMetricsConnection_write2CloudStream_WriteError(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockStreamTunneler{
		WriteErr: errors.New("tunnel write error"),
	}
	stop := make(chan struct{}, 1)

	responseBody := NewMockResponseBody("test data for tunnel")
	mockResponse := &http.Response{
		Body: responseBody,
	}

	metricsConn := &EdgedMetricsConnection{
		MessID: uint64(100),
	}

	go metricsConn.write2CloudStream(mockTunneler, mockResponse, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(1, len(stop))
}

func TestMetricsConnection_Serve(t *testing.T) {
	assert := assert.New(t)

	metricsConn := &EdgedMetricsConnection{
		MessID:   uint64(100),
		ReadChan: make(chan *Message, 10),
		Stop:     make(chan struct{}, 1),
		URL:      url.URL{Scheme: "https", Host: "example.com", Path: "/metrics"},
		Header:   http.Header{},
	}

	mockTunneler := &MockStreamTunneler{}

	responseBody := NewMockResponseBody("test metric data")
	mockResponse := &http.Response{
		StatusCode: 200,
		Body:       responseBody,
	}

	patchNewRequest := gomonkey.ApplyFunc(http.NewRequest,
		func(method, url string, body io.Reader) (*http.Request, error) {
			return &http.Request{
				Header: http.Header{},
			}, nil
		})
	defer patchNewRequest.Reset()

	patchDo := gomonkey.ApplyMethod(reflect.TypeOf(&http.Client{}), "Do",
		func(_ *http.Client, _ *http.Request) (*http.Response, error) {
			return mockResponse, nil
		})
	defer patchDo.Reset()

	go func() {
		time.Sleep(100 * time.Millisecond)
		metricsConn.Stop <- struct{}{}
	}()

	err := metricsConn.Serve(mockTunneler)

	assert.NoError(err)

	found := false
	for _, msg := range mockTunneler.Messages {
		if msg.MessageType == MessageTypeRemoveConnect {
			found = true
			break
		}
	}
	assert.True(found, "Expected a RemoveConnect message to be sent")
}
