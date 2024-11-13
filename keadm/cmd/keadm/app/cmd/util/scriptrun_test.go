package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunScript(t *testing.T) {
	// Create a temporary directory to store script files
	tmpDir := t.TempDir()

	// Test Case 1: Script executes successfully
	t.Run("ScriptExecutesSuccessfully", func(t *testing.T) {
		// Create a temporary script file
		scriptPath := filepath.Join(tmpDir, "success.sh")
		scriptContent := "#!/bin/bash\necho 'Hello, World!'\n"
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0700); err != nil {
			t.Fatalf("failed to create script: %v", err)
		}

		// Run the test
		err := RunScript(scriptPath)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	// Test Case 2: Script does not exist
	t.Run("ScriptDoesNotExist", func(t *testing.T) {
		// Define a path that does not exist
		invalidPath := filepath.Join(tmpDir, "non_existent.sh")
		err := RunScript(invalidPath)
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	// Test Case 3: Script without execution permission
	t.Run("ScriptWithoutExecutionPermission", func(t *testing.T) {
		// Create a temporary script file without execution permissions
		scriptPath := filepath.Join(tmpDir, "no_exec.sh")
		scriptContent := "#!/bin/bash\necho 'No execution permission'\n"
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0000); err != nil {
			t.Fatalf("failed to create script: %v", err)
		}

		// Run the test
		err := RunScript(scriptPath)
		if err == nil {
			t.Error("expected an error due to lack of execute permission, got nil")
		}
	})
}
