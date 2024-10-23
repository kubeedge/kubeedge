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

package authorization

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	cloudhubmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
	"github.com/kubeedge/kubeedge/pkg/viaduct/pkg/conn"
)

func TestAdmitMessage(t *testing.T) {
	tests := []struct {
		name    string
		authz   cloudhubAuthorizer
		message beehivemodel.Message
		hubInfo cloudhubmodel.HubInfo
		allow   bool
	}{
		{
			name:  "authz is disabled",
			authz: cloudhubAuthorizer{enabled: false},
			allow: true,
		},
		{
			name:  "debug mode",
			authz: cloudhubAuthorizer{enabled: true, debug: true, authz: authorizerfactory.NewAlwaysDenyAuthorizer()},
			allow: true,
		},
		{
			name:    "authz reject",
			authz:   cloudhubAuthorizer{enabled: true, authz: authorizerfactory.NewAlwaysDenyAuthorizer()},
			message: beehivemodel.Message{Router: beehivemodel.MessageRoute{Operation: beehivemodel.QueryOperation, Resource: "ns/configmap/test"}},
			allow:   false,
		},
		{
			name:    "authz accept",
			authz:   cloudhubAuthorizer{enabled: true, authz: authorizerfactory.NewAlwaysAllowAuthorizer()},
			message: beehivemodel.Message{Router: beehivemodel.MessageRoute{Operation: beehivemodel.QueryOperation, Resource: "ns/configmap/test"}},
			allow:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.authz.AdmitMessage(tt.message, tt.hubInfo)
			if (err == nil) != tt.allow {
				t.Errorf("AdmitMessage(): expect allow=%+v, got err=%+v", tt.allow, err)
			}
		})
	}
}

func TestAuthenticateConnection(t *testing.T) {
	const testNodeName = "test"
	cert, err := makeTestCert(constants.NodesUserPrefix + testNodeName)
	if err != nil {
		t.Errorf("make test cert failed: %v", err)
	}

	headers := http.Header{}
	headers.Add("node_id", testNodeName)
	tests := []struct {
		name      string
		authz     cloudhubAuthorizer
		connState conn.ConnectionState
		allow     bool
	}{
		{
			name:  "authz is disabled",
			authz: cloudhubAuthorizer{enabled: false},
			allow: true,
		},
		{
			name:  "debug mode",
			authz: cloudhubAuthorizer{enabled: true, debug: true},
			allow: true,
		},
		{
			name:  "authz reject",
			authz: cloudhubAuthorizer{enabled: true},
			connState: conn.ConnectionState{
				Headers:          headers,
				PeerCertificates: []*x509.Certificate{},
			},
			allow: false,
		},
		{
			name:  "authz accept",
			authz: cloudhubAuthorizer{enabled: true},
			connState: conn.ConnectionState{
				Headers:          headers,
				PeerCertificates: []*x509.Certificate{cert},
			},
			allow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := conn.NewWSConn(&conn.ConnectionOptions{
				Base:  &websocket.Conn{},
				State: &tt.connState,
			})
			err := tt.authz.AuthenticateConnection(c)
			if (err == nil) != tt.allow {
				t.Errorf("AuthenticateConnection(): expect allow=%+v, got err=%+v", tt.allow, err)
			}
		})
	}
}

func makeTestCert(cn string) (*x509.Certificate, error) {
	rootCaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		Subject:               pkix.Name{CommonName: "root-ca"},
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootCaCertDer, err := x509.CreateCertificate(rand.Reader, template, template, &rootCaPrivateKey.PublicKey, rootCaPrivateKey)
	if err != nil {
		return nil, err
	}
	rootCaCert, err := x509.ParseCertificate(rootCaCertDer)
	if err != nil {
		return nil, err
	}

	serviceCertPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template = &x509.Certificate{
		Subject:      pkix.Name{CommonName: cn},
		SerialNumber: big.NewInt(2),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	serviceCertDer, err := x509.CreateCertificate(rand.Reader, template, rootCaCert, &serviceCertPrivateKey.PublicKey, rootCaPrivateKey)
	if err != nil {
		return nil, err
	}
	serviceCert, err := x509.ParseCertificate(serviceCertDer)
	if err != nil {
		return nil, err
	}

	hubconfig.Config.Ca = rootCaCertDer
	return serviceCert, nil
}
