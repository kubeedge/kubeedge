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
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
	assert.Equal(dataBytes, mockConn.WrittenData)
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
		ReadError: errors.New("read error"),
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
