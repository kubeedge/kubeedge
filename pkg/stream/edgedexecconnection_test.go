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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockConn struct {
	ReadData  []byte
	ReadErr   error
	WriteData []byte
	WriteErr  error
	Closed    bool
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	if m.ReadErr != nil {
		return 0, m.ReadErr
	}
	if len(m.ReadData) == 0 {
		return 0, io.EOF
	}
	n = copy(b, m.ReadData)
	m.ReadData = m.ReadData[n:]
	return n, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	if m.WriteErr != nil {
		return 0, m.WriteErr
	}
	m.WriteData = append(m.WriteData, b...)
	return len(b), nil
}

func (m *MockConn) Close() error {
	m.Closed = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

type MockTunneler struct {
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

func (m *MockTunneler) WriteMessage(msg *Message) error {
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MockTunneler) WriteControl(messageType int, data []byte, deadline time.Time) error {
	m.ControlType = messageType
	m.ControlData = data
	return m.ControlErr
}

func (m *MockTunneler) NextReader() (messageType int, r io.Reader, err error) {
	if m.ReaderErr != nil {
		return 0, nil, m.ReaderErr
	}
	return m.ReaderType, io.NopCloser(bytes.NewReader(m.ReaderData)), nil
}

func (m *MockTunneler) Close() error {
	m.Closed = true
	return m.CloseErr
}

func TestExecConnection_CleanChannel(t *testing.T) {
	assert := assert.New(t)

	edgedExecConn := &EdgedExecConnection{
		Stop: make(chan struct{}, 2),
	}

	edgedExecConn.Stop <- struct{}{}
	edgedExecConn.Stop <- struct{}{}

	edgedExecConn.CleanChannel()

	assert.Equal(0, len(edgedExecConn.Stop))
}

func TestExecConnection_receiveFromCloudStream(t *testing.T) {
	assert := assert.New(t)

	mockConn := &MockConn{}
	stop := make(chan struct{}, 1)

	edgedExecConn := &EdgedExecConnection{
		MessID:   uint64(100),
		ReadChan: make(chan *Message, 3),
	}

	removeConnMsg := NewMessage(edgedExecConn.MessID, MessageTypeRemoveConnect, nil)

	dataBytes := []byte("test data")
	dataMsg := NewMessage(edgedExecConn.MessID, MessageTypeData, dataBytes)

	edgedExecConn.ReadChan <- removeConnMsg
	edgedExecConn.ReadChan <- dataMsg

	close(edgedExecConn.ReadChan)

	edgedExecConn.receiveFromCloudStream(mockConn, stop)

	assert.Equal(1, len(stop))
	assert.Equal(dataBytes, mockConn.WriteData)
}

func TestExecConnection_write2CloudStream(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	stop := make(chan struct{}, 1)

	mockConn := &MockConn{
		ReadData: []byte("test data for tunnel"),
	}

	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	go edgedExecConn.write2CloudStream(mockTunneler, mockConn, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(1, len(mockTunneler.Messages))
	assert.Equal(MessageTypeData, mockTunneler.Messages[0].MessageType)
	assert.Equal([]byte("test data for tunnel"), mockTunneler.Messages[0].Data)

	assert.Equal(1, len(stop))
}

func TestExecConnection_write2CloudStream_ReadError(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	stop := make(chan struct{}, 1)

	mockConn := &MockConn{
		ReadErr: errors.New("read error"),
	}

	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	go edgedExecConn.write2CloudStream(mockTunneler, mockConn, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(0, len(mockTunneler.Messages))
	assert.Equal(1, len(stop))
}

func TestExecConnection_write2CloudStream_WriteError(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{
		WriteErr: errors.New("tunnel write error"),
	}
	stop := make(chan struct{}, 1)

	mockConn := &MockConn{
		ReadData: []byte("test data for tunnel"),
	}

	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	go edgedExecConn.write2CloudStream(mockTunneler, mockConn, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(1, len(stop))
}

type MockRoundTripper struct {
	DialConn net.Conn
	DialErr  error
}

func (m *MockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, nil
}

func (m *MockRoundTripper) Dial(req *http.Request) (net.Conn, error) {
	if m.DialErr != nil {
		return nil, m.DialErr
	}
	return m.DialConn, nil
}

func TestExecConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)

	edgedExecConn := &EdgedExecConnection{
		MessID: 1,
	}
	msg, err := edgedExecConn.CreateConnectMessage()
	assert.NoError(err)

	exceptedData, err := json.Marshal(edgedExecConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedExecConn.MessID, MessageTypeExecConnect, exceptedData)

	assert.Equal(expectedMessage, msg)
}

func TestExecConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)
	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	stdResult := uint64(100)
	assert.Equal(stdResult, edgedExecConn.MessID)
}

func TestExecConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedExecConn := &EdgedExecConnection{
		MessID: uint64(100),
	}

	result := edgedExecConn.String()
	stdResult := "EDGE_EXEC_CONNECTOR Message MessageID 100"
	assert.Equal(stdResult, result)
}

func TestExecConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedExecConn := &EdgedExecConnection{
		ReadChan: make(chan *Message, 1),
	}
	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedExecConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedExecConn.ReadChan)
}

func TestExecConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedExecConn := &EdgedExecConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		edgedExecConn.CloseReadChannel()
	}()

	_, ok := <-edgedExecConn.ReadChan
	assert.False(ok)
}
