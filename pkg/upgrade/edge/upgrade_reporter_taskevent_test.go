/*
Copyright 2025 The KubeEdge Authors.

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

package edge

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	fsmv1alpha1 "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func TestNewTaskEventReporter(t *testing.T) {
	config := &v1alpha2.EdgeCoreConfig{}
	reporter := NewTaskEventReporter("test-job", "test-event", config)

	assert.NotNil(t, reporter)
	taskEventReporter, ok := reporter.(*TaskEventReporter)
	assert.True(t, ok)
	assert.Equal(t, "test-job", taskEventReporter.JobName)
	assert.Equal(t, "test-event", taskEventReporter.EventType)
	assert.Equal(t, config, taskEventReporter.Config)
}

func TestTaskEventReporter_Report(t *testing.T) {
	t.Run("report success", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		var capturedEvent fsm.Event
		patches.ApplyFunc(ReportTaskResult, func(config *v1alpha2.EdgeCoreConfig, taskType, taskID string, event fsm.Event) error {
			capturedEvent = event
			assert.Equal(t, "test-job", taskID)
			assert.Equal(t, TaskTypeUpgrade, taskType)
			return nil
		})

		config := &v1alpha2.EdgeCoreConfig{}
		reporter := &TaskEventReporter{
			JobName:   "test-job",
			EventType: "test-event",
			Config:    config,
		}

		err := reporter.Report(nil)
		assert.NoError(t, err)
		assert.Equal(t, "test-event", capturedEvent.Type)
		assert.Equal(t, fsmv1alpha1.ActionSuccess, capturedEvent.Action)
		assert.Empty(t, capturedEvent.Msg)
	})

	t.Run("report failure", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		var capturedEvent fsm.Event
		patches.ApplyFunc(ReportTaskResult, func(config *v1alpha2.EdgeCoreConfig, taskType, taskID string, event fsm.Event) error {
			capturedEvent = event
			return nil
		})

		config := &v1alpha2.EdgeCoreConfig{}
		reporter := &TaskEventReporter{
			JobName:   "test-job",
			EventType: "test-event",
			Config:    config,
		}

		testErr := errors.New("test error")
		err := reporter.Report(testErr)
		assert.NoError(t, err)
		assert.Equal(t, "test-event", capturedEvent.Type)
		assert.Equal(t, fsmv1alpha1.ActionFailure, capturedEvent.Action)
		assert.Equal(t, "test error", capturedEvent.Msg)
	})

	t.Run("report task result error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(ReportTaskResult, func(config *v1alpha2.EdgeCoreConfig, taskType, taskID string, event fsm.Event) error {
			return errors.New("report failed")
		})

		config := &v1alpha2.EdgeCoreConfig{}
		reporter := &TaskEventReporter{
			JobName:   "test-job",
			EventType: "test-event",
			Config:    config,
		}

		err := reporter.Report(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "report failed")
	})
}

func TestReportTaskResult(t *testing.T) {
	t.Run("ca file read error", func(t *testing.T) {
		config := &v1alpha2.EdgeCoreConfig{
			Modules: &v1alpha2.Modules{
				Edged: &v1alpha2.Edged{
					TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
						HostnameOverride: "test-node",
					},
				},
				EdgeHub: &v1alpha2.EdgeHub{
					TLSCAFile: "/nonexistent/ca.crt",
				},
			},
		}

		event := fsm.Event{Type: "test", Action: fsmv1alpha1.ActionSuccess}
		err := ReportTaskResult(config, TaskTypeUpgrade, "test-job", event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read ca")
	})

	t.Run("cert file load error", func(t *testing.T) {
		tmpDir := t.TempDir()
		caFile := filepath.Join(tmpDir, "ca.crt")
		certFile := filepath.Join(tmpDir, "cert.crt")
		keyFile := filepath.Join(tmpDir, "key.key")

		err := os.WriteFile(caFile, []byte("fake ca content"), 0644)
		assert.NoError(t, err)

		config := &v1alpha2.EdgeCoreConfig{
			Modules: &v1alpha2.Modules{
				Edged: &v1alpha2.Edged{
					TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
						HostnameOverride: "test-node",
					},
				},
				EdgeHub: &v1alpha2.EdgeHub{
					TLSCAFile:         caFile,
					TLSCertFile:       certFile,
					TLSPrivateKeyFile: keyFile,
					HTTPServer:        "https://test.com",
				},
			},
		}

		event := fsm.Event{Type: "test", Action: fsmv1alpha1.ActionSuccess}
		err = ReportTaskResult(config, TaskTypeUpgrade, "test-job", event)
		assert.Error(t, err)
	})

	t.Run("http post error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		tmpDir := t.TempDir()
		caFile := filepath.Join(tmpDir, "ca.crt")

		validCaCert := []byte(`-----BEGIN CERTIFICATE-----
MIICGTCCAYKgAwIBAgIJALKZKWKUjQJ0MA0GCSqGSIb3DQEBCwUAMCUxIzAhBgNV
BAMTGnRlc3QtY2EtZm9yLXVuaXQtdGVzdGluZzAeFw0yNDAxMDEwMDAwMDBaFw0z
NDAxMDEwMDAwMDBaMCUxIzAhBgNVBAMTGnRlc3QtY2EtZm9yLXVuaXQtdGVzdGlu
ZzCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAqZ6nF+ZsjNv8bPZ1eDO2M8I1
oYKT8z6CJ9sV4X3jKYCNjH0e8b6f2mUXL5+Hn3tTvP3hTn7vT2cLnPq7mRrHjKjP
t7nGpRn3p9vLfJ+NfPmT5kN7NrHj1kT8t7qE1w8t5mL8v7L3nG8D6F1cQwM2z1l1
2cI8b1zD9yJ2X1S7a1cCAwEAAaNQME4wHQYDVR0OBBYEFEf8Fj7wvFyKnPT1e7vj
5wYnM8o2MB8GA1UdIwQYMBaAFEf8Fj7wvFyKnPT1e7vj5wYnM8o2MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQELBQADgYEAWJO7F8TGnYjD2L9V7VYoTmD8hJZiKd6v
F2X1Q2v7lYXZq1aF2K8l5qO6u4XNvTh3k7nH0z2lKnHhE6m6LwZ3E4c8pN0QQvjW
ZXkP1U2l4mYzF8c6o1vZ3K2cF3pJ1F2K6nT8q2sLxY3L5nG6zK2mX1q4vZ1zD4Y7
-----END CERTIFICATE-----`)

		err := os.WriteFile(caFile, validCaCert, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
			return tls.Certificate{}, nil
		})

		config := &v1alpha2.EdgeCoreConfig{
			Modules: &v1alpha2.Modules{
				Edged: &v1alpha2.Edged{
					TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
						HostnameOverride: "test-node",
					},
				},
				EdgeHub: &v1alpha2.EdgeHub{
					TLSCAFile:         caFile,
					TLSCertFile:       "fake-cert",
					TLSPrivateKeyFile: "fake-key",
					HTTPServer:        "https://invalid-url-that-will-fail",
				},
			},
		}

		event := fsm.Event{Type: "test", Action: fsmv1alpha1.ActionSuccess}
		err = ReportTaskResult(config, TaskTypeUpgrade, "test-job", event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "post http request failed")
	})

	t.Run("successful http post", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		tmpDir := t.TempDir()
		caFile := filepath.Join(tmpDir, "ca.crt")

		validCaCert := []byte(`-----BEGIN CERTIFICATE-----
MIICGTCCAYKgAwIBAgIJALKZKWKUjQJ0MA0GCSqGSIb3DQEBCwUAMCUxIzAhBgNV
BAMTGnRlc3QtY2EtZm9yLXVuaXQtdGVzdGluZzAeFw0yNDAxMDEwMDAwMDBaFw0z
NDAxMDEwMDAwMDBaMCUxIzAhBgNVBAMTGnRlc3QtY2EtZm9yLXVuaXQtdGVzdGlu
ZzCBnzANBgkqhkiG9w0BAQEFAAOBjQAwgYkCgYEAqZ6nF+ZsjNv8bPZ1eDO2M8I1
oYKT8z6CJ9sV4X3jKYCNjH0e8b6f2mUXL5+Hn3tTvP3hTn7vT2cLnPq7mRrHjKjP
t7nGpRn3p9vLfJ+NfPmT5kN7NrHj1kT8t7qE1w8t5mL8v7L3nG8D6F1cQwM2z1l1
2cI8b1zD9yJ2X1S7a1cCAwEAAaNQME4wHQYDVR0OBBYEFEf8Fj7wvFyKnPT1e7vj
5wYnM8o2MB8GA1UdIwQYMBaAFEf8Fj7wvFyKnPT1e7vj5wYnM8o2MAwGA1UdEwQF
MAMBAf8wDQYJKoZIhvcNAQELBQADgYEAWJO7F8TGnYjD2L9V7VYoTmD8hJZiKd6v
F2X1Q2v7lYXZq1aF2K8l5qO6u4XNvTh3k7nH0z2lKnHhE6m6LwZ3E4c8pN0QQvjW
ZXkP1U2l4mYzF8c6o1vZ3K2cF3pJ1F2K6nT8q2sLxY3L5nG6zK2mX1q4vZ1zD4Y7
-----END CERTIFICATE-----`)

		err := os.WriteFile(caFile, validCaCert, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(tls.LoadX509KeyPair, func(certFile, keyFile string) (tls.Certificate, error) {
			return tls.Certificate{}, nil
		})

		var httpPostCalled bool
		patches.ApplyMethodFunc(&http.Client{}, "Post", func(url, contentType string, body any) (*http.Response, error) {
			httpPostCalled = true
			resp := &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
			}
			return resp, nil
		})

		config := &v1alpha2.EdgeCoreConfig{
			Modules: &v1alpha2.Modules{
				Edged: &v1alpha2.Edged{
					TailoredKubeletFlag: v1alpha2.TailoredKubeletFlag{
						HostnameOverride: "test-node",
					},
				},
				EdgeHub: &v1alpha2.EdgeHub{
					TLSCAFile:         caFile,
					TLSCertFile:       "fake-cert",
					TLSPrivateKeyFile: "fake-key",
					HTTPServer:        server.URL,
				},
			},
		}

		event := fsm.Event{Type: "test", Action: fsmv1alpha1.ActionSuccess, Msg: "success"}
		err = ReportTaskResult(config, TaskTypeUpgrade, "test-job", event)
		assert.NoError(t, err)
		assert.True(t, httpPostCalled)
	})
}
