/*
Copyright 2022 The KubeEdge Authors.

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
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/common/constants"
	commhttp "github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate/http"
	httpfake "github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate/http/fake"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
)

func TestGetCurrent(t *testing.T) {
	err := genFakeCerts()
	require.NoError(t, err)

	defer func() {
		if err := os.RemoveAll(fakeCertsDir); err != nil {
			t.Error(err)
		}
	}()

	t.Run("invalid node name", func(t *testing.T) {
		cm := &CertManager{
			NodeName: "node1",
			caFile:   filepath.Join(fakeCertsDir, "ca.crt"),
			certFile: filepath.Join(fakeCertsDir, "server.crt"),
			keyFile:  filepath.Join(fakeCertsDir, "server.key"),
		}
		_, err := cm.getCurrent()
		require.ErrorContains(t, err, "certificate CN system:node:testnode does not match node name node1")
	})

	t.Run("get tls certificate successfully", func(t *testing.T) {
		cm := &CertManager{
			NodeName: "testnode",
			caFile:   filepath.Join(fakeCertsDir, "ca.crt"),
			certFile: filepath.Join(fakeCertsDir, "server.crt"),
			keyFile:  filepath.Join(fakeCertsDir, "server.key"),
		}
		tlscert, err := cm.getCurrent()
		require.NoError(t, err)
		require.NotNil(t, tlscert)
	})
}

func TestGetCACert(t *testing.T) {
	const fakehost = "http://localhost"

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(commhttp.SendRequest,
		func(_ *http.Request, _ *http.Client) (*http.Response, error) {
			return &http.Response{Body: httpfake.NewFakeBodyReader([]byte{})}, nil
		})

	_, err := GetCACert(fakehost + "/ca.crt")
	require.NoError(t, err)
}

func TestGetEdgeCert(t *testing.T) {
	const (
		fakehost = "http://localhost"
	)

	t.Run("request failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(commhttp.NewHTTPClientWithCA,
			func(capem []byte, certificate tls.Certificate) (*http.Client, error) {
				return &http.Client{}, nil
			})
		patches.ApplyFunc(commhttp.SendRequest,
			func(_ *http.Request, _ *http.Client) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       httpfake.NewFakeBodyReader([]byte("test error")),
				}, nil
			})

		cm := &CertManager{}
		_, _, err := cm.GetEdgeCert(fakehost+constants.DefaultCAURL, []byte{}, tls.Certificate{}, "")
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to call http, code: 500, message: test error")
	})

	t.Run("request successful", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(commhttp.NewHTTPClientWithCA,
			func(capem []byte, certificate tls.Certificate) (*http.Client, error) {
				return &http.Client{}, nil
			})
		patches.ApplyFunc(commhttp.SendRequest,
			func(_ *http.Request, _ *http.Client) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       httpfake.NewFakeBodyReader([]byte("test cert...")),
				}, nil
			})

		cm := &CertManager{}
		_, _, err := cm.GetEdgeCert(fakehost+constants.DefaultCAURL, []byte{}, tls.Certificate{}, "")
		require.NoError(t, err)
	})
}

const (
	fakeCertsDir = "fake-certs"
)

func genFakeCerts() error {
	cahandler := certs.GetCAHandler(certs.CAHandlerTypeX509)
	pk, err := cahandler.GenPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to generate a private key, err: %v", err)
	}
	caPem, err := cahandler.NewSelfSigned(pk)
	if err != nil {
		return fmt.Errorf("failed to create Certificate Authority, error: %v", err)
	}

	certshandler := certs.GetHandler(certs.HandlerTypeX509)
	csrPem, err := certshandler.CreateCSR(pkix.Name{
		Country:      []string{"CN"},
		Organization: []string{"system:nodes"},
		Locality:     []string{"Hangzhou"},
		Province:     []string{"Zhejiang"},
		CommonName:   "system:node:testnode",
	}, pk, nil)
	if err != nil {
		return fmt.Errorf("failed to create a csr of edge cert, err %v", err)
	}
	certBlock, err := certshandler.SignCerts(certs.SignCertsOptionsWithCSR(
		csrPem.Bytes,
		caPem.Bytes,
		pk.DER(),
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		time.Hour,
	))
	if err != nil {
		return fmt.Errorf("fail to sign certs, err: %v", err)
	}
	fileContents := map[string][]byte{
		filepath.Join(fakeCertsDir, "ca.crt"):     pem.EncodeToMemory(caPem),
		filepath.Join(fakeCertsDir, "server.crt"): pem.EncodeToMemory(certBlock),
		filepath.Join(fakeCertsDir, "server.key"): pk.PEM(),
	}
	if err := os.Mkdir(fakeCertsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create the certs directory %s, err: %v", fakeCertsDir, err)
	}
	for path, content := range fileContents {
		if err := os.WriteFile(path, content, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create the certificate file %s, err: %v", path, err)
		}
	}
	return nil
}
