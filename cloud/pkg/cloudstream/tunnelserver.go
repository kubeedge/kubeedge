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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	streamconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

const (
	// The amount of time the tunnelserver should sleep between retrying node status updates
	DefaultRetrySleepTime          = 20 * time.Second
	DefaultNodeStatusUpdateTimeout = 2 * time.Minute
)

type TunnelServer struct {
	container *restful.Container
	upgrader  websocket.Upgrader
	sync.Mutex
	sessions        map[string]*Session
	nodeNameIP      sync.Map
	tunnelPort      int
	streamPort      int
	iptablesMgrMode v1alpha1.IptablesMgrMode
	kubeClient      v1.CoreV1Interface
	retrySleep      time.Duration
	updateTimeout   time.Duration

	edgeTunnelIPOnce     sync.Once
	edgeTunnelIPDecision bool
}

func newTunnelServer(tunnelPort int, streamPort int, iptablesMgrMode v1alpha1.IptablesMgrMode) *TunnelServer {
	var kubeClient v1.CoreV1Interface
	k8sClient := client.GetKubeClient()
	if k8sClient != nil {
		kubeClient = k8sClient.CoreV1()
	}
	return newTunnelServerWithClient(tunnelPort, streamPort, iptablesMgrMode, kubeClient, DefaultRetrySleepTime, DefaultNodeStatusUpdateTimeout)
}

func newTunnelServerWithClient(tunnelPort int, streamPort int, iptablesMgrMode v1alpha1.IptablesMgrMode, kubeClient v1.CoreV1Interface, retrySleep, updateTimeout time.Duration) *TunnelServer {
	return &TunnelServer{
		container:       restful.NewContainer(),
		sessions:        make(map[string]*Session),
		tunnelPort:      tunnelPort,
		streamPort:      streamPort,
		iptablesMgrMode: iptablesMgrMode,
		kubeClient:      kubeClient,
		retrySleep:      retrySleep,
		updateTimeout:   updateTimeout,
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
		klog.Errorf("Failed to upgrade the HTTP server connection to the WebSocket protocol: %v", err)
		return
	}
	klog.Infof("get a new tunnel agent hostname %v, internalIP %v", hostNameOverride, internalIP)

	session := &Session{
		tunnel:        stream.NewDefaultTunnel(con),
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
		sessionID:     hostNameOverride,
	}

	err = s.updateNodeKubeletEndpoint(hostNameOverride)
	if err != nil {
		msg := stream.NewMessage(0, stream.MessageTypeCloseConnect, []byte(err.Error()))
		if err := session.tunnel.WriteMessage(msg); err == nil {
			klog.V(4).Infof("CloudStream send close connection message to edge successfully")
		} else {
			klog.Errorf("CloudStream failed to send close connection message to edge, error: %v", err)
		}
		return
	}

	if features.DefaultFeatureGate.Enabled(features.EdgeTunnelIP) && s.shouldUseEdgeTunnelIP() {
		if err := s.updateNodeEdgeTunnelIP(hostNameOverride); err != nil {
			klog.Warningf("Failed to set EdgeTunnelIP on node %s: %v", hostNameOverride, err)
		}
	}

	s.addSession(hostNameOverride, session)
	s.addSession(internalIP, session)
	s.addNodeIP(hostNameOverride, internalIP)
	session.Serve()

	if features.DefaultFeatureGate.Enabled(features.EdgeTunnelIP) && s.shouldUseEdgeTunnelIP() {
		s.removeNodeEdgeTunnelIP(hostNameOverride)
	}
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

func (s *TunnelServer) updateNodeKubeletEndpoint(nodeName string) error {
	if s.kubeClient == nil {
		klog.V(4).Info("Skip updating node kubelet endpoint in test mode")
		return fmt.Errorf("kubeclient is nil, cannot update node kubelet endpoint")
	}
	if err := wait.PollUntilContextTimeout(context.Background(), s.retrySleep, s.updateTimeout, true,
		func(ctx context.Context) (bool, error) {
			getNode, err := s.kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Failed while getting a Node to retry updating node KubeletEndpoint Port, node: %s, error: %v", nodeName, err)
				return false, nil
			}

			// When EdgeTunnelIP is both enabled and actually needed (see
			// shouldUseEdgeTunnelIP), API server routes via EdgeTunnelIP to
			// cloudCoreIP and connects on streamPort. Otherwise use tunnelPort
			// for iptableManager DNAT routing. These two conditions must match
			// the gating in connect(): if the port switched to streamPort but
			// the EdgeTunnelIP address was never added (because shouldUse was
			// false), the API server would dial the node's real IP on
			// streamPort, which is not listening there.
			if features.DefaultFeatureGate.Enabled(features.EdgeTunnelIP) && s.shouldUseEdgeTunnelIP() {
				getNode.Status.DaemonEndpoints.KubeletEndpoint.Port = int32(s.streamPort)
			} else {
				getNode.Status.DaemonEndpoints.KubeletEndpoint.Port = int32(s.tunnelPort)
			}

			_, err = s.kubeClient.Nodes().UpdateStatus(ctx, getNode, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Failed to update node KubeletEndpoint Port, node: %s, tunnelPort: %d, err: %v", nodeName, s.tunnelPort, err)
				return false, nil
			}
			return true, nil
		}); err != nil {
		klog.Errorf("Update KubeletEndpoint Port of Node '%v' error: %v. ", nodeName, err)
		return fmt.Errorf("failed to Update KubeletEndpoint Port")
	}
	klog.V(4).Infof("Update node KubeletEndpoint Port successfully, node: %s, tunnelPort: %d", nodeName, s.tunnelPort)
	return nil
}

func (s *TunnelServer) updateNodeEdgeTunnelIP(nodeName string) error {
	if s.kubeClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.updateTimeout)
	defer cancel()
	return wait.PollUntilContextTimeout(ctx, s.retrySleep, s.updateTimeout, true,
		func(ctx context.Context) (bool, error) {
			node, err := s.kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Failed to get node %s for EdgeTunnelIP update: %v", nodeName, err)
				return false, nil
			}
			targetIP := ""
			if node.Annotations != nil {
				targetIP = node.Annotations[constants.EdgeMappingCloudKey]
			}
			if targetIP == "" {
				klog.Warningf("node %s has no cloudcore annotation, skipping EdgeTunnelIP update", nodeName)
				return true, nil
			}
			// Check if EdgeTunnelIP is already set correctly
			for _, addr := range node.Status.Addresses {
				if addr.Type == constants.NodeEdgeTunnelIP && addr.Address == targetIP {
					return true, nil
				}
			}
			// Remove any existing EdgeTunnelIP entries and add the correct one
			var addrs []corev1.NodeAddress
			for _, addr := range node.Status.Addresses {
				if addr.Type != constants.NodeEdgeTunnelIP {
					addrs = append(addrs, addr)
				}
			}
			addrs = append(addrs, corev1.NodeAddress{
				Type:    constants.NodeEdgeTunnelIP,
				Address: targetIP,
			})
			node.Status.Addresses = addrs
			_, err = s.kubeClient.Nodes().UpdateStatus(ctx, node, metav1.UpdateOptions{})
			if err != nil {
				klog.Errorf("Failed to set EdgeTunnelIP on node %s: %v", nodeName, err)
				return false, nil
			}
			klog.V(4).Infof("Set EdgeTunnelIP on node %s to %s", nodeName, targetIP)
			return true, nil
		})
}

func (s *TunnelServer) removeNodeEdgeTunnelIP(nodeName string) {
	if s.kubeClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.updateTimeout)
	defer cancel()
	node, err := s.kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("Failed to get node %s for EdgeTunnelIP removal: %v", nodeName, err)
		}
		return
	}
	var addrs []corev1.NodeAddress
	found := false
	for _, addr := range node.Status.Addresses {
		if addr.Type == constants.NodeEdgeTunnelIP {
			found = true
			continue
		}
		addrs = append(addrs, addr)
	}
	if !found {
		return
	}
	node.Status.Addresses = addrs
	_, err = s.kubeClient.Nodes().UpdateStatus(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to remove EdgeTunnelIP from node %s: %v", nodeName, err)
		return
	}
	klog.V(4).Infof("Removed EdgeTunnelIP from node %s", nodeName)
}

// nodeNameEnvVar is the downward-API environment variable cloudcore's
// deployment injects (fieldRef: spec.nodeName) so a running instance can
// identify which node it is scheduled on.
const nodeNameEnvVar = "NODE_NAME"

// shouldUseEdgeTunnelIP returns true only when iptablesManager is in internal
// mode AND kube-apiserver is not colocated with cloudcore -- i.e. the API
// server cannot reach edge nodes via the iptables DNAT rule installed on
// cloudcore's own node. Colocation is determined by isAPIServerColocated, not
// inferred from configuration presence: advertiseAddress is mandatory and
// auto-defaulted by cloudcore, so checking whether it is set can never
// distinguish colocated from separated deployments.
//
// The result is cached for the process lifetime: node placement cannot
// change without a cloudcore restart.
func (s *TunnelServer) shouldUseEdgeTunnelIP() bool {
	if s.iptablesMgrMode != v1alpha1.InternalMode {
		return false
	}
	s.edgeTunnelIPOnce.Do(func() {
		sameNode, determined := isAPIServerColocated(context.Background(), s.kubeClient, os.Getenv(nodeNameEnvVar))
		// Cannot verify placement (e.g. cloudcore not running as a scheduled
		// pod, NODE_NAME unset, or a lookup failure) -- assume separated
		// nodes. A redundant EdgeTunnelIP is harmless when nodes are
		// actually colocated; a missing one silently breaks kubectl
		// exec/logs/attach when they are not.
		s.edgeTunnelIPDecision = !determined || !sameNode
	})
	return s.edgeTunnelIPDecision
}

// isAPIServerColocated reports whether every reachable kube-apiserver
// endpoint resolves to an address of nodeName, i.e. whether a node-local
// iptables DNAT rule installed on that node would actually intercept the API
// server's outbound traffic.
//
// The real, routable API server address(es) are read from the well-known
// default/kubernetes Endpoints object rather than by inspecting Node or Pod
// objects: that Endpoints object is populated by the API server itself, so
// it is accurate for both self-hosted (kubeadm/static-pod) control planes
// and managed control planes (EKS/GKE/AKS/...) where the API server is not a
// node in the cluster at all.
//
// determined reports whether a definitive answer was possible. Callers must
// not treat (false, false) as "not colocated" -- it means unknown, e.g.
// nodeName is empty, or the lookups failed.
func isAPIServerColocated(ctx context.Context, kubeClient v1.CoreV1Interface, nodeName string) (sameNode bool, determined bool) {
	if kubeClient == nil || nodeName == "" {
		return false, false
	}

	node, err := kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infof("failed to get own node %q for EdgeTunnelIP placement check: %v", nodeName, err)
		return false, false
	}
	nodeIPs := make(map[string]bool, len(node.Status.Addresses))
	for _, addr := range node.Status.Addresses {
		nodeIPs[addr.Address] = true
	}

	endpoints, err := kubeClient.Endpoints(corev1.NamespaceDefault).Get(ctx, "kubernetes", metav1.GetOptions{})
	if err != nil {
		klog.V(4).Infof("failed to get default/kubernetes endpoints for EdgeTunnelIP placement check: %v", err)
		return false, false
	}

	found := false
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			found = true
			if !nodeIPs[addr.IP] {
				// This API server replica is not reachable via nodeName --
				// a DNAT rule installed there cannot intercept it.
				return false, true
			}
		}
	}
	return found, found
}
