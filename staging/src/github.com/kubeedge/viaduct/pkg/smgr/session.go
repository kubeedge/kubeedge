/*
Copyright 2019 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
