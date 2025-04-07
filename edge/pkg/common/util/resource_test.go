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

package util_test

import (
	"testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/util"
)

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
			gotNamespace, gotType, gotID, err := util.ParseResourceEdge(tt.resource, tt.operation)

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
