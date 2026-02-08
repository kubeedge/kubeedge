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

package status

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewEdgeStatus(t *testing.T) {
	cmd := NewEdgeStatus()
	if cmd == nil {
		t.Fatal("NewEdgeStatus returned nil")
	}

	if cmd.Use != "status" {
		t.Errorf("Expected use 'status', got '%s'", cmd.Use)
	}

	if len(cmd.Commands()) != 1 {
		t.Errorf("Expected 1 subcommand, got %d", len(cmd.Commands()))
	}

	edgehubCmd := cmd.Commands()[0]
	if edgehubCmd.Use != "edgehub" {
		t.Errorf("Expected subcommand use 'edgehub', got '%s'", edgehubCmd.Use)
	}
}

func TestNewEdgeHubStatus(t *testing.T) {
	cmd := NewEdgeHubStatus()
	if cmd == nil {
		t.Fatal("NewEdgeHubStatus returned nil")
	}

	if cmd.Use != "edgehub" {
		t.Errorf("Expected use 'edgehub', got '%s'", cmd.Use)
	}

	// Check if node flag is properly set
	flag := cmd.Flag("node")
	if flag == nil {
		t.Error("Expected 'node' flag to be set")
	}
}

func TestEdgeHubStatusOptions_checkEdgeCoreStatus(t *testing.T) {
	tests := []struct {
		name           string
		nodeName       string
		nodeReady      bool
		expectedStatus string
		expectError    bool
	}{
		{
			name:           "Node is ready",
			nodeName:       "test-node",
			nodeReady:      true,
			expectedStatus: "Running",
			expectError:    false,
		},
		{
			name:           "Node is not ready",
			nodeName:       "test-node",
			nodeReady:      false,
			expectedStatus: "Not Ready",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &EdgeHubStatusOptions{NodeName: tt.nodeName}

			// Create fake Kubernetes client
			fakeClient := fake.NewSimpleClientset()

			// Create test node
			node := &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.nodeName,
				},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			}

			if !tt.nodeReady {
				node.Status.Conditions[0].Status = corev1.ConditionFalse
			}

			// Add node to fake client
			_, err := fakeClient.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Failed to create test node: %v", err)
			}

			// Test the function
			err = opts.checkEdgeCoreStatus(fakeClient)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if opts.edgeCoreStatus != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, opts.edgeCoreStatus)
			}
		})
	}
}

func TestEdgeHubStatusOptions_checkEdgeHubConnection(t *testing.T) {
	tests := []struct {
		name           string
		nodeName       string
		runningPods    int
		expectedStatus string
		expectError    bool
	}{
		{
			name:           "Running pods found",
			nodeName:       "test-node",
			runningPods:    3,
			expectedStatus: "Connected",
			expectError:    false,
		},
		{
			name:           "No running pods",
			nodeName:       "test-node",
			runningPods:    0,
			expectedStatus: "Not Connected",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &EdgeHubStatusOptions{NodeName: tt.nodeName}

			// Create fake Kubernetes client
			fakeClient := fake.NewSimpleClientset()

			// Create test pods
			for i := 0; i < tt.runningPods; i++ {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-pod-%d", i),
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						NodeName: tt.nodeName,
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}

				_, err := fakeClient.CoreV1().Pods("default").Create(context.Background(), pod, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create test pod: %v", err)
				}
			}

			// Test the function
			err := opts.checkEdgeHubConnection(fakeClient)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if opts.edgeHubStatus != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, opts.edgeHubStatus)
			}
		})
	}
}

func TestEdgeHubStatusOptions_displayOverallStatus(t *testing.T) {
	tests := []struct {
		name            string
		nodeName        string
		edgeCoreStatus  string
		edgeHubStatus   string
		expectedOverall string
	}{
		{
			name:            "Both healthy",
			nodeName:        "test-node",
			edgeCoreStatus:  "Running",
			edgeHubStatus:   "Connected",
			expectedOverall: "Healthy",
		},
		{
			name:            "EdgeCore not ready",
			nodeName:        "test-node",
			edgeCoreStatus:  "Not Ready",
			edgeHubStatus:   "Connected",
			expectedOverall: "Unhealthy",
		},
		{
			name:            "EdgeHub not connected",
			nodeName:        "test-node",
			edgeCoreStatus:  "Running",
			edgeHubStatus:   "Not Connected",
			expectedOverall: "Unhealthy",
		},
		{
			name:            "Both unhealthy",
			nodeName:        "test-node",
			edgeCoreStatus:  "Not Ready",
			edgeHubStatus:   "Not Connected",
			expectedOverall: "Unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &EdgeHubStatusOptions{
				NodeName:       tt.nodeName,
				edgeCoreStatus: tt.edgeCoreStatus,
				edgeHubStatus:  tt.edgeHubStatus,
			}

			// Capture output
			// Note: In a real test, you might want to capture stdout
			// For now, we just verify the logic doesn't panic
			opts.displayOverallStatus()

			// The actual output verification would require stdout capture
			// This test mainly ensures the function works without panicking
		})
	}
}
