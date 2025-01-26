package fifo

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/kubeedge/beehive/pkg/core/model"
    "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/comm"
)

func TestMessageFifo(t *testing.T) {
    t.Run("Initialize", func(t *testing.T) {
        fifo := NewMessageFifo()
        assert.NotNil(t, fifo)
        assert.NotNil(t, fifo.fifo)
        assert.Equal(t, comm.MessageFiFoSizeMax, cap(fifo.fifo))
    })

    t.Run("PutAndGet", func(t *testing.T) {
        fifo := NewMessageFifo()
        msg := &model.Message{Header: model.MessageHeader{ID: "test-1"}}
        
        fifo.Put(msg)
        
        var received model.Message
        err := fifo.Get(&received)
        assert.NoError(t, err)
        assert.Equal(t, msg.Header.ID, received.Header.ID)
    })

    t.Run("Overflow", func(t *testing.T) {
        fifo := NewMessageFifo()
        msg1 := &model.Message{Header: model.MessageHeader{ID: "test-1"}}
        msg2 := &model.Message{Header: model.MessageHeader{ID: "test-2"}}
        
        for i := 0; i < comm.MessageFiFoSizeMax+1; i++ {
            fifo.Put(msg1)
        }
        
        fifo.Put(msg2)
        
        var received model.Message
        err := fifo.Get(&received)
        assert.NoError(t, err)
        assert.Equal(t, msg1.Header.ID, received.Header.ID)
    })

    t.Run("CloseAndGet", func(t *testing.T) {
        fifo := NewMessageFifo()
        fifo.Close()
        
        var msg model.Message
        err := fifo.Get(&msg)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "fifo is broken")
    })

    t.Run("MultipleClose", func(t *testing.T) {
        fifo := NewMessageFifo()
        fifo.Close()
        fifo.Close()
    })
}