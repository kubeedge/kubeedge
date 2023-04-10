/*
Copyright 2023 The KubeEdge Authors.

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

package certificate

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"
	"time"

	certificates "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

const (
	pairNamePrefix = "metaserver"
	// CertificatesDir defines default certificate directory
	CertificatesDir = "/var/lib/pki/metaserver"
)

type ServerCertificateManager struct {
	certificate.Manager
}

// NewServerCertificateManager creates a certificate manager for the
// metaserver when retrieving a server certificate or returns an error.
func NewServerCertificateManager(
	kubeClient clientset.Interface,
	nodeName types.NodeName,
	ips []net.IP,
	certDirectory string) (*ServerCertificateManager, error) {
	var clientsetFn certificate.ClientsetFunc
	if kubeClient != nil {
		clientsetFn = func(current *tls.Certificate) (clientset.Interface, error) {
			return kubeClient, nil
		}
	}

	certificateStore, err := certificate.NewFileStore(
		pairNamePrefix,
		certDirectory,
		certDirectory,
		"",
		"")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server certificate store: %v", err)
	}

	getTemplate := func() *x509.CertificateRequest {
		// don't return a template if we have no addresses to request for
		if len(ips) == 0 {
			return nil
		}
		return &x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName:   fmt.Sprintf("system:node:%s", nodeName),
				Organization: []string{"system:nodes"},
			},
			IPAddresses: ips,
		}
	}

	name := fmt.Sprintf("metaserver-csr-%s", nodeName)

	m, err := certificate.NewManager(&certificate.Config{
		Name:        name,
		ClientsetFn: clientsetFn,
		GetTemplate: getTemplate,
		SignerName:  certificates.KubeletServingSignerName,
		Usages: []certificates.KeyUsage{
			// https://tools.ietf.org/html/rfc5280#section-4.2.1.3
			//
			// Digital signature allows the certificate to be used to verify
			// digital signatures used during TLS negotiation.
			certificates.UsageDigitalSignature,
			// KeyEncipherment allows the cert/key pair to be used to encrypt
			// keys, including the symmetric keys negotiated during TLS setup
			// and used for data transfer.
			certificates.UsageKeyEncipherment,
			// ServerAuth allows the cert to be used by a TLS server to
			// authenticate itself to a TLS client.
			certificates.UsageServerAuth,
		},
		CertificateStore: certificateStore,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize server certificate manager: %v", err)
	}

	return &ServerCertificateManager{m}, nil
}

func (scm *ServerCertificateManager) ready() bool {
	if cert := scm.Current(); cert != nil {
		return true
	}
	return false
}

func (scm *ServerCertificateManager) WaitForCertReady() error {
	return wait.PollImmediate(5*time.Second, 4*time.Minute, func() (bool, error) {
		isReady := scm.ready()
		if isReady {
			return true, nil
		}
		return false, nil
	})
}

func (scm *ServerCertificateManager) WaitForCAReady() error {
	return wait.PollImmediate(5*time.Second, 4*time.Minute, func() (bool, error) {
		if !cloudconnection.IsConnected() {
			return false, nil
		}
		err := scm.getKubeAPIServerCA()
		if err == nil {
			return true, nil
		}
		return false, err
	})
}

func (scm *ServerCertificateManager) getKubeAPIServerCA() error {
	msg := message.BuildMsg(modules.MetaGroup, "", modules.MetaManagerModuleName, constants.K8sCAResource, model.QueryOperation, nil)
	resp, err := beehiveContext.SendSync(modules.EdgeHubModuleName, *msg, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("send sync message %s failed: %v", msg.GetResource(), err)
	}

	content, err := resp.GetContentData()
	if err != nil {
		return fmt.Errorf("parse message %s err: %v", msg.GetResource(), err)
	}

	if err = cert.WriteCert(fmt.Sprintf("%s/ca.crt", CertificatesDir), content); err != nil {
		return fmt.Errorf("failed to save the k8s CA certificate: %v", err)
	}

	return nil
}
