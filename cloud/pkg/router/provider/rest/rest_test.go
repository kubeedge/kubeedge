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

package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type targetFunc func(map[string]interface{}, chan struct{}) (interface{}, error)

func (f targetFunc) Name() string {
	return "target"
}

func (f targetFunc) GoToTarget(data map[string]interface{}, stop chan struct{}) (interface{}, error) {
	return f(data, stop)
}

func TestForwardReturnsGatewayTimeout(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/edge-node/default/api", nil)
	req.RequestURI = "/edge-node/default/api"

	target := targetFunc(func(_ map[string]interface{}, stop chan struct{}) (interface{}, error) {
		<-stop
		return nil, nil
	})

	got, err := (&Rest{Path: "api"}).Forward(target, map[string]interface{}{
		"messageID": "timeout-message",
		"request":   req,
		"timeout":   time.Millisecond,
		"data":      []byte("request body"),
	})
	if err != nil {
		t.Fatalf("Forward returned error: %v", err)
	}

	response, ok := got.(*http.Response)
	if !ok {
		t.Fatalf("Forward returned %T, want *http.Response", got)
	}
	if response.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusGatewayTimeout)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if string(body) != "wait to get response time out" {
		t.Fatalf("body = %q, want %q", string(body), "wait to get response time out")
	}
}
