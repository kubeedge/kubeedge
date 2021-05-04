/*
Copyright 2019 The KubeEdge Authors.

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

package wsclient

import (
	"crypto/tls"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
	"github.com/kubeedge/viaduct/pkg/server"
)

func handleServer(container *mux.MessageContainer, writer mux.ResponseWriter) {
	klog.Infof("receive message: %s", container.Message.GetContent())
	writer.WriteResponse(&model.Message{}, container.Message.GetContent())
}

func connNotify(conn conn.Connection) {
	klog.Info("receive a connection")
}

// newTestServer() starts a fake server for testing
func newTestServer() error {
	if err := util.GenerateTestCertificate("/tmp/", "edge", "edge"); err != nil {
		return err
	}

	exOpts := api.WSServerOption{
		Path: "/",
	}

	cert, err := tls.LoadX509KeyPair("/tmp/edge.crt", "/tmp/edge.key")
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	httpServer := server.Server{
		Type:       "websocket",
		Addr:       "localhost:9890",
		TLSConfig:  tlsConfig,
		AutoRoute:  true,
		ConnNotify: connNotify,
		ExOpts:     exOpts,
	}

	mux.Entry(mux.NewPattern("*").Op("*"), handleServer)

	go func() {
		err = httpServer.ListenAndServeTLS("", "")
		if err != nil {
			klog.Errorf("listen and serve tls failed, error: %+v", err)
		}
	}()

	return nil
}

func newTestWebSocketClient(api string, certPath string, keyPath string) *WebSocketClient {
	return &WebSocketClient{
		config: &WebSocketConfig{
			URL:              "wss://localhost:9890/" + api,
			CertFilePath:     certPath,
			KeyFilePath:      keyPath,
			HandshakeTimeout: 500 * time.Second,
			WriteDeadline:    100 * time.Second,
			ReadDeadline:     100 * time.Second,
			NodeID:           "test-nodeid",
			ProjectID:        "test-projectid",
		},
	}
}

// TestNewWebSocketClient tests the NewWebSocketClient function that creates the WebSocketClient object
func TestNewWebSocketClient(t *testing.T) {
	tests := []struct {
		name string
		conf *WebSocketConfig
		want *WebSocketClient
	}{
		{"TestNewWebSocketClient: ",
			&WebSocketConfig{
				URL:              "wss://localhost:9890/normal",
				CertFilePath:     "/tmp/edge.crt",
				KeyFilePath:      "/tmp/edge.key",
				HandshakeTimeout: 500 * time.Second,
				WriteDeadline:    100 * time.Second,
				ReadDeadline:     100 * time.Second,
				NodeID:           "test-nodeid",
				ProjectID:        "test-projectid",
			},
			newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
		},
	}

	if err := newTestServer(); err != nil {
		t.Errorf("failed to start server, err: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWebSocketClient(tt.conf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWebSocketClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInit tests the procurement of the WebSocketClient
func TestInit(t *testing.T) {
	config.Config.TLSCAFile = "/tmp/edge.crt"

	tests := []struct {
		name          string
		fields        *WebSocketClient
		expectedError error
	}{
		{
			name:          "TestInit: Success in connection",
			fields:        newTestWebSocketClient("success", "/tmp/edge.crt", "/tmp/edge.key"),
			expectedError: nil,
		},
		{
			name:          "TestInit: If Certificate files not loaded properly",
			fields:        newTestWebSocketClient("fail", "/wrong_path.crt", "/wrong_path.key"),
			expectedError: fmt.Errorf("failed to load x509 key pair, error: open /wrong_path.crt: no such file or directory"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields
			err := wcc.Init()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("WebSocketClient.Init() error = %v, expectedError =  %v", err, tt.expectedError)
			}
		})
	}
}

// TestSend checks send function by sending message to server
func TestSend(t *testing.T) {
	var msg = model.Message{
		Header: model.MessageHeader{
			ID:        uuid.New().String(),
			ParentID:  "1",
			Timestamp: time.Now().UnixNano() / 1e6,
		},
		Content: "test",
	}
	tests := []struct {
		name          string
		fields        *WebSocketClient
		message       model.Message
		expectedError error
	}{
		{
			name:          "Test sending small message",
			fields:        newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
			message:       msg,
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields

			if err := wcc.Init(); err != nil {
				t.Errorf("failed to init, err: %v", err)
			}

			if err := wcc.Send(tt.message); !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("WebSocketClient.Send() error = %v, expectedError = %v", err, tt.expectedError)
			}
		})
	}
}

// TestReceive sends the message through send function then calls receive function to see same message is received or not
func TestReceive(t *testing.T) {
	var msg = model.Message{
		Header: model.MessageHeader{
			ID:        uuid.New().String(),
			ParentID:  "12",
			Timestamp: time.Now().UnixNano() / 1e6,
		},
		Content: "test",
	}
	tests := []struct {
		name          string
		fields        *WebSocketClient
		want          model.Message
		sent          model.Message
		expectedError error
	}{
		{name: "Test Receiving the send message: Success in receiving",
			fields:        newTestWebSocketClient("normal", "/tmp/edge.crt", "/tmp/edge.key"),
			want:          msg,
			sent:          msg,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wcc := tt.fields
			if err := wcc.Init(); err != nil {
				t.Errorf("failed to init, err: %v", err)
			}

			if err := wcc.Send(tt.sent); err != nil {
				t.Errorf("failed to send, err: %v", err)
			}

			got, err := wcc.Receive()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("WebSocketClient.Receive() error = %v, expectedError = %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(fmt.Sprintf("%s", got.GetContent()), fmt.Sprintf("%s", tt.want.GetContent())) {
				t.Errorf("WebSocketClient.Receive() message content: got = %s, want = %s", got.GetContent(), tt.want.GetContent())
			}
		})
	}
}
