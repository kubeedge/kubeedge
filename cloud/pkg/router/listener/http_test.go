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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestForwardWritesOKResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test-Header", "value")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	recorder := httptest.NewRecorder()

	if err := requestForward("127.0.0.1", recorder, req); err != nil {
		t.Fatalf("requestForward returned error: %v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("X-Test-Header"); got != "value" {
		t.Fatalf("X-Test-Header = %q, want %q", got, "value")
	}
	if got := recorder.Body.String(); got != "ok" {
		t.Fatalf("body = %q, want %q", got, "ok")
	}
}

func TestRequestForwardReturnsResponseStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test-Header", "value")
		w.WriteHeader(http.StatusGatewayTimeout)
		if _, err := w.Write([]byte("upstream timeout")); err != nil {
			t.Fatalf("Write returned error: %v", err)
		}
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	recorder := httptest.NewRecorder()

	err = requestForward("127.0.0.1", recorder, req)
	if err == nil {
		t.Fatal("requestForward returned nil, want error")
	}
	respErr, ok := err.(*responseStatusError)
	if !ok {
		t.Fatalf("requestForward returned %T, want *responseStatusError", err)
	}
	if respErr.statusCode != http.StatusGatewayTimeout {
		t.Fatalf("statusCode = %d, want %d", respErr.statusCode, http.StatusGatewayTimeout)
	}
	if got := respErr.header.Get("X-Test-Header"); got != "value" {
		t.Fatalf("X-Test-Header = %q, want %q", got, "value")
	}
	if got := string(respErr.body); got != "upstream timeout" {
		t.Fatalf("body = %q, want %q", got, "upstream timeout")
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("recorder body length = %d, want 0", recorder.Body.Len())
	}
}
