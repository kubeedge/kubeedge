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

package idempotency

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

// idempotencyRecord is a file that is used to avoid mutually exclusive
// commands running at the same time. If the file exist, we don't allow to run
// mutually exclusive commands, we only allow run commands when the file not exist.
var idempotencyRecord = filepath.Join(constants.KubeEdgePath, "idempotency_record")

// Occupy creates a idempotencyRecord file to indicate that mutually exclusive commands are already running.
// If returns true, it means the file already exists or an error an error occurred.
func Occupy() (bool, error) {
	if IsOccupied() {
		return true, nil
	}
	if _, err := os.Create(idempotencyRecord); err != nil {
		return true, fmt.Errorf("failed to create idempotency_record file, err: %v", err)
	}
	return false, nil
}

// IsOccupied returns true if the idempotencyRecord file exists.
// This means that there is a mutually exclusive command running.
func IsOccupied() bool {
	return files.FileExists(idempotencyRecord)
}

// Release deletes the idempotencyRecord file. If the command uses Occupy(),
// this should be called when the command has finished running.
func Release() error {
	if err := os.Remove(idempotencyRecord); err != nil {
		return fmt.Errorf("failed to remove idempotency_record file, err: %v", err)
	}
	return nil
}
