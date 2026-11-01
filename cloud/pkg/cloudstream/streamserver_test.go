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

package cloudstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestGetMetricsDeletesConnectionOnRequestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil).WithContext(ctx)
	req.Host = "edge-node"
	resp := httptest.NewRecorder()

	tunnel := &TunnelServer{
		sessions: make(map[string]*Session),
	}
	mockTunneler := &MockTunneler{}
	session := &Session{
		sessionID:     "edge-node",
		tunnel:        mockTunneler,
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}
	tunnel.addSession("edge-node", session)

	streamServer := &StreamServer{
		tunnel: tunnel,
	}

	streamServer.getMetrics(restful.NewRequest(req), restful.NewResponse(resp))

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, mockTunneler.lastMessage)
	assert.Equal(t, stream.MessageTypeRemoveConnect, mockTunneler.lastMessage.MessageType)
	assert.Empty(t, session.apiServerConn)
}

func TestGetMetricsDeletesConnectionOnRequestCancelWithIPv6Host(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil).WithContext(ctx)
	req.Host = "[2001:db8::1]:10003"
	resp := httptest.NewRecorder()

	tunnel := &TunnelServer{
		sessions: make(map[string]*Session),
	}
	mockTunneler := &MockTunneler{}
	session := &Session{
		sessionID:     "2001:db8::1",
		tunnel:        mockTunneler,
		apiServerConn: make(map[uint64]APIServerConnection),
		apiConnlock:   &sync.RWMutex{},
	}
	tunnel.addSession("2001:db8::1", session)

	streamServer := &StreamServer{
		tunnel: tunnel,
	}

	streamServer.getMetrics(restful.NewRequest(req), restful.NewResponse(resp))

	assert.Equal(t, http.StatusOK, resp.Code)
	require.NotNil(t, mockTunneler.lastMessage)
	assert.Equal(t, stream.MessageTypeRemoveConnect, mockTunneler.lastMessage.MessageType)
	assert.Empty(t, session.apiServerConn)
}
