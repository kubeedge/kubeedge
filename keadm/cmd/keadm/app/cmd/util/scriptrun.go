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

package util

import (
	"fmt"
	"os"
	"os/exec"
)

func RunScript(scriptPath string) error {
	// Create a command to execute the script
	cmd := exec.Command("/bin/bash", scriptPath)

	// Redirect the script's standard output and standard error to the main program's output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the script and check for errors
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute script %s: %v", scriptPath, err)
	}

	return nil
}
