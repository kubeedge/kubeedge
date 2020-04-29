package httpserver

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math"
	"math/big"
	"time"

	certutil "k8s.io/client-go/util/cert"

	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

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

// NewPrivateKey creates an RSA private key
func NewPrivateKey() (crypto.Signer, error) {
	return rsa.GenerateKey(cryptorand.Reader, 2048)
}

// NewSelfSignedCACertDERBytes creates a CA certificate
func NewSelfSignedCACertDERBytes(key crypto.Signer) ([]byte, error) {
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1024),
		Subject: pkix.Name{
			CommonName: "KubeEdge",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365 * 100),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return caDERBytes, err
}

func NewCloudCoreCertDERandKey(cfg *certutil.Config) ([]byte, []byte, error) {
	serverKey, _ := NewPrivateKey()
	keyDER := x509.MarshalPKCS1PrivateKey(serverKey.(*rsa.PrivateKey))

	// get ca from config
	ca := hubconfig.Config.Ca
	caCert, _ := x509.ParseCertificate(ca)
	caKeyDER := hubconfig.Config.CaKey
	caKey, _ := x509.ParsePKCS1PrivateKey(caKeyDER)

	certDER, err := NewCertFromCa(cfg, caCert, serverKey, crypto.Signer(caKey))
	if err != nil {
		fmt.Printf("%v", err)
	}
	return certDER, keyDER, err
}

// NewCertFromCa creates a signed certificate using the given CA certificate and key
func NewCertFromCa(cfg *certutil.Config, caCert *x509.Certificate, clientKey crypto.PublicKey, caKey crypto.Signer) ([]byte, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		fmt.Println("must specify a CommonName")
		return nil, err
	}
	if len(cfg.Usages) == 0 {
		fmt.Println("must specify at least one ExtKeyUsage")
		return nil, err
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(time.Hour * 24 * 365 * 100),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, clientKey, caKey)
	if err != nil {
		return nil, err
	}
	return certDERBytes, err
}

func ParseCertDerToCertificate(certDer, keyDer []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := x509.ParseCertificate(certDer)
	if err != nil {
		fmt.Printf("%v", err)
	}
	key, err := x509.ParsePKCS1PrivateKey(keyDer)
	if err != nil {
		fmt.Printf("%v", err)
	}
	return cert, key, err
}
