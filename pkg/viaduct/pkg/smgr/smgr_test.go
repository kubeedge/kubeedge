package smgr

import (
    "context"
    "testing"
    "time"
    "github.com/quic-go/quic-go"
    "github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

type mockQuicStream struct {
    id quic.StreamID
}

func (m *mockQuicStream) StreamID() quic.StreamID           { return m.id }
func (m *mockQuicStream) Read([]byte) (int, error)          { return 0, nil }
func (m *mockQuicStream) Write([]byte) (int, error)         { return 0, nil }
func (m *mockQuicStream) Close() error                      { return nil }
func (m *mockQuicStream) Context() context.Context          { return context.Background() }
func (m *mockQuicStream) CancelRead(quic.StreamErrorCode)   {}
func (m *mockQuicStream) CancelWrite(quic.StreamErrorCode)  {}
func (m *mockQuicStream) SetReadDeadline(t time.Time) error { return nil }
func (m *mockQuicStream) SetWriteDeadline(t time.Time) error { return nil }
func (m *mockQuicStream) SetDeadline(t time.Time) error      { return nil }

func TestStreamManager(t *testing.T) {
    t.Run("single_stream", func(t *testing.T) {
        sm := NewStreamManager(1, false, nil)
        
        stream := &Stream{
            UseType: api.UseTypeMessage,
            Stream:  &mockQuicStream{id: 1},
        }
        sm.AddStream(stream)

        if got := sm.messagePool.len(); got != 1 {
            t.Errorf("pool size = %d, want 1", got)
        }

        got, err := sm.GetStream(api.UseTypeMessage, false, nil)
        if err != nil || got == nil {
            t.Errorf("GetStream failed: err=%v, stream=%v", err, got)
        }

        sm.ReleaseStream(api.UseTypeMessage, got)
        if idle := sm.messagePool.idlePool.len(); idle != 1 {
            t.Errorf("idle pool size = %d, want 1", idle)
        }
    })
}