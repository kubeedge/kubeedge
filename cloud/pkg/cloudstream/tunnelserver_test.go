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

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	streamconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudstream/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/nodetopology"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/features"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

const (
	testNodeName              = "test-node"
	testTunnelPort            = 10350
	testEdgeTunnelCloudCoreIP = "10.0.0.1"
)

func setupTest(_ *testing.T) (*TunnelServer, *fake.Clientset) {
	fakeClient := fake.NewSimpleClientset()
	ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
	return ts, fakeClient
}

func TestInstallDefaultHandler(t *testing.T) {
	ts := newTunnelServer(testTunnelPort, 0, v1alpha1.InternalMode)
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
		ts := newTunnelServer(testTunnelPort, 0, v1alpha1.InternalMode)
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
		ts := newTunnelServer(testTunnelPort, 0, v1alpha1.InternalMode)

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

		tunnelPort := testTunnelPort
		ts := newTunnelServerWithClient(tunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, int32(tunnelPort), node.Status.DaemonEndpoints.KubeletEndpoint.Port)
	})
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, nil, time.Millisecond*10, time.Millisecond*100)

		err := ts.updateNodeKubeletEndpoint(testNodeName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "kubeclient is nil")
	})

	t.Run("FailureNoNode", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()

		tunnelPort := testTunnelPort
		ts := newTunnelServerWithClient(tunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)

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
		ts := newTunnelServerWithClient(tunnelPort, 0, v1alpha1.InternalMode, customClient, time.Millisecond*10, time.Millisecond*100)

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
			ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)

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

func TestUpdateNodeEdgeTunnelIP(t *testing.T) {
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, nil, time.Millisecond*10, time.Millisecond*100)
		err := ts.updateNodeEdgeTunnelIP(testNodeName)
		assert.NoError(t, err)
	})

	t.Run("NodeHasNoAnnotation", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
		err := ts.updateNodeEdgeTunnelIP(testNodeName)
		assert.NoError(t, err)
		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		for _, addr := range node.Status.Addresses {
			assert.NotEqual(t, constants.NodeEdgeTunnelIP, addr.Type, "no EdgeTunnelIP should be set when annotation is missing")
		}
	})

	t.Run("SuccessWithAnnotation", func(t *testing.T) {
		annotationIP := "192.168.1.100"
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testNodeName,
				Annotations: map[string]string{constants.EdgeMappingCloudKey: annotationIP},
			},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeEdgeTunnelIP(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)

		found := false
		for _, addr := range node.Status.Addresses {
			if addr.Type == constants.NodeEdgeTunnelIP {
				assert.Equal(t, annotationIP, addr.Address, "EdgeTunnelIP should be annotation IP")
				found = true
			}
		}
		assert.True(t, found, "EdgeTunnelIP should be set")
	})

	t.Run("NoAnnotationGracefulSkip", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeEdgeTunnelIP(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		for _, addr := range node.Status.Addresses {
			assert.NotEqual(t, constants.NodeEdgeTunnelIP, addr.Type, "EdgeTunnelIP should not be set when cloudcore annotation is absent")
		}
	})

	t.Run("AlreadyCorrect", func(t *testing.T) {
		cloudCoreIP := testEdgeTunnelCloudCoreIP
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testNodeName,
				Annotations: map[string]string{constants.EdgeMappingCloudKey: cloudCoreIP},
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: constants.NodeEdgeTunnelIP, Address: cloudCoreIP},
				},
			},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeEdgeTunnelIP(testNodeName)
		assert.NoError(t, err)
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*50)

		err := ts.updateNodeEdgeTunnelIP("non-existent-node")
		assert.Error(t, err)
	})

	t.Run("ReplacesExistingEdgeTunnelIP", func(t *testing.T) {
		cloudCoreIP := "10.0.0.2"
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testNodeName,
				Annotations: map[string]string{constants.EdgeMappingCloudKey: cloudCoreIP},
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "192.168.1.5"},
					{Type: constants.NodeEdgeTunnelIP, Address: testEdgeTunnelCloudCoreIP},
				},
			},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeEdgeTunnelIP(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)

		var edgeTunnelIPs []string
		for _, addr := range node.Status.Addresses {
			if addr.Type == constants.NodeEdgeTunnelIP {
				edgeTunnelIPs = append(edgeTunnelIPs, addr.Address)
			}
		}
		assert.Equal(t, 1, len(edgeTunnelIPs), "should have exactly one EdgeTunnelIP after update")
		assert.Equal(t, cloudCoreIP, edgeTunnelIPs[0], "old EdgeTunnelIP must be replaced")
	})
}

func TestRemoveNodeEdgeTunnelIP(t *testing.T) {
	t.Run("NilKubeClient", func(t *testing.T) {
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, nil, time.Millisecond*10, time.Millisecond*100)
		ts.removeNodeEdgeTunnelIP(testNodeName) // must not panic
	})

	t.Run("RemovesEdgeTunnelIPOnly", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "192.168.1.5"},
					{Type: corev1.NodeHostName, Address: testNodeName},
					{Type: constants.NodeEdgeTunnelIP, Address: testEdgeTunnelCloudCoreIP},
				},
			},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		ts.removeNodeEdgeTunnelIP(testNodeName)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		for _, addr := range node.Status.Addresses {
			assert.NotEqual(t, constants.NodeEdgeTunnelIP, addr.Type, "EdgeTunnelIP must be removed")
		}
		assert.Equal(t, 2, len(node.Status.Addresses), "InternalIP and Hostname must remain")
	})

	t.Run("NoEdgeTunnelIPPresent", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "192.168.1.5"},
				},
			},
		})
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		ts.removeNodeEdgeTunnelIP(testNodeName)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(node.Status.Addresses), "address list must be unchanged")
	})

	t.Run("NodeNotFound", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)
		ts.removeNodeEdgeTunnelIP("non-existent-node") // not-found is success; must not panic
	})
}

func TestEdgeTunnelIPPreservation(t *testing.T) {
	// Simulates the upstream.go heartbeat pattern:
	// node has EdgeTunnelIP set, edgecore heartbeat overwrites status
	// with only InternalIP+Hostname, preservation logic re-appends EdgeTunnelIP.
	cloudCoreIP := testEdgeTunnelCloudCoreIP
	realEdgeIP := "192.168.1.5"

	originalAddresses := []corev1.NodeAddress{
		{Type: corev1.NodeInternalIP, Address: realEdgeIP},
		{Type: corev1.NodeHostName, Address: testNodeName},
		{Type: constants.NodeEdgeTunnelIP, Address: cloudCoreIP},
	}

	// Step 1: collect EdgeTunnelIP before overwrite (as upstream.go does)
	var edgeTunnelAddrs []corev1.NodeAddress
	for _, addr := range originalAddresses {
		if addr.Type == constants.NodeEdgeTunnelIP {
			edgeTunnelAddrs = append(edgeTunnelAddrs, addr)
		}
	}

	// Step 2: edgecore heartbeat overwrites status — only InternalIP and Hostname
	heartbeatAddresses := []corev1.NodeAddress{
		{Type: corev1.NodeInternalIP, Address: realEdgeIP},
		{Type: corev1.NodeHostName, Address: testNodeName},
	}

	// Step 3: preservation logic re-appends EdgeTunnelIP (as upstream.go does)
	result := heartbeatAddresses
	if len(edgeTunnelAddrs) > 0 {
		result = append(result, edgeTunnelAddrs...)
	}

	var foundEdgeTunnel bool
	var internalIP string
	for _, addr := range result {
		switch addr.Type {
		case constants.NodeEdgeTunnelIP:
			assert.Equal(t, cloudCoreIP, addr.Address, "preserved EdgeTunnelIP must equal cloudCoreIP")
			foundEdgeTunnel = true
		case corev1.NodeInternalIP:
			internalIP = addr.Address
		}
	}
	assert.True(t, foundEdgeTunnel, "EdgeTunnelIP must survive the heartbeat overwrite")
	assert.Equal(t, realEdgeIP, internalIP, "InternalIP must remain the real edge IP after overwrite")
}

func TestFeatureGateDisabled(t *testing.T) {
	// When feature gate is off, connect() only calls updateNodeKubeletEndpoint
	// (with tunnelPort), never updateNodeEdgeTunnelIP, so no EdgeTunnelIP appears.
	fakeClient := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "192.168.1.5"},
			},
		},
	})
	ts := newTunnelServerWithClient(testTunnelPort, 10003, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

	// Feature gate is disabled (default false) — only updateNodeKubeletEndpoint is called
	err := ts.updateNodeKubeletEndpoint(testNodeName)
	assert.NoError(t, err)

	node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
	assert.NoError(t, err)

	for _, addr := range node.Status.Addresses {
		assert.NotEqual(t, constants.NodeEdgeTunnelIP, addr.Type, "no EdgeTunnelIP when feature gate is off")
	}
	assert.Equal(t, int32(testTunnelPort), node.Status.DaemonEndpoints.KubeletEndpoint.Port,
		"port must be tunnelPort when EdgeTunnelIP feature gate is disabled")
}

func TestUpdateNodeKubeletEndpoint_EdgeTunnelIPEnabled(t *testing.T) {
	streamPort := 10003

	t.Run("EdgeTunnelIPEnabled_UsesStreamPort", func(t *testing.T) {
		if err := features.DefaultMutableFeatureGate.Set("EdgeTunnelIP=true"); err != nil {
			t.Fatalf("failed to enable EdgeTunnelIP feature gate: %v", err)
		}
		defer func() { _ = features.DefaultMutableFeatureGate.Set("EdgeTunnelIP=false") }()

		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		})
		ts := newTunnelServerWithClient(testTunnelPort, streamPort, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, int32(streamPort), node.Status.DaemonEndpoints.KubeletEndpoint.Port,
			"when EdgeTunnelIP is enabled, port must be streamPort (10003)")
	})

	t.Run("EdgeTunnelIPDisabled_UsesTunnelPort", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: testNodeName},
		})
		ts := newTunnelServerWithClient(testTunnelPort, streamPort, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*300)

		err := ts.updateNodeKubeletEndpoint(testNodeName)
		assert.NoError(t, err)

		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), testNodeName, metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, int32(testTunnelPort), node.Status.DaemonEndpoints.KubeletEndpoint.Port,
			"when EdgeTunnelIP is disabled, port must be tunnelPort")
	})
}

func kubernetesEndpoints(ips ...string) *corev1.Endpoints {
	addrs := make([]corev1.EndpointAddress, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, corev1.EndpointAddress{IP: ip})
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: "kubernetes", Namespace: corev1.NamespaceDefault},
		Subsets:    []corev1.EndpointSubset{{Addresses: addrs}},
	}
}

func cloudCoreNode(name string, ips ...string) *corev1.Node {
	addrs := make([]corev1.NodeAddress, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, corev1.NodeAddress{Type: corev1.NodeInternalIP, Address: ip})
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     corev1.NodeStatus{Addresses: addrs},
	}
}

func TestShouldUseEdgeTunnelIP(t *testing.T) {
	const cloudCoreNodeName = "cloudcore-node"

	t.Run("ExternalMode_AlwaysFalse", func(t *testing.T) {
		t.Setenv(nodetopology.NodeNameEnvVar, cloudCoreNodeName)
		fakeClient := fake.NewSimpleClientset(
			cloudCoreNode(cloudCoreNodeName, "10.0.0.9"),
			kubernetesEndpoints("172.16.5.9"), // clearly separated, would be true under InternalMode
		)
		ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.ExternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
		assert.False(t, ts.shouldUseEdgeTunnelIP(), "external iptablesManager mode never needs EdgeTunnelIP")
	})

	t.Run("InternalMode_UndeterminedFallsBackToTrue", func(t *testing.T) {
		t.Setenv(nodetopology.NodeNameEnvVar, "") // cloudcore not running as a scheduled pod -- can't verify placement
		fakeClient := fake.NewSimpleClientset()
		ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
		assert.True(t, ts.shouldUseEdgeTunnelIP(), "undetermined placement must conservatively assume separated nodes")
	})

	t.Run("InternalMode_Colocated_False", func(t *testing.T) {
		t.Setenv(nodetopology.NodeNameEnvVar, cloudCoreNodeName)
		fakeClient := fake.NewSimpleClientset(
			cloudCoreNode(cloudCoreNodeName, "10.0.0.9"),
			kubernetesEndpoints("10.0.0.9"),
		)
		ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
		assert.False(t, ts.shouldUseEdgeTunnelIP(), "apiserver colocated with cloudcore's node: local DNAT already handles routing")
	})

	t.Run("InternalMode_Separated_True", func(t *testing.T) {
		t.Setenv(nodetopology.NodeNameEnvVar, cloudCoreNodeName)
		fakeClient := fake.NewSimpleClientset(
			cloudCoreNode(cloudCoreNodeName, "10.0.0.9"),
			kubernetesEndpoints("172.16.5.9"),
		)
		ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
		assert.True(t, ts.shouldUseEdgeTunnelIP(), "apiserver not reachable via cloudcore's node: needs EdgeTunnelIP")
	})

	t.Run("DecisionIsCachedAfterDetermined", func(t *testing.T) {
		t.Setenv(nodetopology.NodeNameEnvVar, cloudCoreNodeName)
		fakeClient := fake.NewSimpleClientset(
			cloudCoreNode(cloudCoreNodeName, "10.0.0.9"),
			kubernetesEndpoints("10.0.0.9"),
		)
		ts := newTunnelServerWithClient(testTunnelPort, 0, v1alpha1.InternalMode, fakeClient.CoreV1(), time.Millisecond*10, time.Millisecond*100)
		assert.False(t, ts.shouldUseEdgeTunnelIP())

		// Mutate cluster state so a fresh lookup would flip the answer to
		// "separated" -- the cached decision must not change mid-process.
		err := fakeClient.CoreV1().Endpoints(corev1.NamespaceDefault).Delete(context.Background(), "kubernetes", metav1.DeleteOptions{})
		assert.NoError(t, err)
		_, err = fakeClient.CoreV1().Endpoints(corev1.NamespaceDefault).Create(context.Background(), kubernetesEndpoints("172.16.5.9"), metav1.CreateOptions{})
		assert.NoError(t, err)

		assert.False(t, ts.shouldUseEdgeTunnelIP(), "decision must stay cached for the TunnelServer's lifetime")
	})
}

// isAPIServerColocated itself is now in cloud/pkg/common/nodetopology, with
// its own test coverage there (nodetopology_test.go), since edgecontroller's
// patchNode() path uses the same shared implementation.
