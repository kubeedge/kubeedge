package edgestream

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

type mockEdgedConnection struct {
	cleanCalled bool
	closeCalled bool
	cachedMsgs  []*stream.Message
}

func (m *mockEdgedConnection) CreateConnectMessage() (*stream.Message, error) {
	return nil, nil
}
func (m *mockEdgedConnection) Serve(tunnel stream.SafeWriteTunneler) error {
	return nil
}
func (m *mockEdgedConnection) CacheTunnelMessage(msg *stream.Message) {
	m.cachedMsgs = append(m.cachedMsgs, msg)
}
func (m *mockEdgedConnection) GetMessageID() uint64 {
	return 1
}
func (m *mockEdgedConnection) CloseReadChannel() {
	m.closeCalled = true
}
func (m *mockEdgedConnection) CleanChannel() {
	m.cleanCalled = true
}
func (m *mockEdgedConnection) String() string {
	return "mockEdgedConnection"
}

type mockTunneler struct {
	messages []*stream.Message
	index    int
}

func (m *mockTunneler) WriteMessage(message *stream.Message) error {
	return nil
}
func (m *mockTunneler) WriteControl(messageType int, data []byte, deadline time.Time) error {
	return nil
}
func (m *mockTunneler) Close() error {
	return nil
}
func (m *mockTunneler) NextReader() (int, io.Reader, error) {
	if m.index >= len(m.messages) {
		return 0, nil, io.EOF
	}
	msg := m.messages[m.index]
	m.index++
	return websocket.TextMessage, bytes.NewReader(msg.Bytes()), nil
}

func TestServeConnectionUnsupportedMessageType(t *testing.T) {
	session := &TunnelSession{
		localCons: make(map[uint64]stream.EdgedConnection),
	}

	mockConn := &mockEdgedConnection{}
	connectID := uint64(12345)

	session.AddLocalConnection(connectID, mockConn)

	unsupportedMsg := &stream.Message{
		ConnectID:   connectID,
		MessageType: stream.MessageType(999), // Unsupported type
	}

	// This should not panic, and should return early without deleting the local connection
	session.ServeConnection(unsupportedMsg)

	// Verify the connection was not deleted
	_, ok := session.GetLocalConnection(connectID)
	if !ok {
		t.Fatalf("Expected connection with ID %v to still be present, but it was deleted", connectID)
	}

	// Verify that the channels were not closed or cleaned
	if mockConn.cleanCalled {
		t.Errorf("Expected CleanChannel not to be called, but it was")
	}
	if mockConn.closeCalled {
		t.Errorf("Expected CloseReadChannel not to be called, but it was")
	}
}

func TestServeDispatchUnsupportedMessageType(t *testing.T) {
	connectID := uint64(12345)
	mockConn := &mockEdgedConnection{}

	unsupportedMsg := &stream.Message{
		ConnectID:   connectID,
		MessageType: stream.MessageType(999), // Unsupported message type
	}

	validDataMsg := &stream.Message{
		ConnectID:   connectID,
		MessageType: stream.MessageTypeData,
		Data:        []byte("hello world"),
	}

	closeMsg := &stream.Message{
		ConnectID:   connectID,
		MessageType: stream.MessageTypeCloseConnect,
		Data:        []byte("session closed"),
	}

	tunneler := &mockTunneler{
		messages: []*stream.Message{unsupportedMsg, validDataMsg, closeMsg},
	}

	session := &TunnelSession{
		Tunnel:    tunneler,
		localCons: make(map[uint64]stream.EdgedConnection),
	}
	session.AddLocalConnection(connectID, mockConn)

	// Serve loop should process unsupportedMsg, validDataMsg, and exit cleanly on closeMsg
	err := session.Serve()
	if err == nil {
		t.Fatalf("Expected error on CloseConnect message, got nil")
	}

	// 1. Connection should not be deleted by unsupported message type
	_, ok := session.GetLocalConnection(connectID)
	if !ok {
		t.Fatalf("Expected local connection %v to still exist", connectID)
	}

	// 2. Unsupported message should NOT be cached, only valid data message should be cached
	if len(mockConn.cachedMsgs) != 1 {
		t.Fatalf("Expected exactly 1 cached message, got %d", len(mockConn.cachedMsgs))
	}
	if mockConn.cachedMsgs[0].MessageType != stream.MessageTypeData {
		t.Errorf("Expected cached message to be MessageTypeData, got %v", mockConn.cachedMsgs[0].MessageType)
	}

	// 3. Channels should not be closed or cleaned
	if mockConn.cleanCalled || mockConn.closeCalled {
		t.Errorf("Expected clean/close not to be called on local connection")
	}
}
