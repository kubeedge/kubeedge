package clouddatastream

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	streamconfig "github.com/kubeedge/kubeedge/cloud/pkg/clouddatastream/config"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

const (
	// The amount of time the tunnelserver should sleep between retrying node status updates
	retrySleepTime          = 20 * time.Second
	nodeStatusUpdateTimeout = 2 * time.Minute
)

type TunnelServer struct {
	sync.Mutex
	container           *restful.Container
	upgrader            websocket.Upgrader
	sessions            map[string]*Session
	tunnelPort          int
	pendingSessions     map[string]chan struct{}
	pendingSessionsLock sync.Mutex
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
				if err != nil {
					klog.Errorf("failed to write http response, err: %v", err)
				}
			},
		},
		pendingSessions:     make(map[string]chan struct{}),
		pendingSessionsLock: sync.Mutex{},
	}
}

func (ts *TunnelServer) installDefaultHandler() {
	ws := new(restful.WebService)
	ws.Path("/v1/kubeedge/connect")
	ws.Route(ws.GET("/").
		To(ts.connect))
	ts.container.Add(ws)

	ws = new(restful.WebService)
	ws.Path("/v1/kubeedge/videoconnect")
	ws.Route(ws.GET("/").
		To(ts.videoConnect))
	ts.container.Add(ws)
}

func (ts *TunnelServer) addSession(key string, session *Session) {
	ts.Lock()
	ts.sessions[key] = session
	ts.Unlock()
}

func (ts *TunnelServer) getSession(id string) (*Session, bool) {
	ts.Lock()
	defer ts.Unlock()
	sess, ok := ts.sessions[id]
	return sess, ok
}

func (ts *TunnelServer) delSession(id string) {
	ts.Lock()
	delete(ts.sessions, id)
	ts.Unlock()
}

func (ts *TunnelServer) connect(r *restful.Request, w *restful.Response) {
	hostNameOverride := r.HeaderParameter(stream.SessionKeyHostNameOverride)
	internalIP := r.HeaderParameter(stream.SessionKeyInternalIP)
	if internalIP == "" {
		internalIP = strings.Split(r.Request.RemoteAddr, ":")[0]
	}
	con, err := ts.upgrader.Upgrade(w, r.Request, nil)
	if err != nil {
		klog.Errorf("Failed to upgrade the HTTP server connection to the WebSocket protocol: %v", err)
		return
	}
	klog.Infof("get a new tunnel agent hostname %v, internalIP %v", hostNameOverride, internalIP)

	session := &Session{
		sessionID:     hostNameOverride,
		tunnel:        stream.NewDefaultTunnel(con),
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}

	err = ts.updateNodeKubletEndpoint(hostNameOverride)
	if err != nil {
		msg := stream.NewMessage(0, stream.MessageTypeCloseConnect, []byte(err.Error()))
		if err := session.tunnel.WriteMessage(msg); err != nil {
			klog.V(4).Infof("CloudDataStream send close connection message to edge successfully")
		} else {
			klog.Errorf("CloudDataStream failed to send close connection message to edge, error: %v", err)
		}
		return
	}

	ts.addSession(hostNameOverride, session)
	ts.addSession(internalIP, session)
	session.Serve()
}

func (ts *TunnelServer) videoConnect(r *restful.Request, w *restful.Response) {
	hostNameOverride := r.HeaderParameter(stream.SessionKeyHostNameOverride)
	internalIP := r.HeaderParameter(stream.SessionKeyInternalIP)
	if internalIP == "" {
		internalIP = strings.Split(r.Request.RemoteAddr, ":")[0]
	}
	con, err := ts.upgrader.Upgrade(w, r.Request, nil)
	if err != nil {
		klog.Errorf("Failed to upgrade the HTTP server connection to the WebSocket protocol: %v", err)
		return
	}
	klog.Infof("get a new tunnel agent hostname %v, internalIP %v", hostNameOverride, internalIP)

	ep := r.QueryParameter("ep")
	if ep == "" {
		klog.Errorf("videoConnect ep is empty")
		return
	}
	url := r.QueryParameter("url")
	if url == "" {
		klog.Errorf("videoConnect url is empty")
		return
	}

	session := &Session{
		sessionID:     ep,
		tunnel:        stream.NewDefaultTunnel(con),
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}

	ts.addSession(ep, session)
	go session.Serve()
}

func (ts *TunnelServer) Start() {
	ts.installDefaultHandler()
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
		Handler: ts.container,
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

func (s *TunnelServer) updateNodeKubletEndpoint(nodeName string) error {
	if err := wait.PollImmediate(retrySleepTime, nodeStatusUpdateTimeout, func() (bool, error) {
		getNode, err := client.GetKubeClient().CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed while getting a Node to retry updating node KubeletEndpoint Port, node: %s, error: %v", nodeName, err)
			return false, nil
		}

		getNode.Status.DaemonEndpoints.KubeletEndpoint.Port = int32(s.tunnelPort)
		_, err = client.GetKubeClient().CoreV1().Nodes().UpdateStatus(context.Background(), getNode, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to update node KubeletEndpoint Port, node: %s, tunnelPort: %v, err: %v", nodeName, s.tunnelPort, err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		klog.Errorf("Update KubeletEndpoint Port of Node '%v' error: %v. ", nodeName, err)
		return fmt.Errorf("failed to Update KubeletEndpoint Port")
	}
	klog.V(4).Infof("Update node KubeletEndpoint Port successfully, node: %s, tunnelPort: %v", nodeName, s.tunnelPort)
	return nil
}
