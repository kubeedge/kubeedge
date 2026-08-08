//go:build !windows

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

package extsystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemdExtSystem_ServiceCreateAndRemove(t *testing.T) {
	// Create a temporary directory to act as systemdDir
	tmpDir := t.TempDir()
	
	// Backup the original systemdDir and restore after test
	originalSystemdDir := systemdDir
	systemdDir = tmpDir
	defer func() {
		systemdDir = originalSystemdDir
	}()

	sysd := SystemdExtSystem{}
	serviceName := "testservice"
	binPath := "/usr/local/bin/testbin"
	args := []string{"--arg1", "val1"}
	envs := map[string]string{
		"ENV_VAR_1": "value1",
		"ENV_VAR_2": "value2",
	}

	// Test ServiceCreate
	err := sysd.ServiceCreate(serviceName, binPath, args, envs)
	if err != nil {
		t.Fatalf("failed to create systemd service: %v", err)
	}

	expectedFilePath := filepath.Join(tmpDir, serviceName+".service")
	if _, err := os.Stat(expectedFilePath); os.IsNotExist(err) {
		t.Fatalf("expected service file to be created at %s, but it does not exist", expectedFilePath)
	}

	// Read and verify file contents
	contentBytes, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("failed to read service file: %v", err)
	}
	content := string(contentBytes)

	// Verify content contains expected binary path and arguments
	if !strings.Contains(content, "ExecStart=/usr/local/bin/testbin --arg1 val1") {
		t.Errorf("service file content does not contain expected ExecStart: %s", content)
	}

	// Verify content contains environment variables
	if !strings.Contains(content, "Environment=") {
		t.Errorf("service file content does not contain Environment section: %s", content)
	}
	if !strings.Contains(content, "ENV_VAR_1=value1") {
		t.Errorf("service file content does not contain ENV_VAR_1: %s", content)
	}
	if !strings.Contains(content, "ENV_VAR_2=value2") {
		t.Errorf("service file content does not contain ENV_VAR_2: %s", content)
	}

	// Test ServiceRemove
	err = sysd.ServiceRemove(serviceName)
	if err != nil {
		t.Fatalf("failed to remove systemd service: %v", err)
	}

	if _, err := os.Stat(expectedFilePath); !os.IsNotExist(err) {
		t.Errorf("expected service file to be removed at %s, but it still exists", expectedFilePath)
	}
}

func TestOpenRCExtSystem_ServiceCreateAndRemove(t *testing.T) {
	openrc := OpenRCExtSystem{}
	
	// ServiceCreate and ServiceRemove are currently stubs for OpenRC,
	// verifying that calling them doesn't panic and returns nil.
	err := openrc.ServiceCreate("testservice", "/usr/local/bin/testbin", nil, nil)
	if err != nil {
		t.Errorf("expected ServiceCreate to return nil, got: %v", err)
	}

	err = openrc.ServiceRemove("testservice")
	if err != nil {
		t.Errorf("expected ServiceRemove to return nil, got: %v", err)
	}
}
