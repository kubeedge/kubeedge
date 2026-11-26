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

package interlink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestPod(name, namespace string, annotations map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "test-image:latest",
				},
			},
		},
	}
}

func TestIsInterLinkPod(t *testing.T) {
	tests := []struct {
		name     string
		pod      *v1.Pod
		expected bool
	}{
		{
			name:     "nil pod",
			pod:      nil,
			expected: false,
		},
		{
			name:     "pod without annotations",
			pod:      newTestPod("test", "default", nil),
			expected: false,
		},
		{
			name:     "pod with unrelated annotation",
			pod:      newTestPod("test", "default", map[string]string{"foo": "bar"}),
			expected: false,
		},
		{
			name:     "pod with wrong annotation value",
			pod:      newTestPod("test", "default", map[string]string{OffloadAnnotationKey: "other"}),
			expected: false,
		},
		{
			name:     "pod with correct interlink annotation",
			pod:      newTestPod("test", "default", map[string]string{OffloadAnnotationKey: OffloadAnnotationValue}),
			expected: true,
		},
		{
			name: "pod with interlink annotation and others",
			pod: newTestPod("test", "default", map[string]string{
				OffloadAnnotationKey:             OffloadAnnotationValue,
				"slurm.interlink.io/partition":   "gpu",
			}),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInterLinkPod(tt.pod)
			if result != tt.expected {
				t.Errorf("IsInterLinkPod() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClientCreate(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		respBody   string
		wantErr    bool
	}{
		{
			name:       "successful create",
			statusCode: http.StatusOK,
			respBody:   `{"status": "ok"}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			respBody:   `{"error": "internal error"}`,
			wantErr:    true,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			respBody:   `{"error": "bad request"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != createPath {
					t.Errorf("expected path %s, got %s", createPath, r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Errorf("expected method POST, got %s", r.Method)
				}

				// Verify request body is valid
				var req CreateRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}
				if req.Pod.Name != "test-pod" {
					t.Errorf("expected pod name test-pod, got %s", req.Pod.Name)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.respBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, 5*time.Second)
			pod := newTestPod("test-pod", "default", map[string]string{OffloadAnnotationKey: OffloadAnnotationValue})

			err := client.Create(pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != deletePath {
			t.Errorf("expected path %s, got %s", deletePath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	pod := newTestPod("test-pod", "default", nil)

	if err := client.Delete(pod); err != nil {
		t.Errorf("Delete() unexpected error: %v", err)
	}
}

func TestClientStatus(t *testing.T) {
	expectedStatuses := []PodStatusResponse{
		{
			PodName:      "test-pod",
			PodNamespace: "default",
			Containers: []ContainerStatusResponse{
				{
					Name: "test-container",
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{
							StartedAt: metav1.Now(),
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != statusPath {
			t.Errorf("expected path %s, got %s", statusPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedStatuses)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second)
	pod := newTestPod("test-pod", "default", nil)

	statuses, err := client.Status([]*v1.Pod{pod})
	if err != nil {
		t.Fatalf("Status() unexpected error: %v", err)
	}

	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].PodName != "test-pod" {
		t.Errorf("expected pod name test-pod, got %s", statuses[0].PodName)
	}
}

func TestClientConnectionRefused(t *testing.T) {
	// Use a URL that will definitely refuse connections
	client := NewClient("http://127.0.0.1:1", 1*time.Second)
	pod := newTestPod("test-pod", "default", nil)

	if err := client.Create(pod); err == nil {
		t.Error("Create() expected error for connection refused, got nil")
	}

	if err := client.Delete(pod); err == nil {
		t.Error("Delete() expected error for connection refused, got nil")
	}

	if _, err := client.Status([]*v1.Pod{pod}); err == nil {
		t.Error("Status() expected error for connection refused, got nil")
	}
}

func TestMapInterLinkStatusToPodPhase(t *testing.T) {
	tests := []struct {
		name       string
		containers []ContainerStatusResponse
		expected   v1.PodPhase
	}{
		{
			name:       "no containers - pending",
			containers: nil,
			expected:   v1.PodPending,
		},
		{
			name: "container waiting - pending",
			containers: []ContainerStatusResponse{
				{
					Name:  "c1",
					State: v1.ContainerState{Waiting: &v1.ContainerStateWaiting{Reason: "Queued"}},
				},
			},
			expected: v1.PodPending,
		},
		{
			name: "container running - running",
			containers: []ContainerStatusResponse{
				{
					Name:  "c1",
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
			},
			expected: v1.PodRunning,
		},
		{
			name: "all terminated successfully - succeeded",
			containers: []ContainerStatusResponse{
				{
					Name:     "c1",
					State:    v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}},
					ExitCode: 0,
				},
			},
			expected: v1.PodSucceeded,
		},
		{
			name: "terminated with error - failed",
			containers: []ContainerStatusResponse{
				{
					Name:     "c1",
					State:    v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 1}},
					ExitCode: 1,
				},
			},
			expected: v1.PodFailed,
		},
		{
			name: "mixed running and terminated - running",
			containers: []ContainerStatusResponse{
				{
					Name:  "c1",
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
				{
					Name:  "c2",
					State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}},
				},
			},
			expected: v1.PodRunning,
		},
		{
			name: "one waiting among multiple - pending",
			containers: []ContainerStatusResponse{
				{
					Name:  "c1",
					State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
				},
				{
					Name:  "c2",
					State: v1.ContainerState{Waiting: &v1.ContainerStateWaiting{}},
				},
			},
			expected: v1.PodPending,
		},
		{
			name: "multiple terminated, one failed - failed",
			containers: []ContainerStatusResponse{
				{
					Name:  "c1",
					State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 0}},
				},
				{
					Name:  "c2",
					State: v1.ContainerState{Terminated: &v1.ContainerStateTerminated{ExitCode: 137}},
				},
			},
			expected: v1.PodFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapInterLinkStatusToPodPhase(tt.containers)
			if result != tt.expected {
				t.Errorf("MapInterLinkStatusToPodPhase() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewPodStatusPatch(t *testing.T) {
	status := PodStatusResponse{
		PodName:      "test-pod",
		PodNamespace: "default",
		Containers: []ContainerStatusResponse{
			{
				Name:  "c1",
				State: v1.ContainerState{Running: &v1.ContainerStateRunning{}},
			},
		},
	}

	patch := NewPodStatusPatch(status)

	if patch.Status.Phase != v1.PodRunning {
		t.Errorf("expected phase PodRunning, got %v", patch.Status.Phase)
	}

	if len(patch.Status.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(patch.Status.Conditions))
	}

	if patch.Status.Conditions[0].Type != "InterLinkManaged" {
		t.Errorf("expected condition type InterLinkManaged, got %s", patch.Status.Conditions[0].Type)
	}

	if patch.Status.Message == "" {
		t.Error("expected non-empty status message")
	}
}
