/*
Copyright 2024 The Kubernetes Authors.

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
	"context"
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
	"k8s.io/klog/v2"

	beehivecontext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

const (
	pairNamePrefix = "metaserver"

	// CertificatesDir defines default certificate directory
	CertificatesDir = "/etc/kubeedge/pki/metaserver"
)

type ServerCertificateManager struct {
	certificate.Manager
}

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

	certificateStore, err := certificate.NewFileStore(pairNamePrefix, certDirectory, certDirectory, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server certificate store: %v", err)
	}

	getTemplate := func() *x509.CertificateRequest {
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

	m, err := certificate.NewManager(&certificate.Config{
		Name:        fmt.Sprintf("metaserver-csr-%s", nodeName),
		ClientsetFn: clientsetFn,
		GetTemplate: getTemplate,
		SignerName:  certificates.KubeletServingSignerName,
		Usages: []certificates.KeyUsage{
			certificates.UsageDigitalSignature,
			certificates.UsageKeyEncipherment,
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
	if currentCert := scm.Current(); currentCert != nil {
		return true
	}
	return false
}

func (scm *ServerCertificateManager) WaitForCertReady() error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 4*time.Minute, true, func(ctx context.Context) (bool, error) {
		if scm.ready() {
			return true, nil
		}
		return false, nil
	})
}

func (scm *ServerCertificateManager) WaitForCAReady() error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 4*time.Minute, true, func(ctx context.Context) (bool, error) {
		if !cloudconnection.IsConnected() {
			return false, nil
		}
		err := scm.getKubeAPIServerCA()
		if err != nil {
			klog.Errorf("get k8s CA failed, %v", err)
			return false, err
		}
		return true, nil
	})
}

func (scm *ServerCertificateManager) getKubeAPIServerCA() error {
	msg := message.BuildMsg(modules.MetaGroup, "", modules.MetaManagerModuleName, model.ResourceTypeK8sCA, model.QueryOperation, nil)
	resp, err := beehivecontext.SendSync(modules.EdgeHubModuleName, *msg, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("send sync message %s failed: %v", msg.GetResource(), err)
	}

	content, err := resp.GetContentData()
	if err != nil || content == nil {
		return fmt.Errorf("parse message %s failed, err: %v", msg.GetResource(), err)
	}

	if err = cert.WriteCert(fmt.Sprintf("%s/ca.crt", CertificatesDir), content); err != nil {
		return fmt.Errorf("failed to save k8s CA certificate, %v", err)
	}
	return nil
}
