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

package restful

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
)

func TestLogsRequest(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		podName       string
		containerName string
		logsInfo      common.LogsInfo
		expectedPath  string
		containQuery  []string
	}{
		{
			name:          "logs request without query parameters",
			namespace:     "default",
			podName:       "test-pod",
			containerName: "test-container",
			logsInfo:      common.LogsInfo{},
			expectedPath:  "/containerLogs/default/test-pod/test-container",
		},
		{
			name:          "logs request with query parameters",
			namespace:     "kube-system",
			podName:       "edge-app",
			containerName: "main",
			logsInfo: common.LogsInfo{
				Follow:     "true",
				TailLines:  "100",
				Timestamps: "true",
			},
			containQuery: []string{
				"/containerLogs/kube-system/edge-app/main?",
				"follow=true",
				"tailLines=100",
				"timestamps=true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := LogsRequest(tt.namespace, tt.podName, tt.containerName, tt.logsInfo)
			if req == nil {
				t.Fatalf("LogsRequest() returned nil")
			}
			if req.Method != http.MethodGet {
				t.Errorf("LogsRequest() Method = %v, want %v", req.Method, http.MethodGet)
			}
			if strings.HasPrefix(req.Path, "//") {
				t.Errorf("LogsRequest() Path has duplicate leading slash: %v", req.Path)
			}
			if tt.expectedPath != "" && req.Path != tt.expectedPath {
				t.Errorf("LogsRequest() Path = %v, want %v", req.Path, tt.expectedPath)
			}
			for _, query := range tt.containQuery {
				if !strings.Contains(req.Path, query) {
					t.Errorf("LogsRequest() Path %v does not contain %v", req.Path, query)
				}
			}
		})
	}
}

func TestRestfulRequest_NilReceiver(t *testing.T) {
	var req *Request
	_, err := req.RestfulRequest()
	if err == nil {
		t.Errorf("expected error when calling RestfulRequest on nil receiver, got nil")
	}
}

func TestStructToQueryString(t *testing.T) {
	type testStruct struct {
		Name      string `query:"name"`
		Count     int    `query:"count"`
		Empty     string `query:"empty"`
		Namespace string `query:"-"`
	}

	data := testStruct{
		Name:      "kubeedge",
		Count:     42,
		Empty:     "",
		Namespace: "test",
	}

	query, err := structToQueryString(data)
	if err != nil {
		t.Fatalf("structToQueryString() error = %v", err)
	}

	if !strings.Contains(query, "name=kubeedge") {
		t.Errorf("expected query to contain name=kubeedge, got %v", query)
	}
	if !strings.Contains(query, "count=42") {
		t.Errorf("expected query to contain count=42, got %v", query)
	}
	if strings.Contains(query, "empty=") {
		t.Errorf("expected zero field to be skipped, got %v", query)
	}
	if strings.Contains(query, "Namespace=") {
		t.Errorf("expected ignored field to be skipped, got %v", query)
	}
}
