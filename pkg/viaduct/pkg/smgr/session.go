package smgr

import (
	"context"
    "fmt"
    "io"
	"time"

	"github.com/quic-go/quic-go"

	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// wrapper for session manager
type Stream struct {
	// the use type of stream only be stream or message
	UseType api.UseType
	// quic stream
	Stream quic.Stream
	ctx     context.Context
    cancel  context.CancelFunc
}

type Session struct {
	Sess quic.Connection
	ctx  context.Context
}

func NewSession(conn quic.Connection) *Session {
    return &Session{
        Sess: conn,
        ctx:  context.Background(),
    }
}

func (s *Session) OpenStreamSync(streamUse api.UseType) (*Stream, error) {
    // Add timeout context
    ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
    defer cancel()

    stream, err := s.Sess.OpenStreamSync(ctx)
    if err != nil {
        return nil, fmt.Errorf("open stream: %w", err)
    }

    // Set write deadline
    if err := stream.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
        stream.Close()
        return nil, fmt.Errorf("set write deadline: %w", err)
    }

    if _, err = stream.Write([]byte(streamUse)); err != nil {
        stream.Close()
        return nil, fmt.Errorf("write stream type: %w", err)
    }

    streamCtx, streamCancel := context.WithCancel(s.ctx)
    return &Stream{
        UseType: streamUse,
        Stream:  stream,
        ctx:     streamCtx,
        cancel:  streamCancel,
    }, nil
}

func (s *Session) AcceptStream() (*Stream, error) {
    stream, err := s.Sess.AcceptStream(s.ctx)
    if err != nil {
        return nil, fmt.Errorf("accept stream: %w", err)
    }

    // Set read deadline
    if err := stream.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
        stream.Close()
        return nil, fmt.Errorf("set read deadline: %w", err)
    }

    typeBytes := make([]byte, api.UseLen)
    if _, err = io.ReadFull(stream, typeBytes); err != nil {
        stream.Close()
        return nil, fmt.Errorf("read stream type: %w", err)
    }

    streamCtx, streamCancel := context.WithCancel(s.ctx)
    return &Stream{
        UseType: api.UseType(typeBytes),
        Stream:  stream,
        ctx:     streamCtx,
        cancel:  streamCancel,
    }, nil
}

func (s *Session) Close() error {
    return s.Sess.CloseWithError(0, "normal closure")
}

// New method for Stream
func (s *Stream) Close() error {
    s.cancel()
    return s.Stream.Close()
}
