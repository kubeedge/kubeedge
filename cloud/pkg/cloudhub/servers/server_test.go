package servers

import (
    "crypto/rand"
    "crypto/rsa"
    "crypto/x509"
    "crypto/x509/pkix"
    "math/big"
    "testing"
    "time"

    "k8s.io/klog/v2"
)

// helper to make a minimal self-signed cert and key pair for tests
func makeSelfSignedCertAndKey(t *testing.T) (caDER, certDER, keyDER []byte) {
    t.Helper()
    // Create CA key and cert
    caKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        t.Fatalf("failed to create CA key: %v", err)
    }
    caTemplate := &x509.Certificate{
        SerialNumber: big.NewInt(1),
        Subject:      pkixName("test-ca"),
        NotBefore:    time.Now().Add(-time.Hour),
        NotAfter:     time.Now().Add(24 * time.Hour),
        KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
        IsCA:         true,
        BasicConstraintsValid: true,
    }
    caDERBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
    if err != nil {
        t.Fatalf("failed to create CA cert: %v", err)
    }
    caCert, err := x509.ParseCertificate(caDERBytes)
    if err != nil {
        t.Fatalf("failed to parse CA cert: %v", err)
    }

    // Create leaf key and cert signed by CA
    leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        t.Fatalf("failed to create leaf key: %v", err)
    }
    leafTemplate := &x509.Certificate{
        SerialNumber: big.NewInt(2),
        Subject:      pkixName("test-leaf"),
        NotBefore:    time.Now().Add(-time.Hour),
        NotAfter:     time.Now().Add(24 * time.Hour),
        KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
        ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
        BasicConstraintsValid: true,
    }
    certDERBytes, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
    if err != nil {
        t.Fatalf("failed to create leaf cert: %v", err)
    }

    pkcs8, err := x509.MarshalPKCS8PrivateKey(leafKey)
    if err != nil {
        t.Fatalf("failed to marshal private key: %v", err)
    }

    return caDERBytes, certDERBytes, pkcs8
}

// pkixName builds a minimal pkix.Name with a CommonName
func pkixName(cn string) pkix.Name {
    return pkix.Name{CommonName: cn}
}

type exitPanic struct{ code int }

func (e exitPanic) Error() string { return "exit" }

func TestCreateTLSConfig_BadCA(t *testing.T) {
    _, certDER, keyDER := makeSelfSignedCertAndKey(t)

    orig := klog.OsExit
    defer func() { klog.OsExit = orig }()
    klog.OsExit = func(code int) { panic(exitPanic{code}) }

    defer func() {
        if r := recover(); r == nil {
            t.Fatalf("expected exit, but no exit occurred")
        }
    }()

    _ = createTLSConfig([]byte("not-a-ca"), certDER, keyDER)
}

func TestCreateTLSConfig_MismatchedKey(t *testing.T) {
    caDER, certDER, _ := makeSelfSignedCertAndKey(t)

    // generate an unrelated key to force mismatch
    otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        t.Fatalf("failed to create other key: %v", err)
    }
    otherKeyDER, err := x509.MarshalPKCS8PrivateKey(otherKey)
    if err != nil {
        t.Fatalf("failed to marshal other key: %v", err)
    }

    orig := klog.OsExit
    defer func() { klog.OsExit = orig }()
    klog.OsExit = func(code int) { panic(exitPanic{code}) }

    defer func() {
        if r := recover(); r == nil {
            t.Fatalf("expected exit for mismatched key, but no exit occurred")
        }
    }()

    _ = createTLSConfig(caDER, certDER, otherKeyDER)
}


