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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	streamconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

const (
	// The amount of time the tunnelserver should sleep between retrying node status updates
	retrySleepTime        = 20 * time.Second
	nodeStatusUpdateRetry = 3
)

type TunnelServer struct {
	container *restful.Container
	upgrader  websocket.Upgrader
	sync.Mutex
	sessions   map[string]*Session
	nodeNameIP sync.Map
	tunnelPort int
}

func newTunnelServer(tunnelPort int) *TunnelServer {
	return &TunnelServer{
		container:  restful.NewContainer(),
		sessions:   make(map[string]*Session),
		tunnelPort: tunnelPort,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: time.Second * 2,
			ReadBufferSize:   1024,
			Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				w.WriteHeader(status)
				_, err := w.Write([]byte(reason.Error()))
				klog.Errorf("failed to write http response, err: %v", err)
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

func (s *TunnelServer) addNodeIP(node, ip string) {
	s.nodeNameIP.Store(node, ip)
}

func (s *TunnelServer) getNodeIP(node string) (string, bool) {
	ip, ok := s.nodeNameIP.Load(node)
	if !ok {
		return "", ok
	}
	return ip.(string), ok
}

func (s *TunnelServer) connect(r *restful.Request, w *restful.Response) {
	hostNameOverride := r.HeaderParameter(stream.SessionKeyHostNameOverride)
	internalIP := r.HeaderParameter(stream.SessionKeyInternalIP)
	if internalIP == "" {
		internalIP = strings.Split(r.Request.RemoteAddr, ":")[0]
	}
	con, err := s.upgrader.Upgrade(w, r.Request, nil)
	if err != nil {
		return
	}
	klog.Infof("get a new tunnel agent hostname %v, internalIP %v", hostNameOverride, internalIP)

	session := &Session{
		tunnel:        stream.NewDefaultTunnel(con),
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
		sessionID:     hostNameOverride,
	}

	s.updateNodeKubeletEndpoint(hostNameOverride)
	s.addSession(hostNameOverride, session)
	s.addSession(internalIP, session)
	s.addNodeIP(hostNameOverride, internalIP)
	session.Serve()
}

func (s *TunnelServer) Start() {
	s.installDefaultHandler()
	var data []byte
	var key []byte
	var cert []byte

	if streamconfig.Config.Ca != nil {
		data = streamconfig.Config.Ca
		klog.Info("Succeed in loading TunnelCA from local directory")
	} else {
		data = hubconfig.Config.Ca
		klog.Info("Succeed in loading TunnelCA from CloudHub")
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: data}))

	if streamconfig.Config.Key != nil && streamconfig.Config.Cert != nil {
		cert = streamconfig.Config.Cert
		key = streamconfig.Config.Key
		klog.Info("Succeed in loading TunnelCert and Key from local directory")
	} else {
		cert = hubconfig.Config.Cert
		key = hubconfig.Config.Key
		klog.Info("Succeed in loading TunnelCert and Key from CloudHub")
	}

	certificate, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: cert}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}))
	if err != nil {
		klog.Error("Failed to load TLSTunnelCert and Key")
		panic(err)
	}

	tunnelServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", streamconfig.Config.TunnelPort),
		Handler: s.container,
		TLSConfig: &tls.Config{
			ClientCAs:    pool,
			Certificates: []tls.Certificate{certificate},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
			CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
		},
	}
	klog.Infof("Prepare to start tunnel server ...")
	err = tunnelServer.ListenAndServeTLS("", "")
	if err != nil {
		klog.Exitf("Start tunnelServer error %v\n", err)
		return
	}
}

func (s *TunnelServer) updateNodeKubeletEndpoint(nodeName string) {
	if err := wait.PollImmediate(retrySleepTime, nodeStatusUpdateRetry, func() (bool, error) {
		getNode, err := client.GetKubeClient().CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed while getting a Node to retry updating node KubeletEndpoint Port, node: %s, error: %v", nodeName, err)
			return false, nil
		}

		getNode.Status.DaemonEndpoints.KubeletEndpoint.Port = int32(s.tunnelPort)
		_, err = client.GetKubeClient().CoreV1().Nodes().UpdateStatus(context.Background(), getNode, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update node KubeletEndpoint Port, node: %s, tunnelPort: %s, err: %v", nodeName, s.tunnelPort, err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		klog.Errorf("Update KubeletEndpoint Port of Node '%v' error: %v. ", nodeName, err)
		os.Exit(1)
	}
	klog.V(4).Infof("Update node KubeletEndpoint Port successfully, node: %s, tunnelPort: %s", nodeName, s.tunnelPort)
}
