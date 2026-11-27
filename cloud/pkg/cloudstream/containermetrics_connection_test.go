/*
Copyright 2024 The KubeEdge Authors.

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
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestString_Metrics(t *testing.T) {
	assert := assert.New(t)
	metricsConn := &ContainerMetricsConnection{
		MessageID: 100,
	}

	stdResult := "APIServer_MetricsConnection MessageID 100"
	assert.Equal(stdResult, metricsConn.String())
}

func TestWriteToAPIServer_Metrics(t *testing.T) {
	assert := assert.New(t)

	mockWriter := &MockWriter{}
	metricsConn := &ContainerMetricsConnection{
		writer: mockWriter,
	}

	data := []byte("test data")
	dataLength, err := metricsConn.WriteToAPIServer(data)
	assert.NoError(err)
	assert.Equal(len(data), dataLength)
	assert.Equal(data, mockWriter.writeBuffer.Bytes())
}

func TestSetMessageID_Metrics(t *testing.T) {
	assert := assert.New(t)
	metricsConn := &ContainerMetricsConnection{}

	metricsConn.SetMessageID(uint64(100))

	stdResult := uint64(100)
	assert.Equal(stdResult, metricsConn.MessageID)
}

func TestGetMessageID_Metrics(t *testing.T) {
	assert := assert.New(t)

	metricsConn := &ContainerMetricsConnection{
		MessageID: 200,
	}

	stdResult := uint64(200)
	assert.Equal(stdResult, metricsConn.GetMessageID())
}

func TestSetEdgePeerDone_Metrics(t *testing.T) {
	assert := assert.New(t)

	metricsConn := &ContainerMetricsConnection{
		MessageID:    1,
		edgePeerStop: make(chan struct{}),
		closeChan:    make(chan bool),
	}

	go func() {
		metricsConn.SetEdgePeerDone()
	}()

	select {
	case <-metricsConn.edgePeerStop:
		assert.True(true)
	case <-metricsConn.closeChan:
		assert.Fail("Expected edgePeerStop to receive but got closeChan")
	}
}

func TestEdgePeerDone_Metrics(t *testing.T) {
	assert := assert.New(t)

	edgePeerStop := make(chan struct{})
	metricsConn := &ContainerMetricsConnection{
		edgePeerStop: edgePeerStop,
	}

	assert.Equal(edgePeerStop, metricsConn.EdgePeerDone())
}

func TestWriteToTunnel_Metrics(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	metricsConn := &ContainerMetricsConnection{
		MessageID: 1,
		session:   session,
	}

	message := stream.NewMessage(metricsConn.MessageID, stream.MessageTypeData, []byte("test data"))

	err := metricsConn.WriteToTunnel(message)
	assert.NoError(err)
	assert.Equal(mockTunneler.lastMessage, message)
}

func TestSendConnection_Metrics(t *testing.T) {
	assert := assert.New(t)

	// mock HTTP request
	mockReq := &http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "localhost",
			Path:   "/api/v1/nodes",
		},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}
	restfulReq := &restful.Request{Request: mockReq}

	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}

	metricsConn := &ContainerMetricsConnection{
		MessageID: 1,
		r:         restfulReq,
		session:   session,
	}

	conn, err := metricsConn.SendConnection()
	assert.NoError(err)
	assert.NotNil(conn)

	metricsEdgedConn, ok := conn.(*stream.EdgedMetricsConnection)
	assert.True(ok)

	assert.Equal("http", metricsEdgedConn.URL.Scheme)
	assert.Equal(net.JoinHostPort(defaultServerHost, strconv.Itoa(constants.ServerPort)), metricsEdgedConn.URL.Host)

	assert.NotNil(mockTunneler.lastMessage)
	assert.Equal(stream.MessageTypeMetricConnect, mockTunneler.lastMessage.MessageType)
}

func TestMetricsCompletionFromSession(t *testing.T) {
	tests := []struct {
		name       string
		payload    string
		disconnect bool
		wantError  bool
	}{
		{name: "success", payload: "metrics-stream-success-v1"},
		{name: "legacy", wantError: true},
		{name: "invalid", payload: "invalid", wantError: true},
		{name: "future version", payload: "metrics-stream-success-v2", wantError: true},
		{name: "session failure", disconnect: true, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{tunnel: newMockTunnel(), apiConnlock: &sync.RWMutex{}}
			metrics := &ContainerMetricsConnection{
				MessageID: 1, ctx: context.Background(), session: session,
				r:            &restful.Request{Request: &http.Request{URL: &url.URL{Path: "/metrics"}}},
				edgePeerStop: make(chan struct{}), closeChan: make(chan bool),
			}
			session.apiServerConn = map[uint64]APIServerConnection{1: metrics}
			result := make(chan error, 1)
			go func() { result <- metrics.Serve() }()
			if tt.disconnect {
				session.Serve() // NextReader fails; the original Session.Close path runs.
			} else {
				assert.NoError(t, session.ProxyTunnelMessageToApiserver(stream.NewMessage(
					1, stream.MessageTypeRemoveConnect, []byte(tt.payload))))
			}
			select {
			case err := <-result:
				assert.Equal(t, tt.wantError, err != nil, "Serve returned %v", err)
			case <-time.After(time.Second):
				t.Fatal("metrics handler did not finish")
			}
		})
	}
}

func TestMetricsCompletionBeforeServe(t *testing.T) {
	for _, successFirst := range []bool{true, false} {
		metrics := &ContainerMetricsConnection{edgePeerStop: make(chan struct{})}
		done := make(chan struct{})
		go func() {
			metrics.setEdgePeerDone(successFirst)
			metrics.setEdgePeerDone(!successFirst)
			metrics.SetEdgePeerDone() // A subsequent session shutdown must not block.
			close(done)
		}()
		select {
		case <-done:
			<-metrics.EdgePeerDone()
			assert.Equal(t, successFirst, metrics.metricsComplete)
		case <-time.After(time.Second):
			t.Fatal("completion blocked before the metrics handler started")
		}
	}
}
