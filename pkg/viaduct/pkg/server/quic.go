package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/api"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/comm"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/lane"
)

type QuicServer struct {
	options      Options
	exOpts       api.QuicServerOption
	listener     quic.Listener
	listenerLock sync.Mutex
	ctx          context.Context
    cancel       context.CancelFunc
}

func NewQuicServer(opts Options, exOpts interface{}) *QuicServer {
	ctx, cancel := context.WithCancel(context.Background())
	extendOption, ok := exOpts.(api.QuicServerOption)
	if !ok {
		cancel()
		panic("bad extend options")
	}
	return &QuicServer{
		options: opts,
		exOpts:  extendOption,
		ctx:     ctx,
        cancel:  cancel,
	}
}

func (srv *QuicServer) serveTLS(quicConfig *quic.Config) error {
    srv.listenerLock.Lock()
	if srv.listener == (quic.Listener{}) {
        srv.listenerLock.Unlock()
        return fmt.Errorf("ListenAndServeTLS may only be called once")
    }

    listener, err := quic.ListenAddr(srv.options.Addr, srv.options.TLS, quicConfig)
    if err != nil {
        srv.listenerLock.Unlock()
        return fmt.Errorf("listen addr: %w", err)
    }
    srv.listener = *listener
    srv.listenerLock.Unlock()

    for {
        select {
        case <-srv.ctx.Done():
            return nil
        default:
            session, err := listener.Accept(srv.ctx)
            if err != nil {
                if err == context.Canceled {
                    return nil
                }
                return fmt.Errorf("accept connection: %w", err)
            }
            go srv.handleSession(session)
        }
    }
}

// accept control stream
func (srv *QuicServer) acceptControlStream(session quic.Connection) (quic.Stream, error) {
    ctx, cancel := context.WithTimeout(srv.ctx, srv.options.HandshakeTimeout)
    defer cancel()
    
    stream, err := session.AcceptStream(ctx)
    if err != nil {
        return nil, fmt.Errorf("accept stream: %w", err)
    }
    return stream, nil
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
func (srv *QuicServer) handleSession(session quic.Connection) {
	ctrlStream, err := srv.acceptControlStream(session)
	if err != nil {
		klog.Errorf("failed to accept control stream: %v", err)
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
			State:            api.StatConnected,
			Headers:          header,
			PeerCertificates: session.ConnectionState().TLS.PeerCertificates,
		},
		AutoRoute:          srv.options.AutoRoute,
		OnReadTransportErr: srv.options.OnReadTransportErr,
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
        HandshakeIdleTimeout: srv.options.HandshakeTimeout,
        MaxIdleTimeout:       srv.options.HandshakeTimeout * 2,
        EnableDatagrams:      true,
		MaxIncomingStreams:   int64(srv.exOpts.MaxIncomingStreams),
        Allow0RTT:            false,
	}
}

func (srv *QuicServer) ListenAndServeTLS() error {
	config := srv.getQuicConfig()
	return srv.serveTLS(config)
}

func (srv *QuicServer) Close() error {
    srv.cancel()
    srv.listenerLock.Lock()
    defer srv.listenerLock.Unlock()

    if srv.listener != (quic.Listener{}) {
        err := srv.listener.Close()
        srv.listener = (quic.Listener{})
        return err
    }
    return nil
}
