/*
Copyright 2026 The KubeEdge Authors.

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

package handler

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	cloudcoreConfig "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	reliableclient "github.com/kubeedge/api/client/clientset/versioned"
	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/dispatcher"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/session"
	commonclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/mux"
)

type fakeAuthorizer struct {
	authErr  error
	admitErr error
}

func (a *fakeAuthorizer) AdmitMessage(_ beehivemodel.Message, _ model.HubInfo) error {
	return a.admitErr
}

func (a *fakeAuthorizer) AuthenticateConnection(_ conn.Connection) error {
	return a.authErr
}

type fakeDispatcher struct {
	mu                  sync.Mutex
	publishError        error
	publishCount        int
	publishOps          []string
	upstreamDispatches  int
	lastUpstreamNodeID  string
	lastUpstreamProject string
	addedPools          int
	deletedPools        int
	addedCh             chan struct{}
	deletedCh           chan struct{}
}

func (d *fakeDispatcher) DispatchDownstream() {}

func (d *fakeDispatcher) DispatchUpstream(_ *beehivemodel.Message, info *model.HubInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.upstreamDispatches++
	d.lastUpstreamNodeID = info.NodeID
	d.lastUpstreamProject = info.ProjectID
}

func (d *fakeDispatcher) AddNodeMessagePool(_ string, _ *common.NodeMessagePool) {
	d.mu.Lock()
	d.addedPools++
	d.mu.Unlock()
	if d.addedCh != nil {
		select {
		case d.addedCh <- struct{}{}:
		default:
		}
	}
}

func (d *fakeDispatcher) DeleteNodeMessagePool(_ string, _ *common.NodeMessagePool) {
	d.mu.Lock()
	d.deletedPools++
	d.mu.Unlock()
	if d.deletedCh != nil {
		select {
		case d.deletedCh <- struct{}{}:
		default:
		}
	}
}

func (d *fakeDispatcher) GetNodeMessagePool(_ string) *common.NodeMessagePool {
	return nil
}

func (d *fakeDispatcher) Publish(msg *beehivemodel.Message) error {
	d.mu.Lock()
	d.publishCount++
	d.publishOps = append(d.publishOps, msg.GetOperation())
	err := d.publishError
	d.mu.Unlock()
	return err
}

type fakeConn struct {
	state           conn.ConnectionState
	closeCount      int
	writeAsyncCount int
	writeAsyncMsgs  []*beehivemodel.Message
}

func (c *fakeConn) ServeConn() {}

func (c *fakeConn) SetReadDeadline(_ time.Time) error { return nil }

func (c *fakeConn) SetWriteDeadline(_ time.Time) error { return nil }

func (c *fakeConn) Read(_ []byte) (int, error) { return 0, nil }

func (c *fakeConn) Write(raw []byte) (int, error) { return len(raw), nil }

func (c *fakeConn) WriteMessageAsync(msg *beehivemodel.Message) error {
	c.writeAsyncCount++
	c.writeAsyncMsgs = append(c.writeAsyncMsgs, msg)
	return nil
}

func (c *fakeConn) WriteMessageSync(_ *beehivemodel.Message) (*beehivemodel.Message, error) {
	return nil, nil
}

func (c *fakeConn) ReadMessage(_ *beehivemodel.Message) error { return nil }

func (c *fakeConn) RemoteAddr() net.Addr { return nil }

func (c *fakeConn) LocalAddr() net.Addr { return nil }

func (c *fakeConn) ConnectionState() conn.ConnectionState { return c.state }

func (c *fakeConn) Close() error {
	c.closeCount++
	return nil
}

func newFakeConnection(nodeID, projectID string) *fakeConn {
	h := http.Header{}
	h.Set("node_id", nodeID)
	h.Set("project_id", projectID)
	return &fakeConn{state: conn.ConnectionState{Headers: h}}
}

func createTempKubeConfig(t *testing.T, server string) string {
	t.Helper()
	tempDir := t.TempDir()
	kubeConfigPath := filepath.Join(tempDir, "kubeconfig")
	kubeConfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: true
  name: local
contexts:
- context:
    cluster: local
    user: test-user
  name: test
current-context: test
users:
- name: test-user
  user:
    token: test-token
`, server)

	err := os.WriteFile(kubeConfigPath, []byte(kubeConfigContent), 0644)
	assert.NoError(t, err)
	return kubeConfigPath
}

var initClientOnce sync.Once

func ensureCommonClientInitialized(t *testing.T) {
	t.Helper()
	initClientOnce.Do(func() {
		kubeConfig := createTempKubeConfig(t, "http://127.0.0.1:6443")
		cfg := &cloudcoreConfig.KubeAPIConfig{KubeConfig: kubeConfig, QPS: 100, Burst: 200}
		assert.NotPanics(t, func() {
			commonclient.InitKubeEdgeClient(cfg, false)
		})
	})
}

func newMessageHandlerForTest(manager *session.Manager, d dispatcher.MessageDispatcher, a *fakeAuthorizer) *messageHandler {
	return &messageHandler{
		KeepaliveInterval: 60,
		SessionManager:    manager,
		MessageDispatcher: d,
		reliableClient:    reliableclient.Interface(nil),
		authorizer:        a,
	}
}

func TestNewMessageHandler(t *testing.T) {
	h := NewMessageHandler(10, session.NewSessionManager(10), nil, &fakeDispatcher{}, &fakeAuthorizer{})
	assert.NotNil(t, h)
}

func TestHandleMessage(t *testing.T) {
	d := &fakeDispatcher{}
	a := &fakeAuthorizer{}
	mh := newMessageHandlerForTest(session.NewSessionManager(10), d, a)

	container := &mux.MessageContainer{Header: http.Header{}}
	container.Header.Set("node_id", "node1")
	container.Header.Set("project_id", "project1")

	mh.HandleMessage(container, nil)
	assert.Equal(t, 0, d.upstreamDispatches)

	msg := beehivemodel.NewMessage("msg-id").BuildRouter("source", "group", "resource", "operation")
	container.Message = msg
	a.admitErr = errors.New("forbidden")
	mh.HandleMessage(container, nil)
	assert.Equal(t, 0, d.upstreamDispatches)

	a.admitErr = nil
	mh.HandleMessage(container, nil)
	assert.Equal(t, 1, d.upstreamDispatches)
	assert.Equal(t, "node1", d.lastUpstreamNodeID)
	assert.Equal(t, "project1", d.lastUpstreamProject)
}

func TestHandleConnectionEarlyReturnBranches(t *testing.T) {
	t.Run("auth reject", func(t *testing.T) {
		d := &fakeDispatcher{}
		a := &fakeAuthorizer{authErr: errors.New("unauthorized")}
		mh := newMessageHandlerForTest(session.NewSessionManager(10), d, a)

		mh.HandleConnection(newFakeConnection("node-auth", "project-auth"))
		assert.Equal(t, 0, d.publishCount)
	})

	t.Run("reach limit", func(t *testing.T) {
		d := &fakeDispatcher{}
		a := &fakeAuthorizer{}
		mh := newMessageHandlerForTest(session.NewSessionManager(0), d, a)

		mh.HandleConnection(newFakeConnection("node-limit", "project-limit"))
		assert.Equal(t, 0, d.publishCount)
	})

	t.Run("connect event publish error", func(t *testing.T) {
		d := &fakeDispatcher{publishError: errors.New("queue unavailable")}
		a := &fakeAuthorizer{}
		mh := newMessageHandlerForTest(session.NewSessionManager(10), d, a)
		connection := newFakeConnection("node-conn-err", "project-conn-err")

		mh.HandleConnection(connection)

		assert.Equal(t, 1, d.publishCount)
		assert.Equal(t, 1, connection.writeAsyncCount)
	})
}

func TestHandleConnectionSuccessLifecycle(t *testing.T) {
	ensureCommonClientInitialized(t)

	d := &fakeDispatcher{
		addedCh:   make(chan struct{}, 1),
		deletedCh: make(chan struct{}, 1),
	}
	a := &fakeAuthorizer{}
	manager := session.NewSessionManager(10)
	mh := newMessageHandlerForTest(manager, d, a)

	nodeID := "node-success"
	projectID := "project-success"
	connection := newFakeConnection(nodeID, projectID)

	mh.HandleConnection(connection)

	assert.Eventually(t, func() bool {
		_, ok := manager.GetSession(nodeID)
		return ok
	}, 2*time.Second, 20*time.Millisecond)

	// Let UpdateAnnotation retry loop finish so the non-nil error branch is exercised.
	time.Sleep(2500 * time.Millisecond)

	mh.OnReadTransportErr(nodeID, projectID)

	assert.Eventually(t, func() bool {
		_, ok := manager.GetSession(nodeID)
		return !ok
	}, 2*time.Second, 20*time.Millisecond)

	assert.GreaterOrEqual(t, d.addedPools, 1)
	assert.GreaterOrEqual(t, d.deletedPools, 1)
	assert.GreaterOrEqual(t, d.publishCount, 2)
}

func TestOnEdgeNodeConnectAndDisconnect(t *testing.T) {
	info := &model.HubInfo{NodeID: "node-ops", ProjectID: "project-ops"}
	connection := newFakeConnection(info.NodeID, info.ProjectID)
	manager := session.NewSessionManager(10)
	a := &fakeAuthorizer{}

	t.Run("connect success and disconnect publish", func(t *testing.T) {
		d := &fakeDispatcher{}
		mh := newMessageHandlerForTest(manager, d, a)

		err := mh.OnEdgeNodeConnect(info, connection)
		assert.NoError(t, err)
		assert.Equal(t, 1, d.publishCount)

		mh.OnEdgeNodeDisconnect(info, connection)
		assert.Equal(t, 2, d.publishCount)
	})

	t.Run("connect publish error triggers notify", func(t *testing.T) {
		d := &fakeDispatcher{publishError: errors.New("publish failed")}
		mh := newMessageHandlerForTest(manager, d, a)

		err := mh.OnEdgeNodeConnect(info, connection)
		assert.Error(t, err)
		assert.GreaterOrEqual(t, connection.writeAsyncCount, 1)
	})

	t.Run("disconnect publish error", func(t *testing.T) {
		d := &fakeDispatcher{publishError: errors.New("disconnect publish failed")}
		mh := newMessageHandlerForTest(manager, d, a)

		mh.OnEdgeNodeDisconnect(info, connection)
		assert.Equal(t, 1, d.publishCount)
	})
}

func TestOnReadTransportErr(t *testing.T) {
	manager := session.NewSessionManager(10)
	d := &fakeDispatcher{}
	a := &fakeAuthorizer{}
	mh := newMessageHandlerForTest(manager, d, a)

	mh.OnReadTransportErr("missing-node", "project-x")

	nodeID := "node-existing"
	projectID := "project-existing"
	connection := newFakeConnection(nodeID, projectID)
	nodeSession := session.NewNodeSession(nodeID, projectID, connection, time.Second, common.InitNodeMessagePool(nodeID), nil)
	manager.AddSession(nodeSession)

	mh.OnReadTransportErr(nodeID, projectID)
	assert.Equal(t, 1, connection.closeCount)
}
