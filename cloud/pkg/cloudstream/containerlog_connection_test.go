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
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestString_Log(t *testing.T) {
	assert := assert.New(t)
	logsConn := &ContainerLogsConnection{
		MessageID: 100,
	}

	stdResult := "APIServer_LogsConnection MessageID 100"
	assert.Equal(stdResult, logsConn.String())
}

// Using MockWriter to implement io.Writer for testing WriteToAPIServer()

type MockWriter struct {
	writeBuffer bytes.Buffer
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	return m.writeBuffer.Write(p)
}

func TestWriteToAPIServer_Log(t *testing.T) {
	assert := assert.New(t)

	mockWriter := &MockWriter{}
	logsConnection := &ContainerLogsConnection{
		flush: mockWriter,
	}

	data := []byte("test data")
	dataLength, err := logsConnection.WriteToAPIServer(data)
	assert.NoError(err)
	assert.Equal(len(data), dataLength)
	assert.Equal(data, mockWriter.writeBuffer.Bytes())
}

func TestSetMessageID_Log(t *testing.T) {
	assert := assert.New(t)
	logsConn := &ContainerLogsConnection{}

	logsConn.SetMessageID(uint64(100))

	stdResult := uint64(100)
	assert.Equal(stdResult, logsConn.MessageID)
}

func TestGetMessageID_Log(t *testing.T) {
	assert := assert.New(t)

	logsConn := &ContainerLogsConnection{
		MessageID: 200,
	}

	stdResult := uint64(200)
	assert.Equal(stdResult, logsConn.GetMessageID())
}

func TestSetEdgePeerDone_Log(t *testing.T) {
	assert := assert.New(t)

	logsConn := &ContainerLogsConnection{
		MessageID:    1,
		edgePeerStop: make(chan struct{}),
		closeChan:    make(chan bool),
	}

	go func() {
		logsConn.SetEdgePeerDone()
	}()

	select {
	case <-logsConn.edgePeerStop:
		assert.True(true)
	case <-logsConn.closeChan:
		assert.Fail("Expected edgePeerStop to receive but got closeChan")
	}
}

func TestEdgePeerDone_Log(t *testing.T) {
	assert := assert.New(t)

	edgePeerStop := make(chan struct{})
	logsConn := &ContainerLogsConnection{
		edgePeerStop: edgePeerStop,
	}

	assert.Equal(edgePeerStop, logsConn.EdgePeerDone())
}

func TestWriteToTunnel_Log(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	logsConn := &ContainerLogsConnection{
		MessageID: 1,
		session:   session,
	}

	message := stream.NewMessage(logsConn.MessageID, stream.MessageTypeData, []byte("test data"))

	err := logsConn.WriteToTunnel(message)
	assert.NoError(err)
	assert.Equal(mockTunneler.lastMessage, message)
}

func TestSendConnection_Log(t *testing.T) {
	assert := assert.New(t)

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

	logsConnection := &ContainerLogsConnection{
		MessageID: 1,
		r:         r,
		session:   session,
	}

	connector, err := logsConnection.SendConnection()
	assert.NoError(err)

	edgedConnector, ok := connector.(*stream.EdgedLogsConnection)
	assert.True(ok, "Expected connector to be of type *stream.EdgedLogsConnection")
	assert.Equal(logsConnection.MessageID, edgedConnector.MessID)
	expectedURL := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(defaultServerHost, fmt.Sprintf("%v", constants.ServerPort)),
	}
	assert.Equal(expectedURL, edgedConnector.URL)
	assert.Equal(r.Request.Header, edgedConnector.Header)

	assert.Equal(stream.MessageTypeLogsConnect, mockTunneler.lastMessage.MessageType)
	expectedData, _ := edgedConnector.CreateConnectMessage()
	assert.Equal(expectedData.Data, mockTunneler.lastMessage.Data)
}
