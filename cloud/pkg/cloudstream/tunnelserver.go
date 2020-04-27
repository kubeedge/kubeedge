/*
Copyright 2020 The KubeEdge Authors.

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

package cloudstream

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/kubeedge/kubeedge/pkg/stream"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"k8s.io/klog"

	"github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
)

type TunnelServer struct {
	container *restful.Container
	upgrader  websocket.Upgrader
	sync.Mutex
	sessions map[string]*Session
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

func (s *TunnelServer) addSession(key string, session *Session) {
	s.Lock()
	s.sessions[key] = session
	s.Unlock()
}

func (s *TunnelServer) getSession(id string) (*Session, bool) {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.sessions[id]
	return sess, ok
}

func (s *TunnelServer) connect(r *restful.Request, w *restful.Response) {
	hostNameOverride := r.HeaderParameter(stream.SessionKeyHostNameOveride)
	interalIP := r.HeaderParameter(stream.SessionKeyInternalIP)
	con, err := s.upgrader.Upgrade(w, r.Request, nil)
	if err != nil {
		return
	}
	klog.Infof("get a new tunnel agent hostname %v, internalIP %v", hostNameOverride, interalIP)

	session := &Session{
		tunnel:        stream.NewDefaultTunnel(con),
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.Mutex{},
		sessionID:     hostNameOverride,
	}

	s.addSession(hostNameOverride, session)
	s.addSession(interalIP, session)
	session.Serve()
}

func (s *TunnelServer) Start() {
	s.installDefaultHandler()
	data, err := ioutil.ReadFile(config.Config.TLSTunnelCAFile)
	if err != nil {
		klog.Fatalf("Read tls tunnel ca file error %v", err)
		return
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(data)

	cert, err := ioutil.ReadFile(config.Config.TLSTunnelCertFile)
	if err != nil {
		klog.Fatalf("Read cert file %v error %v", config.Config.TLSTunnelCertFile, err)
	}
	key, err := ioutil.ReadFile(config.Config.TLSTunnelPrivateKeyFile)
	if err != nil {
		klog.Fatalf("Read key file %v error %v", config.Config.TLSTunnelPrivateKeyFile, err)
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
	klog.Infof("Prepare to start tunnel server ...")
	err = tunnelServer.ListenAndServeTLS("", "")
	if err != nil {
		klog.Fatalf("Start tunnelServer error %v\n", err)
		return
	}
}
