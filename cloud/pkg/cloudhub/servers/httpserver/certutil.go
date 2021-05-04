package httpserver

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	certutil "k8s.io/client-go/util/cert"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

const validalityPeriod time.Duration = 365 * 100

// NewCertificateAuthorityDer returns certDer and key
func NewCertificateAuthorityDer() ([]byte, crypto.Signer, error) {
	caKey, err := NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	certDER, err := NewSelfSignedCACertDERBytes(caKey)
	if err != nil {
		return nil, nil, err
	}
	return certDER, caKey, nil
}

// NewPrivateKey creates an ECDSA private key
func NewPrivateKey() (crypto.Signer, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// NewSelfSignedCACertDERBytes creates a CA certificate
func NewSelfSignedCACertDERBytes(key crypto.Signer) ([]byte, error) {
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1024),
		Subject: pkix.Name{
			CommonName: "KubeEdge",
		},
		NotBefore: time.Now().UTC(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365 * 100),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return caDERBytes, err
}

// NewCloudCoreCertDERandKey creates certificate and key for CloudCore
func NewCloudCoreCertDERandKey(cfg *certutil.Config) ([]byte, []byte, error) {
	serverKey, err := NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate a privateKey, err: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(serverKey.(*ecdsa.PrivateKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert an EC private key to SEC 1, ASN.1 DER form, err: %v", err)
	}

	// get ca from config
	ca := hubconfig.Config.Ca
	caCert, err := x509.ParseCertificate(ca)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse a caCert from the given ASN.1 DER data, err: %v", err)
	}

	caKeyDER := hubconfig.Config.CaKey
	caKey, err := x509.ParseECPrivateKey(caKeyDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse ECPrivateKey, err: %v", err)
	}

	certDER, err := NewCertFromCa(cfg, caCert, serverKey.Public(), caKey, validalityPeriod)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate a certificate using the given CA certificate and key, err: %v", err)
	}
	return certDER, keyDER, err
}

// NewCertFromCa creates a signed certificate using the given CA certificate and key
func NewCertFromCa(cfg *certutil.Config, caCert *x509.Certificate, serverKey crypto.PublicKey, caKey crypto.Signer, validalityPeriod time.Duration) ([]byte, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    time.Now().UTC(),
		NotAfter:     time.Now().Add(time.Hour * 24 * validalityPeriod),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, serverKey, caKey)
	if err != nil {
		return nil, err
	}
	return certDERBytes, err
}
