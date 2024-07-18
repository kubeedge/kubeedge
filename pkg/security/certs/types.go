/*
Copyright 2024 The KubeEdge Authors.

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
	"time"

	certutil "k8s.io/client-go/util/cert"
)

type PrivateKeyWrap interface {
	Signer() (crypto.Signer, error)
	DER() []byte
	PEM() []byte
}

type CAHandler interface {
	// GenPrivateKey create a private key
	GenPrivateKey() (PrivateKeyWrap, error)

	// New creates CA certificate, returns a pem block.
	NewSelfSigned(key PrivateKeyWrap) (*pem.Block, error)
}

type Handler interface {
	// GenPrivateKey create a private key
	GenPrivateKey() (PrivateKeyWrap, error)

	// CreateCSR create a certificate request, returns a pem block.
	CreateCSR(sub pkix.Name, pkw PrivateKeyWrap, alt *certutil.AltNames) (*pem.Block, error)

	// SignCerts creates a certificate, returns a pem block.
	SignCerts(opts SignCertsOptions) (*pem.Block, error)
}

type SignCertsOptions struct {
	cfg        certutil.Config
	caDER      []byte
	caKeyDER   []byte
	csrDER     []byte
	publicKey  any
	expiration time.Duration
}

func SignCertsOptionsWithCA(cfg certutil.Config, caDER, caKeyDER []byte, publicKey any, expiration time.Duration) SignCertsOptions {
	return SignCertsOptions{
		cfg:        cfg,
		caDER:      caDER,
		caKeyDER:   caKeyDER,
		publicKey:  publicKey,
		expiration: expiration,
	}
}

func SignCertsOptionsWithCSR(csrDER, caDER, caKeyDER []byte, usages []x509.ExtKeyUsage, expiration time.Duration) SignCertsOptions {
	return SignCertsOptions{
		csrDER:   csrDER,
		caDER:    caDER,
		caKeyDER: caKeyDER,
		cfg: certutil.Config{
			Usages: usages,
		},
		expiration: expiration,
	}
}

func SignCertsOptionsWithK8sCSR(csrDER []byte, usages []x509.ExtKeyUsage, expiration time.Duration) SignCertsOptions {
	return SignCertsOptions{
		csrDER: csrDER,
		cfg: certutil.Config{
			Usages: usages,
		},
		expiration: expiration,
	}
}
