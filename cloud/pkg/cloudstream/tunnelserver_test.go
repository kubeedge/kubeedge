/*
Copyright 2025 The KubeEdge Authors.

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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	streamconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

const (
	testNodeName    = "test-node"
	testTunnelPort  = 10350
	testCloudCoreIP = "10.0.2.15"
	testStreamPort  = 10003
	testNamespace   = "kubeedge"
	testServiceName = "edge-node-" + testNodeName
)

func setupTest(_ *testing.T) (*TunnelServer, *fake.Clientset) {
	fakeClient := fake.NewSimpleClientset()
	ts := newTunnelServerWithClient(testTunnelPort, 10003, "", fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
	return ts, fakeClient
}

func TestInstallDefaultHandler(t *testing.T) {
	ts := newTunnelServer(testTunnelPort)
	ts.installDefaultHandler()

	foundHandler := false
	for _, ws := range ts.container.RegisteredWebServices() {
		if ws.RootPath() == "/v1/kubeedge/connect" {
			foundHandler = true
			break
		}
	}

	assert.True(t, foundHandler, "Default handler should be registered")
}

func TestSessionManagement(t *testing.T) {
	t.Run("AddAndGetSession", func(t *testing.T) {
		ts := newTunnelServer(testTunnelPort)
		session := &Session{
			sessionID: "test-session",
		}

		ts.addSession("test-key", session)

		retrievedSession, ok := ts.getSession("test-key")
		assert.True(t, ok, "Session should be found")
		assert.Equal(t, session, retrievedSession)

		retrievedSession, ok = ts.getSession("non-existent-key")
		assert.False(t, ok)
		assert.Nil(t, retrievedSession)
	})

	t.Run("AddAndGetNodeIP", func(t *testing.T) {
		ts := newTunnelServer(testTunnelPort)

		ts.addNodeIP("test-node", "192.168.1.1")

		ip, ok := ts.getNodeIP("test-node")
		assert.True(t, ok)
		assert.Equal(t, "192.168.1.1", ip)

		ip, ok = ts.getNodeIP("non-existent-node")
		assert.False(t, ok)
		assert.Equal(t, "", ip)
	})

	t.Run("SessionConcurrency", func(t *testing.T) {
		ts, _ := setupTest(t)

		session1 := &Session{sessionID: "session1"}
		session2 := &Session{sessionID: "session2"}

		ts.addSession("key1", session1)
		ts.addSession("key2", session2)

		s1, ok1 := ts.getSession("key1")
		s2, ok2 := ts.getSession("key2")

		assert.True(t, ok1, "Session 1 should be found")
		assert.True(t, ok2, "Session 2 should be found")
		assert.Equal(t, session1, s1, "Retrieved session 1 should match")
		assert.Equal(t, session2, s2, "Retrieved session 2 should match")
	})
}

func TestUpdateNodeKubeletEndpoint(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					DaemonEndpoints: corev1.NodeDaemonEndpoints{
						KubeletEndpoint: corev1.DaemonEndpoint{Port: 0},
					},
				},
			},
		)

		streamPort := 10003 // renamed from tunnelPort, matches what updateNodeKubeletEndpoint sets
		ts := newTunnelServerWithClient(testTunnelPort, streamPort, "", fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, int32(streamPort), node.Status.DaemonEndpoints.KubeletEndpoint.Port) // was int32(tunnelPort)
	})
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, 10003, "", nil, time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint(testNodeName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kubeclient is nil")
	})

	t.Run("FailureNoNode", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		tunnelPort := testTunnelPort
		ts := newTunnelServerWithClient(tunnelPort, 10003, "", fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint("non-existent-node")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to Update KubeletEndpoint Port")
	})

	t.Run("UpdateFailure", func(t *testing.T) {
		nodeName := "test-node-immutable"
		fakeClient := fake.NewSimpleClientset()

		customClient := &fakeCoreV1{
			CoreV1Interface: fakeClient.CoreV1(),
			nodeName:        nodeName,
		}

		tunnelPort := testTunnelPort
		ts := newTunnelServerWithClient(tunnelPort, 10003, "", customClient, time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint(nodeName)
		assert.Error(t, err, "updateNodeKubeletEndpoint should return an error when update fails")
		assert.Contains(t, err.Error(), "failed to Update KubeletEndpoint Port", "Error message should indicate failure to update")
	})
}

type fakeCoreV1 struct {
	v1.CoreV1Interface
	nodeName string
}

func (f *fakeCoreV1) Nodes() v1.NodeInterface {
	return &fakeNodeInterface{
		f.CoreV1Interface.Nodes(),
		f.nodeName,
	}
}

type fakeNodeInterface struct {
	v1.NodeInterface
	nodeName string
}

func (f *fakeNodeInterface) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Node, error) {
	if name == f.nodeName {
		return &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: f.nodeName,
			},
			Status: corev1.NodeStatus{
				DaemonEndpoints: corev1.NodeDaemonEndpoints{
					KubeletEndpoint: corev1.DaemonEndpoint{
						Port: 0,
					},
				},
			},
		}, nil
	}
	return nil, errors.New("node not found")
}

func (f *fakeNodeInterface) UpdateStatus(_ context.Context, _ *corev1.Node, _ metav1.UpdateOptions) (*corev1.Node, error) {
	return nil, errors.New("simulated update failure")
}

func TestConnect(t *testing.T) {
	// Test cases
	testCases := []struct {
		name               string
		setupRequest       func(*http.Request)
		setupClient        func() *fake.Clientset
		expectedStatusCode int
		nodeName           string
	}{
		{
			name: "WithInternalIP",
			setupRequest: func(req *http.Request) {
				req.Header.Set(stream.SessionKeyHostNameOverride, testNodeName)
				req.Header.Set(stream.SessionKeyInternalIP, "192.168.1.2")
			},
			setupClient: func() *fake.Clientset {
				return fake.NewSimpleClientset(&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNodeName,
					},
				})
			},
			nodeName:           testNodeName,
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "WithoutInternalIP",
			setupRequest: func(req *http.Request) {
				req.Header.Set(stream.SessionKeyHostNameOverride, testNodeName)
				req.RemoteAddr = "192.168.1.3:12345"
			},
			setupClient: func() *fake.Clientset {
				return fake.NewSimpleClientset(&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNodeName,
					},
				})
			},
			nodeName:           testNodeName,
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "NodeUpdateError",
			setupRequest: func(req *http.Request) {
				req.Header.Set(stream.SessionKeyHostNameOverride, "non-existent-node")
				req.Header.Set(stream.SessionKeyInternalIP, "192.168.1.4")
			},
			setupClient: func() *fake.Clientset {
				return fake.NewSimpleClientset()
			},
			nodeName:           "non-existent-node",
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := tc.setupClient()
			ts := newTunnelServerWithClient(testTunnelPort, 10003, "", fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

			req := httptest.NewRequest("GET", "/v1/kubeedge/connect", nil)
			tc.setupRequest(req)
			resp := httptest.NewRecorder()

			restReq := restful.NewRequest(req)
			restResp := restful.NewResponse(resp)

			origUpgrader := ts.upgrader
			ts.upgrader = websocket.Upgrader{
				HandshakeTimeout: time.Second * 2,
				ReadBufferSize:   1024,
				CheckOrigin:      func(r *http.Request) bool { return true },
				Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
					w.WriteHeader(status)
					_, _ = w.Write([]byte(reason.Error()))
				},
			}

			defer func() {
				ts.upgrader = origUpgrader
			}()

			ts.connect(restReq, restResp)

			// In our test cases, WebSocket upgrade should always fail since we're not setting up
			// a proper WebSocket connection
			assert.True(t, resp.Code != http.StatusOK, "connect should fail when WebSocket upgrade fails")
		})
	}
}

func TestTLSSetup(t *testing.T) {
	fakeCert := []byte("fake-certificate-data")
	fakeKey := []byte("fake-key-data")

	// Save original configs
	origHubCa := hubconfig.Config.Ca
	origHubCert := hubconfig.Config.Cert
	origHubKey := hubconfig.Config.Key
	origStreamCa := streamconfig.Config.Ca
	origStreamCert := streamconfig.Config.Cert
	origStreamKey := streamconfig.Config.Key
	origStreamPort := streamconfig.Config.TunnelPort

	defer func() {
		// Restore original configs
		hubconfig.Config.Ca = origHubCa
		hubconfig.Config.Cert = origHubCert
		hubconfig.Config.Key = origHubKey
		streamconfig.Config.Ca = origStreamCa
		streamconfig.Config.Cert = origStreamCert
		streamconfig.Config.Key = origStreamKey
		streamconfig.Config.TunnelPort = origStreamPort
	}()

	testCases := []struct {
		name        string
		setupConfig func()
	}{
		{
			name: "WithHubConfig",
			setupConfig: func() {
				hubconfig.Config.Ca = fakeCert
				hubconfig.Config.Cert = fakeCert
				hubconfig.Config.Key = fakeKey

				streamconfig.Config.Ca = nil
				streamconfig.Config.Cert = nil
				streamconfig.Config.Key = nil
			},
		},
		{
			name: "WithStreamConfig",
			setupConfig: func() {
				hubconfig.Config.Ca = nil
				hubconfig.Config.Cert = nil
				hubconfig.Config.Key = nil

				streamconfig.Config.Ca = fakeCert
				streamconfig.Config.Cert = fakeCert
				streamconfig.Config.Key = fakeKey
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupConfig()
			streamconfig.Config.TunnelPort = testTunnelPort

			ts, _ := setupTest(t)
			assert.NotNil(t, ts.container, "Container should be initialized")
		})
	}
}

func TestEnsureNodeService(t *testing.T) {
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, nil, time.Millisecond*10, time.Millisecond*300)
		err := ts.ensureNodeService(context.Background(), testNodeName, testCloudCoreIP, testStreamPort)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kubeclient is nil")
	})

	t.Run("CreatesServiceAndEndpointsWhenNotExist", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		_, err := fakeClient.CoreV1().Nodes().Create(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
				},
			},
		}, metav1.CreateOptions{})
		assert.NoError(t, err)

		err = ts.ensureNodeService(context.Background(), testNodeName, testCloudCoreIP, testStreamPort)
		assert.NoError(t, err)

		svc, err := fakeClient.CoreV1().Services(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, int32(testStreamPort), svc.Spec.Ports[0].Port)

		ep, err := fakeClient.CoreV1().Endpoints(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, testCloudCoreIP, ep.Subsets[0].Addresses[0].IP)
		assert.Equal(t, int32(testStreamPort), ep.Subsets[0].Ports[0].Port)
	})

	t.Run("UsesExistingServiceClusterIP", func(t *testing.T) {
		existingClusterIP := "10.96.100.1"
		fakeClient := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: existingClusterIP,
					Ports:     []corev1.ServicePort{{Port: int32(testStreamPort)}},
				},
			},
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.ensureNodeService(context.Background(), testNodeName, testCloudCoreIP, testStreamPort)
		assert.NoError(t, err)

		ep, err := fakeClient.CoreV1().Endpoints(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, testCloudCoreIP, ep.Subsets[0].Addresses[0].IP)
	})
}

func TestEnsureNodeEndpoints(t *testing.T) {
	t.Run("CreatesEndpointsWhenNotExist", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.ensureNodeEndpoints(context.Background(), testNodeName, testCloudCoreIP, testStreamPort, "10.96.1.1", testNamespace)
		assert.NoError(t, err)

		ep, err := fakeClient.CoreV1().Endpoints(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, testCloudCoreIP, ep.Subsets[0].Addresses[0].IP)
		assert.Equal(t, int32(testStreamPort), ep.Subsets[0].Ports[0].Port)
	})

	t.Run("UpdatesExistingEndpoints", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{{IP: "10.0.1.1"}},
						Ports:     []corev1.EndpointPort{{Port: int32(testStreamPort)}},
					},
				},
			},
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.ensureNodeEndpoints(context.Background(), testNodeName, testCloudCoreIP, testStreamPort, "10.96.1.1", testNamespace)
		assert.NoError(t, err)

		ep, err := fakeClient.CoreV1().Endpoints(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, testCloudCoreIP, ep.Subsets[0].Addresses[0].IP)
	})
}

func TestUpdateNodeAddressToClusterIP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		clusterIP := "10.96.50.1"
		fakeClient := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeAddressToClusterIP(context.Background(), testNodeName, clusterIP)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, clusterIP, node.Status.Addresses[0].Address)
	})

	t.Run("AlreadyCorrect", func(t *testing.T) {
		clusterIP := "10.96.50.1"
		fakeClient := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: clusterIP},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeAddressToClusterIP(context.Background(), testNodeName, clusterIP)
		assert.NoError(t, err)
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeAddressToClusterIP(context.Background(), "non-existent", "10.96.50.1")
		assert.Error(t, err)
	})

	t.Run("NoInternalIPAddress", func(t *testing.T) {
		clusterIP := "10.96.50.1"
		fakeClient := fake.NewSimpleClientset(
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeHostName, Address: "ubuntu"},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeAddressToClusterIP(context.Background(), testNodeName, clusterIP)
		assert.NoError(t, err)
	})
}

func TestCleanupNodeService(t *testing.T) {
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, nil, time.Millisecond*10, time.Millisecond*300)
		ts.cleanupNodeService(context.Background(), testNodeName)
	})

	t.Run("DeletesServiceAndEndpoints", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
			},
			&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		ts.cleanupNodeService(context.Background(), testNodeName)

		_, err := fakeClient.CoreV1().Services(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.Error(t, err, "Service should be deleted")

		_, err = fakeClient.CoreV1().Endpoints(testNamespace).Get(context.Background(), testServiceName, metav1.GetOptions{})
		assert.Error(t, err, "Endpoints should be deleted")
	})

	t.Run("NotFoundIsNotAnError", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
		ts.cleanupNodeService(context.Background(), testNodeName)
	})
}

func TestReconcileNodeAddress(t *testing.T) {
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, nil, time.Millisecond*10, time.Millisecond*300)
		ts.reconcileNodeAddress(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		})
	})

	t.Run("EmptyCloudCoreIP", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, "", fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
		ts.reconcileNodeAddress(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		})
	})

	t.Run("NoServiceExists", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
		ts.reconcileNodeAddress(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
				},
			},
		})
	})

	t.Run("AddressAlreadyCorrect", func(t *testing.T) {
		clusterIP := "10.96.50.1"
		fakeClient := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
				Spec: corev1.ServiceSpec{ClusterIP: clusterIP},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
		ts.reconcileNodeAddress(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: clusterIP},
				},
			},
		})
	})

	t.Run("AddressDriftedReconciles", func(t *testing.T) {
		clusterIP := "10.96.50.1"
		fakeClient := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
				Spec: corev1.ServiceSpec{ClusterIP: clusterIP},
			},
			&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
					},
				},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		ts.reconcileNodeAddress(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
				},
			},
		})

		time.Sleep(50 * time.Millisecond)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, clusterIP, node.Status.Addresses[0].Address)
	})

	t.Run("ServiceHasNoClusterIP", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testServiceName,
					Namespace: testNamespace,
				},
				Spec: corev1.ServiceSpec{ClusterIP: "None"},
			},
		)
		ts := newTunnelServerWithClient(testTunnelPort, testStreamPort, testCloudCoreIP, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
		ts.reconcileNodeAddress(context.Background(), &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.0.2.15"},
				},
			},
		})
	})
}
