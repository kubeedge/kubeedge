package lane

import (
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/kubeedge/beehive/pkg/core/model"
)

// MockLane implements Lane interface
type MockLane struct {
    readData    []byte
    writeData   []byte
    readMsg     *model.Message
    writeMsg    *model.Message
    readDeadline  time.Time
    writeDeadline time.Time
}

func (m *MockLane) Read(raw []byte) (int, error) {
    copy(raw, m.readData)
    return len(m.readData), nil
}

func (m *MockLane) Write(raw []byte) (int, error) {
    m.writeData = append(m.writeData, raw...)
    return len(raw), nil
}

func (m *MockLane) ReadMessage(msg *model.Message) error {
    *msg = *m.readMsg
    return nil
}

func (m *MockLane) WriteMessage(msg *model.Message) error {
    m.writeMsg = msg
    return nil
}

func (m *MockLane) SetReadDeadline(t time.Time) error {
    m.readDeadline = t
    return nil
}

func (m *MockLane) SetWriteDeadline(t time.Time) error {
    m.writeDeadline = t
    return nil
}

func TestLaneInterface(t *testing.T) {
    t.Run("data_operations", func(t *testing.T) {
        lane := &MockLane{
            readData: []byte("test data"),
        }
        
        // Test Write
        data := []byte("hello")
        n, err := lane.Write(data)
        assert.NoError(t, err)
        assert.Equal(t, len(data), n)
        assert.Equal(t, data, lane.writeData)
        
        // Test Read
        buf := make([]byte, 100)
        n, err = lane.Read(buf)
        assert.NoError(t, err)
        assert.Equal(t, len(lane.readData), n)
        assert.Equal(t, lane.readData, buf[:n])
    })

    t.Run("message_operations", func(t *testing.T) {
        lane := &MockLane{
            readMsg: &model.Message{
                Header: model.MessageHeader{ID: "test-id"},
            },
        }
        
        // Test WriteMessage
        msg := &model.Message{
            Header: model.MessageHeader{ID: "msg-1"},
        }
        err := lane.WriteMessage(msg)
        assert.NoError(t, err)
        assert.Equal(t, msg, lane.writeMsg)
        
        // Test ReadMessage
        var readMsg model.Message
        err = lane.ReadMessage(&readMsg)
        assert.NoError(t, err)
        assert.Equal(t, lane.readMsg.Header.ID, readMsg.Header.ID)
    })

    t.Run("deadline_operations", func(t *testing.T) {
        lane := &MockLane{}
        now := time.Now()
        
        err := lane.SetReadDeadline(now)
        assert.NoError(t, err)
        assert.Equal(t, now, lane.readDeadline)
        
        err = lane.SetWriteDeadline(now)
        assert.NoError(t, err)
        assert.Equal(t, now, lane.writeDeadline)
    })
}