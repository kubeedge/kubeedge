package httpserver

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

const (
	// PrivateKeyBlockType is a possible value for pem.Block.Type.
	PrivateKeyBlockType = "PRIVATE KEY"
	// PublicKeyBlockType is a possible value for pem.Block.Type.
	PublicKeyBlockType = "PUBLIC KEY"
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	// RSAPrivateKeyBlockType is a possible value for pem.Block.Type.
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
	rsaKeySize             = 2048
)

//Save cert file to filePath
func WriteCertAndKey(pkiPath string, name string, cert *x509.Certificate, key crypto.Signer) error {
	if err := WriteKey(pkiPath, name, key); err != nil {
		return errors.Wrap(err, "couldn't write key")
	}

	return WriteCert(pkiPath, name, cert)
}

// WriteKey stores the given key at the given location
func WriteKey(pkiPath, name string, key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	privateKeyPath := pathForKey(pkiPath, name)
	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return errors.Wrapf(err, "unable to marshal private key to PEM")
	}
	if err := keyutil.WriteKey(privateKeyPath, encoded); err != nil {
		return errors.Wrapf(err, "unable to write private key to file %s", privateKeyPath)
	}

	return nil
}

// WriteCert stores the given certificate at the given location
func WriteCert(pkiPath, name string, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	certificatePath := pathForCert(pkiPath, name)
	if err := certutil.WriteCert(certificatePath, EncodeCertPEM(cert)); err != nil {
		return errors.Wrapf(err, "unable to write certificate to file %s", certificatePath)
	}

	return nil
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw, //mark
	}
	return pem.EncodeToMemory(&block)
}

func pathForKey(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.key", name))
}
func pathForCert(pkiPath, name string) string {
	return filepath.Join(pkiPath, fmt.Sprintf("%s.crt", name))
}
