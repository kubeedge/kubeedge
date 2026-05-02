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
	"testing"

	"github.com/stretchr/testify/assert"

	commonType "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
)

func TestBuildErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		parentID   string
		content    string
		statusCode int
	}{
		{
			name:       "bad request",
			parentID:   "msg-123",
			content:    "invalid format",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "not found",
			parentID:   "msg-456",
			content:    "resource missing",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "server error",
			parentID:   "msg-789",
			content:    "internal failure",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "empty content",
			parentID:   "msg-000",
			content:    "",
			statusCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := buildErrorResponse(tt.parentID, tt.content, tt.statusCode)
			assert.NoError(t, err)
			assert.Equal(t, tt.parentID, msg.GetParentID())

			assert.Equal(t, modules.ServiceBusModuleName, msg.GetSource())
			assert.Equal(t, modules.UserGroup, msg.GetGroup())

			contentData, err := msg.GetContentData()
			assert.NoError(t, err)

			var resp commonType.HTTPResponse
			err = json.Unmarshal(contentData, &resp)
			assert.NoError(t, err)

			assert.Equal(t, tt.statusCode, resp.StatusCode)
			assert.Equal(t, tt.content, string(resp.Body))
		})
	}
}

func TestMarshalResult(t *testing.T) {
	tests := []struct {
		name     string
		response *serverResponse
	}{
		{
			name: "success response",
			response: &serverResponse{
				Code: http.StatusOK,
				Msg:  "ok",
				Body: "data",
			},
		},
		{
			name: "error response",
			response: &serverResponse{
				Code: http.StatusBadRequest,
				Msg:  "error",
				Body: "",
			},
		},
		{
			name: "empty response",
			response: &serverResponse{
				Code: 0,
				Msg:  "",
				Body: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := marshalResult(tt.response)

			var decoded serverResponse
			err := json.Unmarshal(data, &decoded)
			assert.NoError(t, err)

			assert.Equal(t, tt.response.Code, decoded.Code)
			assert.Equal(t, tt.response.Msg, decoded.Msg)
			assert.Equal(t, tt.response.Body, decoded.Body)
		})
	}
}
