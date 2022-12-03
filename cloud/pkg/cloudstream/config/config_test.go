/*
Copyright 2022 The KubeEdge Authors.

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

package config

import (
	"encoding/pem"
	"os"
	"reflect"
	"testing"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

var pemData = `
-----BEGIN CERTIFICATE-----
MA==
-----END CERTIFICATE-----
`

var certificate = &pem.Block{Type: "CERTIFICATE",
	Headers: map[string]string{},
	Bytes:   []uint8{0x30},
}

func TestInitConfigure(t *testing.T) {
	// set ca, cert and key
	rootCA, err := os.CreateTemp("", "rootCA.crt")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		rootCA.Close()
		os.Remove(rootCA.Name())
	}()
	if err := os.WriteFile(rootCA.Name(), []byte(pemData), 0666); err != nil {
		t.Error(err)
	}
	certFile, err := os.CreateTemp("", "server.crt")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		certFile.Close()
		os.Remove(certFile.Name())
	}()
	if err := os.WriteFile(certFile.Name(), []byte(pemData), 0666); err != nil {
		t.Error(err)
	}
	keyFile, err := os.CreateTemp("", "server.key")
	if err != nil {
		t.Error(err)
	}
	defer func() {
		keyFile.Close()
		os.Remove(keyFile.Name())
	}()
	if err := os.WriteFile(keyFile.Name(), []byte(pemData), 0666); err != nil {
		t.Error(err)
	}
	stream := &v1alpha1.CloudStream{
		TLSTunnelCAFile:         rootCA.Name(),
		TLSTunnelCertFile:       certFile.Name(),
		TLSTunnelPrivateKeyFile: keyFile.Name(),
	}
	InitConfigure(stream)
	if !reflect.DeepEqual(Config.Ca, certificate.Bytes) {
		t.Errorf("InitConfigure(): Ca got %v, want %v", Config.Ca, certificate.Bytes)
	}
	if !reflect.DeepEqual(Config.Cert, certificate.Bytes) {
		t.Errorf("InitConfigure(): Cert got %v, want %v", Config.Cert, certificate.Bytes)
	}
	if !reflect.DeepEqual(Config.Key, certificate.Bytes) {
		t.Errorf("InitConfigure(): Key got %v, want %v", Config.Key, certificate.Bytes)
	}
}
