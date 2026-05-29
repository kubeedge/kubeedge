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
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
)

const (
	TestPort       = 10350
	AltTestPort    = 10351
	TestNamespace  = "test-namespace"
	ErrorNamespace = "error-namespace"
	TestPod        = "test-pod"
	TestContainer  = "container1"
	TestNode       = "test-node"
	LocalTestAddr  = "127.0.0.1:12345"
	RemoteTestAddr = "127.0.0.1:54321"
	NetworkTypeTCP = "tcp"
)

var (
	ErrPodNodeName = errors.New("error getting pod node name")
	ErrPodNotFound = errors.New("pod not found")
	ErrHijack      = errors.New("hijack error")
	ErrInvalidPath = errors.New("can not get pod name from url path")
)

type MockAddr struct {
	NetworkVal string
	StringVal  string
}

func (m MockAddr) Network() string { return m.NetworkVal }
func (m MockAddr) String() string  { return m.StringVal }

type MockHijackerConn struct {
	readFunc             func(b []byte) (n int, err error)
	writeFunc            func(b []byte) (n int, err error)
	closeFunc            func() error
	localAddrFunc        func() net.Addr
	remoteAddrFunc       func() net.Addr
	setDeadlineFunc      func(t time.Time) error
	setReadDeadlineFunc  func(t time.Time) error
	setWriteDeadlineFunc func(t time.Time) error
}

func (m *MockHijackerConn) Read(b []byte) (n int, err error) {
	if m.readFunc != nil {
		return m.readFunc(b)
	}
	return 0, nil
}

func (m *MockHijackerConn) Write(b []byte) (n int, err error) {
	if m.writeFunc != nil {
		return m.writeFunc(b)
	}
	return len(b), nil
}

func (m *MockHijackerConn) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *MockHijackerConn) LocalAddr() net.Addr {
	if m.localAddrFunc != nil {
		return m.localAddrFunc()
	}
	return MockAddr{NetworkVal: NetworkTypeTCP, StringVal: LocalTestAddr}
}

func (m *MockHijackerConn) RemoteAddr() net.Addr {
	if m.remoteAddrFunc != nil {
		return m.remoteAddrFunc()
	}
	return MockAddr{NetworkVal: NetworkTypeTCP, StringVal: RemoteTestAddr}
}

func (m *MockHijackerConn) SetDeadline(t time.Time) error {
	if m.setDeadlineFunc != nil {
		return m.setDeadlineFunc(t)
	}
	return nil
}

func (m *MockHijackerConn) SetReadDeadline(t time.Time) error {
	if m.setReadDeadlineFunc != nil {
		return m.setReadDeadlineFunc(t)
	}
	return nil
}

func (m *MockHijackerConn) SetWriteDeadline(t time.Time) error {
	if m.setWriteDeadlineFunc != nil {
		return m.setWriteDeadlineFunc(t)
	}
	return nil
}

type MockHijacker struct {
	httptest.ResponseRecorder
	conn      net.Conn
	hijackErr error
}

func (m *MockHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if m.hijackErr != nil {
		return nil, nil, m.hijackErr
	}
	return m.conn, nil, nil
}

type MockFlushWriter struct {
	httptest.ResponseRecorder
	flushed bool
}

func (m *MockFlushWriter) Flush() {
	m.flushed = true
}

func mockGetPodNodeName(namespaceName, podName string) (string, error) {
	if namespaceName == TestNamespace && podName == TestPod {
		return TestNode, nil
	}
	if namespaceName == ErrorNamespace {
		return "", ErrPodNodeName
	}
	return "", ErrPodNotFound
}

func newTestSession(id string) *Session {
	return &Session{
		sessionID:     id,
		tunnel:        &mockTunnel{},
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}
}

func TestGetSessionKeyValidPath(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	key, err := s.getSessionKey("/containerLogs/" + TestNamespace + "/" + TestPod + "/" + TestContainer)

	assert.NoError(t, err)
	assert.Equal(t, TestNode, key)
}

func TestGetSessionKeyNodeNameError(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	_, err := s.getSessionKey("/containerLogs/" + ErrorNamespace + "/" + TestPod + "/" + TestContainer)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrPodNodeName.Error())
}

func TestGetSessionKeyInvalidPath(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	_, err := s.getSessionKey("/invalid/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrInvalidPath.Error())
}

func TestGetContainerLogsWithInvalidSessionKey(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/invalid", nil)
	resp := httptest.NewRecorder()

	s.getContainerLogs(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetContainerLogsWithNonFlushableWriter(_ *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	customWriter := struct {
		http.ResponseWriter
	}{
		ResponseWriter: httptest.NewRecorder(),
	}

	customResponse := restful.NewResponse(customWriter)

	req := httptest.NewRequest("GET", "/containerLogs/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)

	s.getContainerLogs(restful.NewRequest(req), customResponse)
}

func TestGetMetricsSessionNotFound(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Host = "nonexistent-host:8080"
	resp := httptest.NewRecorder()

	s.getMetrics(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetMetricsWithXForwardedUri(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Host = "api-server:8080"
	req.Header.Set("X-Forwarded-Uri", "/api/v1/nodes/nonexistent-node/proxy/metrics")
	resp := httptest.NewRecorder()

	s.getMetrics(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetMetricsWithXForwardedUriNotNodePath(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Host = "api-server:8080"
	req.Header.Set("X-Forwarded-Uri", "/api/v1/something-else")
	resp := httptest.NewRecorder()

	s.getMetrics(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetExecWithInvalidSessionKey(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/invalid", nil)
	resp := httptest.NewRecorder()

	s.getExec(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetExecSessionNotFound(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/exec/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	resp := httptest.NewRecorder()

	s.getExec(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetExecNonUpgradeRequest(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/exec/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	resp := httptest.NewRecorder()

	s.getExec(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetExecNonHijackableWriter(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/exec/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp := httptest.NewRecorder()

	s.getExec(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetExecHijackError(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/exec/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	mockHijacker := &MockHijacker{
		ResponseRecorder: *httptest.NewRecorder(),
		hijackErr:        ErrHijack,
	}

	s.getExec(restful.NewRequest(req), restful.NewResponse(mockHijacker))

	assert.Equal(t, http.StatusInternalServerError, mockHijacker.Code)
}

func TestGetExecAddAPIServerConnectionError(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	session.tunnelClosed = true
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/exec/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	mockConn := &MockHijackerConn{
		closeFunc: func() error { return nil },
	}

	mockHijacker := &MockHijacker{
		ResponseRecorder: *httptest.NewRecorder(),
		conn:             mockConn,
	}

	s.getExec(restful.NewRequest(req), restful.NewResponse(mockHijacker))

	assert.Equal(t, http.StatusInternalServerError, mockHijacker.Code)
}

func TestGetAttachWithInvalidSessionKey(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/invalid", nil)
	resp := httptest.NewRecorder()

	s.getAttach(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetAttachSessionNotFound(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/attach/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	resp := httptest.NewRecorder()

	s.getAttach(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetAttachNonUpgradeRequest(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/attach/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	resp := httptest.NewRecorder()

	s.getAttach(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetAttachNonHijackableWriter(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/attach/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp := httptest.NewRecorder()

	s.getAttach(restful.NewRequest(req), restful.NewResponse(resp))
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestGetAttachHijackError(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/attach/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	mockHijacker := &MockHijacker{
		ResponseRecorder: *httptest.NewRecorder(),
		hijackErr:        ErrHijack,
	}

	s.getAttach(restful.NewRequest(req), restful.NewResponse(mockHijacker))

	assert.Equal(t, http.StatusInternalServerError, mockHijacker.Code)
}

func TestGetAttachAddAPIServerConnectionError(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	session := newTestSession(TestNode)
	session.tunnelClosed = true
	tunnelServer.sessions[TestNode] = session

	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	req := httptest.NewRequest("GET", "/attach/"+TestNamespace+"/"+TestPod+"/"+TestContainer, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	mockConn := &MockHijackerConn{
		closeFunc: func() error { return nil },
	}

	mockHijacker := &MockHijacker{
		ResponseRecorder: *httptest.NewRecorder(),
		conn:             mockConn,
	}

	s.getAttach(restful.NewRequest(req), restful.NewResponse(mockHijacker))

	assert.Equal(t, http.StatusInternalServerError, mockHijacker.Code)
}

func TestSessionKeyExtractionFromDifferentURLs(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	oldFunc := getPodNodeNameFunc
	getPodNodeNameFunc = mockGetPodNodeName
	defer func() { getPodNodeNameFunc = oldFunc }()

	testCases := []struct {
		name          string
		path          string
		expectedKey   string
		expectedError bool
	}{
		{
			name:          "Standard path",
			path:          "/containerLogs/" + TestNamespace + "/" + TestPod + "/" + TestContainer,
			expectedKey:   TestNode,
			expectedError: false,
		},
		{
			name:          "Path with UID",
			path:          "/exec/" + TestNamespace + "/" + TestPod + "/uid123/" + TestContainer,
			expectedKey:   TestNode,
			expectedError: false,
		},
		{
			name:          "Invalid path - too short",
			path:          "/containerLogs/" + TestNamespace,
			expectedKey:   "",
			expectedError: true,
		},
		{
			name:          "Pod not found",
			path:          "/containerLogs/" + TestNamespace + "/unknown-pod/" + TestContainer,
			expectedKey:   "",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := s.getSessionKey(tc.path)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedKey, key)
			}
		})
	}
}

func TestInstallDebugHandler(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)
	s.installDebugHandler()

	expectedPaths := []string{
		"/containerLogs",
		"/exec",
		"/attach",
		"/stats",
		"/metrics",
	}

	registeredPaths := make(map[string]bool)
	for _, ws := range s.container.RegisteredWebServices() {
		registeredPaths[ws.RootPath()] = true
	}

	for _, path := range expectedPaths {
		assert.True(t, registeredPaths[path], "Path %s should be registered", path)
	}

	for _, ws := range s.container.RegisteredWebServices() {
		assert.NotEmpty(t, ws.Routes(), "WebService %s should have routes", ws.RootPath())
	}

	routeCounts := map[string]int{
		"/containerLogs": 0,
		"/exec":          0,
		"/attach":        0,
		"/stats":         0,
		"/metrics":       0,
	}

	for _, ws := range s.container.RegisteredWebServices() {
		routeCounts[ws.RootPath()] += len(ws.Routes())
	}

	assert.Equal(t, 1, routeCounts["/containerLogs"], "containerLogs should have 1 route")
	assert.Equal(t, 4, routeCounts["/exec"], "exec should have 4 routes")
	assert.Equal(t, 4, routeCounts["/attach"], "attach should have 4 routes")
	assert.Equal(t, 5, routeCounts["/stats"], "stats should have 5 routes")
	assert.Equal(t, 4, routeCounts["/metrics"], "metrics should have 4 routes")
}

func TestNewStreamServer(t *testing.T) {
	tunnelServer := newTunnelServer(TestPort)
	s := newStreamServer(tunnelServer)

	assert.NotNil(t, s)
	assert.NotNil(t, s.container)
	assert.Equal(t, tunnelServer, s.tunnel)
	assert.Equal(t, uint64(0), s.nextMessageID)
}

func TestServeErrorHandling(t *testing.T) {
	t.Skip("Skipping test that requires access to mockAPIServerConnection.serveErr")
}

func TestStreamServerWithCustomTunnel(t *testing.T) {
	customTunnel := newTunnelServer(AltTestPort)
	s := newStreamServer(customTunnel)

	assert.Equal(t, customTunnel, s.tunnel)
}
