package util

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateTestCertificateSuccess(t *testing.T) {
	tmp := t.TempDir()

	err := GenerateTestCertificate(tmp, "cert", "key")
	require.NoError(t, err)

	certPath := filepath.Join(tmp, "cert.crt")
	keyPath := filepath.Join(tmp, "key.key")

	_, err = os.Stat(certPath)
	require.NoError(t, err)

	_, err = os.Stat(keyPath)
	require.NoError(t, err)

	certInfo, err := os.Stat(certPath)
	require.NoError(t, err)
	require.Greater(t, certInfo.Size(), int64(0))

	keyInfo, err := os.Stat(keyPath)
	require.NoError(t, err)
	require.Greater(t, keyInfo.Size(), int64(0))
}

func TestGenerateTestCertificateValidPair(t *testing.T) {
	tmp := t.TempDir()

	err := GenerateTestCertificate(tmp, "cert", "key")
	require.NoError(t, err)

	certPEM, err := os.ReadFile(filepath.Join(tmp, "cert.crt"))
	require.NoError(t, err)
	keyPEM, err := os.ReadFile(filepath.Join(tmp, "key.key"))
	require.NoError(t, err)

	certBlock, _ := pem.Decode(certPEM)
	require.NotNil(t, certBlock)
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	require.NoError(t, err)

	keyBlock, _ := pem.Decode(keyPEM)
	require.NotNil(t, keyBlock)
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	require.NoError(t, err)

	require.True(t, key.PublicKey.Equal(cert.PublicKey), "cert public key must match generated private key")
}

func TestGenerateTestCertificateInvalidPath(t *testing.T) {
	tmp := t.TempDir()

	file := filepath.Join(tmp, "existing-file")
	err := os.WriteFile(file, []byte("data"), 0644)
	require.NoError(t, err)

	err = GenerateTestCertificate(file, "cert", "key")
	require.Error(t, err)
}

func TestCreatePEMFileSuccess(t *testing.T) {
	tmp := t.TempDir()

	path := filepath.Join(tmp, "test.pem")

	block := pem.Block{
		Type:  "TEST",
		Bytes: []byte("hello"),
	}

	err := createPEMfile(path, block)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}

func TestCreatePEMFileError(t *testing.T) {
	tmp := t.TempDir()

	dir := filepath.Join(tmp, "dir")
	require.NoError(t, os.Mkdir(dir, 0755))

	block := pem.Block{
		Type:  "TEST",
		Bytes: []byte("hello"),
	}

	err := createPEMfile(dir, block)
	require.Error(t, err)
}
