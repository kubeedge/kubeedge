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
	assert.NoError(t, err)

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(99),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")}, // ← add this
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	assert.NoError(t, err)

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
	// ── Build a test CA ──────────────────────────────────────────────────────
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

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
	assert.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	assert.NoError(t, err)

	// Write CA cert PEM to a temp file — this is what os.ReadFile reads in
	// ReportTaskResult.
	caFile, err := os.CreateTemp("", "test-ca-*.crt")
	assert.NoError(t, err)
	defer os.Remove(caFile.Name())
	assert.NoError(t, pem.Encode(caFile, &pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	assert.NoError(t, caFile.Close())

	// Empty temp files for TLSCertFile / TLSPrivateKeyFile.
	// LoadX509KeyPair will fail on these but the error is ignored in source.
	dummyCert, err := os.CreateTemp("", "dummy-cert-*.crt")
	assert.NoError(t, err)
	defer os.Remove(dummyCert.Name())
	dummyCert.Close()

	dummyKey, err := os.CreateTemp("", "dummy-key-*.key")
	assert.NoError(t, err)
	defer os.Remove(dummyKey.Name())
	dummyKey.Close()

	// ── Table-driven cases ───────────────────────────────────────────────────
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
			// Buffered so the handler never blocks.
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
					// mockServer.URL is "https://127.0.0.1:<port>"
					HTTPServer: mockServer.URL,
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
			// TLS handshake succeeds (server cert chained to our CA) and the
			// mock returns 200, so Report() should return nil.
			assert.NoError(t, reportErr)

			select {
			case got := <-actionCh:
				assert.Equal(t, tt.expectedAction, got,
					"action sent in POST body did not match expected mapping")
			default:
				t.Error("mock server did not receive a request — check TLS handshake")
			}
		})
	}
}

// TestReportTaskResult_MissingCAFile verifies that a missing CA file produces
// a clear error before any network activity occurs.
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
