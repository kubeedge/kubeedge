package smgr

import (
	"io"
	"time"

	"github.com/lucas-clemente/quic-go"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
)

// DefaultStreamHandshakeTimeout bounds how long OpenStreamSync/AcceptStream
// wait to write/read the stream-type handshake byte(s) on a new QUIC stream,
// used when Session.HandshakeTimeout is unset.
const DefaultStreamHandshakeTimeout = 5 * time.Second

// wrapper for session manager
type Stream struct {
	// the use type of stream only be stream or message
	UseType api.UseType
	// quic stream
	Stream quic.Stream
}

type Session struct {
	Sess quic.Session
	// HandshakeTimeout bounds the write/read deadline applied to the
	// stream-type handshake performed by OpenStreamSync/AcceptStream.
	// If zero, DefaultStreamHandshakeTimeout is used.
	HandshakeTimeout time.Duration
}

func (s *Session) handshakeTimeout() time.Duration {
	if s.HandshakeTimeout <= 0 {
		return DefaultStreamHandshakeTimeout
	}
	return s.HandshakeTimeout
}

func (s *Session) OpenStreamSync(streamUse api.UseType) (*Stream, error) {
	stream, err := s.Sess.OpenStreamSync()
	if err != nil {
		klog.Errorf("failed to open stream, error: %+v", err)
		return nil, err
	}

	if err := stream.SetWriteDeadline(time.Now().Add(s.handshakeTimeout())); err != nil {
		klog.Warningf("failed to set write deadline for stream handshake, error: %+v", err)
	}
	_, err = stream.Write([]byte(streamUse))
	if err != nil {
		klog.Errorf("write stream type, error: %+v", err)
		return nil, err
	}
	if err := stream.SetWriteDeadline(time.Time{}); err != nil {
		klog.Warningf("failed to clear write deadline after stream handshake, error: %+v", err)
	}

	return &Stream{
		UseType: streamUse,
		Stream:  stream,
	}, nil
}

func (s *Session) AcceptStream() (*Stream, error) {
	stream, err := s.Sess.AcceptStream()
	if err != nil {
		klog.Errorf("failed to accept stream, error: %+v", err)
		return nil, err
	}

	if err := stream.SetReadDeadline(time.Now().Add(s.handshakeTimeout())); err != nil {
		klog.Warningf("failed to set read deadline for stream handshake, error: %+v", err)
	}
	typeBytes := make([]byte, api.UseLen)
	_, err = io.ReadFull(stream, typeBytes)
	if err != nil {
		klog.Errorf("read stream type, error: %+v", err)
		return nil, err
	}
	if err := stream.SetReadDeadline(time.Time{}); err != nil {
		klog.Warningf("failed to clear read deadline after stream handshake, error: %+v", err)
	}

	klog.Infof("receive a stream(%s)", string(typeBytes))

	return &Stream{
		UseType: api.UseType(typeBytes),
		Stream:  stream,
	}, nil
}

func (s *Session) Close() error {
	return s.Sess.Close()
}
