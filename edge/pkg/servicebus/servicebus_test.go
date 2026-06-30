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

package servicebus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core"

	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	"github.com/kubeedge/kubeedge/pkg/features"
)

// TestNewServicebus tests the constructor for servicebus.
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
			assert := assert.New(t)
			sb := newServicebus(tt.enable, tt.server, tt.port, tt.timeout)
			assert.NotNil(sb, "newServicebus() should not return nil")
			assert.Equal(tt.enable, sb.enable, "servicebus.enable = %v, want %v", sb.enable, tt.enable)
			assert.Equal(tt.server, sb.server, "servicebus.server = %v, want %v", sb.server, tt.server)
			assert.Equal(tt.port, sb.port, "servicebus.port = %v, want %v", sb.port, tt.port)
			assert.Equal(tt.timeout, sb.timeout, "servicebus.timeout = %v, want %v", sb.timeout, tt.timeout)
			assert.NotNil(sb.sbs, "servicebus.sbs (ServiceBusService) should not be nil")
		})
	}
}

func TestName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "ServiceBusNameTest",
			want: modules.ServiceBusModuleName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			sb := &servicebus{}
			assert.Equal(tt.want, sb.Name(), "servicebus.Name() = %v, want %v", sb.Name(), tt.want)
		})
	}
}

// TestGroup tests the Group() method of servicebus.
func TestGroup(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "ServiceBusGroupTest",
			want: modules.BusGroup,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			sb := &servicebus{}
			assert.Equal(tt.want, sb.Group(), "servicebus.Group() = %v, want %v", sb.Group(), tt.want)
		})
	}
}

func TestEnable(t *testing.T) {
	tests := []struct {
		name string
		sb   *servicebus
		want bool
	}{
		{
			name: "Enable true",
			want: true,
			sb:   &servicebus{enable: true},
		},
		{
			name: "Enable false",
			want: false,
			sb:   &servicebus{enable: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tt.want, tt.sb.Enable(),
				"servicebus.Enable() returned unexpected result. got = %v, want = %v", tt.sb.Enable(), tt.want)
		})
	}
}

// TestRestartPolicy tests the RestartPolicy() method with feature gate toggling.
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
			assert := assert.New(t)
			err := features.DefaultMutableFeatureGate.SetFromMap(
				map[string]bool{string(features.ModuleRestart): tt.featureEnabled})
			assert.NoError(err, "Failed to set feature gate")

			sb := &servicebus{}
			got := sb.RestartPolicy()

			if tt.wantNil {
				assert.Nil(got, "RestartPolicy() should return nil when feature gate is disabled")
				return
			}

			assert.NotNil(got, "RestartPolicy() should return non-nil policy when feature gate is enabled")
			assert.Equal(core.RestartTypeOnFailure, got.RestartType,
				"RestartType = %v, want %v", got.RestartType, core.RestartTypeOnFailure)
			assert.Equal(2.0, got.IntervalTimeGrowthRate,
				"IntervalTimeGrowthRate = %v, want 2.0", got.IntervalTimeGrowthRate)
		})
	}
}

// TestBuildErrorResponse tests the buildErrorResponse helper with various status codes and messages.
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
			assert := assert.New(t)
			msg, err := buildErrorResponse(tt.parentID, tt.content, tt.statusCode)
			assert.NoError(err, "buildErrorResponse() returned unexpected error")

			assert.Equal(tt.parentID, msg.GetParentID(),
				"ParentID = %q, want %q", msg.GetParentID(), tt.parentID)

			assert.Equal(modules.ServiceBusModuleName, msg.GetSource(),
				"Source = %q, want %q", msg.GetSource(), modules.ServiceBusModuleName)
			assert.Equal(modules.UserGroup, msg.GetGroup(),
				"Group = %q, want %q", msg.GetGroup(), modules.UserGroup)

			body := msg.GetContent()
			httpResp, ok := body.(commonType.HTTPResponse)
			assert.True(ok, "Content type = %T, want commonType.HTTPResponse", body)

			assert.Equal(tt.statusCode, httpResp.StatusCode,
				"HTTPResponse.StatusCode = %d, want %d", httpResp.StatusCode, tt.statusCode)
			assert.Equal(tt.content, string(httpResp.Body),
				"HTTPResponse.Body = %q, want %q", string(httpResp.Body), tt.content)
			assert.Equal("kubeedge-edgecore", httpResp.Header.Get("Server"),
				"HTTPResponse.Header[Server] = %q, want %q", httpResp.Header.Get("Server"), "kubeedge-edgecore")
		})
	}
}

// TestMarshalResult tests the marshalResult helper for JSON serialization.
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
			assert := assert.New(t)
			result := marshalResult(tt.resp)
			assert.NotNil(result, "marshalResult should not return nil")

			var decoded serverResponse
			err := json.Unmarshal(result, &decoded)
			assert.NoError(err, "marshalResult produced invalid JSON")

			assert.Equal(tt.resp.Code, decoded.Code,
				"Code = %d, want %d", decoded.Code, tt.resp.Code)
			assert.Equal(tt.resp.Msg, decoded.Msg,
				"Msg = %q, want %q", decoded.Msg, tt.resp.Msg)
			assert.Equal(tt.resp.Body, decoded.Body,
				"Body = %q, want %q", decoded.Body, tt.resp.Body)

			// Verify expected JSON field names
			var raw map[string]json.RawMessage
			err = json.Unmarshal(result, &raw)
			assert.NoError(err, "failed to unmarshal as map")
			for _, key := range []string{"code", "msg", "body"} {
				_, ok := raw[key]
				assert.True(ok, "JSON output missing expected key %q", key)
			}
		})
	}
}

// TestBuildBasicHandler_InvalidBody tests the handler with an invalid JSON body.
func TestBuildBasicHandler_InvalidBody(t *testing.T) {
	assert := assert.New(t)

	handler := buildBasicHandler(5 * time.Second)
	body := strings.NewReader("this is not json")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(err, "failed to unmarshal response")
	assert.Equal(http.StatusBadRequest, resp.Code)
	assert.Equal("invalid params", resp.Msg)
}

// TestBuildBasicHandler_EmptyBody tests the handler with an empty body.
func TestBuildBasicHandler_EmptyBody(t *testing.T) {
	assert := assert.New(t)

	handler := buildBasicHandler(5 * time.Second)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(err, "failed to unmarshal response")
	assert.Equal(http.StatusBadRequest, resp.Code)
}

// TestBuildBasicHandler_OversizedBody tests the handler rejects bodies exceeding maxBodySize.
func TestBuildBasicHandler_OversizedBody(t *testing.T) {
	assert := assert.New(t)

	handler := buildBasicHandler(5 * time.Second)
	oversized := strings.Repeat("x", int(maxBodySize)+1)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(oversized))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp serverResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(err, "failed to unmarshal response")
	assert.Equal(http.StatusBadRequest, resp.Code)
	assert.Equal("can't read data from body of the http's request", resp.Msg)
}

func TestBuildBasicHandler_ReturnsHandler(t *testing.T) {
	assert := assert.New(t)

	handler := buildBasicHandler(10 * time.Second)
	assert.NotNil(handler, "buildBasicHandler should return a non-nil handler")
}
