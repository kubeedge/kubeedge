package certutil

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

const (
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
)

func WriteKeyAndCert(keyFile string, certFile string, key crypto.Signer, cert *x509.Certificate) error {
	err := WriteKey(keyFile, key)
	if err != nil {
		return err
	}
	err = WriteCert(certFile, cert)
	if err != nil {
		return err
	}
	return nil
}

// WriteKey stores the given key at the given location
func WriteKey(pkiPath string, key crypto.Signer) error {
	if key == nil {
		return errors.New("private key cannot be nil when writing to file")
	}

	encoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return errors.Wrapf(err, "unable to marshal private key to PEM")
	}
	if err := keyutil.WriteKey(pkiPath, encoded); err != nil {
		return errors.Wrapf(err, "unable to write private key to file %s", pkiPath)
	}

	return nil
}

// WriteCert stores the given certificate at the given location
func WriteCert(certPath string, cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("certificate cannot be nil when writing to file")
	}

	if err := certutil.WriteCert(certPath, EncodeCertPEM(cert)); err != nil {
		return errors.Wrapf(err, "unable to write certificate to file %s", certPath)
	}

	return nil
}

// EncodeCertPEM returns PEM-encoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
