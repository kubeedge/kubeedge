package certificate

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/cert"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
	"github.com/kubeedge/kubeedge/pkg/security/token"
)

// jitteryDuration uses some jitter to set the rotation threshold so each node
// will rotate at approximately 70-90% of the total lifetime of the
// certificate.  With jitter, if a number of nodes are added to a cluster at
// approximately the same time (such as cluster creation time), they won't all
// try to rotate certificates at the same time for the rest of the life of the
// cluster.
//
// This function is represented as a variable to allow replacement during testing.
var jitteryDuration = func(totalDuration float64) time.Duration {
	return wait.Jitter(time.Duration(totalDuration), 0.2) - time.Duration(totalDuration*0.3)
}

var CleanupTokenChan = make(chan struct{}, 1)

type CertManager struct {
	RotateCertificates bool
	NodeName           string

	caFile   string
	certFile string
	keyFile  string

	token string
	// Set to time.Now but can be stubbed out for testing
	now func() time.Time

	caURL   string
	certURL string
	Done    chan struct{}
}

// NewCertManager creates a CertManager for edge certificate management according to EdgeHub config
func NewCertManager(edgehub v1alpha2.EdgeHub, nodename string) CertManager {
	return CertManager{
		RotateCertificates: edgehub.RotateCertificates,
		NodeName:           nodename,
		token:              edgehub.Token,
		caFile:             edgehub.TLSCAFile,
		certFile:           edgehub.TLSCertFile,
		keyFile:            edgehub.TLSPrivateKeyFile,
		now:                time.Now,
		caURL:              edgehub.HTTPServer + constants.DefaultCAURL,
		certURL:            edgehub.HTTPServer + constants.DefaultCertURL,
		Done:               make(chan struct{}),
	}
}

// Start starts the CertManager
func (cm *CertManager) Start() {
	if _, err := cm.getCurrent(); err != nil {
		klog.Infof("unable to get the current edge certs, reason: %v", err)
		if err = cm.applyCerts(); err != nil {
			klog.Exitf("failed to apply the edge certs, err: %v", err)
		}
		// inform to cleanup token in configuration edgecore.yaml
		CleanupTokenChan <- struct{}{}
	}
	if cm.RotateCertificates {
		cm.rotate()
	}
}

// getCurrent returns current edge certificate
func (cm *CertManager) getCurrent() (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(cm.certFile, cm.keyFile)
	if err != nil {
		return nil, err
	}
	certs, err := x509.ParseCertificates(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate data: %v", err)
	}
	cert.Leaf = certs[0]
	return &cert, nil
}

// applyCerts realizes the certificate application by token
func (cm *CertManager) applyCerts() error {
	cacert, err := GetCACert(cm.caURL)
	if err != nil {
		return fmt.Errorf("failed to get CA certificate, err: %v", err)
	}

	// validate the CA certificate by hashcode
	realToken, err := token.VerifyCAAndGetRealToken(cm.token, cacert)
	if err != nil {
		return err
	}

	// save the ca.crt to file
	caPem, err := certs.WriteDERToPEMFile(cm.caFile, cert.CertificateBlockType, cacert)
	if err != nil {
		return fmt.Errorf("failed to save the CA certificate to file: %s, error: %v", cm.caFile, err)
	}
	certDER, keyDER, err := cm.GetEdgeCert(cm.certURL, pem.EncodeToMemory(caPem), tls.Certificate{}, realToken)
	if err != nil {
		return fmt.Errorf("failed to get edge certificate from the cloudcore, error: %v", err)
	}
	// save the edge.crt to the file
	if _, err := certs.WriteDERToPEMFile(cm.certFile,
		certutil.CertificateBlockType, certDER); err != nil {
		return fmt.Errorf("failed to save the certificate file %s, err: %v", cm.certFile, err)
	}
	if _, err := certs.WriteDERToPEMFile(cm.keyFile,
		keyutil.ECPrivateKeyBlockType, keyDER); err != nil {
		return fmt.Errorf("failed to save the certificate key file %s, err: %v", cm.keyFile, err)
	}
	return nil
}

// rotate starts edge certificate rotation process
func (cm *CertManager) rotate() {
	klog.Infof("Certificate rotation is enabled.")
	go wait.Forever(func() {
		deadline, err := cm.nextRotationDeadline()
		if err != nil {
			klog.Errorf("failed to get next rotation deadline:%v", err)
		}
		if sleepInterval := deadline.Sub(cm.now()); sleepInterval > 0 {
			klog.V(2).Infof("Waiting %v for next certificate rotation", sleepInterval)

			timer := time.NewTimer(sleepInterval)
			defer timer.Stop()

			<-timer.C // unblock when deadline expires
		}

		backoff := wait.Backoff{
			Duration: 2 * time.Second,
			Factor:   2,
			Jitter:   0.1,
			Steps:    5,
		}
		if err := wait.ExponentialBackoff(backoff, cm.rotateCert); err != nil {
			utilruntime.HandleError(fmt.Errorf("reached backoff limit, still unable to rotate certs: %v", err))
			if err := wait.PollInfinite(32*time.Second, cm.rotateCert); err != nil {
				// TODO: handle error
				klog.Error(err)
			}
		}
	}, time.Second)
}

// nextRotationDeadline returns the rotation deadline. It is different in every rotation.
func (cm *CertManager) nextRotationDeadline() (time.Time, error) {
	cert, err := cm.getCurrent()
	if err != nil {
		return time.Time{}, fmt.Errorf("faild to get current certificate")
	}
	notAfter := cert.Leaf.NotAfter
	totalDuration := float64(notAfter.Sub(cert.Leaf.NotBefore))
	deadline := cert.Leaf.NotBefore.Add(jitteryDuration(totalDuration))
	klog.V(2).Infof("Certificate expiration is %v, rotation deadline is %v", notAfter, deadline)

	return deadline, nil
}

// rotateCert realizes the specific process of edge certificate rotation.
func (cm *CertManager) rotateCert() (bool, error) {
	klog.Infof("Rotating certificates")

	tlsCert, err := cm.getCurrent()
	if err != nil {
		klog.Errorf("failed to get current certificate:%v", err)
		return false, nil
	}
	caPem, err := cm.getCA()
	if err != nil {
		klog.Errorf("failed to get CA certificate locally:%v", err)
		return false, nil
	}
	certDER, keyDER, err := cm.GetEdgeCert(cm.certURL, caPem, *tlsCert, "")
	if err != nil {
		klog.Errorf("failed to get edge certificate from CloudCore:%v", err)
		return false, nil
	}
	if _, err := certs.WriteDERToPEMFile(cm.certFile,
		certutil.CertificateBlockType, certDER); err != nil {
		klog.Errorf("failed to save the certificate file %s, err: %v", cm.certFile, err)
		return false, nil
	}
	if _, err := certs.WriteDERToPEMFile(cm.keyFile,
		keyutil.ECPrivateKeyBlockType, keyDER); err != nil {
		klog.Errorf("failed to save the certificate key file %s, err: %v", cm.keyFile, err)
		return false, nil
	}

	klog.Info("succeeded to rotate certificate")

	cm.Done <- struct{}{}

	return true, nil
}

// getCA returns the CA in pem format.
func (cm *CertManager) getCA() ([]byte, error) {
	return os.ReadFile(cm.caFile)
}

// GetCACert gets the cloudcore CA certificate
func GetCACert(url string) ([]byte, error) {
	client := http.NewHTTPClient()
	req, err := http.BuildRequest(nethttp.MethodGet, url, nil, "", "")
	if err != nil {
		return nil, err
	}
	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	caCert, err := io.ReadAll(io.LimitReader(res.Body, constants.MaxRespBodyLength))
	if err != nil {
		return nil, err
	}

	return caCert, nil
}

// GetEdgeCert applies for the certificate from cloudcore
func (cm *CertManager) GetEdgeCert(url string, capem []byte, tlscert tls.Certificate, token string,
) ([]byte, []byte, error) {
	h := certs.GetHandler(certs.HandlerTypeX509)
	pkw, err := h.GenPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate a private key of edge cert, err: %v", err)
	}
	csrPem, err := h.CreateCSR(pkix.Name{
		Country:      []string{"CN"},
		Organization: []string{"system:nodes"},
		Locality:     []string{"Hangzhou"},
		Province:     []string{"Zhejiang"},
		CommonName:   fmt.Sprintf("system:node:%s", cm.NodeName),
	}, pkw, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a csr of edge cert, err %v", err)
	}

	client, err := http.NewHTTPClientWithCA(capem, tlscert)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a http client, err: %v", err)
	}

	req, err := http.BuildRequest(nethttp.MethodGet, url, bytes.NewReader(csrPem.Bytes), token, cm.NodeName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate a http request, err: %v", err)
	}

	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to request the cloudcore server, err: %v", err)
	}
	defer res.Body.Close()

	content, err := io.ReadAll(io.LimitReader(res.Body, constants.MaxRespBodyLength))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body, err: %v", err)
	}
	if res.StatusCode != nethttp.StatusOK {
		return nil, nil, fmt.Errorf("failed to call http, code: %d, message: %s", res.StatusCode, string(content))
	}
	return content, pkw.DER(), nil
}
