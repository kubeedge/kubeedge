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

package servicebus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"

	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
	"github.com/kubeedge/kubeedge/pkg/features"
)

func TestNewServicebus(t *testing.T) {
	tests := []struct {
		name    string
		enable  bool
		server  string
		port    int
		timeout int
	}{
		{
			name:    "default values",
			enable:  true,
			server:  "127.0.0.1",
			port:    9060,
			timeout: 60,
		},
		{
			name:    "disabled with custom server",
			enable:  false,
			server:  "0.0.0.0",
			port:    8080,
			timeout: 30,
		},
		{
			name:    "zero timeout",
			enable:  true,
			server:  "localhost",
			port:    0,
			timeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := newServicebus(tt.enable, tt.server, tt.port, tt.timeout)
			require.NotNil(t, sb, "newServicebus() should not return nil")
			assert.Equal(t, tt.enable, sb.enable)
			assert.Equal(t, tt.server, sb.server)
			assert.Equal(t, tt.port, sb.port)
			assert.Equal(t, tt.timeout, sb.timeout)
			assert.NotNil(t, sb.sbs, "servicebus.sbs (ServiceBusService) should not be nil")
		})
	}
}
func TestRestartPolicy(t *testing.T) {
	originalState := features.DefaultFeatureGate.Enabled(features.ModuleRestart)
	t.Cleanup(func() {
		_ = features.DefaultMutableFeatureGate.SetFromMap(
			map[string]bool{string(features.ModuleRestart): originalState})
	})

	tests := []struct {
		name           string
		featureEnabled bool
		wantNil        bool
	}{
		{
			name:           "feature gate disabled returns nil",
			featureEnabled: false,
			wantNil:        true,
		},
		{
			name:           "feature gate enabled returns policy",
			featureEnabled: true,
			wantNil:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := features.DefaultMutableFeatureGate.SetFromMap(
				map[string]bool{string(features.ModuleRestart): tt.featureEnabled})
			require.NoError(t, err, "Failed to set feature gate")

			sb := &servicebus{}
			got := sb.RestartPolicy()

			if tt.wantNil {
				assert.Nil(t, got, "RestartPolicy() should return nil when feature gate is disabled")
				return
			}

			require.NotNil(t, got, "RestartPolicy() should return non-nil policy when feature gate is enabled")
			assert.Equal(t, core.RestartTypeOnFailure, got.RestartType)
			assert.Equal(t, 2.0, got.IntervalTimeGrowthRate)
		})
	}
}

func TestBuildErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		parentID   string
		content    string
		statusCode int
	}{
		{
			name:       "bad request error",
			parentID:   "msg-001",
			content:    "the format of resource is incorrect",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "not found error",
			parentID:   "msg-002",
			content:    "error to call service",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "internal server error",
			parentID:   "msg-003",
			content:    "error to receive response, err: connection reset",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "empty parent ID",
			parentID:   "",
			content:    "some error",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "empty content",
			parentID:   "msg-004",
			content:    "",
			statusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := buildErrorResponse(tt.parentID, tt.content, tt.statusCode)
			require.NoError(t, err, "buildErrorResponse() returned unexpected error")

			assert.Equal(t, tt.parentID, msg.GetParentID())
			assert.Equal(t, modules.ServiceBusModuleName, msg.GetSource())
			assert.Equal(t, modules.UserGroup, msg.GetGroup())

			body := msg.GetContent()
			httpResp, ok := body.(commonType.HTTPResponse)
			require.True(t, ok, "Content type = %T, want commonType.HTTPResponse", body)

			assert.Equal(t, tt.statusCode, httpResp.StatusCode)
			assert.Equal(t, tt.content, string(httpResp.Body))
			assert.Equal(t, "kubeedge-edgecore", httpResp.Header.Get("Server"))
		})
	}
}

func TestMarshalResult(t *testing.T) {
	tests := []struct {
		name string
		resp *serverResponse
	}{
		{
			name: "success response",
			resp: &serverResponse{
				Code: http.StatusOK,
				Msg:  "receive response from cloud successfully",
				Body: `{"key":"value"}`,
			},
		},
		{
			name: "error response with empty body",
			resp: &serverResponse{
				Code: http.StatusBadRequest,
				Msg:  "invalid params",
				Body: "",
			},
		},
		{
			name: "zero value response",
			resp: &serverResponse{},
		},
		{
			name: "response with special characters in body",
			resp: &serverResponse{
				Code: http.StatusOK,
				Msg:  "ok",
				Body: `{"msg":"hello \"world\"","list":[1,2,3]}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marshalResult(tt.resp)
			require.NotNil(t, result, "marshalResult should not return nil")

			var decoded serverResponse
			err := json.Unmarshal(result, &decoded)
			require.NoError(t, err, "marshalResult produced invalid JSON")

			assert.Equal(t, tt.resp.Code, decoded.Code)
			assert.Equal(t, tt.resp.Msg, decoded.Msg)
			assert.Equal(t, tt.resp.Body, decoded.Body)

			var raw map[string]json.RawMessage
			err = json.Unmarshal(result, &raw)
			require.NoError(t, err, "failed to unmarshal as map")
			for _, key := range []string{"code", "msg", "body"} {
				_, ok := raw[key]
				assert.True(t, ok, "JSON output missing expected key %q", key)
			}
		})
	}
}

func TestBuildBasicHandler_InvalidBody(t *testing.T) {
	handler := buildBasicHandlerWithDeps(5*time.Second, basicHandlerDeps{
		getURLByKey: func(string) (*models.TargetUrls, error) { return nil, nil },
		sendSync: func(string, beehiveModel.Message, time.Duration) (beehiveModel.Message, error) {
			return beehiveModel.Message{}, nil
		},
	})
	body := strings.NewReader("this is not json")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response")
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "invalid params", resp.Msg)
}

func TestBuildBasicHandler_OversizedBody(t *testing.T) {
	handler := buildBasicHandlerWithDeps(5*time.Second, basicHandlerDeps{
		getURLByKey: func(string) (*models.TargetUrls, error) { return nil, nil },
		sendSync: func(string, beehiveModel.Message, time.Duration) (beehiveModel.Message, error) {
			return beehiveModel.Message{}, nil
		},
	})
	oversized := strings.Repeat("x", int(maxBodySize)+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(oversized))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response")
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "can't read data from body of the http's request", resp.Msg)
}


func TestBuildBasicHandler_UnregisteredURL(t *testing.T) {
	handler := buildBasicHandlerWithDeps(5*time.Second, basicHandlerDeps{
		getURLByKey: func(string) (*models.TargetUrls, error) {
			return nil, nil
		},
		sendSync: func(string, beehiveModel.Message, time.Duration) (beehiveModel.Message, error) {
			t.Fatal("sendSync should not be called for unregistered URL")
			return beehiveModel.Message{}, nil
		},
	})

	reqBody, err := json.Marshal(serverRequest{
		Method:    "GET",
		TargetURL: "http://example.com/unregistered",
		Payload:   nil,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(reqBody)))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response")
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Contains(t, resp.Msg, "is not allowed")
	assert.Contains(t, resp.Msg, "http://example.com/unregistered")
}

func TestBuildBasicHandler_SendSyncFailure(t *testing.T) {
	handler := buildBasicHandlerWithDeps(5*time.Second, basicHandlerDeps{
		getURLByKey: func(string) (*models.TargetUrls, error) {
			return &models.TargetUrls{URL: "http://example.com/api"}, nil
		},
		sendSync: func(string, beehiveModel.Message, time.Duration) (beehiveModel.Message, error) {
			return beehiveModel.Message{}, fmt.Errorf("edge hub unreachable")
		},
	})

	reqBody, err := json.Marshal(serverRequest{
		Method:    "POST",
		TargetURL: "http://example.com/api",
		Payload:   map[string]string{"key": "value"},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(reqBody)))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response")
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "edge hub unreachable", resp.Msg)
}

func TestBuildBasicHandler_InvalidResponseContent(t *testing.T) {
	handler := buildBasicHandlerWithDeps(5*time.Second, basicHandlerDeps{
		getURLByKey: func(string) (*models.TargetUrls, error) {
			return &models.TargetUrls{URL: "http://example.com/api"}, nil
		},
		sendSync: func(string, beehiveModel.Message, time.Duration) (beehiveModel.Message, error) {
			msg := beehiveModel.NewMessage("").FillBody(make(chan int))
			return *msg, nil
		},
	})

	reqBody, err := json.Marshal(serverRequest{
		Method:    "GET",
		TargetURL: "http://example.com/api",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(reqBody)))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response")
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
}

func TestBuildBasicHandler_SuccessPath(t *testing.T) {
	cloudResponse := `{"result":"ok","count":42}`

	handler := buildBasicHandlerWithDeps(5*time.Second, basicHandlerDeps{
		getURLByKey: func(key string) (*models.TargetUrls, error) {
			return &models.TargetUrls{URL: key}, nil
		},
		sendSync: func(module string, msg beehiveModel.Message, timeout time.Duration) (beehiveModel.Message, error) {
			assert.Equal(t, modules.EdgeHubModuleName, module)
			respMsg := beehiveModel.NewMessage("").FillBody([]byte(cloudResponse))
			return *respMsg, nil
		},
	})

	reqBody, err := json.Marshal(serverRequest{
		Method:    "GET",
		TargetURL: "http://example.com/api",
		Payload:   nil,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(reqBody)))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to unmarshal response")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "receive response from cloud successfully", resp.Msg)
	assert.Equal(t, cloudResponse, resp.Body)
}
