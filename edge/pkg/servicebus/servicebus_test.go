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
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	beehiveModel "github.com/kubeedge/beehive/pkg/core/model"
	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestProcessMessageRejectsOversizedResponse(t *testing.T) {
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(&common.ModuleInfo{ModuleName: modules.EdgeHubModuleName, ModuleType: common.MsgCtxTypeChannel})
	beehiveContext.AddModuleGroup(modules.EdgeHubModuleName, modules.HubGroup)
	uc.Client = &http.Client{Timeout: 10 * time.Second}

	tests := []struct {
		name       string
		bodySize   int
		wantStatus int
	}{
		{name: "body at the limit is forwarded", bodySize: maxBodySize, wantStatus: http.StatusOK},
		{name: "body over the limit is rejected", bodySize: maxBodySize + 1, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.Repeat([]byte("a"), tt.bodySize)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(body)
			}))
			defer backend.Close()
			_, port, err := net.SplitHostPort(backend.Listener.Addr().String())
			require.NoError(t, err)

			msg := beehiveModel.NewMessage("").
				SetRoute(sourceType, modules.UserGroup).
				SetResourceOperation(port+":/", "request").
				FillBody(commonType.HTTPRequest{Method: http.MethodGet})
			processMessage(msg)

			resp, err := beehiveContext.Receive(modules.EdgeHubModuleName)
			require.NoError(t, err)
			content, err := resp.GetContentData()
			require.NoError(t, err)
			var httpResp commonType.HTTPResponse
			require.NoError(t, json.Unmarshal(content, &httpResp))
			require.Equal(t, tt.wantStatus, httpResp.StatusCode)
			if tt.wantStatus == http.StatusOK {
				require.Len(t, httpResp.Body, tt.bodySize)
			}
		})
	}
}
