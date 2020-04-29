package certutil

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/config"
	"io/ioutil"
	"os"
)

const privateKeyBits = 2048

// GetCACert gets the cloudcore CA certificate
func GetCACert(url string) ([]byte, error) {
	client := http.NewHTTPClient()
	req, _ := http.BuildRequest("GET", url, nil, "")
	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	cacert, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return cacert, nil
}

func getCSR() ([]byte, error) {
	pk, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// save the private key
	if err := WriteKey(config.Config.TLSPrivateKeyFile, pk); err != nil {
		return nil, err
	}

	certReq := &x509.CertificateRequest{
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Organization:       []string{"kubeEdge"},
			OrganizationalUnit: []string{},
			Locality:           []string{"Hangzhou"},
			Province:           []string{"Zhejiang"},
			CommonName:         "kubeedge.io",
		},
	}
	return x509.CreateCertificateRequest(rand.Reader, certReq, pk)
}

// GetEdgeCert applies for the certificate from cloudcore
func GetEdgeCert(url string, cacert []byte, token string) ([]byte, error) {
	csr, err := getCSR()
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %v", err)
	}
	client, err := http.NewHTTPclientWithCA(cacert)
	req, _ := http.BuildRequest("GET", url, bytes.NewReader(csr), token)
	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	edgecert, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return edgecert, nil
}

// SaveToFile saves the certificate or private key
func SaveToFile(data []byte, file string, pemBlockType string) error {
	out, err := os.Create(file)
	defer out.Close()
	if err != nil {
		return fmt.Errorf("failed to create file: %s", file)
	}
	if err = pem.Encode(out, &pem.Block{Type: pemBlockType, Bytes: data}); err != nil {
		return err
	}
	return nil
}

func hashCA(cacerts []byte) string {
	digest := sha256.Sum256(cacerts)
	return hex.EncodeToString(digest[:])
}

// ValidateCACerts validates the CA certificate by hash code
func ValidateCACerts(cacerts []byte, hash string) (bool, string, string) {
	if len(cacerts) == 0 && hash == "" {
		return true, "", ""
	}

	newHash := hashCA(cacerts)
	return hash == newHash, hash, newHash
}
