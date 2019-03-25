package server

import (
	glog "log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
)

type WSServer struct {
	options Options
	exOpts  api.WSServerOption
	server  *http.Server
}

type LoggerFilter struct{}

func (f *LoggerFilter) Write(p []byte) (n int, err error) {
	output := string(p)
	if strings.Contains(output, "http: TLS handshake error from") {
		return 0, nil
	}
	log.LOGGER.Error(output)
	return os.Stderr.Write(p)
}

func NewWSServer(opts Options, exOpts interface{}) *WSServer {
	extendOption, ok := exOpts.(api.WSServerOption)
	if !ok {
		panic("bad websocket option")
	}

	server := http.Server{
		Addr:      opts.Addr,
		TLSConfig: opts.TLS,
		ErrorLog:  glog.New(&LoggerFilter{}, "", glog.LstdFlags),
	}

	wsServer := &WSServer{
		options: opts,
		exOpts:  extendOption,
		server:  &server,
	}
	http.HandleFunc(extendOption.Path, wsServer.ServeHTTP)
	return wsServer
}

func (srv *WSServer) upgrade(w http.ResponseWriter, r *http.Request) *websocket.Conn {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: srv.options.HandshakeTimeout,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.LOGGER.Errorf("failed to upgrade to websocket")
		return nil
	}
	return conn
}

func (srv *WSServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if srv.exOpts.Filter != nil {
		if filtered := srv.exOpts.Filter(w, req); filtered {
			log.LOGGER.Warnf("failed to filter req")
			return
		}
	}

	wsConn := srv.upgrade(w, req)
	if wsConn == nil {
		return
	}

	conn := conn.NewConnection(&conn.ConnectionOptions{
		ConnType: api.ProtocolTypeWS,
		Base:     wsConn,
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

func (srv *WSServer) ListenAndServeTLS() error {
	return srv.server.ListenAndServeTLS("", "")
}

func (srv *WSServer) Close() error {
	if srv.server != nil {
		return srv.server.Close()
	}
	return nil
}
