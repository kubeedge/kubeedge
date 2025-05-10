/*
Copyright 2025 The KubeEdge Authors.

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

package certs

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	certutil "k8s.io/client-go/util/cert"
)

type MockPrivateKeyWrap struct {
	DERValue   []byte
	PEMValue   []byte
	SignerFunc func() (crypto.Signer, error)
	SignerErr  error
}

func (m *MockPrivateKeyWrap) Signer() (crypto.Signer, error) {
	if m.SignerErr != nil {
		return nil, m.SignerErr
	}
	if m.SignerFunc != nil {
		signer, err := m.SignerFunc()
		if err != nil {
			return nil, err
		}
		return signer, nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (m *MockPrivateKeyWrap) DER() []byte {
	if m.DERValue != nil {
		return m.DERValue
	}
	return []byte("mock-der-value")
}

func (m *MockPrivateKeyWrap) PEM() []byte {
	if m.PEMValue != nil {
		return m.PEMValue
	}
	return []byte("mock-pem-value")
}

func TestGenPrivateKey(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		handler := x509CertsHandler{}
		key, err := handler.GenPrivateKey()

		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.NotNil(t, key.DER())
		assert.NotNil(t, key.PEM())

		signer, err := key.Signer()
		assert.NoError(t, err)
		assert.NotNil(t, signer)
	})

	t.Run("Error generating key", func(t *testing.T) {
		handler := x509CertsHandler{}
		expectedErr := errors.New("generate key error")

		patch := gomonkey.ApplyFunc(ecdsa.GenerateKey,
			func(c elliptic.Curve, random io.Reader) (*ecdsa.PrivateKey, error) {
				return nil, expectedErr
			})
		defer patch.Reset()

		key, err := handler.GenPrivateKey()
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("Error marshaling key", func(t *testing.T) {
		handler := x509CertsHandler{}
		expectedErr := errors.New("marshal key error")

		realKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		patch1 := gomonkey.ApplyFunc(ecdsa.GenerateKey,
			func(c elliptic.Curve, random io.Reader) (*ecdsa.PrivateKey, error) {
				return realKey, nil
			})
		defer patch1.Reset()

		patch2 := gomonkey.ApplyFunc(x509.MarshalECPrivateKey,
			func(key *ecdsa.PrivateKey) ([]byte, error) {
				return nil, expectedErr
			})
		defer patch2.Reset()

		key, err := handler.GenPrivateKey()
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestCreateCSR(t *testing.T) {
	handler := x509CertsHandler{}

	t.Run("Success without altnames", func(t *testing.T) {
		key, err := handler.GenPrivateKey()
		assert.NoError(t, err)

		subject := pkix.Name{
			CommonName:   "test-cn",
			Organization: []string{"test-org"},
		}

		csrPEM, err := handler.CreateCSR(subject, key, nil)
		assert.NoError(t, err)
		assert.NotNil(t, csrPEM)
		assert.Equal(t, csrPEM.Type, certutil.CertificateRequestBlockType)

		csr, err := x509.ParseCertificateRequest(csrPEM.Bytes)
		assert.NoError(t, err)
		assert.Equal(t, subject.CommonName, csr.Subject.CommonName)
		assert.Equal(t, subject.Organization, csr.Subject.Organization)
	})

	t.Run("Error from Signer", func(t *testing.T) {
		expectedErr := errors.New("signer error")

		mockKey := &MockPrivateKeyWrap{
			SignerErr: expectedErr,
		}

		subject := pkix.Name{
			CommonName:   "test-cn",
			Organization: []string{"test-org"},
		}

		csrPEM, err := handler.CreateCSR(subject, mockKey, nil)
		assert.Error(t, err)
		assert.Nil(t, csrPEM)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("Error creating CSR", func(t *testing.T) {
		key, err := handler.GenPrivateKey()
		assert.NoError(t, err)

		subject := pkix.Name{
			CommonName:   "test-cn",
			Organization: []string{"test-org"},
		}

		expectedErr := errors.New("create CSR error")

		patch := gomonkey.ApplyFunc(x509.CreateCertificateRequest,
			func(random io.Reader, template *x509.CertificateRequest, priv any) ([]byte, error) {
				return nil, expectedErr
			})
		defer patch.Reset()

		csrPEM, err := handler.CreateCSR(subject, key, nil)
		assert.Error(t, err)
		assert.Nil(t, csrPEM)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestSignCerts(t *testing.T) {
	handler := x509CertsHandler{}

	setupCA := func() (caDER, caKeyDER []byte, caKey *ecdsa.PrivateKey, err error) {
		caKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, nil, err
		}

		caKeyDER, err = x509.MarshalECPrivateKey(caKey)
		if err != nil {
			return nil, nil, nil, err
		}

		caTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject: pkix.Name{
				CommonName:   "Test CA",
				Organization: []string{"Test Org"},
			},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().Add(time.Hour),
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
			BasicConstraintsValid: true,
			IsCA:                  true,
		}

		caDER, err = x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
		if err != nil {
			return nil, nil, nil, err
		}

		return caDER, caKeyDER, caKey, nil
	}

	t.Run("Error parsing CSR", func(t *testing.T) {
		caDER, caKeyDER, _, err := setupCA()
		assert.NoError(t, err)

		invalidCSR := []byte("invalid-csr")
		opts := SignCertsOptionsWithCSR(
			invalidCSR,
			caDER,
			caKeyDER,
			[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			time.Hour*24,
		)

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), "failed to parse csr")
	})

	t.Run("Error empty CommonName", func(t *testing.T) {
		caDER, caKeyDER, _, err := setupCA()
		assert.NoError(t, err)

		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		cfg := certutil.Config{
			Organization: []string{"Test Org"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		opts := SignCertsOptionsWithCA(
			cfg,
			caDER,
			caKeyDER,
			&clientKey.PublicKey,
			time.Hour*24,
		)

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), "must specify a CommonName")
	})

	t.Run("Error empty Usages", func(t *testing.T) {
		caDER, caKeyDER, _, err := setupCA()
		assert.NoError(t, err)

		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		cfg := certutil.Config{
			CommonName:   "Test Client",
			Organization: []string{"Test Org"},
		}

		opts := SignCertsOptionsWithCA(
			cfg,
			caDER,
			caKeyDER,
			&clientKey.PublicKey,
			time.Hour*24,
		)

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), "must specify at least one ExtKeyUsage")
	})

	t.Run("Error generating serial number", func(t *testing.T) {
		caDER, caKeyDER, _, err := setupCA()
		assert.NoError(t, err)

		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		cfg := certutil.Config{
			CommonName:   "Test Client",
			Organization: []string{"Test Org"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		opts := SignCertsOptionsWithCA(
			cfg,
			caDER,
			caKeyDER,
			&clientKey.PublicKey,
			time.Hour*24,
		)

		expectedErr := errors.New("random int error")

		patch := gomonkey.ApplyFunc(rand.Int,
			func(random io.Reader, max *big.Int) (*big.Int, error) {
				return nil, expectedErr
			})
		defer patch.Reset()

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})

	t.Run("Error parsing CA private key", func(t *testing.T) {
		caDER, _, _, err := setupCA()
		assert.NoError(t, err)

		invalidCAKey := []byte("invalid-ca-key")

		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		cfg := certutil.Config{
			CommonName:   "Test Client",
			Organization: []string{"Test Org"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		opts := SignCertsOptionsWithCA(
			cfg,
			caDER,
			invalidCAKey,
			&clientKey.PublicKey,
			time.Hour*24,
		)

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), "failed to parse CA private key")
	})

	t.Run("Error parsing CA certificate", func(t *testing.T) {
		_, caKeyDER, _, err := setupCA()
		assert.NoError(t, err)

		invalidCADER := []byte("invalid-ca-der")

		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		cfg := certutil.Config{
			CommonName:   "Test Client",
			Organization: []string{"Test Org"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		opts := SignCertsOptionsWithCA(
			cfg,
			invalidCADER,
			caKeyDER,
			&clientKey.PublicKey,
			time.Hour*24,
		)

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), "failed to parse CA")
	})

	t.Run("Error creating certificate", func(t *testing.T) {
		caDER, caKeyDER, _, err := setupCA()
		assert.NoError(t, err)

		clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		assert.NoError(t, err)

		cfg := certutil.Config{
			CommonName:   "Test Client",
			Organization: []string{"Test Org"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		opts := SignCertsOptionsWithCA(
			cfg,
			caDER,
			caKeyDER,
			&clientKey.PublicKey,
			time.Hour*24,
		)

		expectedErr := errors.New("create certificate error")

		patch := gomonkey.ApplyFunc(x509.CreateCertificate,
			func(random io.Reader, template, parent *x509.Certificate, pub, priv any) ([]byte, error) {
				return nil, expectedErr
			})
		defer patch.Reset()

		certPEM, err := handler.SignCerts(opts)
		assert.Error(t, err)
		assert.Nil(t, certPEM)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}
