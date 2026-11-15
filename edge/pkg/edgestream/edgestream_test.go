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

package edgestream

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestCert returns a self-signed certificate without SANs and its PEM encoding.
func newTestCert(t *testing.T, usage x509.ExtKeyUsage) (tls.Certificate, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "kubeedge"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{usage},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestNewTunnelTLSConfig(t *testing.T) {
	serverCert, serverPEM := newTestCert(t, x509.ExtKeyUsageServerAuth)
	_, otherPEM := newTestCert(t, x509.ExtKeyUsageServerAuth)
	clientAuthCert, clientAuthPEM := newTestCert(t, x509.ExtKeyUsageClientAuth)
	clientCert, _ := newTestCert(t, x509.ExtKeyUsageClientAuth)

	cases := []struct {
		name       string
		serverCert tls.Certificate
		caPEM      []byte
		wantErr    bool
	}{
		{
			name:       "server certificate signed by tunnel ca",
			serverCert: serverCert,
			caPEM:      serverPEM,
		},
		{
			name:       "server certificate signed by another ca",
			serverCert: serverCert,
			caPEM:      otherPEM,
			wantErr:    true,
		},
		{
			name:       "server certificate without server auth usage",
			serverCert: clientAuthCert,
			caPEM:      clientAuthPEM,
			wantErr:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			server.TLS = &tls.Config{Certificates: []tls.Certificate{c.serverCert}}
			server.StartTLS()
			defer server.Close()

			caFile := filepath.Join(t.TempDir(), "ca.crt")
			if err := os.WriteFile(caFile, c.caPEM, 0600); err != nil {
				t.Fatalf("failed to write ca file: %v", err)
			}
			tlsConfig, err := newTunnelTLSConfig(caFile, clientCert)
			if err != nil {
				t.Fatalf("newTunnelTLSConfig() error = %v", err)
			}

			conn, err := tls.Dial("tcp", server.Listener.Addr().String(), tlsConfig)
			if err == nil {
				conn.Close()
			}
			if (err != nil) != c.wantErr {
				t.Errorf("tls.Dial() error = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

func TestNewTunnelTLSConfigInvalidCA(t *testing.T) {
	clientCert, _ := newTestCert(t, x509.ExtKeyUsageClientAuth)
	if _, err := newTunnelTLSConfig(filepath.Join(t.TempDir(), "missing.crt"), clientCert); err == nil {
		t.Error("expected error for missing ca file")
	}

	caFile := filepath.Join(t.TempDir(), "ca.crt")
	if err := os.WriteFile(caFile, []byte("not a certificate"), 0600); err != nil {
		t.Fatalf("failed to write ca file: %v", err)
	}
	if _, err := newTunnelTLSConfig(caFile, clientCert); err == nil {
		t.Error("expected error for invalid ca file")
	}
}
