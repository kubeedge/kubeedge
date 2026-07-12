package stream

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPortForwardConnection_CreateConnectMessage(t *testing.T) {
	assert := assert.New(t)
	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID: 1,
	}

	msg, err := edgedPortForwardConn.CreateConnectMessage()
	assert.NoError(err)
	expectedData, err := json.Marshal(edgedPortForwardConn)
	assert.NoError(err)
	expectedMessage := NewMessage(edgedPortForwardConn.MessID, MessageTypePortForwardConnect, expectedData)

	assert.Equal(expectedMessage, msg)
}

func TestPortForwardConnection_GetMessageID(t *testing.T) {
	assert := assert.New(t)

	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID: uint64(100),
	}

	messID := edgedPortForwardConn.GetMessageID()
	stdResult := uint64(100)

	assert.Equal(messID, stdResult)
}

func TestPortForwardConnection_String(t *testing.T) {
	assert := assert.New(t)

	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID: uint64(100),
	}

	result := edgedPortForwardConn.String()
	stdResult := "EDGE_PORT_FORWARD_CONNECTOR Message MessageID 100"
	assert.Equal(stdResult, result)
}

func TestPortForwardConnection_CacheTunnelMessage(t *testing.T) {
	assert := assert.New(t)
	edgedPortForwardConn := &EdgedPortForwardConnection{
		ReadChan: make(chan *Message, 1),
	}

	msg := &Message{ConnectID: 100, MessageType: MessageTypeData, Data: []byte("test data")}
	edgedPortForwardConn.CacheTunnelMessage(msg)

	assert.Equal(msg, <-edgedPortForwardConn.ReadChan)
}

func TestPortForwardConnection_CloseReadChannel(t *testing.T) {
	assert := assert.New(t)
	edgedPortForwardConn := &EdgedPortForwardConnection{
		ReadChan: make(chan *Message),
	}

	go func() {
		time.Sleep(1 * time.Second)
		edgedPortForwardConn.CloseReadChannel()
	}()

	_, ok := <-edgedPortForwardConn.ReadChan
	assert.False(ok)
}

func TestPortForwardConnection_CleanChannel(t *testing.T) {
	assert := assert.New(t)

	forwardPortConn := &EdgedPortForwardConnection{
		Stop: make(chan struct{}, 2),
	}

	forwardPortConn.Stop <- struct{}{}
	forwardPortConn.Stop <- struct{}{}

	forwardPortConn.CleanChannel()

	assert.Equal(0, len(forwardPortConn.Stop))
}

func TestPortForwardConnection_receiveFromCloudStream(t *testing.T) {
	assert := assert.New(t)

	// reusing the MockConn from exec
	portForwardMockConn := &execMockConn{}
	stop := make(chan struct{}, 1)

	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID:   uint64(100),
		ReadChan: make(chan *Message, 3),
	}

	removeConnMsg := NewMessage(edgedPortForwardConn.MessID, MessageTypeRemoveConnect, nil)

	dataBytes := []byte("test data")
	dataMsg := NewMessage(edgedPortForwardConn.MessID, MessageTypeData, dataBytes)

	edgedPortForwardConn.ReadChan <- removeConnMsg
	edgedPortForwardConn.ReadChan <- dataMsg

	close(edgedPortForwardConn.ReadChan)

	edgedPortForwardConn.receiveFromCloudStream(portForwardMockConn, stop)

	assert.Equal(1, len(stop))
	assert.Equal(dataBytes, portForwardMockConn.WriteData)
}

func TestPortForwardConnection_write2CloudStream(t *testing.T) {
	assert := assert.New(t)

	// reusing the MockTunneler from attach
	mockTunneler := &MockTunneler{}
	stop := make(chan struct{}, 1)

	// reusing the MockConn from exec
	portForwardMockConn := &execMockConn{
		ReadData: []byte("test data for tunnel"),
	}

	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID: uint64(100),
	}

	go edgedPortForwardConn.write2CloudStream(mockTunneler, portForwardMockConn, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(1, len(mockTunneler.Messages))
	assert.Equal(MessageTypeData, mockTunneler.Messages[0].MessageType)
	assert.Equal([]byte("test data for tunnel"), mockTunneler.Messages[0].Data)

	assert.Equal(1, len(stop))
}

func TestPortForwardConnection_write2CloudStream_ReadError(t *testing.T) {
	assert := assert.New(t)

	// reusing the MockTunneler from attach
	mockTunneler := &MockTunneler{}
	stop := make(chan struct{}, 1)

	// reusing the MockConn from exec
	portForwardMockConn := &execMockConn{
		ReadErr: errors.New("read error"),
	}

	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID: uint64(100),
	}

	go edgedPortForwardConn.write2CloudStream(mockTunneler, portForwardMockConn, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(0, len(mockTunneler.Messages))
	assert.Equal(1, len(stop))
}

func TestPortForwardConnection_write2CloudStream_WriteError(t *testing.T) {
	assert := assert.New(t)

	// reusing the MockTunneler from attach
	mockTunneler := &MockTunneler{
		WriteErr: errors.New("tunnel write error"),
	}
	stop := make(chan struct{}, 1)

	// reusing the MockConn from exec
	portForwardMockConn := &execMockConn{
		ReadData: []byte("test data for tunnel"),
	}

	edgedPortForwardConn := &EdgedPortForwardConnection{
		MessID: uint64(100),
	}

	go edgedPortForwardConn.write2CloudStream(mockTunneler, portForwardMockConn, stop)

	time.Sleep(100 * time.Millisecond)

	assert.Equal(1, len(stop))
}
