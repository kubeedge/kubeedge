package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"k8s.io/klog"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/comm"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/lane"
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

		go srv.handleSession(session)
	}
}

// accept control stream
func (srv *QuicServer) acceptControlStream(session quic.Session) quic.Stream {
	stream, err := session.AcceptStream()
	if err != nil {
		klog.Errorf("failed to accept stream, error:%+v", err)
		return nil
	}
	return stream
}

// receive header from control lane
func (srv *QuicServer) receiveHeader(lane lane.Lane) (http.Header, error) {
	var msg model.Message
	// read control message
	err := lane.ReadMessage(&msg)
	if err != nil {
		klog.Error("failed read control message")
		return nil, err
	}

	// process control message
	result := comm.RespTypeAck
	headers := make(http.Header)
	err = json.Unmarshal(msg.GetContent().([]byte), &headers)
	if err != nil {
		klog.Errorf("failed to unmarshal header, error: %+v", err)
		result = comm.RespTypeNack
	}

	// feedback the response
	resp := msg.NewRespByMessage(&msg, result)
	err = lane.WriteMessage(resp)
	if err != nil {
		klog.Errorf("failed to send response back, error:%+v", err)
		return nil, err
	}
	return headers, nil
}

// handle session
// 1) get connection
// 2) notify connection event
// 3) add connection into manager
// 4) auto route to entries
func (srv *QuicServer) handleSession(session quic.Session) {
	ctrlStream := srv.acceptControlStream(session)
	if ctrlStream == nil {
		klog.Error("failed to accept control stream")
		return
	}

	ctrlLane := lane.NewLane(api.ProtocolTypeQuic, ctrlStream)
	header, err := srv.receiveHeader(ctrlLane)
	if err != nil {
		klog.Errorf("failed to complete get header, error: %+v", err)
	}

	conn := conn.NewConnection(&conn.ConnectionOptions{
		ConnType: api.ProtocolTypeQuic,
		// TODO:
		ConnUse:  api.UseTypeShare,
		Consumer: srv.options.Consumer,
		Base:     session,
		CtrlLane: lane.NewLane(api.ProtocolTypeQuic, ctrlStream),
		Handler:  srv.options.Handler,
		State: &conn.ConnectionState{
			State:   api.StatConnected,
			Headers: header,
		},
		AutoRoute: srv.options.AutoRoute,
	})

	// connection callback
	if srv.options.ConnNotify != nil {
		srv.options.ConnNotify(conn)
	}

	// connection manager
	if srv.options.ConnMgr != nil {
		srv.options.ConnMgr.AddConnection(conn)
	}

	// serve connection
	conn.ServeConn()
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
