/*
Copyright 2026 The KubeEdge Authors.

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

package controllermanager

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Controller Manager Webhook Server", func() {
	It("should serve AdmissionReview over TLS", func() {
		By("waiting for webhook server readiness")
		healthClient := &http.Client{Timeout: 2 * time.Second}
		Eventually(func() bool {
			resp, err := healthClient.Get(fmt.Sprintf("http://127.0.0.1:%d/readyz", healthProbePort))
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 20*time.Second, 250*time.Millisecond).Should(BeTrue())

		By("loading webhook TLS certificate")
		certPEM, err := os.ReadFile(filepath.Join(webhookCertDir, "tls.crt"))
		Expect(err).NotTo(HaveOccurred())
		roots := x509.NewCertPool()
		Expect(roots.AppendCertsFromPEM(certPEM)).To(BeTrue())

		httpsClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    roots,
					MinVersion: tls.VersionTLS12,
				},
			},
			Timeout: 5 * time.Second,
		}

		const requestUID = "webhook-test-uid"
		review := admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
			Request: &admissionv1.AdmissionRequest{
				UID:       types.UID(requestUID),
				Kind:      metav1.GroupVersionKind{Version: "v1", Kind: "Pod"},
				Resource:  metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
				Name:      "webhook-test",
				Namespace: "default",
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{Raw: []byte(`{
"apiVersion":"v1",
"kind":"Pod",
"metadata":{"name":"webhook-test","namespace":"default"}
}`)},
			},
		}

		body, err := json.Marshal(review)
		Expect(err).NotTo(HaveOccurred())
		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("https://localhost:%d/test-admission", webhookPort),
			bytes.NewReader(body),
		)
		Expect(err).NotTo(HaveOccurred())
		req.Header.Set("Content-Type", "application/json")

		By("sending AdmissionReview request over TLS")
		resp, err := httpsClient.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		var responseReview admissionv1.AdmissionReview
		Expect(json.NewDecoder(resp.Body).Decode(&responseReview)).To(Succeed())
		Expect(responseReview.Response).NotTo(BeNil())
		Expect(responseReview.Response.Allowed).To(BeTrue())
		Expect(responseReview.Response.UID).To(Equal(types.UID(requestUID)))
	})
})

func freeWebhookTestPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func writeWebhookTestTLSCert(certDir string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment |
			x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	var certPEM bytes.Buffer
	if err := pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	var keyPEM bytes.Buffer
	if err := pem.Encode(&keyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(certDir, "tls.crt"), certPEM.Bytes(), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(certDir, "tls.key"), keyPEM.Bytes(), 0o600)
}
