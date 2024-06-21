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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

type x509CertsHandler struct{}

// check implements Handler
var _ Handler = (*x509CertsHandler)(nil)

func (h x509CertsHandler) GenPrivateKey() (PrivateKeyWrap, error) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate private key, err: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to convert an EC private key to SEC 1, ASN.1 DER form, err: %v", err)
	}
	return &x509PrivateKeyWrap{der: keyDER}, nil
}

func (h x509CertsHandler) CreateCSR(sub pkix.Name, pkw PrivateKeyWrap, alt *certutil.AltNames) (*pem.Block, error) {
	tpl := x509.CertificateRequest{
		Subject: sub,
	}
	if alt != nil {
		tpl.DNSNames = alt.DNSNames
		tpl.IPAddresses = alt.IPs
	}
	pk, err := pkw.Signer()
	if err != nil {
		return nil, fmt.Errorf("faild to parse the private key der to Signer, err: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &tpl, pk)
	if err != nil {
		return nil, fmt.Errorf("faild to create x509 certificate request, err %v", err)
	}
	return &pem.Block{Type: certutil.CertificateRequestBlockType, Bytes: csrDER}, nil
}

func (h x509CertsHandler) SignCerts(opts SignCertsOptions) (*pem.Block, error) {
	pubkey := opts.publicKey
	if opts.csrDER != nil {
		csr, err := x509.ParseCertificateRequest(opts.csrDER)
		if err != nil {
			return nil, fmt.Errorf("failed to parse csr, err: %v", err)
		}
		opts.cfg.CommonName = csr.Subject.CommonName
		opts.cfg.Organization = csr.Subject.Organization
		opts.cfg.AltNames.DNSNames = csr.DNSNames
		opts.cfg.AltNames.IPs = csr.IPAddresses
		pubkey = csr.PublicKey
	}
	if len(opts.cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(opts.cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number, err: %v", err)
	}

	caKey, err := x509PrivateKeyWrap{der: opts.caKeyDER}.Signer()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA private key, err: %v", err)
	}

	ca, err := x509.ParseCertificate(opts.caDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA, err: %v", err)
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   opts.cfg.CommonName,
			Organization: opts.cfg.Organization,
		},
		DNSNames:     opts.cfg.AltNames.DNSNames,
		IPAddresses:  opts.cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    time.Now().UTC(),
		NotAfter:     time.Now().Add(opts.expiration),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  opts.cfg.Usages,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &certTmpl, ca, pubkey, caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate, err: %v", err)
	}

	return &pem.Block{Type: certutil.CertificateBlockType, Bytes: certDER}, nil
}
