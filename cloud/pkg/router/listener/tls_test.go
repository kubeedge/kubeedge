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

package listener

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	routerConfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
)

type testCA struct {
	cert *x509.Certificate
	key  crypto.Signer
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &testCA{cert: cert, key: key}
}

// issue returns the DER certificate and PKCS #8 DER key of a server certificate
// that is only valid for the given DNS name, not for 127.0.0.1.
func (ca *testCA) issue(t *testing.T, dnsName string) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: dnsName},
		DNSNames:     []string{dnsName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, key.Public(), ca.key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return certDER, keyDER
}

func writeKeyPair(t *testing.T, certDER, keyDER []byte) (string, string) {
	t.Helper()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "router.crt")
	keyFile := filepath.Join(dir, "router.key")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600))
	return certFile, keyFile
}

func newTargetCloudCore(t *testing.T, certDER, keyDER []byte) *httptest.Server {
	t.Helper()
	cert, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
	)
	require.NoError(t, err)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

func TestNewServerTLSConfig(t *testing.T) {
	ca := newTestCA(t)
	certDER, keyDER := ca.issue(t, "router.example")
	certFile, keyFile := writeKeyPair(t, certDER, keyDER)

	oldCert, oldKey := hubconfig.Config.Cert, hubconfig.Config.Key
	t.Cleanup(func() {
		hubconfig.Config.Cert, hubconfig.Config.Key = oldCert, oldKey
	})
	hubCertDER, hubKeyDER := ca.issue(t, "cloudcore.example")
	hubconfig.Config.Cert, hubconfig.Config.Key = hubCertDER, hubKeyDER

	cases := []struct {
		name     string
		config   *v1alpha1.RouterTLS
		wantCert []byte
		wantErr  bool
	}{
		{
			name:     "configured certificate",
			config:   &v1alpha1.RouterTLS{Enable: true, CertFile: certFile, PrivateKeyFile: keyFile},
			wantCert: certDER,
		},
		{
			name:     "CloudCore certificate when no certificate is configured",
			config:   &v1alpha1.RouterTLS{Enable: true},
			wantCert: hubCertDER,
		},
		{
			name:    "configured certificate does not exist",
			config:  &v1alpha1.RouterTLS{Enable: true, CertFile: certFile + ".missing", PrivateKeyFile: keyFile},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tlsConfig, err := newServerTLSConfig(c.config)
			if c.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, tlsConfig.Certificates, 1)
			assert.Equal(t, c.wantCert, tlsConfig.Certificates[0].Certificate[0])
			assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
		})
	}
}

func TestNewForwardTLSClient(t *testing.T) {
	cloudCoreCA := newTestCA(t)
	otherCA := newTestCA(t)

	// The target is reached by 127.0.0.1, which is not in its certificate.
	trustedCert, trustedKey := cloudCoreCA.issue(t, "cloudcore.example")
	untrustedCert, untrustedKey := otherCA.issue(t, "cloudcore.example")

	cases := []struct {
		name          string
		targetCert    []byte
		targetKey     []byte
		serverCertDER []byte
		wantErr       bool
	}{
		{
			name:       "target certificate issued by the CloudCore CA",
			targetCert: trustedCert,
			targetKey:  trustedKey,
		},
		{
			name:       "target certificate issued by another CA",
			targetCert: untrustedCert,
			targetKey:  untrustedKey,
			wantErr:    true,
		},
		{
			name:          "target presents the same certificate as this listener",
			targetCert:    untrustedCert,
			targetKey:     untrustedKey,
			serverCertDER: untrustedCert,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := newTargetCloudCore(t, c.targetCert, c.targetKey)
			client, err := newForwardTLSClient(cloudCoreCA.cert.Raw, c.serverCertDER, 5*time.Second)
			require.NoError(t, err)

			resp, err := client.Get(srv.URL)
			if c.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}

	t.Run("invalid CloudCore CA", func(t *testing.T) {
		_, err := newForwardTLSClient([]byte("invalid"), nil, time.Second)
		assert.Error(t, err)
	})
}

func TestServeTLS(t *testing.T) {
	ca := newTestCA(t)
	certDER, keyDER := ca.issue(t, "router.example")
	certFile, keyFile := writeKeyPair(t, certDER, keyDER)

	oldTLS := routerConfig.Config.TLS
	t.Cleanup(func() {
		routerConfig.Config.TLS = oldTLS
	})
	routerConfig.Config.TLS = &v1alpha1.RouterTLS{Enable: true, CertFile: certFile, PrivateKeyFile: keyFile}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	rh := &RestHandler{
		bindAddress: "127.0.0.1",
		port:        port,
		tlsEnabled:  true,
		restTimeout: time.Second,
	}
	go rh.Serve()

	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)
	var conn *tls.Conn
	require.Eventually(t, func() bool {
		conn, err = tls.Dial("tcp", l.Addr().String(), &tls.Config{
			RootCAs:    roots,
			ServerName: "router.example",
			MinVersion: tls.VersionTLS12,
		})
		return err == nil
	}, 5*time.Second, 50*time.Millisecond)
	defer conn.Close()

	assert.Equal(t, certDER, conn.ConnectionState().PeerCertificates[0].Raw)
}
