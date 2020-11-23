package certificate

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/certutil"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"
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

type CertManager struct {
	RotateCertificates bool
	NodeName           string
	CR                 *x509.CertificateRequest

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
func NewCertManager(edgehub v1alpha1.EdgeHub, nodename string) CertManager {
	certReq := &x509.CertificateRequest{
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"kubeEdge"},
			Locality:     []string{"Hangzhou"},
			Province:     []string{"Zhejiang"},
			CommonName:   "kubeedge.io",
		},
	}
	return CertManager{
		RotateCertificates: edgehub.RotateCertificates,
		NodeName:           nodename,
		token:              edgehub.Token,
		CR:                 certReq,
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
	_, err := cm.getCurrent()
	if err != nil {
		err = cm.applyCerts()
		if err != nil {
			klog.Fatalf("Error: %v", err)
		}
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
	tokenParts := strings.Split(cm.token, ".")
	if len(tokenParts) != 4 {
		return fmt.Errorf("token credentials are in the wrong format")
	}
	ok, hash, newHash := ValidateCACerts(cacert, tokenParts[0])
	if !ok {
		return fmt.Errorf("failed to validate CA certificate. tokenCAhash: %s, CAhash: %s", hash, newHash)
	}

	// save the ca.crt to file
	ca, err := x509.ParseCertificate(cacert)
	if err != nil {
		return fmt.Errorf("failed to parse the CA certificate, error: %v", err)
	}

	if err = certutil.WriteCert(cm.caFile, ca); err != nil {
		return fmt.Errorf("failed to save the CA certificate to file: %s, error: %v", cm.caFile, err)
	}

	// get the edge.crt
	caPem := pem.EncodeToMemory(&pem.Block{Bytes: cacert, Type: "CERTIFICATE"})
	pk, edgeCert, err := cm.GetEdgeCert(cm.certURL, caPem, tls.Certificate{}, strings.Join(tokenParts[1:], "."))
	if err != nil {
		return fmt.Errorf("failed to get edge certificate from the cloudcore, error: %v", err)
	}

	// save the edge.crt to the file
	cert, _ := x509.ParseCertificate(edgeCert)
	if err = certutil.WriteKeyAndCert(cm.keyFile, cm.certFile, pk, cert); err != nil {
		return fmt.Errorf("failed to save the edge key and certificate to file: %s, error: %v", cm.certFile, err)
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
			wait.PollInfinite(32*time.Second, cm.rotateCert)
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
	pk, edgecert, err := cm.GetEdgeCert(cm.certURL, caPem, *tlsCert, "")
	if err != nil {
		klog.Errorf("failed to get edge certificate from CloudCore:%v", err)
		return false, nil
	}
	// save the edge.crt to the file
	cert, err := x509.ParseCertificate(edgecert)
	if err != nil {
		klog.Errorf("failed to parse edge certificate:%v", err)
		return false, nil
	}
	if err = certutil.WriteKeyAndCert(cm.keyFile, cm.certFile, pk, cert); err != nil {
		klog.Errorf("failed to save edge key and certificate:%v", err)
		return false, nil
	}

	klog.Info("succeeded to rotate certificate")

	cm.Done <- struct{}{}

	return true, nil
}

// getCA returns the CA in pem format.
func (cm *CertManager) getCA() ([]byte, error) {
	return ioutil.ReadFile(cm.caFile)
}

// GetCACert gets the cloudcore CA certificate
func GetCACert(url string) ([]byte, error) {
	client := http.NewHTTPClient()
	req, err := http.BuildRequest("GET", url, nil, "", "")
	if err != nil {
		return nil, err
	}
	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	caCert, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return caCert, nil
}

// GetEdgeCert applies for the certificate from cloudcore
func (cm *CertManager) GetEdgeCert(url string, capem []byte, cert tls.Certificate, token string) (*ecdsa.PrivateKey, []byte, error) {
	pk, csr, err := cm.getCSR()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CSR: %v", err)
	}

	client, err := http.NewHTTPClientWithCA(capem, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("falied to create http client:%v", err)
	}

	req, err := http.BuildRequest("GET", url, bytes.NewReader(csr), token, cm.NodeName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate http request:%v", err)
	}

	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != 200 {
		return nil, nil, fmt.Errorf(string(content))
	}

	return pk, content, nil
}

func (cm *CertManager) getCSR() (*ecdsa.PrivateKey, []byte, error) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, cm.CR, pk)
	if err != nil {
		return nil, nil, err
	}

	return pk, csr, nil
}

// ValidateCACerts validates the CA certificate by hash code
func ValidateCACerts(cacerts []byte, hash string) (bool, string, string) {
	if len(cacerts) == 0 && hash == "" {
		return true, "", ""
	}

	newHash := hashCA(cacerts)
	return hash == newHash, hash, newHash
}

func hashCA(cacerts []byte) string {
	digest := sha256.Sum256(cacerts)
	return hex.EncodeToString(digest[:])
}
