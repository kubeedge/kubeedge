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

package message

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestBuildMsg(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		name      string
		parentID  string
		group     string
		resource  string
		source    string
		operation string
		content   interface{}
		result    model.Message
	}{
		{
			name:      "Non Empty group, resource, operation and content",
			parentID:  "parent1",
			group:     "resource",
			source:    "edgehub",
			operation: "publish",
			resource:  "node/connection",
			content:   "This is a content",
			result: model.Message{
				Router: model.MessageRoute{
					Group:     "resource",
					Resource:  "node/connection",
					Operation: "publish",
					Source:    "edgehub",
				},
				Content: "This is a content",
			},
		},
		{
			name:      "Empty parentID and content",
			parentID:  "",
			group:     "twin",
			source:    "edgehub",
			operation: "subscribe",
			resource:  "node/connection",
			content:   "",
			result: model.Message{
				Header: model.MessageHeader{
					ParentID: "",
				},
				Router: model.MessageRoute{
					Group:     "twin",
					Resource:  "node/connection",
					Operation: "subscribe",
					Source:    "edgehub",
				},
				Content: "",
			},
		},
		{
			name:      "Empty group , parentID , source , resource , operation and content",
			parentID:  "",
			group:     "",
			source:    "",
			operation: "",
			content:   "",
			resource:  "",
			result: model.Message{
				Header: model.MessageHeader{
					ParentID: "",
				},
				Router: model.MessageRoute{
					Group:     "",
					Resource:  "",
					Operation: "",
					Source:    "",
				},
				Content: "",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(_ *testing.T) {
			result := BuildMsg(test.group, test.parentID, test.source, test.resource, test.operation, test.content)
			assert.NotNil(result)
			assert.Equal(test.result.Router.Group, result.GetGroup())
			assert.Equal(test.result.Router.Resource, result.GetResource())
			assert.Equal(test.result.Router.Operation, result.GetOperation())
			assert.Equal(test.result.Router.Source, result.GetSource())
			assert.Equal(test.result.Content, result.GetContent())
		})
	}
}

func TestParseResourceEdge(t *testing.T) {
	tests := []struct {
		name           string
		resource       string
		operation      string
		wantNamespace  string
		wantType       string
		wantID         string
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name:          "Valid resource with namespace/type/id",
			resource:      "namespace/resourceType/resourceID",
			operation:     model.UpdateOperation,
			wantNamespace: "namespace",
			wantType:      "resourceType",
			wantID:        "resourceID",
			wantErr:       false,
		},
		{
			name:          "Valid query operation with namespace/type",
			resource:      "namespace/resourceType",
			operation:     model.QueryOperation,
			wantNamespace: "namespace",
			wantType:      "resourceType",
			wantID:        "",
			wantErr:       false,
		},
		{
			name:          "Valid response operation with namespace/type",
			resource:      "namespace/resourceType",
			operation:     model.ResponseOperation,
			wantNamespace: "namespace",
			wantType:      "resourceType",
			wantID:        "",
			wantErr:       false,
		},
		{
			name:           "Invalid operation with incomplete resource",
			resource:       "namespace/resourceType",
			operation:      model.UpdateOperation,
			wantNamespace:  "",
			wantType:       "",
			wantID:         "",
			wantErr:        true,
			expectedErrMsg: "resource: namespace/resourceType format incorrect, or Operation: update is not query/response",
		},
		{
			name:           "Empty resource and operation",
			resource:       "",
			operation:      "",
			wantNamespace:  "",
			wantType:       "",
			wantID:         "",
			wantErr:        true,
			expectedErrMsg: "resource:  format incorrect, or Operation:  is not query/response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNamespace, gotType, gotID, err := ParseResourceEdge(tt.resource, tt.operation)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResourceEdge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil || err.Error() != tt.expectedErrMsg {
					t.Errorf("ParseResourceEdge() error message = %v, expected %v", err, tt.expectedErrMsg)
				}
				return
			}

			if gotNamespace != tt.wantNamespace {
				t.Errorf("ParseResourceEdge() gotNamespace = %v, want %v", gotNamespace, tt.wantNamespace)
			}

			if gotType != tt.wantType {
				t.Errorf("ParseResourceEdge() gotType = %v, want %v", gotType, tt.wantType)
			}

			if gotID != tt.wantID {
				t.Errorf("ParseResourceEdge() gotID = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}
