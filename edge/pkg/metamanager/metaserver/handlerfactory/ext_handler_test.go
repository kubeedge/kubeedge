/*
Copyright 2024 The KubeEdge Authors.

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

package handlerfactory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/task/taskexecutor"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/upgradedb"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
	"github.com/kubeedge/kubeedge/pkg/util/fsm"
)

func TestLogs(t *testing.T) {
	type testCase struct {
		requestInfo            *request.RequestInfo
		queryParams            string
		isGetLogsFail          bool
		isUnexpectedStatusCode bool
		isStreamingErr         bool
	}

	cases := testCase{}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	f := NewFactory()

	patch.ApplyMethod(reflect.TypeOf(f.storage), "Logs", func(s *storage.REST, ctx context.Context, info common.LogsInfo) (*types.LogsResponse, *http.Response) {
		if cases.isGetLogsFail {
			return &types.LogsResponse{
				LogMessages: []string{},
				ErrMessages: []string{"failed to get logs for container"},
			}, nil
		}
		if cases.isUnexpectedStatusCode {
			return &types.LogsResponse{}, &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString("")),
			}
		}

		if cases.queryParams == "follow=true" {
			if cases.isStreamingErr {
				return &types.LogsResponse{}, &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}
			}
			return &types.LogsResponse{}, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("log line 1\nlog line 2\n")),
			}
		}
		return &types.LogsResponse{
				LogMessages: []string{},
				ErrMessages: []string{},
			}, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("log line 1\nlog line 2\n")),
			}
	})

	tests := []struct {
		name           string
		cases          testCase
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Logs with success response",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams: "",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"logMessages":["log line 1\nlog line 2\n"],"errMessages":null}`,
		},
		{
			name: "Logs(follow) with success response",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams: "follow=true",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "log line 1\nlog line 2\n",
		},
		{
			name: "Logs with get logs fail",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams:   "",
				isGetLogsFail: true,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to get logs from edged\n",
		},
		{
			name: "Logs with unexpected status code",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams:            "",
				isUnexpectedStatusCode: true,
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   fmt.Sprintf("Unexpected status code from edged: %d\n", http.StatusInternalServerError),
		},
		{
			name: "Logs with streaming error",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams:    "follow=true",
				isStreamingErr: true,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/logs?"+tt.cases.queryParams, nil)
			w := httptest.NewRecorder()

			cases = tt.cases

			handler := f.Logs(tt.cases.requestInfo)
			handler.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.expectedStatus == http.StatusOK && tt.cases.queryParams == "follow=true" {
				assert.Equal(t, tt.expectedBody, string(body))
			} else if tt.expectedStatus == http.StatusInternalServerError {
				assert.Equal(t, tt.expectedBody, string(body))
			} else {
				var logsResponse types.LogsResponse
				err := json.Unmarshal(body, &logsResponse)
				assert.NoError(t, err)
				expectedResponse := types.LogsResponse{}
				if err = json.Unmarshal([]byte(tt.expectedBody), &expectedResponse); err != nil {
					t.Errorf("failed to unmarshal expected body: %v", err)
				}
				assert.Equal(t, expectedResponse, logsResponse)
			}
		})
	}
}

func TestExec(t *testing.T) {
	type testCase struct {
		requestInfo    *request.RequestInfo
		queryParams    string
		isExecFail     bool
		isHandlerExist bool
	}

	cases := testCase{}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	f := NewFactory()

	patch.ApplyMethod(reflect.TypeOf(f.storage), "Exec", func(s *storage.REST, ctx context.Context, info common.ExecInfo) (*types.ExecResponse, http.Handler) {
		if cases.isExecFail {
			return &types.ExecResponse{
				ErrMessages: []string{"failed to exec command in container"},
			}, nil
		}
		if cases.isHandlerExist {
			return &types.ExecResponse{}, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("handler response"))
				assert.NoError(t, err)
			})
		}
		return &types.ExecResponse{
			ErrMessages:    []string{},
			RunOutMessages: []string{"exec output"},
			RunErrMessages: []string{},
		}, nil
	})

	tests := []struct {
		name           string
		cases          testCase
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Exec with success response",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams: "command=ls&container=test-container&stdout=true",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"runOutMessages":["exec output"], "runErrMessages":null, "errMessages":null}`,
		},
		{
			name: "Exec with handler response",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams:    "command=ls&container=test-container&stdout=true&tty=true&stdin=true",
				isHandlerExist: true,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "handler response",
		},
		{
			name: "Exec with exec fail",
			cases: testCase{
				requestInfo: &request.RequestInfo{
					Name:      "test-pod",
					Namespace: "default",
				},
				queryParams: "command=ls&container=test-container&stdout=true",
				isExecFail:  true,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"runOutMessages":null,"runErrMessages":null,"errMessages":["failed to exec command in container"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/exec?"+tt.cases.queryParams, nil)
			w := httptest.NewRecorder()

			cases = tt.cases

			handler := f.Exec(tt.cases.requestInfo)
			handler.ServeHTTP(w, req)

			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if tt.expectedStatus == http.StatusOK && tt.cases.isHandlerExist {
				assert.Equal(t, tt.expectedBody, string(body))
			} else if tt.expectedStatus == http.StatusInternalServerError {
				assert.Equal(t, tt.expectedBody, string(body))
			} else {
				var execResponse types.ExecResponse
				err := json.Unmarshal(body, &execResponse)
				assert.NoError(t, err)
				expectedResponse := types.ExecResponse{}
				if err = json.Unmarshal([]byte(tt.expectedBody), &expectedResponse); err != nil {
					t.Errorf("failed to unmarshal expected body: %v", err)
				}
				assert.Equal(t, expectedResponse, execResponse)
			}
		})
	}
}

func TestRestart(t *testing.T) {
	patch := gomonkey.NewPatches()
	defer patch.Reset()

	f := NewFactory()

	mockRestartResponse := &types.RestartResponse{}

	patch.ApplyFunc(limitedReadBody, func(req *http.Request, size int64) ([]byte, error) {
		if req.Body == nil {
			return []byte("[]"), nil
		}
		data, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	})

	patch.ApplyMethod(reflect.TypeOf(f.storage), "Restart",
		func(_ *storage.REST, _ context.Context, _ common.RestartInfo) *types.RestartResponse {
			return mockRestartResponse
		})

	originalMarshal := json.Marshal

	patch.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
		if _, ok := v.(*types.RestartResponse); ok {
			return []byte(`{"success":true,"message":"Pods restarted successfully"}`), nil
		}
		return originalMarshal(v)
	})

	t.Run("Successful restart", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/restart", strings.NewReader(`["pod1", "pod2"]`))
		w := httptest.NewRecorder()

		handler := f.Restart("default")
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		assert.JSONEq(t, `{"success":true,"message":"Pods restarted successfully"}`, string(body))
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		invalidJSONPatch := gomonkey.ApplyFunc(limitedReadBody, func(_ *http.Request, _ int64) ([]byte, error) {
			return []byte("invalid json"), nil
		})
		defer invalidJSONPatch.Reset()

		jsonUnmarshalPatch := gomonkey.ApplyFunc(json.Unmarshal, func(data []byte, v interface{}) error {
			if string(data) == "invalid json" {
				return errors.New("invalid character 'i' looking for beginning of value")
			}
			return nil
		})
		defer jsonUnmarshalPatch.Reset()

		req := httptest.NewRequest("POST", "/restart", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		handler := f.Restart("default")
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "invalid character")
	})

	t.Run("Marshal error", func(t *testing.T) {
		marshalErrorPatch := gomonkey.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
			if _, ok := v.(*types.RestartResponse); ok {
				return nil, errors.New("marshal error")
			}
			return nil, nil
		})
		defer marshalErrorPatch.Reset()

		req := httptest.NewRequest("POST", "/restart", strings.NewReader(`["pod1"]`))
		w := httptest.NewRecorder()

		handler := f.Restart("default")
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "marshal error")
	})
}

func TestConfirmUpgrade(t *testing.T) {
	patch := gomonkey.NewPatches()
	defer patch.Reset()

	f := NewFactory()

	patch.ApplyFunc(options.GetEdgeCoreOptions, func() *options.EdgeCoreOptions {
		return &options.EdgeCoreOptions{
			ConfigFile: "/etc/kubeedge/config/edgecore.yaml",
		}
	})

	patch.ApplyFunc(upgradedb.QueryNodeTaskRequestFromMetaV2, func() (types.NodeTaskRequest, error) {
		return types.NodeTaskRequest{
			TaskID: "task-123",
			Type:   "upgrade",
		}, nil
	})

	patch.ApplyFunc(upgradedb.QueryNodeUpgradeJobRequestFromMetaV2, func() (types.NodeUpgradeJobRequest, error) {
		return types.NodeUpgradeJobRequest{
			UpgradeID: "upgrade-123",
			HistoryID: "history-123",
			Version:   "v1.12.0",
			Image:     "kubeedge/installation-package:v1.12.0",
		}, nil
	})

	executorMock := &mockExecutor{}
	patch.ApplyFunc(taskexecutor.GetExecutor, func(taskType string) (taskexecutor.Executor, error) {
		return executorMock, nil
	})

	patch.ApplyFunc(klog.Errorf, func(format string, args ...interface{}) {})
	patch.ApplyFunc(klog.Info, func(args ...interface{}) {})
	patch.ApplyFunc(klog.Infof, func(format string, args ...interface{}) {})

	patch.ApplyFunc(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})

	patch.ApplyMethod((*exec.Cmd)(nil), "CombinedOutput", func(_ *exec.Cmd) ([]byte, error) {
		return []byte("upgrade successful"), nil
	})

	patch.ApplyFunc(upgradedb.DeleteNodeTaskRequestFromMetaV2, func() error {
		return nil
	})

	patch.ApplyFunc(upgradedb.DeleteNodeUpgradeJobRequestFromMetaV2, func() error {
		return nil
	})

	t.Run("ConfirmUpgrade success", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/confirm-upgrade", nil)
		w := httptest.NewRecorder()

		handler := f.ConfirmUpgrade()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("ConfirmUpgrade command error", func(t *testing.T) {
		cmdErrorPatch := gomonkey.ApplyMethod((*exec.Cmd)(nil), "CombinedOutput",
			func(_ *exec.Cmd) ([]byte, error) {
				return []byte("command failed"), errors.New("command failed")
			})
		defer cmdErrorPatch.Reset()

		req := httptest.NewRequest("POST", "/confirm-upgrade", nil)
		w := httptest.NewRecorder()

		handler := f.ConfirmUpgrade()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Contains(t, string(body), "command failed")
	})

	t.Run("ConfirmUpgrade with db error handling", func(t *testing.T) {
		dbErrorPatch := gomonkey.ApplyFunc(upgradedb.DeleteNodeTaskRequestFromMetaV2, func() error {
			return errors.New("db delete error")
		})
		defer dbErrorPatch.Reset()

		req := httptest.NewRequest("POST", "/confirm-upgrade", nil)
		w := httptest.NewRecorder()

		handler := f.ConfirmUpgrade()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// mockExecutor implements the taskexecutor.Executor interface
type mockExecutor struct{}

func (m *mockExecutor) Name() string {
	return "mockExecutor"
}

func (m *mockExecutor) Do(req types.NodeTaskRequest) (fsm.Event, error) {
	return fsm.Event{
		Type:   "UpgradeConfirmed",
		Action: "Success",
	}, nil
}
