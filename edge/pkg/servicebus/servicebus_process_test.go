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
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

// TestProcessMessageTargetWithColon checks that processMessage forwards a
// request to the local service when the resource is "<port>:<path>" and the
// path itself contains a colon. Colons are valid in URL paths (for example the
// custom method in "/v1/models/resnet:predict") and in query values.
func TestProcessMessageTargetWithColon(t *testing.T) {
	const hubStub = "servicebus-test-hub"

	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(&common.ModuleInfo{
		ModuleName: hubStub,
		ModuleType: common.MsgCtxTypeChannel,
	})
	beehiveContext.AddModuleGroup(hubStub, modules.HubGroup)
	t.Cleanup(func() { beehiveContext.Cleanup(hubStub) })

	oldClient := uc.Client
	uc.Client = &http.Client{Timeout: 5 * time.Second}
	t.Cleanup(func() { uc.Client = oldClient })

	tests := []struct {
		name      string
		target    string
		wantPath  string
		wantQuery string
	}{
		{
			name:     "path without colon",
			target:   "/v1/models/resnet",
			wantPath: "/v1/models/resnet",
		},
		{
			name:     "colon in path",
			target:   "/v1/models/resnet:predict",
			wantPath: "/v1/models/resnet:predict",
		},
		{
			name:      "colon in query value",
			target:    "/events?since=10:30",
			wantPath:  "/events",
			wantQuery: "since=10:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
				_, _ = w.Write([]byte("served"))
			}))
			defer server.Close()
			port := strconv.Itoa(server.Listener.Addr().(*net.TCPAddr).Port)

			msg := beehiveModel.NewMessage("").
				BuildRouter(sourceType, modules.UserGroup, port+":"+tt.target, http.MethodGet).
				FillBody(commonType.HTTPRequest{Method: http.MethodGet})

			processMessage(msg)

			respCh := make(chan beehiveModel.Message, 1)
			go func() {
				resp, err := beehiveContext.Receive(hubStub)
				if err == nil {
					respCh <- resp
				}
			}()
			var resp beehiveModel.Message
			select {
			case resp = <-respCh:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for the response message")
			}

			content, err := resp.GetContentData()
			if err != nil {
				t.Fatalf("failed to get response content: %v", err)
			}
			var httpResp commonType.HTTPResponse
			if err := json.Unmarshal(content, &httpResp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if httpResp.StatusCode != http.StatusOK {
				t.Fatalf("expected status %d, got %d with body %q",
					http.StatusOK, httpResp.StatusCode, string(httpResp.Body))
			}
			if string(httpResp.Body) != "served" {
				t.Errorf("expected body %q, got %q", "served", string(httpResp.Body))
			}
			if gotPath != tt.wantPath || gotQuery != tt.wantQuery {
				t.Errorf("service got path %q query %q, want path %q query %q",
					gotPath, gotQuery, tt.wantPath, tt.wantQuery)
			}
		})
	}
}
