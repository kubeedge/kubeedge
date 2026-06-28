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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// TestMarshalResult verifies that marshalResult correctly serialises a
// serverResponse into JSON and that the zero-value case is valid JSON.
func TestMarshalResult(t *testing.T) {
	tests := []struct {
		name  string
		input *serverResponse
	}{
		{
			name:  "zero value",
			input: &serverResponse{},
		},
		{
			name: "with fields",
			input: &serverResponse{
				Code: http.StatusOK,
				Msg:  "ok",
				Body: "data",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalResult(tt.input)
			if len(got) == 0 {
				t.Fatal("marshalResult returned empty bytes")
			}
			var roundtrip serverResponse
			if err := json.Unmarshal(got, &roundtrip); err != nil {
				t.Fatalf("marshalResult output is not valid JSON: %v", err)
			}
			if roundtrip.Code != tt.input.Code {
				t.Errorf("Code mismatch: got %d, want %d", roundtrip.Code, tt.input.Code)
			}
			if roundtrip.Msg != tt.input.Msg {
				t.Errorf("Msg mismatch: got %q, want %q", roundtrip.Msg, tt.input.Msg)
			}
		})
	}
}

// TestBuildErrorResponse verifies that buildErrorResponse produces a message
// with the correct routing and a non-empty body.
func TestBuildErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		parentID   string
		content    string
		statusCode int
	}{
		{
			name:       "bad-request",
			parentID:   "parent-1",
			content:    "bad request body",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "internal-server-error",
			parentID:   "",
			content:    "internal error",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "not-found",
			parentID:   "parent-2",
			content:    "resource not found",
			statusCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := buildErrorResponse(tt.parentID, tt.content, tt.statusCode)
			if err != nil {
				t.Fatalf("buildErrorResponse returned unexpected error: %v", err)
			}
			// The route source should be the ServiceBus module.
			if msg.GetSource() != modules.ServiceBusModuleName {
				t.Errorf("source: got %q, want %q", msg.GetSource(), modules.ServiceBusModuleName)
			}
			// The parent message ID should be preserved as the response's parent ID.
			if msg.GetParentID() != tt.parentID {
				t.Errorf("parentID: got %q, want %q", msg.GetParentID(), tt.parentID)
			}
			if msg.Content == nil {
				t.Error("Content should not be nil")
			}
		})
	}
}

// TestBuildBasicHandlerInvalidJSON verifies that buildBasicHandler returns
// StatusBadRequest when the request body is not valid JSON.
func TestBuildBasicHandlerInvalidJSON(t *testing.T) {
	h := buildBasicHandler(5 * time.Second)
	body := strings.NewReader("not-json")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	var sResp serverResponse
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}
	if sResp.Code != http.StatusBadRequest {
		t.Errorf("Code: got %d, want %d", sResp.Code, http.StatusBadRequest)
	}
}


// TestNewServicebus verifies that newServicebus correctly stores the provided
// configuration values.
func TestNewServicebus(t *testing.T) {
	sb := newServicebus(true, "127.0.0.1", 9060, 30)
	if !sb.enable {
		t.Error("expected enable=true")
	}
	if sb.server != "127.0.0.1" {
		t.Errorf("server: got %q, want %q", sb.server, "127.0.0.1")
	}
	if sb.port != 9060 {
		t.Errorf("port: got %d, want 9060", sb.port)
	}
	if sb.timeout != 30 {
		t.Errorf("timeout: got %d, want 30", sb.timeout)
	}
	if sb.sbs == nil {
		t.Error("sbs should not be nil")
	}
}

// TestServicebusModuleMetadata verifies Name, Group and Enable on the
// servicebus struct without requiring a running beehive context.
func TestServicebusModuleMetadata(t *testing.T) {
	sb := newServicebus(false, "127.0.0.1", 9060, 10)
	if sb.Name() != modules.ServiceBusModuleName {
		t.Errorf("Name(): got %q, want %q", sb.Name(), modules.ServiceBusModuleName)
	}
	if sb.Group() != modules.BusGroup {
		t.Errorf("Group(): got %q, want %q", sb.Group(), modules.BusGroup)
	}
	if sb.Enable() {
		t.Error("Enable() should return false when initialised with enable=false")
	}

	sbEnabled := newServicebus(true, "127.0.0.1", 9060, 10)
	if !sbEnabled.Enable() {
		t.Error("Enable() should return true when initialised with enable=true")
	}
}
