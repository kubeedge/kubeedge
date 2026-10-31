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
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pool *x509.CertPool
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-48 * time.Hour),
		NotAfter:              time.Now().Add(48 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &testCA{cert: cert, key: key, pool: pool}
}

// writeKeyPair issues a certificate valid in [notBefore, notAfter] and writes
// it with its key to certFile and keyFile in PEM format.
func (ca *testCA) writeKeyPair(t *testing.T, serial int64, notBefore, notAfter time.Time, certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "system:node:test-node"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
}

func leafSerial(t *testing.T, cert *tls.Certificate) int64 {
	t.Helper()
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	return leaf.SerialNumber.Int64()
}

func TestNewCertReloaderMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := newCertReloader(filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")); err == nil {
		t.Fatal("expected an error for missing key pair files")
	}
}

func TestCertReloaderPicksUpRotatedCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	ca := newTestCA(t)
	now := time.Now()

	ca.writeKeyPair(t, 100, now.Add(-time.Hour), now.Add(time.Hour), certFile, keyFile)
	r, err := newCertReloader(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := r.GetClientCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := leafSerial(t, cert); got != 100 {
		t.Fatalf("expected serial 100, got %d", got)
	}

	ca.writeKeyPair(t, 200, now.Add(-time.Hour), now.Add(time.Hour), certFile, keyFile)
	cert, err = r.GetClientCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := leafSerial(t, cert); got != 200 {
		t.Fatalf("expected rotated serial 200, got %d", got)
	}
}

func TestCertReloaderFallsBackOnInvalidPair(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	ca := newTestCA(t)
	now := time.Now()

	ca.writeKeyPair(t, 100, now.Add(-time.Hour), now.Add(time.Hour), certFile, keyFile)
	r, err := newCertReloader(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a rotation caught between writing the new certificate and the
	// new key: the certificate on disk no longer matches the key.
	otherKeyFile := filepath.Join(dir, "other.key")
	ca.writeKeyPair(t, 200, now.Add(-time.Hour), now.Add(time.Hour), certFile, otherKeyFile)

	cert, err := r.GetClientCertificate(nil)
	if err != nil {
		t.Fatalf("expected fallback to the previous certificate, got error: %v", err)
	}
	if got := leafSerial(t, cert); got != 100 {
		t.Fatalf("expected previous serial 100, got %d", got)
	}
}

// TestTunnelReconnectAfterCertRotation verifies that once the startup
// certificate has expired and been rotated on disk, a reconnect with the same
// tls.Config presents the rotated certificate and is accepted by a server that
// verifies client certificates, as CloudStream's tunnel server does.
func TestTunnelReconnectAfterCertRotation(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key")
	ca := newTestCA(t)
	now := time.Now()

	serverCertFile, serverKeyFile := filepath.Join(dir, "cloud.crt"), filepath.Join(dir, "cloud.key")
	ca.writeKeyPair(t, 1, now.Add(-time.Hour), now.Add(time.Hour), serverCertFile, serverKeyFile)
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    ca.pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	handshakes := make(chan error)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			handshakes <- conn.(*tls.Conn).Handshake()
			conn.Close()
		}
	}()

	connect := func(cfg *tls.Config) error {
		conn, err := tls.Dial("tcp", ln.Addr().String(), cfg)
		if err == nil {
			// With TLS 1.3 the server verifies the client certificate after the
			// client handshake returns, so the server's result is authoritative.
			conn.Close()
		}
		return <-handshakes
	}

	// The certificate loaded at startup has since expired.
	ca.writeKeyPair(t, 100, now.Add(-2*time.Hour), now.Add(-time.Hour), certFile, keyFile)
	r, err := newCertReloader(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify:   true,
		GetClientCertificate: r.GetClientCertificate,
	}
	if err := connect(tlsConfig); err == nil {
		t.Fatal("expected the server to reject the expired client certificate")
	}

	// EdgeHub rotates the certificate on disk.
	ca.writeKeyPair(t, 200, now.Add(-time.Hour), now.Add(time.Hour), certFile, keyFile)
	if err := connect(tlsConfig); err != nil {
		t.Fatalf("expected reconnect with the rotated certificate to succeed, got: %v", err)
	}
}
