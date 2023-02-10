/*
Copyright 2022 The KubeEdge Authors.

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

package quicclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
	"github.com/kubeedge/viaduct/pkg/api"
	"github.com/kubeedge/viaduct/pkg/conn"
	"github.com/kubeedge/viaduct/pkg/mux"
	"github.com/kubeedge/viaduct/pkg/server"
)

func init() {
	err := util.PrepareTestCerts()
	if err != nil {
		fmt.Printf("Failed to create certificate: %v\n", err)
		os.Exit(1)
	}
}

func newTestQuicClient(api string, certPath string, keyPath string, cacertPath string) *QuicClient {
	return &QuicClient{
		config: &QuicConfig{
			Addr:             net.JoinHostPort("127.0.0.1", "10001"),
			CaFilePath:       cacertPath,
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

func connNotify(conn conn.Connection) {
	klog.Info("receive a connection")
}

func handleServer(container *mux.MessageContainer, writer mux.ResponseWriter) {
	klog.Infof("receive message: %s", container.Message.GetContent())
	writer.WriteResponse(&model.Message{}, container.Message.GetContent())
}

var once sync.Once

func newTestServer(t *testing.T) {
	// new a QUIC server only once
	once.Do(func() {
		exOpts := api.QuicServerOption{
			MaxIncomingStreams: 100,
		}

		cert, err := tls.LoadX509KeyPair("/tmp/edge.crt", "/tmp/edge.key")
		if err != nil {
			t.Fatalf("failed to load certificate: %v", err)
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		httpServer := server.Server{
			Type:       "quic",
			Addr:       net.JoinHostPort("127.0.0.1", "10001"),
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
				os.Exit(1)
			}
		}()
	})
}
func TestNewQuicClient(t *testing.T) {
	tests := []struct {
		name string
		conf *QuicConfig
		want *QuicClient
	}{
		{
			"TestNewQuicClient",
			&QuicConfig{
				Addr:             "",
				CaFilePath:       "",
				CertFilePath:     "",
				KeyFilePath:      "",
				HandshakeTimeout: time.Second * 2,
				ReadDeadline:     time.Second * 2,
				WriteDeadline:    time.Second * 2,
				NodeID:           "nodeid",
				ProjectID:        "project_id",
			},
			&QuicClient{
				config: &QuicConfig{
					Addr:             "",
					CaFilePath:       "",
					CertFilePath:     "",
					KeyFilePath:      "",
					HandshakeTimeout: time.Second * 2,
					ReadDeadline:     time.Second * 2,
					WriteDeadline:    time.Second * 2,
					NodeID:           "nodeid",
					ProjectID:        "project_id",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewQuicClient(tt.conf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewQuicClient() got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestInit(t *testing.T) {
	newTestServer(t)

	tests := []struct {
		name          string
		client        *QuicClient
		expectedError error
	}{
		{
			name:          "QuicClient with valid config",
			client:        newTestQuicClient("success", "/tmp/edge.crt", "/tmp/edge.key", "/tmp/edge.crt"),
			expectedError: nil,
		},
		{
			name:          "QuicClient with invalid cert and key",
			client:        newTestQuicClient("fail", "/tmp/invalid/edge.crt", "/tmp/invalid/edge.key", "/tmp/invalid/edge.crt"),
			expectedError: fmt.Errorf("failed to load x509 key pair, error: open /tmp/invalid/edge.crt: no such file or directory"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.Init()
			if !reflect.DeepEqual(tt.expectedError, err) {
				t.Errorf("Init() failed. got = %v, want = %v", err, tt.expectedError)
			}
		})
	}
}

func TestSend(t *testing.T) {
	newTestServer(t)

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
		fields        *QuicClient
		message       model.Message
		expectedError error
	}{
		{
			name:          "Test sending small message",
			fields:        newTestQuicClient("normal", "/tmp/edge.crt", "/tmp/edge.key", "/tmp/edge.crt"),
			message:       msg,
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := tt.fields

			if err := qc.Init(); err != nil {
				t.Fatalf("failed to init, err: %v", err)
			}

			if err := qc.Send(tt.message); !reflect.DeepEqual(err, tt.expectedError) {
				t.Fatalf("QuicClient.Send() error = %v, expectedError = %v", err, tt.expectedError)
			}
		})
	}
}

func TestReceive(t *testing.T) {
	newTestServer(t)

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
		fields        *QuicClient
		want          model.Message
		sent          model.Message
		expectedError error
	}{
		{name: "Test Receiving the send message: Success in receiving",
			fields:        newTestQuicClient("normal", "/tmp/edge.crt", "/tmp/edge.key", "/tmp/edge.crt"),
			want:          msg,
			sent:          msg,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := tt.fields
			if err := qc.Init(); err != nil {
				t.Fatalf("failed to init, err: %v", err)
			}

			if err := qc.Send(tt.sent); err != nil {
				t.Fatalf("failed to send, err: %v", err)
			}

			got, err := qc.Receive()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Fatalf("QuicClient.Receive() error = %v, expectedError = %v", err, tt.expectedError)
			}
			if !reflect.DeepEqual(fmt.Sprintf("%s", got.GetContent()), fmt.Sprintf("%s", tt.want.GetContent())) {
				t.Fatalf("QuicClient.Receive() message content: got = %s, want = %s", got.GetContent(), tt.want.GetContent())
			}
		})
	}
}
