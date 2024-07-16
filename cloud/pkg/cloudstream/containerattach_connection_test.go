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

package cloudstream

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestString(t *testing.T) {
	assert := assert.New(t)
	attachConn := &ContainerAttachConnection{
		MessageID: 100,
	}

	stdResult := "APIServer_AttachConnection MessageID 100"
	assert.Equal(stdResult, attachConn.String())
}

// Using MockConn to implement net.Conn for testing WriteToAPIServer(), SendConnection() and WriteToTunnel()

type MockConn struct {
	readBuffer  bytes.Buffer
	writeBuffer bytes.Buffer
	closeCalled bool
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *MockConn) Close() error {
	m.closeCalled = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return &net.IPAddr{}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &net.IPAddr{}
}

func (m *MockConn) SetDeadline(time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(time.Time) error {
	return nil
}

// Using MockTunneler to implement a tunneler for testing WriteToAPIServer(), SendConnection() and WriteToTunnel()

type MockTunneler struct {
	lastMessage *stream.Message
	err         error
}

func (m *MockTunneler) WriteMessage(message *stream.Message) error {
	if m.err != nil {
		return m.err
	}
	m.lastMessage = message
	return nil
}

func (m *MockTunneler) NextReader() (int, io.Reader, error) {
	return 0, nil, nil
}

func (m *MockTunneler) Close() error {
	return nil
}

func (m *MockTunneler) WriteControl(int, []byte, time.Time) error {
	return nil
}

func TestWriteToAPIServer(t *testing.T) {
	assert := assert.New(t)
	mockConn := &MockConn{}
	attachConn := &ContainerAttachConnection{
		Conn: mockConn,
	}

	data := []byte("test data")
	dataLength, err := attachConn.WriteToAPIServer(data)
	assert.NoError(err)
	assert.Equal(9, dataLength)
	assert.Equal(data, mockConn.writeBuffer.Bytes())
}

func TestSetMessageID(t *testing.T) {
	assert := assert.New(t)
	attachConn := &ContainerAttachConnection{}

	attachConn.SetMessageID(uint64(100))

	stdResult := uint64(100)
	assert.Equal(stdResult, attachConn.MessageID)
}

func TestGetMessageID(t *testing.T) {
	assert := assert.New(t)

	attachConn := &ContainerAttachConnection{
		MessageID: 200,
	}

	stdResult := uint64(200)
	assert.Equal(stdResult, attachConn.GetMessageID())
}

func TestSetEdgePeerDone(t *testing.T) {
	assert := assert.New(t)

	attachConn := &ContainerAttachConnection{
		MessageID:    1,
		edgePeerStop: make(chan struct{}),
		closeChan:    make(chan bool),
	}

	go func() {
		attachConn.SetEdgePeerDone()
	}()

	select {
	case <-attachConn.edgePeerStop:
		assert.True(true)
	case <-attachConn.closeChan:
		assert.Fail("Expected edgePeerStop to receive but got closeChan")
	}
}

func TestEdgePeerDone(t *testing.T) {
	assert := assert.New(t)

	edgePeerStop := make(chan struct{})
	attachConn := &ContainerAttachConnection{
		edgePeerStop: edgePeerStop,
	}

	assert.Equal(edgePeerStop, attachConn.EdgePeerDone())
}

func TestWriteToTunnel(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	attachConn := &ContainerAttachConnection{
		MessageID: 1,
		session:   session,
	}

	message := stream.NewMessage(attachConn.MessageID, stream.MessageTypeData, []byte("test data"))

	err := attachConn.WriteToTunnel(message)
	assert.NoError(err)
	assert.Equal(mockTunneler.lastMessage, message)
}

func TestSendConnection(t *testing.T) {
	assert := assert.New(t)

	mockConn := &MockConn{}
	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	r := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{},
			Header: http.Header{},
		},
	}

	attachConn := &ContainerAttachConnection{
		MessageID: 1,
		r:         r,
		Conn:      mockConn,
		session:   session,
	}

	connector, err := attachConn.SendConnection()
	assert.NoError(err)

	edgedConnector, ok := connector.(*stream.EdgedAttachConnection)
	assert.True(ok, "Expected connector should be of type *stream.EdgedAttachConnection")
	assert.Equal(attachConn.MessageID, edgedConnector.MessID)
	assert.Equal(r.Request.Method, edgedConnector.Method)
	expectedURL := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:10350",
	}
	assert.Equal(expectedURL, edgedConnector.URL)
	assert.Equal(r.Request.Header, edgedConnector.Header)

	assert.Equal(mockTunneler.lastMessage.MessageType, stream.MessageTypeAttachConnect)
	expectedData, _ := edgedConnector.CreateConnectMessage()
	assert.Equal(mockTunneler.lastMessage.Data, expectedData.Data)
}
