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

package nodetask

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/taskmanager/v1alpha1/util"
	commontypes "github.com/kubeedge/kubeedge/common/types"
)

func TestBuildUpgradeMessage(t *testing.T) {
	tests := []struct {
		name         string
		resp         commontypes.NodeUpgradeJobResponse
		wantResource string
		wantEvent    string
		wantAction   api.Action
		wantNodeName string
		wantReason   string
	}{
		{
			name: "upgrade success",
			resp: commontypes.NodeUpgradeJobResponse{
				UpgradeID: "job-123",
				NodeName:  "edge-node-1",
				Status:    UpgradeSuccess,
			},
			wantResource: "task/job-123/node/edge-node-1",
			wantEvent:    "Upgrade",
			wantAction:   api.ActionSuccess,
			wantNodeName: "edge-node-1",
		},
		{
			name: "upgrade failed rollback success",
			resp: commontypes.NodeUpgradeJobResponse{
				UpgradeID: "job-456",
				NodeName:  "edge-node-2",
				Status:    UpgradeFailedRollbackSuccess,
				Reason:    "upgrade failed, rolled back",
			},
			wantResource: "task/job-456/node/edge-node-2",
			wantEvent:    "Rollback",
			wantAction:   api.ActionSuccess,
			wantNodeName: "edge-node-2",
			wantReason:   "upgrade failed, rolled back",
		},
		{
			name: "upgrade failed rollback failed",
			resp: commontypes.NodeUpgradeJobResponse{
				UpgradeID: "job-789",
				NodeName:  "edge-node-3",
				Status:    UpgradeFailedRollbackFailed,
				Reason:    "upgrade and rollback both failed",
			},
			wantResource: "task/job-789/node/edge-node-3",
			wantEvent:    "Rollback",
			wantAction:   api.ActionFailure,
			wantNodeName: "edge-node-3",
			wantReason:   "upgrade and rollback both failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildUpgradeMessage(tt.resp)

			assert.Equal(t, modules.CloudHubModuleName, msg.GetSource())
			assert.Equal(t, modules.CloudHubModuleGroup, msg.GetGroup())
			assert.Equal(t, tt.wantResource, msg.GetResource())
			assert.Equal(t, util.TaskUpgrade, msg.GetOperation())

			content, ok := msg.GetContent().(commontypes.NodeTaskResponse)
			require.True(t, ok, "expected NodeTaskResponse content")
			assert.Equal(t, tt.wantNodeName, content.NodeName)
			assert.Equal(t, tt.wantEvent, content.Event)
			assert.Equal(t, tt.wantAction, content.Action)
			assert.Equal(t, tt.wantReason, content.Reason)
		})
	}
}

func TestUpgradeEdgeInvalidRequests(t *testing.T) {
	ws := new(restful.WebService)
	ws.Route(ws.POST("/upgrade").To(UpgradeEdge))
	container := restful.NewContainer()
	container.Add(ws)

	t.Run("invalid json returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/upgrade", bytes.NewBufferString("{invalid"))
		rec := httptest.NewRecorder()
		container.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("oversized body returns 400", func(t *testing.T) {
		oversized := bytes.Repeat([]byte("a"), int(millionByte)+1)
		req := httptest.NewRequest(http.MethodPost, "/upgrade", bytes.NewReader(oversized))
		rec := httptest.NewRecorder()
		container.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
