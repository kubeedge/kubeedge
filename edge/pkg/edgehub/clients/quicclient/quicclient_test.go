package quicclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"reflect"
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
	_, err := os.Stat("/tmp/edge.crt")
	if err != nil {
		err := util.GenerateTestCertificate("/tmp/", "edge", "edge")

		if err != nil {
			fmt.Printf("Failed to create certificate: %v\n", err)
		}
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

func newTestServer(t *testing.T) {
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
		err = httpServer.ListenAndServeTLS("/tmp/edge.crt", "/tmp/edge.key")
		if err != nil {
			klog.Errorf("listen and serve tls failed, error: %+v", err)
		}
	}()
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
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.client.Init()
			if tt.expectedError != nil && err != nil && err.Error() != tt.expectedError.Error() {
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
				t.Errorf("failed to init, err: %v", err)
			}

			if err := qc.Send(tt.message); !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("QuicClient.Send() error = %v, expectedError = %v", err, tt.expectedError)
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
				t.Errorf("failed to init, err: %v", err)
			}

			if err := qc.Send(tt.sent); err != nil {
				t.Errorf("failed to send, err: %v", err)
			}

			got, err := qc.Receive()
			if !reflect.DeepEqual(err, tt.expectedError) {
				t.Errorf("QuicClient.Receive() error = %v, expectedError = %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(fmt.Sprintf("%s", got.GetContent()), fmt.Sprintf("%s", tt.want.GetContent())) {
				t.Errorf("QuicClient.Receive() message content: got = %s, want = %s", got.GetContent(), tt.want.GetContent())
			}
		})
	}
}
