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

package nodeselect

import (
	"testing"
)

func TestNewNodeSelector(t *testing.T) {
	ns := NewNodeSelector()
	if ns == nil {
		t.Fatal("NewNodeSelector returned nil")
	}
	if len(ns.nodes) != 0 {
		t.Errorf("Expected empty nodes, got %d nodes", len(ns.nodes))
	}
	if len(ns.selector) != 0 {
		t.Errorf("Expected empty selector, got %d selectors", len(ns.selector))
	}
}

func TestNodeSelector_AddNodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Single node",
			input:    "node1",
			expected: 1,
		},
		{
			name:     "Multiple nodes",
			input:    "node1,node2,node3",
			expected: 3,
		},
		{
			name:     "Nodes with spaces",
			input:    "node1, node2 , node3",
			expected: 3,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "Empty elements",
			input:    "node1,,node2",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := NewNodeSelector()
			ns.AddNodes(tt.input)
			if len(ns.GetNodes()) != tt.expected {
				t.Errorf("Expected %d nodes, got %d", tt.expected, len(ns.GetNodes()))
			}
		})
	}
}

func TestNodeSelector_AddSelector(t *testing.T) {
	ns := NewNodeSelector()
	ns.AddSelector("region", "us-west")
	ns.AddSelector("env", "production")

	if !ns.HasSelector() {
		t.Error("Expected HasSelector to return true")
	}

	selector := ns.GetSelector()
	if len(selector) != 2 {
		t.Errorf("Expected 2 selectors, got %d", len(selector))
	}

	if selector["region"] != "us-west" {
		t.Errorf("Expected region=us-west, got %s", selector["region"])
	}
}

func TestNodeSelector_AddSelector_EmptyValues(t *testing.T) {
	ns := NewNodeSelector()
	ns.AddSelector("", "value")
	ns.AddSelector("key", "")
	ns.AddSelector("", "")

	if ns.HasSelector() {
		t.Error("Expected HasSelector to return false for empty key/value pairs")
	}
}

func TestNodeSelector_Validate(t *testing.T) {
	tests := []struct {
		name      string
		nodes     string
		selector  map[string]string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "No selection",
			nodes:     "",
			selector:  map[string]string{},
			wantError: true,
			errorMsg:  "must specify either",
		},
		{
			name:      "Valid nodes",
			nodes:     "node1,node2",
			selector:  map[string]string{},
			wantError: false,
		},
		{
			name:      "Valid selector",
			nodes:     "",
			selector:  map[string]string{"region": "us-west"},
			wantError: false,
		},
		{
			name:      "Both specified",
			nodes:     "node1",
			selector:  map[string]string{"key": "value"},
			wantError: true,
			errorMsg:  "cannot specify both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := NewNodeSelector()
			if tt.nodes != "" {
				ns.AddNodes(tt.nodes)
			}
			for k, v := range tt.selector {
				ns.AddSelector(k, v)
			}

			err := ns.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if err != nil && tt.errorMsg != "" {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message = %v, want to contain %v", err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

func TestNodeSelector_Count(t *testing.T) {
	ns := NewNodeSelector()
	
	if ns.Count() != 0 {
		t.Errorf("Expected count 0, got %d", ns.Count())
	}

	ns.AddNodes("node1,node2,node3")
	if ns.Count() != 3 {
		t.Errorf("Expected count 3, got %d", ns.Count())
	}
}

func TestNodeSelector_String(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*NodeSelector)
		expected string
	}{
		{
			name:     "Empty selector",
			setup:    func(ns *NodeSelector) {},
			expected: "empty selector",
		},
		{
			name: "With nodes",
			setup: func(ns *NodeSelector) {
				ns.AddNodes("node1,node2")
			},
			expected: "nodes:",
		},
		{
			name: "With selector",
			setup: func(ns *NodeSelector) {
				ns.AddSelector("region", "us-west")
			},
			expected: "selector:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := NewNodeSelector()
			tt.setup(ns)
			result := ns.String()
			if !contains(result, tt.expected) {
				t.Errorf("String() = %v, want to contain %v", result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}