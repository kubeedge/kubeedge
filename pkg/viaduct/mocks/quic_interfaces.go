package mocks

import (
    "context"
    "time"
    "net"
    "github.com/quic-go/quic-go"
)

type StreamInterface interface {
    StreamID() quic.StreamID
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    CancelRead(quic.StreamErrorCode)
    CancelWrite(quic.StreamErrorCode)
    SetReadDeadline(time.Time) error
    SetWriteDeadline(time.Time) error
    SetDeadline(time.Time) error
    Context() context.Context
}

type ConnectionInterface interface {
    AcceptStream(context.Context) (quic.Stream, error)
    OpenStream() (quic.Stream, error)
    OpenStreamSync(context.Context) (quic.Stream, error)
    LocalAddr() net.Addr
    RemoteAddr() net.Addr
    CloseWithError(quic.ApplicationErrorCode, string) error
    Context() context.Context
    ConnectionState() quic.ConnectionState
}