package certificate

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"time"

	certificates "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/certificate"
)

// NewMetaServerCertificateManager creates a certificate manager for the metaserver when
// retrieving a server certificate or returns an error.
func NewMetaServerCertificateManager(kubeClient clientset.Interface, nodeName types.NodeName, ips []net.IP, certDirectory string) (certificate.Manager, error) {
	var clientsetFn certificate.ClientsetFunc
	if kubeClient != nil {
		clientsetFn = func(current *tls.Certificate) (clientset.Interface, error) {
			return kubeClient, nil
		}
	}
	certificateStore, err := certificate.NewFileStore(
		"metaserver",
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

	m, err := certificate.NewManager(&certificate.Config{
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

	return m, nil
}

func ready(manager certificate.Manager) bool {
	if cert := manager.Current(); cert != nil {
		return true
	}
	return false
}

func WaitForCertReady(manager certificate.Manager) error {
	return wait.PollImmediate(5*time.Second, 4*time.Minute, func() (bool, error) {
		isReady := ready(manager)
		if isReady {
			return true, nil
		}
		return false, nil
	})
}
