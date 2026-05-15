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
	"k8s.io/client-go/tools/cache"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	streamconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/common/constants"
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
	sessions      map[string]*Session
	nodeNameIP    sync.Map
	tunnelPort    int
	streamPort    int
	cloudCoreIP   string
	kubeClient    v1.CoreV1Interface
	retrySleep    time.Duration
	updateTimeout time.Duration
}

func newTunnelServer(tunnelPort int) *TunnelServer {
	var kubeClient v1.CoreV1Interface
	k8sClient := client.GetKubeClient()
	if k8sClient != nil {
		kubeClient = k8sClient.CoreV1()
	}

	cloudCoreIP := ""
	if len(hubconfig.Config.AdvertiseAddress) > 0 {
		cloudCoreIP = hubconfig.Config.AdvertiseAddress[0]
	}

	return newTunnelServerWithClient(tunnelPort, int(streamconfig.Config.StreamPort), cloudCoreIP, kubeClient, DefaultRetrySleepTime, DefaultNodeStatusUpdateTimeout)
}

func newTunnelServerWithClient(tunnelPort int, streamPort int, cloudCoreIP string, kubeClient v1.CoreV1Interface, retrySleep, updateTimeout time.Duration) *TunnelServer {
	return &TunnelServer{
		container:     restful.NewContainer(),
		sessions:      make(map[string]*Session),
		tunnelPort:    tunnelPort,
		streamPort:    streamPort,
		cloudCoreIP:   cloudCoreIP,
		kubeClient:    kubeClient,
		retrySleep:    retrySleep,
		updateTimeout: updateTimeout,
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
	if s.cloudCoreIP != "" {
		if err := s.ensureNodeService(context.Background(), hostNameOverride, s.cloudCoreIP, s.streamPort); err != nil {
			klog.Warningf("Failed to ensure node service for %s: %v", hostNameOverride, err)
		}
	}
	if err != nil {
		msg := stream.NewMessage(0, stream.MessageTypeCloseConnect, []byte(err.Error()))
		if err := session.tunnel.WriteMessage(msg); err == nil {
			klog.V(4).Infof("CloudStream send close connection message to edge successfully")
		} else {
			klog.Errorf("CloudStream failed to send close connection message to edge, error: %v", err)
		}
		return
	}

	if s.cloudCoreIP != "" {
		if err := s.updateNodeCloudCoreAddress(hostNameOverride, s.cloudCoreIP); err != nil {
			klog.Warningf("Failed to update node %s CloudCore address, falling back to iptables: %v", hostNameOverride, err)
		}
	}
	s.addSession(hostNameOverride, session)
	s.addSession(internalIP, session)
	s.addNodeIP(hostNameOverride, internalIP)
	session.Serve()
	if s.cloudCoreIP != "" {
		s.cleanupNodeService(context.Background(), hostNameOverride)
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
	ctx, cancel := context.WithTimeout(context.Background(), s.updateTimeout)
	defer cancel()
	if err := wait.PollUntilContextTimeout(ctx, s.retrySleep, s.updateTimeout, true, func(ctx context.Context) (bool, error) {
		getNode, err := s.kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed while getting a Node to retry updating node KubeletEndpoint Port, node: %s, error: %v", nodeName, err)
			return false, nil
		}

		getNode.Status.DaemonEndpoints.KubeletEndpoint.Port = int32(s.streamPort)
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

func (s *TunnelServer) updateNodeCloudCoreAddress(nodeName, cloudCoreIP string) error {
	if s.kubeClient == nil {
		return fmt.Errorf("kubeclient is nil, cannot update node address")
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.updateTimeout)
	defer cancel()
	if err := wait.PollUntilContextTimeout(ctx, s.retrySleep, s.updateTimeout, true, func(ctx context.Context) (bool, error) {
		node, err := s.kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed to get node %s for address update, err: %v", nodeName, err)
			return false, nil
		}

		for i, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				node.Status.Addresses[i].Address = cloudCoreIP
				_, err = s.kubeClient.Nodes().UpdateStatus(ctx, node, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf("Failed to update node %s address to CloudCore IP %s, err: %v", nodeName, cloudCoreIP, err)
					return false, nil
				}
				klog.V(4).Infof("Updated node %s InternalIP to CloudCore IP %s", nodeName, cloudCoreIP)
				return true, nil
			}
		}
		klog.Warningf("Node %s has no InternalIP address to update", nodeName)
		return true, nil
	}); err != nil {
		return fmt.Errorf("failed to update node %s address to CloudCore IP: %w", nodeName, err)
	}
	return nil
}

func (s *TunnelServer) ensureNodeService(ctx context.Context, nodeName, cloudCoreIP string, streamPort int) error {
	if s.kubeClient == nil {
		return fmt.Errorf("kubeclient is nil, cannot ensure node service")
	}

	namespace := constants.SystemNamespace
	serviceName := fmt.Sprintf("edge-node-%s", nodeName)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels: map[string]string{
				"kubeedge.io/node-name": nodeName,
				"kubeedge.io/managed":   "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     int32(streamPort),
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	existing, err := s.kubeClient.Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get service %s: %w", serviceName, err)
		}
		created, err := s.kubeClient.Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create service %s: %w", serviceName, err)
		}
		klog.V(4).Infof("Created service %s with ClusterIP %s for node %s", serviceName, created.Spec.ClusterIP, nodeName)
		return s.ensureNodeEndpoints(ctx, nodeName, cloudCoreIP, streamPort, created.Spec.ClusterIP, namespace)
	}

	return s.ensureNodeEndpoints(ctx, nodeName, cloudCoreIP, streamPort, existing.Spec.ClusterIP, namespace)
}

func (s *TunnelServer) ensureNodeEndpoints(ctx context.Context, nodeName, cloudCoreIP string, streamPort int, clusterIP, namespace string) error {
	serviceName := fmt.Sprintf("edge-node-%s", nodeName)

	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{IP: cloudCoreIP},
				},
				Ports: []corev1.EndpointPort{
					{
						Port:     int32(streamPort),
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	_, err := s.kubeClient.Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get endpoints %s: %w", serviceName, err)
		}
		_, err = s.kubeClient.Endpoints(namespace).Create(ctx, endpoints, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create endpoints %s: %w", serviceName, err)
		}
	} else {
		_, err = s.kubeClient.Endpoints(namespace).Update(ctx, endpoints, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update endpoints %s: %w", serviceName, err)
		}
	}

	klog.V(4).Infof("Ensured endpoints for service %s pointing to CloudCore %s:%d", serviceName, cloudCoreIP, streamPort)
	return s.updateNodeAddressToClusterIP(ctx, nodeName, clusterIP)
}

func (s *TunnelServer) updateNodeAddressToClusterIP(ctx context.Context, nodeName, clusterIP string) error {
	return wait.PollUntilContextTimeout(ctx, s.retrySleep, s.updateTimeout, true, func(ctx context.Context) (bool, error) {
		node, err := s.kubeClient.Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Failed to get node %s: %v", nodeName, err)
			return false, nil
		}
		for i, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				if addr.Address == clusterIP {
					return true, nil
				}
				node.Status.Addresses[i].Address = clusterIP
				_, err = s.kubeClient.Nodes().UpdateStatus(ctx, node, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf("Failed to update node %s InternalIP to ClusterIP %s: %v", nodeName, clusterIP, err)
					return false, nil
				}
				klog.V(4).Infof("Updated node %s InternalIP to Service ClusterIP %s", nodeName, clusterIP)
				return true, nil
			}
		}
		return true, nil
	})
}

func (s *TunnelServer) cleanupNodeService(ctx context.Context, nodeName string) {
	if s.kubeClient == nil {
		return
	}
	namespace := constants.SystemNamespace
	serviceName := fmt.Sprintf("edge-node-%s", nodeName)
	if err := s.kubeClient.Services(namespace).Delete(ctx, serviceName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Failed to delete service %s: %v", serviceName, err)
	}
	if err := s.kubeClient.Endpoints(namespace).Delete(ctx, serviceName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("Failed to delete endpoints %s: %v", serviceName, err)
	}
	klog.V(4).Infof("Cleaned up service and endpoints for node %s", nodeName)
}

func (s *TunnelServer) startNodeAddressReconciler(ctx context.Context) {
	nodeInformer := informers.GetInformersManager().GetKubeInformerFactory().Core().V1().Nodes()
	if _, err := nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			node, ok := newObj.(*corev1.Node)
			if !ok {
				return
			}
			s.reconcileNodeAddress(ctx, node)
		},
	}); err != nil {
		klog.Errorf("Failed to add event handler for node informer: %v", err)
		return
	}
}

func (s *TunnelServer) reconcileNodeAddress(ctx context.Context, node *corev1.Node) {
	if s.kubeClient == nil || s.cloudCoreIP == "" {
		return
	}

	namespace := constants.SystemNamespace
	serviceName := fmt.Sprintf("edge-node-%s", node.Name)

	svc, err := s.kubeClient.Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return
	}

	clusterIP := svc.Spec.ClusterIP
	if clusterIP == "" || clusterIP == "None" {
		return
	}

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP && addr.Address == clusterIP {
			return
		}
	}

	klog.V(4).Infof("Node %s InternalIP drifted from ClusterIP %s, reconciling", node.Name, clusterIP)
	if err := s.updateNodeAddressToClusterIP(ctx, node.Name, clusterIP); err != nil {
		klog.Errorf("Failed to reconcile node %s address to ClusterIP %s: %v", node.Name, clusterIP, err)
	}
}
