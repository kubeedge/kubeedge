package lane

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/quic-go/quic-go"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/packer"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/translator"
)

type QuicLane struct {
	writeDeadline time.Time
	readDeadline  time.Time
	stream        quic.Stream
	ctx           context.Context
    cancel        context.CancelFunc
}

func NewQuicLane(van interface{}) *QuicLane {
    if stream, ok := van.(quic.Stream); ok {
        ctx, cancel := context.WithCancel(context.Background())
        return &QuicLane{
            stream: stream,
            ctx:    ctx,
            cancel: cancel,
        }
    }
	klog.Error("oops! bad type of van")
	return nil
}

func (l *QuicLane) ReadMessage(msg *model.Message) error {
    if err := l.ctx.Err(); err != nil {
        return fmt.Errorf("stream closed: %w", err)
    }
    
    rawData, err := packer.NewReader(l.stream).Read()
    if err != nil {
        if err == io.EOF {
            return err
        }
        return fmt.Errorf("read message: %w", err)
    }

    if err := translator.NewTran().Decode(rawData, msg); err != nil {
        return fmt.Errorf("decode message: %w", err)
    }

	return nil
}

func (l *QuicLane) WriteMessage(msg *model.Message) error {
    if err := l.ctx.Err(); err != nil {
        return fmt.Errorf("stream closed: %w", err)
    }

    rawData, err := translator.NewTran().Encode(msg)
    if err != nil {
        return fmt.Errorf("encode message: %w", err)
    }

	_, err = packer.NewWriter(l.stream).Write(rawData)
	return err
}

func (l *QuicLane) Read(raw []byte) (int, error) {
	return l.stream.Read(raw)
}

func (l *QuicLane) Write(raw []byte) (int, error) {
	return l.stream.Write(raw)
}

func (l *QuicLane) SetReadDeadline(t time.Time) error {
	l.readDeadline = t
	return l.stream.SetReadDeadline(t)
}

func (l *QuicLane) SetWriteDeadline(t time.Time) error {
	l.writeDeadline = t
	return l.stream.SetWriteDeadline(t)
}

func (l *QuicLane) Close() error {
    l.cancel()
    return l.stream.Close()
}
