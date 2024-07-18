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
	"encoding/pem"

	"k8s.io/client-go/util/keyutil"
)

type x509PrivateKeyWrap struct {
	der []byte
}

func (k x509PrivateKeyWrap) Signer() (crypto.Signer, error) {
	return x509.ParseECPrivateKey(k.der)
}

func (k x509PrivateKeyWrap) DER() []byte {
	return k.der
}

func (k x509PrivateKeyWrap) PEM() []byte {
	privateKeyPemBlock := &pem.Block{
		Type:  keyutil.ECPrivateKeyBlockType,
		Bytes: k.der,
	}
	return pem.EncodeToMemory(privateKeyPemBlock)
}
