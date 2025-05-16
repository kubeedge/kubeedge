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
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	certutil "k8s.io/client-go/util/cert"
)

func TestSignCertsOptionsWithCA(t *testing.T) {
	cfg := certutil.Config{
		CommonName:   "test-cn",
		Organization: []string{"test-org"},
		AltNames: certutil.AltNames{
			DNSNames: []string{"example.com"},
			IPs:      []net.IP{net.ParseIP("192.168.1.1")},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	caDER := []byte("ca-der-data")
	caKeyDER := []byte("ca-key-der-data")
	publicKey := "public-key"
	expiration := 24 * time.Hour

	opts := SignCertsOptionsWithCA(cfg, caDER, caKeyDER, publicKey, expiration)

	assert.Equal(t, cfg, opts.cfg)
	assert.Equal(t, caDER, opts.caDER)
	assert.Equal(t, caKeyDER, opts.caKeyDER)
	assert.Equal(t, publicKey, opts.publicKey)
	assert.Equal(t, expiration, opts.expiration)

	assert.Empty(t, opts.csrDER)
}

func TestSignCertsOptionsWithCSR(t *testing.T) {
	csrDER := []byte("csr-der-data")
	caDER := []byte("ca-der-data")
	caKeyDER := []byte("ca-key-der-data")
	usages := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	expiration := 48 * time.Hour

	opts := SignCertsOptionsWithCSR(csrDER, caDER, caKeyDER, usages, expiration)

	assert.Equal(t, csrDER, opts.csrDER)
	assert.Equal(t, caDER, opts.caDER)
	assert.Equal(t, caKeyDER, opts.caKeyDER)
	assert.Equal(t, expiration, opts.expiration)

	assert.Equal(t, usages, opts.cfg.Usages)
	assert.Empty(t, opts.cfg.CommonName)
	assert.Empty(t, opts.cfg.Organization)
	assert.Empty(t, opts.cfg.AltNames.DNSNames)
	assert.Empty(t, opts.cfg.AltNames.IPs)

	assert.Nil(t, opts.publicKey)
}

func TestSignCertsOptionsWithK8sCSR(t *testing.T) {
	csrDER := []byte("k8s-csr-der-data")
	usages := []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}
	expiration := 12 * time.Hour

	opts := SignCertsOptionsWithK8sCSR(csrDER, usages, expiration)

	assert.Equal(t, csrDER, opts.csrDER)
	assert.Equal(t, expiration, opts.expiration)

	assert.Equal(t, usages, opts.cfg.Usages)
	assert.Empty(t, opts.cfg.CommonName)
	assert.Empty(t, opts.cfg.Organization)
	assert.Empty(t, opts.cfg.AltNames.DNSNames)
	assert.Empty(t, opts.cfg.AltNames.IPs)

	assert.Empty(t, opts.caDER)
	assert.Empty(t, opts.caKeyDER)
	assert.Nil(t, opts.publicKey)
}

func TestPrivateKeyWrapInterface(t *testing.T) {
	mock := &mockPrivateKeyWrap{
		der: []byte("mock-der-data"),
		pem: []byte("mock-pem-data"),
	}

	var _ PrivateKeyWrap = mock

	der := mock.DER()
	pem := mock.PEM()

	assert.Equal(t, []byte("mock-der-data"), der)
	assert.Equal(t, []byte("mock-pem-data"), pem)

	patches := gomonkey.ApplyMethod(reflect.TypeOf(mock), "Signer",
		func(*mockPrivateKeyWrap) (crypto.Signer, error) {
			return nil, nil
		})
	defer patches.Reset()

	signer, err := mock.Signer()
	assert.Nil(t, signer)
	assert.Nil(t, err)
}

func TestHandlerInterface(t *testing.T) {
	mock := &mockHandler{}

	var _ Handler = mock

	patches := gomonkey.ApplyMethod(reflect.TypeOf(mock), "GenPrivateKey",
		func(*mockHandler) (PrivateKeyWrap, error) {
			return nil, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mock), "CreateCSR",
		func(*mockHandler, pkix.Name, PrivateKeyWrap, *certutil.AltNames) (*pem.Block, error) {
			return nil, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mock), "SignCerts",
		func(*mockHandler, SignCertsOptions) (*pem.Block, error) {
			return nil, nil
		})
	defer patches.Reset()

	key, err := mock.GenPrivateKey()
	assert.Nil(t, key)
	assert.Nil(t, err)

	csrBlock, err := mock.CreateCSR(pkix.Name{}, nil, nil)
	assert.Nil(t, csrBlock)
	assert.Nil(t, err)

	certBlock, err := mock.SignCerts(SignCertsOptions{})
	assert.Nil(t, certBlock)
	assert.Nil(t, err)
}

func TestCAHandlerInterface(t *testing.T) {
	mock := &mockCAHandler{}

	var _ CAHandler = mock

	patches := gomonkey.ApplyMethod(reflect.TypeOf(mock), "GenPrivateKey",
		func(*mockCAHandler) (PrivateKeyWrap, error) {
			return nil, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mock), "NewSelfSigned",
		func(*mockCAHandler, PrivateKeyWrap) (*pem.Block, error) {
			return nil, nil
		})
	defer patches.Reset()

	key, err := mock.GenPrivateKey()
	assert.Nil(t, key)
	assert.Nil(t, err)

	certBlock, err := mock.NewSelfSigned(nil)
	assert.Nil(t, certBlock)
	assert.Nil(t, err)
}

type mockPrivateKeyWrap struct {
	der       []byte
	pem       []byte
	signerErr error
}

func (m *mockPrivateKeyWrap) Signer() (crypto.Signer, error) {
	if m.signerErr != nil {
		return nil, m.signerErr
	}
	return nil, nil
}

func (m *mockPrivateKeyWrap) DER() []byte {
	return m.der
}

func (m *mockPrivateKeyWrap) PEM() []byte {
	return m.pem
}

type mockHandler struct{}

func (m *mockHandler) GenPrivateKey() (PrivateKeyWrap, error) {
	return nil, nil
}

func (m *mockHandler) CreateCSR(sub pkix.Name, pkw PrivateKeyWrap, alt *certutil.AltNames) (*pem.Block, error) {
	return nil, nil
}

func (m *mockHandler) SignCerts(opts SignCertsOptions) (*pem.Block, error) {
	return nil, nil
}

type mockCAHandler struct{}

func (m *mockCAHandler) GenPrivateKey() (PrivateKeyWrap, error) {
	return nil, nil
}

func (m *mockCAHandler) NewSelfSigned(key PrivateKeyWrap) (*pem.Block, error) {
	return nil, nil
}
