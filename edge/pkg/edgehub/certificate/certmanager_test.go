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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	commhttp "github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate/http"
	httpfake "github.com/kubeedge/kubeedge/edge/pkg/edgehub/certificate/http/fake"
	"github.com/kubeedge/kubeedge/pkg/security/certs"
	"github.com/kubeedge/kubeedge/pkg/security/token"
)

func TestGetCurrent(t *testing.T) {
	cm := &CertManager{
		NodeName: "test-node",
		certFile: "testdata/server.crt",
		keyFile:  "testdata/server.key",
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
		return tls.Certificate{
			Certificate: [][]byte{{1, 2, 3}},
		}, nil
	})

	patches.ApplyFunc(x509.ParseCertificates, func(der []byte) ([]*x509.Certificate, error) {
		cert := &x509.Certificate{
			Subject: pkix.Name{
				CommonName: fmt.Sprintf("system:node:%s", cm.NodeName),
			},
		}
		return []*x509.Certificate{cert}, nil
	})

	cert, err := cm.getCurrent()

	require.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotNil(t, cert.Leaf)
}

func TestNextRotationDeadline(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	cm := &CertManager{
		NodeName: "test-node",
		certFile: "testdata/server.crt",
		keyFile:  "testdata/server.key",
		now: func() time.Time {
			return fixedTime
		},
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	notBefore := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	notAfter := time.Date(2023, 1, 11, 0, 0, 0, 0, time.UTC)

	patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
		return tls.Certificate{
			Certificate: [][]byte{{1, 2, 3}},
		}, nil
	})

	patches.ApplyFunc(x509.ParseCertificates, func(der []byte) ([]*x509.Certificate, error) {
		cert := &x509.Certificate{
			Subject: pkix.Name{
				CommonName: fmt.Sprintf("system:node:%s", cm.NodeName),
			},
			NotBefore: notBefore,
			NotAfter:  notAfter,
		}
		return []*x509.Certificate{cert}, nil
	})

	originalJitteryDuration := jitteryDuration
	defer func() { jitteryDuration = originalJitteryDuration }()

	jitteryDuration = func(totalDuration float64) time.Duration {
		return time.Duration(totalDuration * 0.8)
	}

	deadline, err := cm.nextRotationDeadline()

	require.NoError(t, err)

	totalDuration := float64(notAfter.Sub(notBefore))
	expectedDeadline := notBefore.Add(time.Duration(totalDuration * 0.8))

	assert.Equal(t, expectedDeadline, deadline)
}

func TestStart(t *testing.T) {
	t.Run("certificates already exist", func(t *testing.T) {
		cm := &CertManager{
			RotateCertificates: false,
			NodeName:           "test-node",
			Done:               make(chan struct{}),
			certFile:           "testdata/server.crt",
			keyFile:            "testdata/server.key",
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
			return tls.Certificate{
				Certificate: [][]byte{{1, 2, 3}},
			}, nil
		})

		patches.ApplyFunc(x509.ParseCertificates, func(der []byte) ([]*x509.Certificate, error) {
			cert := &x509.Certificate{
				Subject: pkix.Name{
					CommonName: fmt.Sprintf("system:node:%s", cm.NodeName),
				},
			}
			return []*x509.Certificate{cert}, nil
		})

		klogCallCount := 0
		klogMessages := []string{}

		patches.ApplyFunc(klog.Infof, func(format string, args ...interface{}) {
			klogCallCount++
			klogMessages = append(klogMessages, format)
		})

		cm.Start()

		rotationMessage := "Certificate rotation is enabled."
		messageFound := false
		for _, msg := range klogMessages {
			if msg == rotationMessage {
				messageFound = true
				break
			}
		}
		assert.False(t, messageFound, "rotate should not be called when RotateCertificates is false")
	})

	t.Run("certificates need to be applied", func(t *testing.T) {
		cm := &CertManager{
			RotateCertificates: false,
			NodeName:           "test-node",
			Done:               make(chan struct{}),
			certFile:           "testdata/server.crt",
			keyFile:            "testdata/server.key",
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		loadKeyPairCalled := false
		patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
			loadKeyPairCalled = true
			return tls.Certificate{}, errors.New("certificate not found")
		})

		patches.ApplyFunc(GetCACert, func(url string) ([]byte, error) {
			return []byte("test CA cert"), nil
		})

		patches.ApplyFunc(token.VerifyCAAndGetRealToken, func(token string, ca []byte) (string, error) {
			return "verified-token", nil
		})

		patches.ApplyFunc(certs.WriteDERToPEMFile, func(filename, blockType string, data []byte) (*pem.Block, error) {
			return &pem.Block{Type: blockType, Bytes: data}, nil
		})

		patches.ApplyFunc(commhttp.NewHTTPClientWithCA, func(caCrt []byte, certificate tls.Certificate) (*http.Client, error) {
			return &http.Client{}, nil
		})

		patches.ApplyFunc(commhttp.BuildRequest, func(method, urlStr string, body io.Reader, token, nodeName string) (*http.Request, error) {
			req, err := http.NewRequest(method, urlStr, body)
			if err != nil {
				return nil, err
			}
			return req, nil
		})

		patches.ApplyFunc(commhttp.SendRequest, func(req *http.Request, client *http.Client) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       httpfake.NewFakeBodyReader([]byte("test cert data")),
			}, nil
		})

		patches.ApplyFunc(klog.Warningf, func(format string, args ...interface{}) {
		})

		patches.ApplyFunc(klog.Info, func(args ...interface{}) {
		})

		cm.Start()

		assert.True(t, loadKeyPairCalled, "LoadX509KeyPair should be called")

		select {
		case <-CleanupTokenChan:
		default:
			assert.Fail(t, "Expected CleanupTokenChan to have a value")
		}
	})
}

func TestRotateCert(t *testing.T) {
	const testCAData = "test CA certificate data"
	testCertDER := []byte("test certificate DER data")
	testKeyDER := []byte("test key DER data")

	cm := &CertManager{
		NodeName: "test-node",
		caFile:   "testdata/ca.crt",
		certFile: "testdata/server.crt",
		keyFile:  "testdata/server.key",
		certURL:  "https://localhost:10002/edge.crt",
		Done:     make(chan struct{}, 1),
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
		return tls.Certificate{
			Certificate: [][]byte{{1, 2, 3}},
		}, nil
	})

	patches.ApplyFunc(x509.ParseCertificates, func(der []byte) ([]*x509.Certificate, error) {
		cert := &x509.Certificate{
			Subject: pkix.Name{
				CommonName: fmt.Sprintf("system:node:%s", cm.NodeName),
			},
			NotBefore: time.Now().Add(-1 * time.Hour),
			NotAfter:  time.Now().Add(24 * time.Hour),
		}
		return []*x509.Certificate{cert}, nil
	})

	patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		if filename == cm.caFile {
			return []byte(testCAData), nil
		}
		return nil, fmt.Errorf("unexpected file: %s", filename)
	})

	patches.ApplyFunc(commhttp.NewHTTPClientWithCA, func(caCrt []byte, certificate tls.Certificate) (*http.Client, error) {
		return &http.Client{}, nil
	})

	patches.ApplyFunc(commhttp.BuildRequest, func(method, urlStr string, body io.Reader, token, nodeName string) (*http.Request, error) {
		req, err := http.NewRequest(method, urlStr, body)
		if err != nil {
			return nil, err
		}
		return req, nil
	})

	patches.ApplyFunc(commhttp.SendRequest, func(req *http.Request, client *http.Client) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       httpfake.NewFakeBodyReader(testCertDER),
		}, nil
	})

	patches.ApplyFunc(certs.WriteDERToPEMFile, func(filename, blockType string, data []byte) (*pem.Block, error) {
		if len(data) == 0 {
			data = testKeyDER
		}
		return &pem.Block{Type: blockType, Bytes: data}, nil
	})

	success, err := cm.rotateCert()

	require.NoError(t, err)
	assert.True(t, success)

	select {
	case <-cm.Done:
	default:
		assert.Fail(t, "Expected Done channel to have a value")
	}
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

func TestNewCertManager(t *testing.T) {
	edgeHub := v1alpha2.EdgeHub{
		HTTPServer:         "https://localhost:10002",
		Token:              "test-token",
		TLSCAFile:          "/var/lib/kubeedge/ca.crt",
		TLSCertFile:        "/var/lib/kubeedge/certs/server.crt",
		TLSPrivateKeyFile:  "/var/lib/kubeedge/certs/server.key",
		RotateCertificates: true,
	}
	nodeName := "test-node"

	cm := NewCertManager(edgeHub, nodeName)

	assert.Equal(t, edgeHub.RotateCertificates, cm.RotateCertificates)
	assert.Equal(t, nodeName, cm.NodeName)
	assert.Equal(t, edgeHub.Token, cm.token)
	assert.Equal(t, edgeHub.TLSCAFile, cm.caFile)
	assert.Equal(t, edgeHub.TLSCertFile, cm.certFile)
	assert.Equal(t, edgeHub.TLSPrivateKeyFile, cm.keyFile)
	assert.Equal(t, edgeHub.HTTPServer+constants.DefaultCAURL, cm.caURL)
	assert.Equal(t, edgeHub.HTTPServer+constants.DefaultCertURL, cm.certURL)
	assert.NotNil(t, cm.now)
	assert.NotNil(t, cm.Done)
}

func TestJitteryDuration(t *testing.T) {
	testDurationNs := float64(100 * time.Second)

	result := jitteryDuration(testDurationNs)

	minExpected := time.Duration(testDurationNs * 0.7)
	maxExpected := time.Duration(testDurationNs * 0.9)

	assert.GreaterOrEqual(t, result, minExpected, "Expected result to be at least 70% of input duration")
	assert.LessOrEqual(t, result, maxExpected, "Expected result to be at most 90% of input duration")
}

func TestGetCA(t *testing.T) {
	testCAContent := []byte("test CA content")
	tmpCAFile, err := os.CreateTemp("", "ca-*.crt")
	require.NoError(t, err)
	defer os.Remove(tmpCAFile.Name())

	_, err = tmpCAFile.Write(testCAContent)
	require.NoError(t, err)
	require.NoError(t, tmpCAFile.Close())

	cm := &CertManager{
		caFile: tmpCAFile.Name(),
	}

	result, err := cm.getCA()

	require.NoError(t, err)
	assert.Equal(t, testCAContent, result)
}

func TestApplyCerts(t *testing.T) {
	const (
		testCAData    = "test CA certificate data"
		testToken     = "test.token.part1.part2"
		testRealToken = "token.part1.part2"
		testNodeName  = "test-node"
	)
	testCertDER := []byte("test certificate DER data")
	testKeyDER := []byte("test key DER data")
	_ = testKeyDER

	testCAPem := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte(testCAData),
	}

	tmpCAFile, err := os.CreateTemp("", "ca-*.crt")
	require.NoError(t, err)
	defer os.Remove(tmpCAFile.Name())

	tmpCertFile, err := os.CreateTemp("", "cert-*.crt")
	require.NoError(t, err)
	defer os.Remove(tmpCertFile.Name())

	tmpKeyFile, err := os.CreateTemp("", "key-*.key")
	require.NoError(t, err)
	defer os.Remove(tmpKeyFile.Name())

	cm := &CertManager{
		NodeName: testNodeName,
		token:    testToken,
		caFile:   tmpCAFile.Name(),
		certFile: tmpCertFile.Name(),
		keyFile:  tmpKeyFile.Name(),
		caURL:    "https://localhost:10002/ca.crt",
		certURL:  "https://localhost:10002/edge.crt",
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(GetCACert, func(url string) ([]byte, error) {
		return []byte(testCAData), nil
	})

	patches.ApplyFunc(token.VerifyCAAndGetRealToken, func(token string, ca []byte) (string, error) {
		return testRealToken, nil
	})

	patches.ApplyFunc(certs.WriteDERToPEMFile, func(filename string, blockType string, data []byte) (*pem.Block, error) {
		return testCAPem, nil
	})

	patches.ApplyMethod((*CertManager)(nil), "GetEdgeCert",
		func(cm *CertManager, url string, capem []byte, tlscert tls.Certificate, token string) ([]byte, []byte, error) {
			return testCertDER, testKeyDER, nil
		})

	err = cm.applyCerts()

	require.NoError(t, err)
}
