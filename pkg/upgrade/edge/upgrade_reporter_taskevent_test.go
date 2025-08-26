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
		certFile := filepath.Join(tmpDir, "cert.crt")
		keyFile := filepath.Join(tmpDir, "key.key")

		caCert := []byte(`-----BEGIN CERTIFICATE-----
MIIC9TCCAd2gAwIBAgIJAL1g+5hHh1KXMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNV
BAMMB3Rlc3QtY2EwHhcNMjUwMTI4MDAwMDAwWhcNMzUwMTI4MDAwMDAwWjASMRAw
DgYDVQQDDAd0ZXN0LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
test-fake-ca-content-for-testing
-----END CERTIFICATE-----`)

		clientCert := []byte(`-----BEGIN CERTIFICATE-----
MIIC9TCCAd2gAwIBAgIJAL1g+5hHh1KXMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNV
BAMMB3Rlc3QtY2EwHhcNMjUwMTI4MDAwMDAwWhcNMzUwMTI4MDAwMDAwWjASMRAw
DgYDVQQDDAd0ZXN0LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
test-fake-cert-content-for-testing
-----END CERTIFICATE-----`)

		clientKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAtest-fake-key-content-for-testing
-----END RSA PRIVATE KEY-----`)

		err := os.WriteFile(caFile, caCert, 0644)
		assert.NoError(t, err)
		err = os.WriteFile(certFile, clientCert, 0644)
		assert.NoError(t, err)
		err = os.WriteFile(keyFile, clientKey, 0644)
		assert.NoError(t, err)

		patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
			if filename == caFile {
				return caCert, nil
			}
			return nil, errors.New("file not found")
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
					TLSCertFile:       certFile,
					TLSPrivateKeyFile: keyFile,
					HTTPServer:        "https://invalid-url",
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

		caCert := []byte(`-----BEGIN CERTIFICATE-----
MIIC9TCCAd2gAwIBAgIJAL1g+5hHh1KXMA0GCSqGSIb3DQEBCwUAMBIxEDAOBgNV
BAMMB3Rlc3QtY2EwHhcNMjUwMTI4MDAwMDAwWhcNMzUwMTI4MDAwMDAwWjASMRAw
DgYDVQQDDAd0ZXN0LWNhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
test-fake-ca-content-for-testing
-----END CERTIFICATE-----`)

		err := os.WriteFile(caFile, caCert, 0644)
		assert.NoError(t, err)

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
