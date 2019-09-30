package smgr

import (
	"io"

	"k8s.io/klog"

	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/lucas-clemente/quic-go"
)

// wrapper for session manager
type Stream struct {
	// the use type of stream only be stream or message
	UseType api.UseType
	// quic stream
	Stream quic.Stream
}

type Session struct {
	Sess quic.Session
}

func (s *Session) OpenStreamSync(streamUse api.UseType) (*Stream, error) {
	stream, err := s.Sess.OpenStreamSync()
	if err != nil {
		klog.Errorf("failed to open stream, error: %+v", err)
		return nil, err
	}

	// TODO: add write timeout
	_, err = stream.Write([]byte(streamUse))
	if err != nil {
		klog.Errorf("write stream type, error: %+v", err)
		return nil, err
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

	// TODO: add read timeout
	typeBytes := make([]byte, api.UseLen)
	_, err = io.ReadFull(stream, typeBytes)
	if err != nil {
		klog.Errorf("read stream type, error: %+v", err)
		return nil, err
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
