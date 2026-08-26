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

package admissioncontroller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeKeyPair writes a self signed key pair issued for commonName to certFile and keyFile.
func writeKeyPair(t *testing.T, certFile, keyFile, commonName string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(certFile,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0600))
	require.NoError(t, os.WriteFile(keyFile,
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600))
}

// servedCommonName returns the common name of the certificate getCertificate returns.
func servedCommonName(t *testing.T, getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)) string {
	t.Helper()

	cert, err := getCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	return leaf.Subject.CommonName
}

func TestNewCertReloader(t *testing.T) {
	assert := assert.New(t)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "tls.crt")
	keyFile := filepath.Join(dir, "tls.key")

	_, err := newCertReloader(certFile, keyFile)
	assert.Error(err)

	writeKeyPair(t, certFile, keyFile, "first")
	reloader, err := newCertReloader(certFile, keyFile)
	assert.NoError(err)
	assert.Equal("first", servedCommonName(t, reloader.GetCertificate))
}

func TestCertReloaderGetCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "tls.crt")
	keyFile := filepath.Join(dir, "tls.key")

	writeKeyPair(t, certFile, keyFile, "first")
	reloader, err := newCertReloader(certFile, keyFile)
	require.NoError(t, err)

	t.Run("rotated key pair is served", func(t *testing.T) {
		writeKeyPair(t, certFile, keyFile, "second")
		assert.Equal(t, "second", servedCommonName(t, reloader.GetCertificate))
	})

	t.Run("half rotated key pair keeps the previous one", func(t *testing.T) {
		// The certificate is replaced but the matching key is not written yet.
		writeKeyPair(t, filepath.Join(dir, "next.crt"), filepath.Join(dir, "next.key"), "third")
		next, err := os.ReadFile(filepath.Join(dir, "next.crt"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(certFile, next, 0600))

		assert.Equal(t, "second", servedCommonName(t, reloader.GetCertificate))
	})

	t.Run("unreadable key keeps the previous key pair", func(t *testing.T) {
		require.NoError(t, os.Remove(keyFile))
		assert.Equal(t, "second", servedCommonName(t, reloader.GetCertificate))
	})

	t.Run("unreadable certificate keeps the previous key pair", func(t *testing.T) {
		require.NoError(t, os.Remove(certFile))
		assert.Equal(t, "second", servedCommonName(t, reloader.GetCertificate))
	})
}
