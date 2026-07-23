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

package edge

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	edgeconfig "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	fsmv1alpha1 "github.com/kubeedge/api/apis/fsm/v1alpha1"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func makeTLSServerSignedByCA(
	t *testing.T,
	caCert *x509.Certificate,
	caKey *rsa.PrivateKey,
	handler http.Handler,
) *httptest.Server {
	t.Helper()

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(99),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{serverDER},
				PrivateKey:  serverKey,
			},
		},
	}
	srv.StartTLS()
	return srv
}

func TestNewTaskEventReporter(t *testing.T) {
	config := &edgeconfig.EdgeCoreConfig{}
	reporter := NewTaskEventReporter("job1", "upgrade", config)

	r, ok := reporter.(*TaskEventReporter)
	assert.True(t, ok)
	assert.Equal(t, "job1", r.JobName)
	assert.Equal(t, "upgrade", r.EventType)
	assert.Equal(t, config, r.Config)
}

func TestReport(t *testing.T) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	caFile, err := os.CreateTemp("", "test-ca-*.crt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(caFile.Name()) }()
	require.NoError(t, pem.Encode(caFile, &pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	require.NoError(t, caFile.Close())

	// Generate a valid client certificate signed by the CA
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caTemplate, &clientKey.PublicKey, caKey)
	require.NoError(t, err)

	dummyCert, err := os.CreateTemp("", "test-client-*.crt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(dummyCert.Name()) }()
	require.NoError(t, pem.Encode(dummyCert, &pem.Block{Type: "CERTIFICATE", Bytes: clientDER}))
	require.NoError(t, dummyCert.Close())

	dummyKey, err := os.CreateTemp("", "test-client-*.key")
	require.NoError(t, err)
	defer func() { _ = os.Remove(dummyKey.Name()) }()
	clientKeyBytes, err := x509.MarshalPKCS8PrivateKey(clientKey)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(dummyKey, &pem.Block{Type: "PRIVATE KEY", Bytes: clientKeyBytes}))
	require.NoError(t, dummyKey.Close())

	tests := []struct {
		name           string
		reportErr      error
		expectedAction fsmv1alpha1.Action
	}{
		{
			name:           "nil error maps to ActionSuccess",
			reportErr:      nil,
			expectedAction: fsmv1alpha1.ActionSuccess,
		},
		{
			name:           "non-nil error maps to ActionFailure",
			reportErr:      errors.New("something went wrong"),
			expectedAction: fsmv1alpha1.ActionFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actionCh := make(chan fsmv1alpha1.Action, 1)

			mockServer := makeTLSServerSignedByCA(t, caCert, caKey,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var body commontypes.NodeTaskResponse
					if decErr := json.NewDecoder(r.Body).Decode(&body); decErr == nil {
						actionCh <- body.Action
					}
					w.WriteHeader(http.StatusOK)
				}),
			)
			defer mockServer.Close()

			config := &edgeconfig.EdgeCoreConfig{}
			config.Modules = &edgeconfig.Modules{
				EdgeHub: &edgeconfig.EdgeHub{
					TLSCAFile:         caFile.Name(),
					TLSCertFile:       dummyCert.Name(),
					TLSPrivateKeyFile: dummyKey.Name(),
					HTTPServer:        mockServer.URL,
				},
				Edged: &edgeconfig.Edged{
					TailoredKubeletFlag: edgeconfig.TailoredKubeletFlag{
						HostnameOverride: "test-node",
					},
				},
			}

			reporter := &TaskEventReporter{
				JobName:   "test-job",
				EventType: "upgrade",
				Config:    config,
			}

			reportErr := reporter.Report(tt.reportErr)
			assert.NoError(t, reportErr)

			select {
			case got := <-actionCh:
				assert.Equal(t, tt.expectedAction, got,
					"action sent in POST body did not match expected mapping")
			case <-time.After(3 * time.Second):
				t.Error("timed out waiting for mock server to receive request")
			}
		})
	}
}

func TestReportTaskResult_MissingCAFile(t *testing.T) {
	config := &edgeconfig.EdgeCoreConfig{}
	config.Modules = &edgeconfig.Modules{
		EdgeHub: &edgeconfig.EdgeHub{
			TLSCAFile: "/testdata/ca.crt",
		},
		Edged: &edgeconfig.Edged{
			TailoredKubeletFlag: edgeconfig.TailoredKubeletFlag{
				HostnameOverride: "test-node",
			},
		},
	}

	err := ReportTaskResult(config, TaskTypeUpgrade, "job1", fsm.Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read ca")
}

func TestReportTaskResult_InvalidClientCert(t *testing.T) {
	caFile, err := os.CreateTemp("", "test-ca-*.crt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(caFile.Name()) }()
	
	// Create dummy non-empty file for CA to pass the read check
	_, err = caFile.Write([]byte("dummy-ca"))
	require.NoError(t, err)
	require.NoError(t, caFile.Close())

	config := &edgeconfig.EdgeCoreConfig{}
	config.Modules = &edgeconfig.Modules{
		EdgeHub: &edgeconfig.EdgeHub{
			TLSCAFile:         caFile.Name(),
			TLSCertFile:       "/invalid/cert/path",
			TLSPrivateKeyFile: "/invalid/key/path",
		},
		Edged: &edgeconfig.Edged{
			TailoredKubeletFlag: edgeconfig.TailoredKubeletFlag{
				HostnameOverride: "test-node",
			},
		},
	}

	err = ReportTaskResult(config, TaskTypeUpgrade, "job1", fsm.Event{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load client certificate")
}
