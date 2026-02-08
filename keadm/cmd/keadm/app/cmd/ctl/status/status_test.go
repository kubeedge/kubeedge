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
	"testing"
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
