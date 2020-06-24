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

package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"time"

	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/cmgr"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
)

// notify a new connection
type ConnNotify func(conn.Connection)

// protocol server
type ProtocolServer interface {
	ListenAndServeTLS() error
	Close() error
}

// server options
type Options struct {
	Addr             string
	TLS              *tls.Config
	ConnNotify       ConnNotify
	ConnMgr          *cmgr.ConnectionManager
	ConnNumMax       int
	AutoRoute        bool
	HandshakeTimeout time.Duration
	Handler          mux.Handler
	Consumer         io.Writer
}

type Server struct {
	// protocol type used
	Type string
	// the server address
	Addr string
	// the tls config
	TLSConfig *tls.Config
	// ConnNotify will be called when a new connection coming
	ConnNotify ConnNotify
	// the local connection store
	ConnMgr *cmgr.ConnectionManager

	//auto route
	AutoRoute bool
	// handshake timeout
	HandshakeTimeout time.Duration
	// mux handler
	Handler mux.Handler
	// consumer for raw data
	Consumer io.Writer
	// extend options
	ExOpts interface{}

	// protocol server
	protoServer ProtocolServer
}

// get tls config
func (s *Server) getTLSConfig(cert, key string) (*tls.Config, error) {
	var tlsConfig *tls.Config

	if s.TLSConfig == nil {
		tlsConfig = &tls.Config{}
	} else {
		tlsConfig = s.TLSConfig.Clone()
	}

	hasCert := false
	if len(tlsConfig.Certificates) > 0 ||
		tlsConfig.GetCertificate != nil {
		hasCert = true
	}
	if !hasCert || cert != "" || key != "" {
		var err error
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
	}

	return tlsConfig, nil
}

// get the protocol server by protocol type
func (s *Server) getProtoServer(opts Options) error {
	switch s.Type {
	case api.ProtocolTypeQuic:
		s.protoServer = NewQuicServer(opts, s.ExOpts)
		return nil
	case api.ProtocolTypeWS:
		s.protoServer = NewWSServer(opts, s.ExOpts)
		return nil
	}
	return fmt.Errorf("bad protocol type(%s)", s.Type)
}

// listen and serve
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	tlsConfig, err := s.getTLSConfig(certFile, keyFile)
	if err != nil {
		return err
	}

	err = s.getProtoServer(Options{
		Addr:             s.Addr,
		TLS:              tlsConfig,
		ConnNotify:       s.ConnNotify,
		ConnMgr:          s.ConnMgr,
		HandshakeTimeout: s.HandshakeTimeout,
		AutoRoute:        s.AutoRoute,
		Handler:          s.Handler,
		Consumer:         s.Consumer,
	})
	if err != nil {
		return err
	}

	return s.protoServer.ListenAndServeTLS()
}

// close the server
func (s *Server) Close() error {
	return s.protoServer.Close()
}
