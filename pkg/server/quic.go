package server

import (
	"fmt"
	"sync"

	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/lucas-clemente/quic-go"
)

type QuicServer struct {
	options      Options
	exOpts       api.QuicServerOption
	listener     quic.Listener
	listenerLock sync.Mutex
}

func NewQuicServer(opts Options, exOpts interface{}) *QuicServer {
	extendOption, ok := exOpts.(api.QuicServerOption)
	if !ok {
		panic("bad extend options")
	}
	return &QuicServer{
		options: opts,
		exOpts:  extendOption,
	}
}

func (srv *QuicServer) serveTLS(quicConfig *quic.Config) error {
	srv.listenerLock.Lock()
	if srv.listener != nil {
		srv.listenerLock.Unlock()
		return fmt.Errorf("ListenAndServeTLS may only be called once")
	}

	listener, err := quic.ListenAddr(srv.options.Addr, srv.options.TLS, quicConfig)
	if err != nil {
		srv.listenerLock.Unlock()
		return err
	}
	srv.listener = listener
	srv.listenerLock.Unlock()

	for {
		session, err := listener.Accept()
		if err != nil {
			return err
		}
		srv.handleSession(session)
	}
}

func (srv *QuicServer) handleSession(session quic.Session) {
	conn := conn.NewConnection(&conn.ConnectionOptions{
		ConnType: api.ProtocolTypeQuic,
		Base:     session,
		Handler:  srv.options.Handler,
	})

	// connection callback
	if srv.options.ConnNotify != nil {
		srv.options.ConnNotify(conn)
	}

	// connection manager
	if srv.options.ConnMgr != nil {
		srv.options.ConnMgr.AddConnection(conn)
	}

	// auto route message to entry
	if srv.options.AutoRoute {
		go conn.ServeConn()
	}
}

func (srv *QuicServer) getQuicConfig() *quic.Config {
	return &quic.Config{
		HandshakeTimeout:   srv.options.HandshakeTimeout,
		KeepAlive:          true,
		MaxIncomingStreams: srv.exOpts.MaxIncomingStreams,
	}
}

func (srv *QuicServer) ListenAndServeTLS() error {
	config := srv.getQuicConfig()
	return srv.serveTLS(config)
}

func (srv *QuicServer) Close() error {
	srv.listenerLock.Lock()
	defer srv.listenerLock.Unlock()

	if srv.listener != nil {
		err := srv.listener.Close()
		srv.listener = nil
		return err
	}

	return nil
}
