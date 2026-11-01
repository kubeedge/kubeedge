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

package listener

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"time"

	certutil "k8s.io/client-go/util/cert"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	hubconfig "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/config"
)

const (
	// forwardTimeoutMargin is added to the REST timeout when forwarding a request
	// to another CloudCore, so the forwarding side does not give up before the
	// target CloudCore has answered.
	forwardTimeoutMargin = 10 * time.Second

	forwardIdleConnTimeout = 90 * time.Second
)

// newServerTLSConfig creates the TLS config of the REST listener. The configured
// certificate is used if set, otherwise the CloudCore certificate is used.
func newServerTLSConfig(c *v1alpha1.RouterTLS) (*tls.Config, error) {
	var (
		cert tls.Certificate
		err  error
	)
	if c != nil && (c.CertFile != "" || c.PrivateKeyFile != "") {
		cert, err = tls.LoadX509KeyPair(c.CertFile, c.PrivateKeyFile)
	} else {
		cert, err = tls.X509KeyPair(
			pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: hubconfig.Config.Cert}),
			pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: hubconfig.Config.Key}),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load the router certificate: %v", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// newForwardTLSClient creates the client used to forward requests to the REST
// listener of another CloudCore over TLS. The target must present a certificate
// issued by the CloudCore CA, or the same certificate as this listener (when all
// CloudCores share a configured certificate). The host name is not checked,
// because CloudCores are addressed by their pod IP, which is usually not in the
// certificate.
func newForwardTLSClient(caDER, serverCertDER []byte, timeout time.Duration) (*http.Client, error) {
	roots := x509.NewCertPool()
	if len(caDER) > 0 {
		ca, err := x509.ParseCertificate(caDER)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the CloudCore CA: %v", err)
		}
		roots.AddCert(ca)
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// The default verification also requires the host name to match, so it
		// is replaced by VerifyPeerCertificate, which still verifies the chain.
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: verifyForwardTarget(roots, serverCertDER),
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			IdleConnTimeout: forwardIdleConnTimeout,
		},
	}, nil
}

func verifyForwardTarget(roots *x509.CertPool, serverCertDER []byte) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("the target CloudCore presented no certificate")
		}
		if len(serverCertDER) > 0 && bytes.Equal(rawCerts[0], serverCertDER) {
			return nil
		}

		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, raw := range rawCerts {
			cert, err := x509.ParseCertificate(raw)
			if err != nil {
				return fmt.Errorf("failed to parse the target CloudCore certificate: %v", err)
			}
			certs = append(certs, cert)
		}
		intermediates := x509.NewCertPool()
		for _, cert := range certs[1:] {
			intermediates.AddCert(cert)
		}
		_, err := certs[0].Verify(x509.VerifyOptions{
			Roots:         roots,
			Intermediates: intermediates,
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		})
		if err != nil {
			return fmt.Errorf("failed to verify the target CloudCore certificate: %v", err)
		}
		return nil
	}
}
