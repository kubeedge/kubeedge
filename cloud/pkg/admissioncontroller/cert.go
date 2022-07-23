/*
CHANGELOG
KubeEdge Authors:
- This File is drived from github.com/karmada-io/karmada/pkg/karmadactl/cmdinit/cert/cert.go
*/

package admissioncontroller

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"path/filepath"
	"time"

	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/util/sets"
)

const (
	// certificateBlockType is a possible value for pem.Block.Type.
	certificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
	// Duration365d Certificate validity period
	Duration365d = time.Hour * 24 * 365
)

// NewPrivateKey returns a new private key.
var NewPrivateKey = GeneratePrivateKey

// GeneratePrivateKey Generate CA Private Key
func GeneratePrivateKey(keyType x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	if keyType == x509.ECDSA {
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// CertsConfig is a wrapper around certutil.Config extending it with PublicKeyAlgorithm.
type CertsConfig struct {
	certutil.Config
	NotAfter           *time.Time
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority(config *CertsConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating CA certificate %v", err)
	}

	cert, err := certutil.NewSelfSignedCACert(config.Config, key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create self-signed CA certificate %v", err)
	}

	return cert, key, nil
}

// NewCACertAndKey The public and private keys of the root certificate are returned
func NewCACertAndKey() (*x509.Certificate, *crypto.Signer, error) {
	certCfg := &CertsConfig{Config: certutil.Config{
		CommonName:   "ca",
		Organization: []string{"kubeedge"},
	},
	}
	caCert, caKey, err := NewCertificateAuthority(certCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while generating CA certificate and key: %v", err)
	}

	return caCert, &caKey, nil
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg *CertsConfig, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, isCA bool) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if isCA {
		keyUsage |= x509.KeyUsageCertSign
	}

	RemoveDuplicateAltNames(&cfg.AltNames)

	notAfter := time.Now().Add(Duration365d).UTC()
	if cfg.NotAfter != nil {
		notAfter = *cfg.NotAfter
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:              cfg.AltNames.DNSNames,
		IPAddresses:           cfg.AltNames.IPs,
		SerialNumber:          serial,
		NotBefore:             caCert.NotBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           cfg.Usages,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// RemoveDuplicateAltNames removes duplicate items in altNames.
func RemoveDuplicateAltNames(altNames *certutil.AltNames) {
	if altNames == nil {
		return
	}

	if altNames.DNSNames != nil {
		altNames.DNSNames = sets.NewString(altNames.DNSNames...).List()
	}

	ipsKeys := make(map[string]struct{})
	var ips []net.IP
	for _, one := range altNames.IPs {
		if _, ok := ipsKeys[one.String()]; !ok {
			ipsKeys[one.String()] = struct{}{}
			ips = append(ips, one)
		}
	}
	altNames.IPs = ips
}

// NewCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func NewCertAndKey(caCert *x509.Certificate, caKey crypto.Signer, config *CertsConfig) (*x509.Certificate, crypto.Signer, error) {
	if len(config.Usages) == 0 {
		return nil, nil, errors.New("must specify at least one ExtKeyUsage")
	}

	key, err := NewPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key %v", err)
	}

	cert, err := NewSignedCert(config, key, caCert, caKey, false)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign certificate. %v", err)
	}

	return cert, key, nil
}

// PathForKey returns the paths for the key given the path and basename.
func PathForKey(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.key", name))
}

// PathForCert returns the paths for the certificate given the path and basename.
func PathForCert(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.crt", name))
}

// WriteCert stores the given certificate at the given location
func WriteCert(pkiPath, name string, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	certificatePath := PathForCert(pkiPath, name)
	if err := certutil.WriteCert(certificatePath, EncodeCertPEM(cert)); err != nil {
		return fmt.Errorf("unable to write certificate to file %v", err)
	}

	return nil
}

// WriteKey stores the given key at the given location
func WriteKey(pkiPath, name string, key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	privateKeyPath := PathForKey(pkiPath, name)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key to PEM %v", err)
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return fmt.Errorf("unable to write private key to file %v", err)
	}

	return nil
}

// WriteCertAndKey Write certificate and key to file.
func WriteCertAndKey(pkiPath, pkiName string, ca *x509.Certificate, key *crypto.Signer) error {
	if err := WriteKey(pkiPath, pkiName, *key); err != nil {
		return err
	}

	if err := WriteCert(pkiPath, pkiName, ca); err != nil {
		return err
	}

	klog.Infof("Generate %s certificate success.", pkiName)
	return nil
}

// NewCertConfig create new CertConfig
func NewCertConfig(cn string, org []string, altNames certutil.AltNames, notAfter *time.Time) *CertsConfig {
	return &CertsConfig{
		Config: certutil.Config{
			CommonName:   cn,
			Organization: org,
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			AltNames:     altNames,
		},
		NotAfter: notAfter,
	}
}

// GenCerts Create CA certificate and sign kubeedge certificate.
func GenCerts(certCfg *CertsConfig) error {
	caCert, caKey, err := NewCACertAndKey()
	if err != nil {
		return err
	}

	if err = WriteCertAndKey(caPkiPath, caPkiName, caCert, caKey); err != nil {
		return err
	}

	kubeedgeCert, kubeedgeKey, err := NewCertAndKey(caCert, *caKey, certCfg)
	if err != nil {
		return err
	}

	if err = WriteCertAndKey(admissionPkiPath, admissionPkiName, kubeedgeCert, &kubeedgeKey); err != nil {
		return err
	}

	return nil
}
