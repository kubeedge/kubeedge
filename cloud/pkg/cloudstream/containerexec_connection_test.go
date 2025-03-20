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
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/pkg/stream"
)

func TestString_Exec(t *testing.T) {
	assert := assert.New(t)
	execConn := &ContainerExecConnection{
		MessageID: 100,
	}

	stdResult := "APIServer_ExecConnection MessageID 100"
	assert.Equal(stdResult, execConn.String())
}

func TestWriteToAPIServer_Exec(t *testing.T) {
	assert := assert.New(t)
	mockConn := &MockConn{}
	execConn := &ContainerExecConnection{
		Conn: mockConn,
	}

	data := []byte("test data")
	dataLength, err := execConn.WriteToAPIServer(data)
	assert.NoError(err)
	assert.Equal(9, dataLength)
	assert.Equal(data, mockConn.writeBuffer.Bytes())
}

func TestSetMessageID_Exec(t *testing.T) {
	assert := assert.New(t)
	execConn := &ContainerExecConnection{}

	execConn.SetMessageID(uint64(100))

	stdResult := uint64(100)
	assert.Equal(stdResult, execConn.MessageID)
}

func TestGetMessageID_Exec(t *testing.T) {
	assert := assert.New(t)

	execConn := &ContainerExecConnection{
		MessageID: 200,
	}

	stdResult := uint64(200)
	assert.Equal(stdResult, execConn.GetMessageID())
}

func TestSetEdgePeerDone_Exec(t *testing.T) {
	assert := assert.New(t)

	execConn := &ContainerExecConnection{
		MessageID:    1,
		edgePeerStop: make(chan struct{}),
		closeChan:    make(chan bool),
	}

	received := make(chan bool, 1)
	go func() {
		select {
		case <-execConn.edgePeerStop:
			received <- true
		case <-time.After(100 * time.Millisecond):
			received <- false
		}
	}()

	execConn.SetEdgePeerDone()

	assert.True(<-received, "edgePeerStop channel should receive a signal")
}

func TestEdgePeerDone_Exec(t *testing.T) {
	assert := assert.New(t)

	edgePeerStop := make(chan struct{})
	execConn := &ContainerExecConnection{
		edgePeerStop: edgePeerStop,
	}

	assert.Equal(edgePeerStop, execConn.EdgePeerDone())
}

func TestWriteToTunnel_Exec(t *testing.T) {
	assert := assert.New(t)

	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	execConn := &ContainerExecConnection{
		MessageID: 1,
		session:   session,
	}

	message := stream.NewMessage(execConn.MessageID, stream.MessageTypeData, []byte("test data"))

	err := execConn.WriteToTunnel(message)
	assert.NoError(err)
	assert.Equal(mockTunneler.lastMessage, message)
}

func TestSendConnection_Exec(t *testing.T) {
	assert := assert.New(t)

	mockConn := &MockConn{}
	mockTunneler := &MockTunneler{}
	session := &Session{
		tunnel: mockTunneler,
	}
	r := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{},
			Header: http.Header{},
		},
	}

	execConn := &ContainerExecConnection{
		MessageID: 1,
		r:         r,
		Conn:      mockConn,
		session:   session,
	}

	connector, err := execConn.SendConnection()
	assert.NoError(err)

	edgedConnector, ok := connector.(*stream.EdgedExecConnection)
	assert.True(ok, "Expected connector should be of type *stream.EdgedExecConnection")
	assert.Equal(execConn.MessageID, edgedConnector.MessID)
	assert.Equal(r.Request.Method, edgedConnector.Method)
	expectedURL := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:10350",
	}
	assert.Equal(expectedURL, edgedConnector.URL)
	assert.Equal(r.Request.Header, edgedConnector.Header)

	assert.Equal(mockTunneler.lastMessage.MessageType, stream.MessageTypeExecConnect)
	expectedData, _ := edgedConnector.CreateConnectMessage()
	assert.Equal(mockTunneler.lastMessage.Data, expectedData.Data)
}

func TestServe_Exec(t *testing.T) {
	testCases := []struct {
		name          string
		setup         func(t *testing.T) (*ContainerExecConnection, context.CancelFunc)
		expectedError bool
		errorContains string
	}{
		{
			name: "Context done",
			setup: func(t *testing.T) (*ContainerExecConnection, context.CancelFunc) {
				conn := &MockConn{}
				conn.readBuffer.Write([]byte("test data"))

				tun := &MockTunneler{}

				ctx, cancel := context.WithCancel(context.Background())

				session := &Session{
					tunnel: tun,
				}

				r := &restful.Request{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{},
						Header: http.Header{},
					},
				}

				execConn := &ContainerExecConnection{
					MessageID:    1,
					ctx:          ctx,
					r:            r,
					Conn:         conn,
					session:      session,
					edgePeerStop: make(chan struct{}),
					closeChan:    make(chan bool),
				}

				cancel()

				return execConn, cancel
			},
			expectedError: false,
		},
		{
			name: "Edge peer done",
			setup: func(t *testing.T) (*ContainerExecConnection, context.CancelFunc) {
				conn := &MockConn{}
				conn.readBuffer.Write([]byte("test data"))

				tun := &MockTunneler{}

				ctx, cancel := context.WithCancel(context.Background())

				session := &Session{
					tunnel: tun,
				}

				r := &restful.Request{
					Request: &http.Request{
						Method: "GET",
						URL:    &url.URL{},
						Header: http.Header{},
					},
				}

				execConn := &ContainerExecConnection{
					MessageID:    1,
					ctx:          ctx,
					r:            r,
					Conn:         conn,
					session:      session,
					edgePeerStop: make(chan struct{}),
					closeChan:    make(chan bool),
				}

				go func() {
					time.Sleep(50 * time.Millisecond)
					execConn.EdgePeerDone() <- struct{}{}
				}()

				return execConn, cancel
			},
			expectedError: true,
			errorContains: "find edge peer done",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			execConn, cancel := tc.setup(t)
			defer cancel()

			done := make(chan error)

			go func() {
				err := execConn.Serve()
				done <- err
			}()

			var err error
			select {
			case err = <-done:
			case <-time.After(500 * time.Millisecond):
				cancel()
				err = <-done
			}

			if tc.expectedError {
				assert.Error(err)
				if tc.errorContains != "" {
					assert.Contains(err.Error(), tc.errorContains)
				}
			} else {
				if err != nil {
					t.Logf("Expected no error but got: %v. This is acceptable in this test case.", err)
				}
			}
		})
	}
}

func TestSetEdgePeerDone_Closed(t *testing.T) {
	assert := assert.New(t)

	execConn := &ContainerExecConnection{
		MessageID:    1,
		edgePeerStop: make(chan struct{}),
		closeChan:    make(chan bool),
	}

	close(execConn.closeChan)

	received := make(chan bool, 1)
	go func() {
		select {
		case <-execConn.edgePeerStop:
			received <- true
		case <-time.After(100 * time.Millisecond):
			received <- false
		}
	}()

	execConn.SetEdgePeerDone()

	assert.False(<-received, "edgePeerStop channel should not receive a signal when closeChan is closed")
}

func TestSendConnection_WriteToTunnelError(t *testing.T) {
	assert := assert.New(t)

	mockConn := &MockConn{}
	mockTunneler := &MockTunneler{
		err: errors.New("write error"),
	}
	session := &Session{
		tunnel: mockTunneler,
	}
	r := &restful.Request{
		Request: &http.Request{
			Method: "GET",
			URL:    &url.URL{},
			Header: http.Header{},
		},
	}

	execConn := &ContainerExecConnection{
		MessageID: 1,
		r:         r,
		Conn:      mockConn,
		session:   session,
	}

	connector, err := execConn.SendConnection()
	assert.Error(err)
	assert.Contains(err.Error(), "write error")
	assert.Nil(connector)
}
