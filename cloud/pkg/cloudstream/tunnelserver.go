package cloudstream

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
)

type TunnelServer struct {
	container *restful.Container
	upgrader  websocket.Upgrader
	sync.Mutex
	sessions map[string]*Session // key 是根据agent 发来的id 做区分
}

func newTunnelServer() *TunnelServer {
	return &TunnelServer{
		container: restful.NewContainer(),
		sessions:  make(map[string]*Session),
		upgrader: websocket.Upgrader{
			HandshakeTimeout: time.Second * 2,
			ReadBufferSize:   1024,
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				w.WriteHeader(status)
				w.Write([]byte(reason.Error()))
			},
		},
	}
}

func (s *TunnelServer) installDefaultHandler() {
	ws := new(restful.WebService)
	ws.Path("/v1/kubeedge/connect")
	ws.Route(ws.GET("/").
		To(s.connect))
	s.container.Add(ws)
}

func (s *TunnelServer) addSession(id string, con *websocket.Conn) *Session {
	session := &Session{
		nextID:        0,
		tunnelCon:     con,
		apiServerConn: make(map[uint64]ApiServerConnection),
		sessionID:     id,
	}
	s.Lock()
	s.sessions[id] = session
	s.Unlock()
	return session
}

func (s *TunnelServer) getSession(id string) (*Session, bool) {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

func (s *TunnelServer) connect(r *restful.Request, w *restful.Response) {
	// TODO change Host to overrider
	id := r.HeaderParameter("Host")
	con, err := s.upgrader.Upgrade(w, r.Request, nil)
	if err != nil {
		return
	}
	klog.Infof("get a new tunnelCon agent %v", id)
	session := s.addSession(id, con)
	session.Serve()
}

func (s *TunnelServer) Start() {

	s.installDefaultHandler()
	data, err := ioutil.ReadFile(config.Config.TLSTunnelCAFile)
	if err != nil {
		klog.Fatalf("read tls tunnel ca file error %v", err)
		return
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(data)

	cert, err := ioutil.ReadFile(config.Config.TLSTunnelCertFile)
	if err != nil {
		klog.Fatalf("read cert file %v error %v", config.Config.TLSTunnelCertFile, err)
	}
	key, err := ioutil.ReadFile(config.Config.TLSTunnelPrivateKeyFile)
	if err != nil {
		klog.Fatalf("read key file %v error %v", config.Config.TLSTunnelPrivateKeyFile, err)
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}

	tunnelServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Config.TunnelPort),
		Handler: s.container,
		TLSConfig: &tls.Config{
			ClientCAs:    pool,
			Certificates: []tls.Certificate{certificate},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
	}
	klog.Infof("prepare to start tunnel server ...")
	err = tunnelServer.ListenAndServeTLS("", "")
	if err != nil {
		klog.Fatalf("start tunnelServer error %v\n", err)
		return
	}
}
