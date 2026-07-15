package cloudstream

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestPortForward_String(t *testing.T) {
	assert := assert.New(t)
	portForwardConn := &ContainerPortForwardConnection{
		MessageID: 100,
	}

	stdResult := "APIServer_PortForwardConnection MessageID 100"
	assert.Equal(stdResult, portForwardConn.String())
}

func TestPortForward_WriteToAPIServer(t *testing.T) {
	assert := assert.New(t)
	mockConn := &MockConn{}
	portForwardConn := &ContainerPortForwardConnection{
		Conn: mockConn,
	}

	data := []byte("test data")
	dataLength, err := portForwardConn.WriteToAPIServer(data)
	assert.NoError(err)
	assert.Equal(9, dataLength)
	assert.Equal(data, mockConn.writeBuffer.Bytes())
}

func TestPortForward_SetMessageID(t *testing.T) {
	assert := assert.New(t)
	portForwardConn := &ContainerPortForwardConnection{}

	portForwardConn.SetMessageID(uint64(100))

	stdResult := uint64(100)
	assert.Equal(stdResult, portForwardConn.MessageID)
}

func TestPortForward_GetMessageID(t *testing.T) {
	assert := assert.New(t)

	portForwardConn := &ContainerPortForwardConnection{
		MessageID: 200,
	}

	stdResult := uint64(200)
	assert.Equal(stdResult, portForwardConn.GetMessageID())
}

func TestPortForward_SetEdgePeerDone(t *testing.T) {
	assert := assert.New(t)

	portForwardConn := &ContainerPortForwardConnection{
		MessageID:    1,
		edgePeerStop: make(chan struct{}),
		closeChan:    make(chan bool),
	}

	go func() {
		portForwardConn.SetEdgePeerDone()
	}()

	select {
	case <-portForwardConn.edgePeerStop:
		assert.True(true)
	case <-portForwardConn.closeChan:
		assert.Fail("Expected edgePeerStop to receive but got closeChan")
	}
}

func TestPortForward_EdgePeerDone(t *testing.T) {
	assert := assert.New(t)

	edgePeerStop := make(chan struct{})
	portForwardConn := &ContainerPortForwardConnection{
		edgePeerStop: edgePeerStop,
	}

	assert.Equal(edgePeerStop, portForwardConn.EdgePeerDone())
}

func TestPortForward_WriteToTunnel(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	portForwardConn := &ContainerPortForwardConnection{
		MessageID: 1,
		session:   session,
	}

	message := stream.NewMessage(portForwardConn.MessageID, stream.MessageTypeData, []byte("test data"))

	err := portForwardConn.WriteToTunnel(message)
	assert.NoError(err)
	assert.Equal(mockTunneler.lastMessage, message)
}

func TestPortForward_SendConnection(t *testing.T) {
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

	portForwardConn := &ContainerPortForwardConnection{
		MessageID: 1,
		r:         r,
		Conn:      mockConn,
		session:   session,
	}

	connector, err := portForwardConn.SendConnection()
	assert.NoError(err)

	edgedConnector, ok := connector.(*stream.EdgedPortForwardConnection)
	assert.True(ok, "Expected connector should be of type *stream.EdgedPortForwardConnection")
	assert.Equal(portForwardConn.MessageID, edgedConnector.MessID)
	assert.Equal(r.Request.Method, edgedConnector.Method)
	expectedURL := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:10350",
	}
	assert.Equal(expectedURL, edgedConnector.URL)
	assert.Equal(r.Request.Header, edgedConnector.Header)

	assert.Equal(mockTunneler.lastMessage.MessageType, stream.MessageTypePortForwardConnect)
	expectedData, _ := edgedConnector.CreateConnectMessage()
	assert.Equal(mockTunneler.lastMessage.Data, expectedData.Data)
}
