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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
)

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
