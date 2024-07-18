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
	"fmt"
	"math/big"
	"time"

	certutil "k8s.io/client-go/util/cert"

	"github.com/kubeedge/kubeedge/common/constants"
)

type x509CAHandler struct{}

// check implements CAHandler
var _ CAHandler = (*x509CAHandler)(nil)

func (h x509CAHandler) GenPrivateKey() (PrivateKeyWrap, error) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate self signed CA private key, err: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to convert an EC private key to SEC 1, ASN.1 DER form, err: %v", err)
	}
	return &x509PrivateKeyWrap{der: keyDER}, nil
}

func (h x509CAHandler) NewSelfSigned(key PrivateKeyWrap) (*pem.Block, error) {
	const year100 = time.Hour * 24 * 364 * 100

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1024),
		Subject: pkix.Name{
			CommonName: constants.ProjectName,
		},
		NotBefore:             time.Now().UTC(),
		NotAfter:              time.Now().Add(year100),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	pk, err := key.Signer()
	if err != nil {
		return nil, fmt.Errorf("failed parse CA key der to private key, err: %v", err)
	}
	caDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, pk.Public(), pk)
	if err != nil {
		return nil, fmt.Errorf("failed to generate self signed CA cert, err: %v", err)
	}
	return &pem.Block{Type: certutil.CertificateBlockType, Bytes: caDER}, nil
}
