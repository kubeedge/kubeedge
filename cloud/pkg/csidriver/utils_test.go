/*
Copyright 2024 The KubeEdge Authors.

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

package csidriver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/beehive/pkg/core/model"
)

func TestParseEndpoint(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		input          string
		expectedStrOne string
		expectedStrTwo string
		expectError    bool
	}{
		{
			input:          "unix:///tmp/test.sock",
			expectedStrOne: "unix",
			expectedStrTwo: "/tmp/test.sock",
			expectError:    false,
		},
		{
			input:          "unix://tmp/test.sock",
			expectedStrOne: "unix",
			expectedStrTwo: "tmp/test.sock",
			expectError:    false,
		},
		{
			input:          "tcp://127.0.0.1:8080",
			expectedStrOne: "tcp",
			expectedStrTwo: "127.0.0.1:8080",
			expectError:    false,
		},
		{
			input:          "/tmp/test.sock",
			expectedStrOne: "",
			expectedStrTwo: "",
			expectError:    true,
		},
		{
			input:          "unix://",
			expectedStrOne: "",
			expectedStrTwo: "",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		firstStr, secondStr, err := parseEndpoint(tc.input)
		if tc.expectError {
			assert.Error(err)
		} else {
			assert.NoError(err)
			assert.Equal(tc.expectedStrOne, firstStr)
			assert.Equal(tc.expectedStrTwo, secondStr)
		}
	}
}

func TestBuildResource(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name         string
		nodeID       string
		namespace    string
		resourceType string
		resourceID   string
		want         string
		wantErr      bool
	}{
		{
			name:         "Valid resource without resourceID",
			nodeID:       "node1",
			namespace:    "default",
			resourceType: "volume",
			resourceID:   "",
			want:         "node/node1/default/volume",
			wantErr:      false,
		},
		{
			name:         "Valid resource with resourceID",
			nodeID:       "node1",
			namespace:    "default",
			resourceType: "volume",
			resourceID:   "vol1",
			want:         "node/node1/default/volume/vol1",
			wantErr:      false,
		},
		{
			name:         "Resource without nodeID",
			nodeID:       "",
			namespace:    "default",
			resourceType: "volume",
			resourceID:   "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "Resource missing namespace",
			nodeID:       "node1",
			namespace:    "",
			resourceType: "volume",
			resourceID:   "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "Resource without resourceType",
			nodeID:       "node1",
			namespace:    "default",
			resourceType: "",
			resourceID:   "",
			want:         "",
			wantErr:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := buildResource(test.nodeID, test.namespace, test.resourceType, test.resourceID)
			if test.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(test.want, got)
			}
		})
	}
}

func TestExtractMessage(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name    string
		context string
		wantErr bool
	}{
		{
			name:    "Valid JSON",
			context: `{"header":{"namespace":"default"},"router":{"resource":"test"},"content":"test"}`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			context: `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "Empty context",
			context: "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			msg, err := extractMessage(test.context)
			if test.wantErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.NotNil(msg)
				assert.IsType(&model.Message{}, msg)
			}
		})
	}
}
