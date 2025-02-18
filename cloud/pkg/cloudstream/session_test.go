package cloudstream

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

// mockTunnel implements stream.SafeWriteTunneler for testing
type mockTunnel struct {
	messageType int
	data        []byte
	closed      bool
	err         error
	readCount   int
	messages    []*struct {
		messageType int
		data        []byte
		err         error
	}
}

func newMockTunnel() *mockTunnel {
	return &mockTunnel{
		messages: make([]*struct {
			messageType int
			data        []byte
			err         error
		}, 0),
	}
}

func (m *mockTunnel) WriteMessage(_ *stream.Message) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockTunnel) WriteControl(_ int, _ []byte, _ time.Time) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockTunnel) Close() error {
	m.closed = true
	return nil
}

func (m *mockTunnel) NextReader() (messageType int, reader io.Reader, err error) {
	if len(m.messages) == 0 {
		return 0, nil, fmt.Errorf("no more messages")
	}

	msg := m.messages[0]
	m.messages = m.messages[1:]

	if msg.err != nil {
		return 0, nil, msg.err
	}

	return msg.messageType, bytes.NewReader(msg.data), nil
}

func (m *mockTunnel) AddMessage(messageType int, data []byte, err error) {
	m.messages = append(m.messages, &struct {
		messageType int
		data        []byte
		err         error
	}{
		messageType: messageType,
		data:        data,
		err:         err,
	})
}

// mockAPIServerConnection implements APIServerConnection for testing
type mockAPIServerConnection struct {
	messageID uint64
	done      bool
	writeErr  error
	writeBuf  []byte
	doneChan  chan struct{}
}

func newMockAPIServerConnection() *mockAPIServerConnection {
	return &mockAPIServerConnection{
		doneChan: make(chan struct{}),
	}
}

func (m *mockAPIServerConnection) SendConnection() (stream.EdgedConnection, error) {
	return nil, nil
}

func (m *mockAPIServerConnection) WriteToTunnel(_ *stream.Message) error {
	return nil
}

func (m *mockAPIServerConnection) WriteToAPIServer(data []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.writeBuf = append(m.writeBuf, data...)
	return len(data), nil
}

func (m *mockAPIServerConnection) SetMessageID(id uint64) {
	m.messageID = id
}

func (m *mockAPIServerConnection) GetMessageID() uint64 {
	return m.messageID
}

func (m *mockAPIServerConnection) Serve() error {
	return nil
}

func (m *mockAPIServerConnection) SetEdgePeerDone() {
	m.done = true
	close(m.doneChan)
}

func (m *mockAPIServerConnection) EdgePeerDone() chan struct{} {
	return m.doneChan
}

func (m *mockAPIServerConnection) String() string {
	return fmt.Sprintf("mock connection %d", m.messageID)
}

func TestSession_WriteMessageToTunnel(t *testing.T) {
	mockTun := &mockTunnel{}
	session := &Session{
		sessionID:     "test-session",
		tunnel:        mockTun,
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}

	msg := &stream.Message{
		MessageType: stream.MessageTypeData,
		ConnectID:   1,
		Data:        []byte("test data"),
	}

	err := session.WriteMessageToTunnel(msg)
	assert.NoError(t, err)

	// Test error case
	mockTun.err = fmt.Errorf("write error")
	err = session.WriteMessageToTunnel(msg)
	assert.Error(t, err)
}

func TestSession_Close(t *testing.T) {
	mockTun := &mockTunnel{}
	mockConn1 := newMockAPIServerConnection()
	mockConn2 := newMockAPIServerConnection()

	session := &Session{
		sessionID: "test-session",
		tunnel:    mockTun,
		apiServerConn: map[uint64]APIServerConnection{
			1: mockConn1,
			2: mockConn2,
		},
		apiConnlock: &sync.RWMutex{},
	}

	session.Close()

	assert.True(t, mockTun.closed)
	assert.True(t, mockConn1.done)
	assert.True(t, mockConn2.done)
	assert.True(t, session.tunnelClosed)
}
func TestSession_ProxyTunnelMessageToApiserver(t *testing.T) {
	tests := []struct {
		name        string
		message     *stream.Message
		setupMock   func() (*Session, *mockAPIServerConnection)
		expectError bool
	}{
		{
			name: "valid data message",
			message: &stream.Message{
				MessageType: stream.MessageTypeData,
				ConnectID:   1,
				Data:        []byte("test data"),
			},
			setupMock: func() (*Session, *mockAPIServerConnection) {
				mockConn := newMockAPIServerConnection()
				session := &Session{
					sessionID: "test-session",
					apiServerConn: map[uint64]APIServerConnection{
						1: mockConn,
					},
					apiConnlock: &sync.RWMutex{},
				}
				return session, mockConn
			},
			expectError: false,
		},
		{
			name: "connection not found",
			message: &stream.Message{
				MessageType: stream.MessageTypeData,
				ConnectID:   2,
				Data:        []byte("test data"),
			},
			setupMock: func() (*Session, *mockAPIServerConnection) {
				mockConn := newMockAPIServerConnection()
				session := &Session{
					sessionID: "test-session",
					apiServerConn: map[uint64]APIServerConnection{
						1: mockConn,
					},
					apiConnlock: &sync.RWMutex{},
				}
				return session, mockConn
			},
			expectError: true,
		},
		{
			name: "remove connection message",
			message: &stream.Message{
				MessageType: stream.MessageTypeRemoveConnect,
				ConnectID:   1,
			},
			setupMock: func() (*Session, *mockAPIServerConnection) {
				mockConn := newMockAPIServerConnection()
				session := &Session{
					sessionID: "test-session",
					apiServerConn: map[uint64]APIServerConnection{
						1: mockConn,
					},
					apiConnlock: &sync.RWMutex{},
				}
				return session, mockConn
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, mockConn := tt.setupMock()
			err := session.ProxyTunnelMessageToApiserver(tt.message)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.message.MessageType == stream.MessageTypeData {
					assert.Equal(t, tt.message.Data, mockConn.writeBuf)
				}
				if tt.message.MessageType == stream.MessageTypeRemoveConnect {
					assert.True(t, mockConn.done)
				}
			}
		})
	}
}

func TestSession_AddAPIServerConnection(t *testing.T) {
	mockTun := &mockTunnel{}
	session := &Session{
		sessionID:     "test-session",
		tunnel:        mockTun,
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}

	streamServer := &StreamServer{
		nextMessageID: 0,
	}

	tests := []struct {
		name         string
		tunnelClosed bool
		expectError  bool
	}{
		{
			name:         "successful addition",
			tunnelClosed: false,
			expectError:  false,
		},
		{
			name:         "tunnel closed",
			tunnelClosed: true,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session.tunnelClosed = tt.tunnelClosed
			mockConn := newMockAPIServerConnection()

			conn, err := session.AddAPIServerConnection(streamServer, mockConn)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, conn)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conn)
				assert.Equal(t, uint64(1), mockConn.GetMessageID())
				assert.Contains(t, session.apiServerConn, uint64(1))
			}
		})
	}
}

func TestSession_DeleteAPIServerConnection(t *testing.T) {
	mockConn := newMockAPIServerConnection()
	mockConn.SetMessageID(1)

	session := &Session{
		sessionID: "test-session",
		apiServerConn: map[uint64]APIServerConnection{
			1: mockConn,
		},
		apiConnlock: &sync.RWMutex{},
	}

	session.DeleteAPIServerConnection(mockConn)

	assert.NotContains(t, session.apiServerConn, uint64(1))
}

func TestSession_Serve(t *testing.T) {
	validMessage := stream.Message{
		MessageType: stream.MessageTypeData,
		ConnectID:   1,
		Data:        []byte("test data"),
	}
	validMessageBytes := validMessage.Bytes()

	tests := []struct {
		name            string
		setupMock       func() (*Session, *mockTunnel)
		setupMessages   func(*mockTunnel)
		expectedClosed  bool
		expectedTimeout bool
	}{
		{
			name: "successful message processing",
			setupMock: func() (*Session, *mockTunnel) {
				mockTun := newMockTunnel()
				mockConn := newMockAPIServerConnection()
				session := &Session{
					sessionID:     "test-session",
					tunnel:        mockTun,
					apiServerConn: map[uint64]APIServerConnection{1: mockConn},
					apiConnlock:   &sync.RWMutex{},
				}
				return session, mockTun
			},
			setupMessages: func(m *mockTunnel) {
				m.AddMessage(websocket.TextMessage, validMessageBytes, nil)
				m.AddMessage(0, nil, fmt.Errorf("exit test"))
			},
			expectedClosed:  true,
			expectedTimeout: false,
		},
		{
			name: "non-text message type",
			setupMock: func() (*Session, *mockTunnel) {
				mockTun := newMockTunnel()
				session := &Session{
					sessionID:     "test-session",
					tunnel:        mockTun,
					apiServerConn: make(map[uint64]APIServerConnection),
					apiConnlock:   &sync.RWMutex{},
				}
				return session, mockTun
			},
			setupMessages: func(m *mockTunnel) {
				m.AddMessage(websocket.BinaryMessage, []byte("binary data"), nil)
			},
			expectedClosed:  true,
			expectedTimeout: false,
		},
		{
			name: "invalid message format",
			setupMock: func() (*Session, *mockTunnel) {
				mockTun := newMockTunnel()
				session := &Session{
					sessionID:     "test-session",
					tunnel:        mockTun,
					apiServerConn: make(map[uint64]APIServerConnection),
					apiConnlock:   &sync.RWMutex{},
				}
				return session, mockTun
			},
			setupMessages: func(m *mockTunnel) {
				m.AddMessage(websocket.TextMessage, []byte("invalid message format"), nil)
			},
			expectedClosed:  true,
			expectedTimeout: false,
		},
		{
			name: "reader error",
			setupMock: func() (*Session, *mockTunnel) {
				mockTun := newMockTunnel()
				session := &Session{
					sessionID:     "test-session",
					tunnel:        mockTun,
					apiServerConn: make(map[uint64]APIServerConnection),
					apiConnlock:   &sync.RWMutex{},
				}
				return session, mockTun
			},
			setupMessages: func(m *mockTunnel) {
				m.AddMessage(0, nil, fmt.Errorf("reader error"))
			},
			expectedClosed:  true,
			expectedTimeout: false,
		},
		{
			name: "proxy error then success",
			setupMock: func() (*Session, *mockTunnel) {
				mockTun := newMockTunnel()
				session := &Session{
					sessionID:     "test-session",
					tunnel:        mockTun,
					apiServerConn: make(map[uint64]APIServerConnection),
					apiConnlock:   &sync.RWMutex{},
				}
				return session, mockTun
			},
			setupMessages: func(m *mockTunnel) {
				invalidMsg := stream.Message{
					MessageType: stream.MessageTypeData,
					ConnectID:   999, // Invalid connection ID
					Data:        []byte("test data"),
				}
				invalidMsgBytes := invalidMsg.Bytes()

				// First send message that will cause proxy error
				m.AddMessage(websocket.TextMessage, invalidMsgBytes, nil)
				// Then send valid message
				m.AddMessage(websocket.TextMessage, validMessageBytes, nil)
				// Finally exit
				m.AddMessage(0, nil, fmt.Errorf("exit test"))
			},
			expectedClosed:  true,
			expectedTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, mockTun := tt.setupMock()
			tt.setupMessages(mockTun)

			done := make(chan struct{})
			go func() {
				session.Serve()
				close(done)
			}()

			select {
			case <-done:
				if tt.expectedTimeout {
					t.Error("Test completed before timeout when timeout was expected")
				}
			case <-time.After(100 * time.Millisecond):
				if !tt.expectedTimeout {
					t.Error("Test timed out when timeout was not expected")
				}
				session.Close()
				<-done
			}

			assert.Equal(t, tt.expectedClosed, session.tunnelClosed)
		})
	}
}
