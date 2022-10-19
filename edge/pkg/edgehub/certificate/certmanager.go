package certificate

import (
	"bytes"
	"crypto"
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
	nethttp "net/http"
	"strings"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/certutil"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
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

	// Set to time.Now but can be stubbed out for testing
	now func() time.Time

	Done chan struct{}

	certificateRetriever Retriever
}

// Retriever defines an API to decouple the _retrieval_ of
// certificates from the actual usage. This allows plugging in different mechanisms
// how the certificates are retrieved and integrates with the certificate rotation
// mechanism already in place. The original implementation (retrieving edge certificates
// via token) has been moved (unchanged) into an realization of this interface.
type Retriever interface {
	RetrieveCertificate() (err error)
}

// CloudEdgeCertRetriever encapsulates the orignal functionality of retrieving the
// certificate from the cloudedge server
type CloudEdgeCertRetriever struct {
	caFile   string
	certFile string
	keyFile  string
	caURL    string
	certURL  string
	now      func() time.Time
	token    string
	CR       *x509.CertificateRequest
	NodeName string
}

func (cecr *CloudEdgeCertRetriever) RetrieveCertificate() (err error) {
	cacert, err := GetCACert(cecr.caURL)
	if err != nil {
		return fmt.Errorf("failed to get CA certificate, err: %v", err)
	}

	// validate the CA certificate by hashcode
	tokenParts := strings.Split(cecr.token, ".")
	if len(tokenParts) != 4 {
		return fmt.Errorf("token credentials are in the wrong format")
	}
	ok, hash, newHash := cecr.validateCACerts(cacert, tokenParts[0])
	if !ok {
		return fmt.Errorf("failed to validate CA certificate. tokenCAhash: %s, CAhash: %s", hash, newHash)
	}

	// save the ca.crt to file
	ca, err := x509.ParseCertificate(cacert)
	if err != nil {
		return fmt.Errorf("failed to parse the CA certificate, error: %v", err)
	}

	if err = certutil.WriteCert(cecr.caFile, ca); err != nil {
		return fmt.Errorf("failed to save the CA certificate to file: %s, error: %v", cecr.caFile, err)
	}

	// get the edge.crt
	caPem := pem.EncodeToMemory(&pem.Block{Bytes: cacert, Type: cert.CertificateBlockType})
	pk, edgeCert, err := cecr.GetEdgeCert(cecr.certURL, caPem, tls.Certificate{}, strings.Join(tokenParts[1:], "."))
	if err != nil {
		return fmt.Errorf("failed to get edge certificate from the cloudcore, error: %v", err)
	}

	// save the edge.crt to the file
	crt, _ := x509.ParseCertificate(edgeCert)
	if err = certutil.WriteKeyAndCert(cecr.keyFile, cecr.certFile, pk, crt); err != nil {
		return fmt.Errorf("failed to save the edge key and certificate to file: %s, error: %v", cecr.certFile, err)
	}

	return nil
}

// NewCertManager creates a CertManager for edge certificate management according to EdgeHub config
func NewCertManager(edgehub v1alpha2.EdgeHub, nodename string) CertManager {
	certReq := &x509.CertificateRequest{
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"kubeEdge"},
			Locality:     []string{"Hangzhou"},
			Province:     []string{"Zhejiang"},
			CommonName:   "kubeedge.io",
		},
	}
	var certRetriever Retriever
	if edgehub.Vault.Enable {
		var err error
		certRetriever, err = NewVaultRetriever(edgehub)
		if err != nil {
			// it is a fatal error, when the certificate retriever should be used,
			// but cannot be created due to a misconfiguration. Bailing...
			klog.Exitf("Failed to create cert retriever: %v", err)
		}
	} else {
		certRetriever = &CloudEdgeCertRetriever{
			caFile:   edgehub.TLSCAFile,
			certFile: edgehub.TLSCertFile,
			keyFile:  edgehub.TLSPrivateKeyFile,
			now:      time.Now,
			caURL:    edgehub.HTTPServer + constants.DefaultCAURL,
			certURL:  edgehub.HTTPServer + constants.DefaultCertURL,
			token:    edgehub.Token,
			CR:       certReq,
		}
	}
	return CertManager{
		RotateCertificates:   edgehub.RotateCertificates,
		NodeName:             nodename,
		caFile:               edgehub.TLSCAFile,
		certFile:             edgehub.TLSCertFile,
		keyFile:              edgehub.TLSPrivateKeyFile,
		now:                  time.Now,
		Done:                 make(chan struct{}),
		certificateRetriever: certRetriever,
	}
}

// Start starts the CertManager
func (cm *CertManager) Start() {
	_, err := cm.getCurrent()
	if err != nil {
		err = cm.certificateRetriever.RetrieveCertificate()
		if err != nil {
			klog.Exitf("Error: %v", err)
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
	if err != nil {
		return nil, fmt.Errorf("failed to read ca file: %v", err)
	}
	caPool := x509.NewCertPool()
	caCer, err := readCertificate(cm.caFile)
	if err != nil {
		return nil, err
	}
	caPool.AddCert(caCer)
	_, err = certs[0].Verify(x509.VerifyOptions{
		Roots:     caPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate certificates: %v", err)
	}
	return &cert, nil
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
	if err := cm.certificateRetriever.RetrieveCertificate(); err != nil {
		return false, err
	}
	klog.Info("succeeded to rotate certificate")

	cm.Done <- struct{}{}

	return true, nil
}

// getCA returns the CA in pem format.
func (cm *CertManager) getCA() ([]byte, error) {
	return ioutil.ReadFile(cm.caFile)
}

func (cm *CertManager) getCACertificate() (*x509.Certificate, error) {
	caPEM, err := cm.getCA()
	if err != nil {
		return nil, err
	}
	caDER, _ := pem.Decode(caPEM)

	cert, err := x509.ParseCertificate(caDER.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, err
}

// readCertificate reads a PEM encoded file an returns a x509 certificate
func readCertificate(filename string) (*x509.Certificate, error) {
	certPEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	certDER, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(certDER.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// readKey reads a PEM encoded private key and decodes it.
// elliptic curve resp. PKCS#1 keys are handled automatically
func readKey(filename string) (crypto.Signer, error) {
	var err error
	keyPEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	keyDER, _ := pem.Decode(keyPEM)
	var key crypto.Signer
	key, err = x509.ParseECPrivateKey(keyDER.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(keyDER.Bytes)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
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
	caCert, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return caCert, nil
}

// GetEdgeCert applies for the certificate from cloudcore
func (cecr *CloudEdgeCertRetriever) GetEdgeCert(url string, capem []byte, cert tls.Certificate, token string) (*ecdsa.PrivateKey, []byte, error) {
	pk, csr, err := cecr.getCSR()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create CSR: %v", err)
	}

	client, err := http.NewHTTPClientWithCA(capem, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create http client:%v", err)
	}

	req, err := http.BuildRequest(nethttp.MethodGet, url, bytes.NewReader(csr), token, cecr.NodeName)
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

func (cecr *CloudEdgeCertRetriever) getCSR() (*ecdsa.PrivateKey, []byte, error) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, cecr.CR, pk)
	if err != nil {
		return nil, nil, err
	}

	return pk, csr, nil
}

// ValidateCACerts validates the CA certificate by hash code
func (cecr *CloudEdgeCertRetriever) validateCACerts(cacerts []byte, hash string) (bool, string, string) {
	if len(cacerts) == 0 && hash == "" {
		return true, "", ""
	}

	newHash := cecr.hashCA(cacerts)
	return hash == newHash, hash, newHash
}

func (cecr *CloudEdgeCertRetriever) hashCA(cacerts []byte) string {
	digest := sha256.Sum256(cacerts)
	return hex.EncodeToString(digest[:])
}
