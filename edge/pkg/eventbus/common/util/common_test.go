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

package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
)

var clientOptions = MQTT.NewClientOptions()

func init() {
	nodeName := "testEdge"
	cfg := v1alpha2.NewDefaultEdgeCoreConfig()
	eventconfig.InitConfigure(cfg.Modules.EventBus, nodeName)
}

// TestCheckKeyExist checks the functionality of CheckKeyExist function
func TestCheckKeyExist(t *testing.T) {
	tests := []struct {
		name          string
		keys          []string
		disinfo       map[string]interface{}
		expectedError error
	}{
		{
			name:          "TestCheckKeyExist: Key exists in passed map",
			keys:          []string{"key1"},
			disinfo:       map[string]interface{}{"key1": "value1"},
			expectedError: nil,
		},
		{
			name:          "TestCheckKeyExist: Key does not exists in passed map",
			keys:          []string{"key1"},
			disinfo:       map[string]interface{}{"key2": "value2"},
			expectedError: errors.New("key not found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckKeyExist(tt.keys, tt.disinfo)
			if !reflect.DeepEqual(tt.expectedError, err) {
				t.Errorf("Expected error contain %s, but error is %v", tt.expectedError, err)
			}
		})
	}
}

// TestCheckClientToken checks client token received
func TestCheckClientToken(t *testing.T) {
	tests := []struct {
		name          string
		token         MQTT.Token
		expectedError string
	}{
		{
			name:          "TestCheckClientToken: Client Token with no error",
			token:         MQTT.NewClient(clientOptions).Connect(),
			expectedError: "",
		},
		{
			name:          "TestCheckClientToken: Client token created with error",
			token:         MQTT.NewClient(HubClientInit("tcp://127.0.0:8000", "12345", "", "")).Connect(),
			expectedError: "Network Error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, err := CheckClientToken(tt.token)
			fmt.Printf("rs  =  %v", rs)
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error contain %s, but error is %v", tt.expectedError, err)
			}
		})
	}
}

// TestPathExist checks the functionality of PathExist function
func TestPathExist(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "TestPathExist: Path Exist",
			path: "/",
			want: true,
		},
		{
			name: "TestPathExist: Path does not Exist",
			path: "Wrong_Path",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathExist(tt.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("common.TestPathExist() got = %v, want =  %v", got, tt.want)
			}
		})
	}
}

// TestHubClientInit checks the HubClientInit method that it returns the same clientOptions object or not
func TestHubClientInit(t *testing.T) {
	tests := []struct {
		name          string
		server        string
		clientID      string
		username      string
		password      string
		want          *MQTT.ClientOptions
		expectedError error
	}{
		{
			name:          "TestHubclientInit: given username and password",
			server:        "tcp://127.0.0.1:1880",
			clientID:      "12345",
			username:      "test_user",
			password:      "123456789",
			want:          clientOptions,
			expectedError: nil,
		},
		{
			name:          "TestHubclientInit: given username and no password",
			server:        "tcp://127.0.0.1:1882",
			clientID:      "12345",
			username:      "test_user",
			password:      "",
			want:          clientOptions,
			expectedError: nil,
		},
		{
			name:          "TestHubclientInit: no username and password",
			server:        "tcp://127.0.0.1:1883",
			clientID:      "12345",
			username:      "",
			password:      "",
			want:          clientOptions,
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brokerURI, _ := url.Parse(tt.server)
			tt.want.Servers = append([]*url.URL{}, brokerURI)
			tt.want.ClientID = tt.clientID
			tt.want.Username = tt.username
			tt.want.Password = tt.password
			tt.want.TLSConfig = &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
			got := HubClientInit(tt.server, tt.clientID, tt.username, tt.password)
			if !reflect.DeepEqual(tt.want.Servers, got.Servers) || tt.want.ClientID != got.ClientID || tt.want.CleanSession != got.CleanSession ||
				tt.want.Username != got.Username || tt.want.Password != got.Password || !reflect.DeepEqual(tt.want.TLSConfig, got.TLSConfig) {
				t.Errorf("expected %#v, but got %#v", tt.want, got)
			}
		})
	}
}

// TestLoopConnect checks LoopConnect to connect to MQTT broker
func TestLoopConnect(t *testing.T) {
	tests := []struct {
		name          string
		client        MQTT.Client
		clientID      string
		clientOptions *MQTT.ClientOptions
		connect       bool
	}{
		{
			name:          "TestLoopConnect: success in connection",
			clientID:      "12345",
			clientOptions: MQTT.NewClientOptions(),
			connect:       true,
		},
		{
			name:          "TestLoopConnect: Connection error",
			clientID:      "12345",
			clientOptions: HubClientInit("tcp://127.0.0.1:1882", "12345", "test_user", "123456789"),
			connect:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.client = MQTT.NewClient(tt.clientOptions)
			go LoopConnect(tt.clientID, tt.client)
			time.Sleep(5 * time.Millisecond)
			if !tt.client.IsConnected() {
				if len(tt.clientOptions.Servers) != 0 {
					if tt.connect {
						t.Errorf("common.TestLoopConnect() Options.Servers = %v, want connect =  %v", tt.clientOptions.Servers, tt.connect)
					}
				}
				klog.Info("No servers defined to connect to")
			}
		})
	}
}

// TestTLSConfig tests the TLS configuration code paths
func TestTLSConfig(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certContent := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTIzMDQwODE5NDMwNloXDTI5MDQwODE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)

	keyContent := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)

	caContent := certContent

	certFile := tempDir + "/cert.pem"
	keyFile := tempDir + "/key.pem"
	caFile := tempDir + "/ca.pem"
	invalidFile := tempDir + "/invalid.pem"

	if err := os.WriteFile(certFile, certContent, 0644); err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyContent, 0644); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := os.WriteFile(caFile, caContent, 0644); err != nil {
		t.Fatalf("Failed to create CA file: %v", err)
	}
	if err := os.WriteFile(invalidFile, []byte("invalid content"), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	originalTLSConfig := eventconfig.Config.TLS
	defer func() {
		eventconfig.Config.TLS = originalTLSConfig
	}()

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadX509KeyPair failed: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatalf("Certificate is empty")
	}

	caCert, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatalf("Failed to read CA file: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		t.Fatalf("Failed to append CA certificate to pool")
	}

	t.Run("TLS enabled", func(t *testing.T) {
		eventconfig.Config.TLS.Enable = true
		eventconfig.Config.TLS.TLSMqttCertFile = certFile
		eventconfig.Config.TLS.TLSMqttPrivateKeyFile = keyFile
		eventconfig.Config.TLS.TLSMqttCAFile = caFile

		opts := HubClientInit("tcp://127.0.0.1:1883", "testClient", "", "")
		if opts == nil {
			t.Fatal("HubClientInit returned nil with valid certificate")
		}
		if opts.TLSConfig == nil {
			t.Fatal("TLS config is nil")
		}
		if opts.TLSConfig.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should be false")
		}
		if opts.TLSConfig.RootCAs == nil {
			t.Error("RootCAs is nil")
		}
	})

	t.Run("Invalid CA file", func(t *testing.T) {
		eventconfig.Config.TLS.Enable = true
		eventconfig.Config.TLS.TLSMqttCertFile = certFile
		eventconfig.Config.TLS.TLSMqttPrivateKeyFile = keyFile
		eventconfig.Config.TLS.TLSMqttCAFile = invalidFile

		opts := HubClientInit("tcp://127.0.0.1:1883", "testClient", "", "")
		if opts != nil {
			t.Error("Expected nil result with invalid CA")
		}
	})

	t.Run("Non-existent CA file", func(t *testing.T) {
		eventconfig.Config.TLS.Enable = true
		eventconfig.Config.TLS.TLSMqttCertFile = certFile
		eventconfig.Config.TLS.TLSMqttPrivateKeyFile = keyFile
		eventconfig.Config.TLS.TLSMqttCAFile = tempDir + "/nonexistent.pem"

		opts := HubClientInit("tcp://127.0.0.1:1883", "testClient", "", "")
		if opts != nil {
			t.Errorf("Expected nil result with non-existent CA file")
		}
	})

	t.Run("TLS disabled", func(t *testing.T) {
		eventconfig.Config.TLS.Enable = false
		opts := HubClientInit("tcp://127.0.0.1:1883", "testClient", "", "")

		if opts == nil {
			t.Error("Expected non-nil result with TLS disabled")
		} else if opts.TLSConfig == nil {
			t.Error("TLS config should not be nil")
		} else if !opts.TLSConfig.InsecureSkipVerify {
			t.Error("InsecureSkipVerify should be true when TLS is disabled")
		}
	})
}
