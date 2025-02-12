package cmgr

import (
    "crypto/x509"
    "net"
    "net/http"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/kubeedge/beehive/pkg/core/model"
    "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
)

type MockConnection struct {
    addr  net.Addr
    state conn.ConnectionState
}

func (m *MockConnection) ServeConn() {}

func (m *MockConnection) SetReadDeadline(t time.Time) error {
    return nil
}

func (m *MockConnection) SetWriteDeadline(t time.Time) error {
    return nil
}

func (m *MockConnection) Read(raw []byte) (int, error) {
    return 0, nil
}

func (m *MockConnection) Write(raw []byte) (int, error) {
    return len(raw), nil
}

func (m *MockConnection) WriteMessageAsync(msg *model.Message) error {
    return nil
}

func (m *MockConnection) WriteMessageSync(msg *model.Message) (*model.Message, error) {
    return msg, nil
}

func (m *MockConnection) ReadMessage(msg *model.Message) error {
    return nil
}

func (m *MockConnection) RemoteAddr() net.Addr {
    return m.addr
}

func (m *MockConnection) LocalAddr() net.Addr {
    return m.addr
}

func (m *MockConnection) ConnectionState() conn.ConnectionState {
    return m.state
}

func (m *MockConnection) Close() error {
    return nil
}

func TestConnectionManager(t *testing.T) {
    mockAddr := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
    mockState := conn.ConnectionState{
        State:   "connected",
        Headers: http.Header{"Test": []string{"value"}},
        PeerCertificates: []*x509.Certificate{},
    }
    mockConn := &MockConnection{
        addr:  mockAddr,
        state: mockState,
    }

    t.Run("TestNewManager_WithDefaultKey", func(t *testing.T) {
        mgr := NewManager(nil)
        assert.NotNil(t, mgr)
        assert.NotNil(t, mgr.connKey)
        
        key := mgr.connKey(mockConn)
        assert.Equal(t, "127.0.0.1:8080", key)
    })

    t.Run("TestNewManager_WithCustomKey", func(t *testing.T) {
        customKey := func(c conn.Connection) string {
            return c.ConnectionState().Headers.Get("Test")
        }
        mgr := NewManager(customKey)
        assert.NotNil(t, mgr)
        
        key := mgr.connKey(mockConn)
        assert.Equal(t, "value", key)
    })

    t.Run("TestAddAndGetConnection", func(t *testing.T) {
        mgr := NewManager(nil)
        mgr.AddConnection(mockConn)
        
        conn, exists := mgr.GetConnection("127.0.0.1:8080")
        assert.True(t, exists)
        assert.Equal(t, mockConn, conn)
    })

    t.Run("TestDeleteConnection", func(t *testing.T) {
        mgr := NewManager(nil)
        mgr.AddConnection(mockConn)
        mgr.DelConnection(mockConn)
        
        _, exists := mgr.GetConnection("127.0.0.1:8080")
        assert.False(t, exists)
    })

    t.Run("TestGetNonExistentConnection", func(t *testing.T) {
        mgr := NewManager(nil)
        conn, exists := mgr.GetConnection("non-existent")
        assert.False(t, exists)
        assert.Nil(t, conn)
    })

    t.Run("TestRangeConnections", func(t *testing.T) {
        mgr := NewManager(nil)
        mgr.AddConnection(mockConn)
        
        count := 0
        mgr.Range(func(key, value interface{}) bool {
            count++
            assert.Equal(t, "127.0.0.1:8080", key)
            assert.Equal(t, mockConn, value)
            return true
        })
        assert.Equal(t, 1, count)
    })
}