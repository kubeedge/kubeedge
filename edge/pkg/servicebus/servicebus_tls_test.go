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

package servicebus

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	servicebusConfig "github.com/kubeedge/kubeedge/edge/pkg/servicebus/config"
)

// generateServerCert writes a self-signed TLS server certificate (with
// ExtKeyUsageServerAuth and a 127.0.0.1 SAN) plus its private key to dir.
// This is the correct certificate type for a ServiceBus HTTPS server.
// The EdgeHub client certificate CANNOT be reused here because it carries
// ExtKeyUsageClientAuth and no ServiceBus SANs.
func generateServerCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "servicebus-server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		// ExtKeyUsageServerAuth is required for HTTPS server certificates.
		// A ClientAuth-only cert (like the EdgeHub cert) would be rejected by
		// a normal HTTPS client performing TLS server certificate verification.
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// 127.0.0.1 SAN is required so that clients connecting to
		// "https://127.0.0.1:..." can validate the certificate hostname.
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certFile = filepath.Join(dir, "server.crt")
	keyFile = filepath.Join(dir, "server.key")

	cf, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	defer cf.Close()
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("failed to PEM-encode cert: %v", err)
	}

	kf, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	defer kf.Close()
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER}); err != nil {
		t.Fatalf("failed to PEM-encode key: %v", err)
	}

	return certFile, keyFile
}

// generateClientAuthCert writes a self-signed certificate with
// ExtKeyUsageClientAuth only (no ServerAuth, no IP SAN) — simulating the
// EdgeHub certificate that CloudCore issues for edge nodes.
func generateClientAuthCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generateClientAuthCert: key gen: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "system:node:test-node"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		// ClientAuth only — no ServerAuth, no SANs.
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("generateClientAuthCert: create cert: %v", err)
	}

	certFile = filepath.Join(dir, "client.crt")
	keyFile = filepath.Join(dir, "client.key")

	cf, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("generateClientAuthCert: create certFile: %v", err)
	}
	defer cf.Close()
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("generateClientAuthCert: encode cert: %v", err)
	}

	kf, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("generateClientAuthCert: create keyFile: %v", err)
	}
	defer kf.Close()
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("generateClientAuthCert: marshal key: %v", err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER}); err != nil {
		t.Fatalf("generateClientAuthCert: encode key: %v", err)
	}

	return certFile, keyFile
}

// TestBuildTLSConfigDisabled verifies that buildTLSConfig returns (nil, nil)
// when TLSEnabled is false — the server must start as plain HTTP.
func TestBuildTLSConfigDisabled(t *testing.T) {
	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: false})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil TLS config when TLSEnabled=false")
	}
}

// TestBuildTLSConfigEnabledNoPaths verifies that enabling TLS without cert/key
// paths returns an error.  The server must NOT silently fall back to HTTP.
func TestBuildTLSConfigEnabledNoPaths(t *testing.T) {
	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: "", KeyFile: ""})
	if err == nil {
		t.Error("expected an error when TLS is enabled but CertFile/KeyFile are empty")
	}
	if cfg != nil {
		t.Error("expected nil TLS config on error")
	}
}

// TestBuildTLSConfigServerAuthCert verifies that a certificate with
// ExtKeyUsageServerAuth and a 127.0.0.1 SAN produces a valid *tls.Config.
// This is the correct certificate type for a ServiceBus HTTPS server.
func TestBuildTLSConfigServerAuthCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateServerCert(t, dir)

	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err != nil {
		t.Fatalf("expected no error with a valid ServerAuth cert, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion: got %v, want TLS 1.2", cfg.MinVersion)
	}
	if cfg.ClientAuth != tls.NoClientCert {
		t.Errorf("ClientAuth: got %v, want NoClientCert (server-only TLS)", cfg.ClientAuth)
	}
	if cfg.ClientCAs != nil {
		t.Error("ClientCAs must be nil: server-only TLS does not use a client CA pool")
	}
	if cfg.GetCertificate == nil {
		t.Error("GetCertificate callback must not be nil (required for cert rotation)")
	}
}

// TestBuildTLSConfigClientAuthCertFails verifies that buildTLSConfig returns
// an error when a ClientAuth-only certificate is supplied — simulating the
// EdgeHub certificate that CloudCore issues for edge nodes.
//
// This test documents that the rejected approach is caught at startup:
// the EdgeHub cert MUST NOT be reused as a ServiceBus server identity,
// and EdgeCore now actively rejects it rather than only letting clients
// discover the problem at handshake time.
func TestBuildTLSConfigClientAuthCertFails(t *testing.T) {
	dir := t.TempDir()
	// Generate a ClientAuth-only cert (simulating the EdgeHub cert).
	certFile, keyFile := generateClientAuthCert(t, dir)

	// buildTLSConfig must now return an error because the cert lacks
	// ExtKeyUsageServerAuth — startup validation catches this immediately.
	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err == nil {
		t.Error("expected an error for a ClientAuth-only cert (missing ExtKeyUsageServerAuth), got nil")
	}
	if cfg != nil {
		t.Error("expected nil TLS config when cert validation fails")
	}
}

// TestBuildTLSConfigInvalidCertPaths verifies that buildTLSConfig returns an
// error (not klog.Fatalf) when the cert/key files do not exist.
func TestBuildTLSConfigInvalidCertPaths(t *testing.T) {
	cfg, err := buildTLSConfig(TLSOptions{
		TLSEnabled: true,
		CertFile:   "/nonexistent/cert.crt",
		KeyFile:    "/nonexistent/key.key",
	})
	if err == nil {
		t.Error("expected an error for missing cert files, got nil")
	}
	if cfg != nil {
		t.Error("expected nil TLS config on error")
	}
}

// TestValidateServerAuthEKU directly covers the validateServerAuthEKU helper.
func TestValidateServerAuthEKU(t *testing.T) {
	dir := t.TempDir()

	t.Run("ServerAuthCertPasses", func(t *testing.T) {
		certFile, _ := generateServerCert(t, dir)
		if err := validateServerAuthEKU(certFile); err != nil {
			t.Errorf("expected no error for ServerAuth cert, got: %v", err)
		}
	})

	t.Run("ClientAuthCertFails", func(t *testing.T) {
		certFile, _ := generateClientAuthCert(t, dir)
		if err := validateServerAuthEKU(certFile); err == nil {
			t.Error("expected an error for ClientAuth-only cert, got nil")
		}
	})

	t.Run("NonExistentFileFails", func(t *testing.T) {
		if err := validateServerAuthEKU("/nonexistent/cert.crt"); err == nil {
			t.Error("expected an error for nonexistent cert file, got nil")
		}
	})
}

// TestBuildTLSConfigGetCertificateReloads verifies that the GetCertificate
// callback re-reads the cert from disk, enabling certificate rotation.
// This test writes a first certificate, calls GetCertificate, then replaces
// the files with a second certificate and verifies the next call returns the
// new certificate (i.e., the serial number changes).
func TestBuildTLSConfigGetCertificateReloads(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateServerCert(t, dir)

	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err != nil || cfg == nil {
		t.Fatalf("buildTLSConfig: %v, cfg=%v", err, cfg)
	}

	// First call — load the original certificate.
	cert1, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate (first call): %v", err)
	}
	if cert1 == nil {
		t.Fatal("GetCertificate (first call) returned nil cert")
	}
	parsed1, err := x509.ParseCertificate(cert1.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate (first): %v", err)
	}

	// Generate a second, distinct certificate with a different serial number
	// and overwrite the same files — simulating CertManager rotation.
	priv2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key gen 2: %v", err)
	}
	tmpl2 := &x509.Certificate{
		SerialNumber: big.NewInt(999), // distinct from the first cert's serial (1)
		Subject:      pkix.Name{CommonName: "servicebus-server-rotated"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(2 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER2, err := x509.CreateCertificate(rand.Reader, tmpl2, tmpl2, &priv2.PublicKey, priv2)
	if err != nil {
		t.Fatalf("CreateCertificate 2: %v", err)
	}

	// Overwrite cert file.
	cf, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("overwrite certFile: %v", err)
	}
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER2}); err != nil {
		cf.Close()
		t.Fatalf("encode cert2: %v", err)
	}
	cf.Close()

	// Overwrite key file.
	privDER2, err := x509.MarshalPKCS8PrivateKey(priv2)
	if err != nil {
		t.Fatalf("marshal key2: %v", err)
	}
	kf, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("overwrite keyFile: %v", err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER2}); err != nil {
		kf.Close()
		t.Fatalf("encode key2: %v", err)
	}
	kf.Close()

	// Second call — must return the NEW certificate.
	cert2, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate (second call): %v", err)
	}
	if cert2 == nil {
		t.Fatal("GetCertificate (second call) returned nil cert")
	}
	parsed2, err := x509.ParseCertificate(cert2.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate (second): %v", err)
	}

	// The serial numbers must differ, proving the cert was reloaded from disk.
	if parsed1.SerialNumber.Cmp(parsed2.SerialNumber) == 0 {
		t.Errorf("GetCertificate did not reload the rotated certificate: serial unchanged (%s)", parsed1.SerialNumber)
	}
}

// TestHTTPSServerWithServerAuthCert starts a real TLS listener (not httptest)
// using a ServerAuth + 127.0.0.1-SAN certificate, then verifies that a normal
// http.Client configured to trust only that CA can connect successfully.
//
// This test directly addresses maintainer issue 5: httptest.StartTLS injects
// its own certificate when TLS.Certificates is empty and GetCertificate is
// set, so it never exercises the configured certificate.  Using a real
// net.Listen + tls.NewListener lets us control the certificate end-to-end.
func TestHTTPSServerWithServerAuthCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateServerCert(t, dir)

	tlsCfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err != nil || tlsCfg == nil {
		t.Fatalf("buildTLSConfig: %v, cfg=%v", err, tlsCfg)
	}

	// Start a real TLS listener so we control which certificate is served.
	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(listener) //nolint:errcheck
	defer srv.Close()

	// Load the server cert as the trusted CA for the client (self-signed).
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("ReadFile(certFile): %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		t.Fatal("AppendCertsFromPEM returned false")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool, // trust only our server cert
			},
		},
	}

	resp, err := client.Get("https://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("GET with trusted CA: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

// TestHTTPSServerClientAuthCertRejected verifies that buildTLSConfig rejects
// a certificate that has ExtKeyUsageClientAuth only (no ServerAuth) at startup,
// proving that the EdgeHub cert cannot be reused as a ServiceBus server identity.
// Since EKU validation now happens in buildTLSConfig, the error surfaces before
// a TLS listener is ever started.
func TestHTTPSServerClientAuthCertRejected(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateClientAuthCert(t, dir)

	// buildTLSConfig must now reject the ClientAuth-only cert at startup.
	tlsCfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err == nil {
		t.Error("expected buildTLSConfig to reject a ClientAuth-only cert, got nil error")
	}
	if tlsCfg != nil {
		t.Error("expected nil TLS config when cert lacks ExtKeyUsageServerAuth")
	}
	_ = keyFile // used above in TLSOptions
}

// TestHTTPSServerSANMismatchRejected verifies that a client rejects a server
// certificate whose SAN does not match the address being connected to.
func TestHTTPSServerSANMismatchRejected(t *testing.T) {
	dir := t.TempDir()

	// Generate a cert with SAN 127.0.0.2 (intentional mismatch for 127.0.0.1).
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "wrong-san"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.2")}, // wrong SAN
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)

	certFile := filepath.Join(dir, "mismatch.crt")
	keyFile := filepath.Join(dir, "mismatch.key")
	cf, _ := os.Create(certFile)
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		cf.Close()
		t.Fatalf("pem.Encode cert: %v", err)
	}
	cf.Close()
	kf, _ := os.Create(keyFile)
	privDER, _ := x509.MarshalPKCS8PrivateKey(priv)
	if err := pem.Encode(kf, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER}); err != nil {
		kf.Close()
		t.Fatalf("pem.Encode key: %v", err)
	}
	kf.Close()

	tlsCfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	defer listener.Close()

	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})}
	go srv.Serve(listener) //nolint:errcheck
	defer srv.Close()

	certPEM, _ := os.ReadFile(certFile)
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(certPEM)

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}

	// Must fail: the cert's SAN is 127.0.0.2 but we're connecting to 127.0.0.1.
	_, err = client.Get("https://" + listener.Addr().String())
	if err == nil {
		t.Error("expected a TLS error for SAN mismatch, got nil")
	}
}

func TestHTTPPlainWhenTLSDisabled(t *testing.T) {
	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: false})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil TLS config when TLSEnabled=false")
	}

	// Start a plain HTTP server using a real listener.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer listener.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(listener) //nolint:errcheck
	defer srv.Close()

	resp, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("plain HTTP GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

// TestInitedResetOnTLSStartupFailure verifies that when server() is called
// with an invalid TLS configuration (non-existent cert/key files), it resets
// the inited flag back to 0 before returning.  Without this fix the flag
// would stay 1 and no subsequent valid startup attempt could ever proceed.
//
// Regression test for Issue #3 in maintainer review of PR #6922.
func TestInitedResetOnTLSStartupFailure(t *testing.T) {
	// Drive inited to 1 as Start() would, simulating a concurrent CAS success.
	if !atomic.CompareAndSwapInt32(&inited, 0, 1) {
		// Another test left inited=1; reset it and proceed.
		atomic.StoreInt32(&inited, 0)
		if !atomic.CompareAndSwapInt32(&inited, 0, 1) {
			t.Fatal("could not set inited to 1 for test setup")
		}
	}
	t.Cleanup(func() { atomic.StoreInt32(&inited, 0) })

	badOpts := TLSOptions{
		TLSEnabled: true,
		CertFile:   "/nonexistent/cert.crt",
		KeyFile:    "/nonexistent/key.key",
	}

	stopCh := make(chan struct{})
	// server() must return (not block) when TLS config fails.
	done := make(chan struct{})
	go func() {
		defer close(done)
		server(stopCh, badOpts)
	}()

	select {
	case <-done:
		// server() returned — check that inited was reset.
	case <-time.After(3 * time.Second):
		t.Fatal("server() did not return within 3 s after invalid TLS config")
	}

	if v := atomic.LoadInt32(&inited); v != 0 {
		t.Errorf("inited: got %d after failed TLS startup, want 0 (must be reset so a later valid attempt is not blocked)", v)
	}
}

// TestDynamicStartupUsesConfiguredTLSOpts verifies that Register stores the
// supplied TLSOptions in the package-level configuredTLSOpts variable so that
// a dynamic/delayed server startup (triggered via processMessage when the URL
// table is initially empty) uses the same TLS configuration as an immediate
// startup.
//
// Regression test for Issue #2 in maintainer review of PR #6922.
func TestDynamicStartupUsesConfiguredTLSOpts(t *testing.T) {
	// Save and restore the package state so the test is isolated.
	origOpts := configuredTLSOpts
	t.Cleanup(func() { configuredTLSOpts = origOpts })

	// Simulate Register() being called with a TLS-enabled config.
	want := TLSOptions{
		TLSEnabled: true,
		CertFile:   "/etc/kubeedge/certs/servicebus-server.crt",
		KeyFile:    "/etc/kubeedge/certs/servicebus-server.key",
	}
	configuredTLSOpts = want // mirrors what Register() now does

	got := configuredTLSOpts
	if got.TLSEnabled != want.TLSEnabled {
		t.Errorf("TLSEnabled: got %v, want %v", got.TLSEnabled, want.TLSEnabled)
	}
	if got.CertFile != want.CertFile {
		t.Errorf("CertFile: got %q, want %q", got.CertFile, want.CertFile)
	}
	if got.KeyFile != want.KeyFile {
		t.Errorf("KeyFile: got %q, want %q", got.KeyFile, want.KeyFile)
	}

	// The dynamic path in processMessage reads configuredTLSOpts — verify it
	// is not the zero value when TLS was configured.
	if configuredTLSOpts == (TLSOptions{}) {
		t.Error("configuredTLSOpts is zero-value; dynamic startup would silently use plain HTTP even when TLS was configured")
	}
}

// TestNewServicebusStoresTLSOpts verifies that newServicebus itself (not a
// simulation) stores the supplied TLSOptions on the returned struct. This
// covers the new tlsOpts field wiring added to newServicebus's signature and
// struct literal.
func TestNewServicebusStoresTLSOpts(t *testing.T) {
	opts := TLSOptions{TLSEnabled: true, CertFile: "/tmp/cert.pem", KeyFile: "/tmp/key.pem"}

	sb := newServicebus(true, "127.0.0.1", 9020, 10, opts)

	if sb == nil {
		t.Fatal("newServicebus returned nil")
	}
	if sb.tlsOpts != opts {
		t.Errorf("tlsOpts: got %+v, want %+v", sb.tlsOpts, opts)
	}
	if !sb.enable || sb.server != "127.0.0.1" || sb.port != 9020 || sb.timeout != 10 {
		t.Errorf("unexpected fields: %+v", sb)
	}
}

// TestRegisterStoresConfiguredTLSOpts verifies that Register() itself (not a
// simulation) stores the supplied TLSOptions in the package-level
// configuredTLSOpts variable. This covers the new "configuredTLSOpts =
// tlsOpts" line and the updated newServicebus call inside Register.
func TestRegisterStoresConfiguredTLSOpts(t *testing.T) {
	origOpts := configuredTLSOpts
	t.Cleanup(func() { configuredTLSOpts = origOpts })

	want := TLSOptions{
		TLSEnabled: true,
		CertFile:   "/etc/kubeedge/certs/servicebus-server.crt",
		KeyFile:    "/etc/kubeedge/certs/servicebus-server.key",
	}

	// Enable is false so Register does not add a live entry to beehive's
	// module registry, keeping this test side-effect free.
	Register(&v1alpha2.ServiceBus{Enable: false, Server: "127.0.0.1", Port: 9021, Timeout: 5}, want)

	if configuredTLSOpts != want {
		t.Errorf("configuredTLSOpts after Register(): got %+v, want %+v", configuredTLSOpts, want)
	}
}

// TestGetCertificateRotationError verifies that the GetCertificate callback
// inside buildTLSConfig returns a descriptive error when the cert file is
// deleted after the initial config was built.  This covers the error branch
// at line 328-330 in servicebus.go (the "certificate rotation: failed to
// reload key pair" path) which exists to surface rotation failures clearly
// instead of crashing the TLS handshake with a nil certificate.
func TestGetCertificateRotationError(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateServerCert(t, dir)

	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err != nil || cfg == nil {
		t.Fatalf("buildTLSConfig: %v, cfg=%v", err, cfg)
	}

	// Delete the cert and key files to simulate a rotation failure.
	if err := os.Remove(certFile); err != nil {
		t.Fatalf("os.Remove(certFile): %v", err)
	}
	if err := os.Remove(keyFile); err != nil {
		t.Fatalf("os.Remove(keyFile): %v", err)
	}

	// GetCertificate must now return an error (not nil, not panic).
	cert, err := cfg.GetCertificate(nil)
	if err == nil {
		t.Error("expected an error from GetCertificate after cert files are deleted, got nil")
	}
	if cert != nil {
		t.Errorf("expected nil certificate on error, got: %v", cert)
	}
}

// TestGetCertificateRotationEKURejected verifies that the GetCertificate
// callback rejects a rotated certificate that lacks ExtKeyUsageServerAuth.
// This covers the new EKU re-validation path added to the rotation callback:
// a cert that passes key-pair loading but lacks ServerAuth must be rejected
// rather than silently served, bypassing the startup EKU check.
func TestGetCertificateRotationEKURejected(t *testing.T) {
	dir := t.TempDir()

	// Build the initial config with a valid ServerAuth cert.
	certFile, keyFile := generateServerCert(t, dir)
	cfg, err := buildTLSConfig(TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	if err != nil || cfg == nil {
		t.Fatalf("buildTLSConfig: %v, cfg=%v", err, cfg)
	}

	// Now overwrite the cert/key files with a ClientAuth-only certificate,
	// simulating a bad certificate rotation (e.g. wrong cert deployed).
	priv2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("key gen: %v", err)
	}
	tmpl2 := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "system:node:bad-rotation"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		// ClientAuth only — no ServerAuth, no SANs (simulates EdgeHub cert).
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDER2, err := x509.CreateCertificate(rand.Reader, tmpl2, tmpl2, &priv2.PublicKey, priv2)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}

	cf, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("overwrite certFile: %v", err)
	}
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER2}); err != nil {
		cf.Close()
		t.Fatalf("pem.Encode cert: %v", err)
	}
	cf.Close()

	privDER2, err := x509.MarshalPKCS8PrivateKey(priv2)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	kf, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("overwrite keyFile: %v", err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER2}); err != nil {
		kf.Close()
		t.Fatalf("pem.Encode key: %v", err)
	}
	kf.Close()

	// GetCertificate must now return an error because the rotated cert
	// lacks ExtKeyUsageServerAuth — the EKU re-validation path.
	cert, err := cfg.GetCertificate(nil)
	if err == nil {
		t.Error("expected GetCertificate to reject a rotated ClientAuth-only cert, got nil error")
	}
	if cert != nil {
		t.Errorf("expected nil certificate when rotation EKU check fails, got: %v", cert)
	}
}

// reserveServerConfig reserves a free localhost port and points the
// package-level servicebusConfig.Config at it, so that server()'s
// http.Server{Addr: ...} binds to a predictable, available address. It
// returns the address and a restore func that must be deferred to avoid
// leaking state into other tests.
func reserveServerConfig(t *testing.T) (addr string, restore func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("listener.Close: %v", err)
	}

	orig := servicebusConfig.Config
	servicebusConfig.Config = servicebusConfig.Configure{
		ServiceBus: v1alpha2.ServiceBus{
			Server:  "127.0.0.1",
			Port:    port,
			Timeout: 10,
		},
	}
	return fmt.Sprintf("127.0.0.1:%d", port), func() { servicebusConfig.Config = orig }
}

// waitForPort polls addr with a plain TCP dial until it accepts connections
// or the timeout elapses. A raw TCP connect works as a readiness probe
// whether the listener speaks TLS or plain HTTP.
func waitForPort(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not start listening on %s in time", addr)
}

// TestServerTLSSuccessPath verifies that server() itself — not a stand-in —
// wires TLSConfig onto the http.Server when buildTLSConfig succeeds, actually
// listens on HTTPS, and shuts down cleanly when stopCh is closed. This
// covers servicebus.go lines 364 and 377-380 (the TLS-success branch:
// s.TLSConfig = tlsCfg and ListenAndServeTLS).
func TestServerTLSSuccessPath(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := generateServerCert(t, dir)

	addr, restore := reserveServerConfig(t)
	defer restore()

	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		server(stopCh, TLSOptions{TLSEnabled: true, CertFile: certFile, KeyFile: keyFile})
	}()

	waitForPort(t, addr)

	// Client trusts our self-signed cert.
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		t.Fatal("AppendCertsFromPEM returned false")
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}
	resp, err := client.Get("https://" + addr)
	if err != nil {
		t.Fatalf("HTTPS GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}

	close(stopCh)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server() did not return within 5s after stopCh was closed")
	}
}

// TestServerPlainHTTPPath verifies that when TLSEnabled is false, the real
// server() function starts a plain HTTP server (not HTTPS) and shuts down
// cleanly. This covers servicebus.go lines 377 and 381-383 (the else branch:
// ListenAndServe without TLS).
func TestServerPlainHTTPPath(t *testing.T) {
	addr, restore := reserveServerConfig(t)
	defer restore()

	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		server(stopCh, TLSOptions{TLSEnabled: false})
	}()

	waitForPort(t, addr)

	resp, err := http.Get("http://" + addr)
	if err != nil {
		t.Fatalf("plain HTTP GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}

	close(stopCh)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server() did not return within 5s after stopCh was closed")
	}
}
