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

package tasklog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/kubeedge/api/apis/common/constants"
)

const (
	dirMode  os.FileMode = 0750
	fileMode os.FileMode = 0600
)

// OpenKeadmLog opens a Keadm task log in the KubeEdge-owned log directory.
func OpenKeadmLog(name string, flag int) (*os.File, error) {
	return OpenKeadmLogAt(constants.KubeEdgeLogPath, name, flag)
}

// OpenKeadmLogAt opens a Keadm task log under logDir.
func OpenKeadmLogAt(logDir, name string, flag int) (*os.File, error) {
	name = SafeName(name)
	if name == "" || name == "." {
		return nil, fmt.Errorf("invalid keadm log file name %q", name)
	}
	if err := os.MkdirAll(logDir, dirMode); err != nil {
		return nil, fmt.Errorf("failed to create keadm log directory: %w", err)
	}
	// MkdirAll does not change the mode of a directory that already exists,
	// so an existing dir created with permissive bits would otherwise stay that way.
	if err := os.Chmod(logDir, dirMode); err != nil {
		return nil, fmt.Errorf("failed to chmod keadm log directory: %w", err)
	}

	path := filepath.Join(logDir, name)
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("refuse to open symlinked keadm log file %s", path)
	} else if err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("refuse to open non-regular keadm log file %s", path)
		}
		// os.OpenFile only applies fileMode when creating a new file, so an existing
		// file with permissive bits needs an explicit chmod to be corrected.
		if err := os.Chmod(path, fileMode); err != nil {
			return nil, fmt.Errorf("failed to chmod keadm log file %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to inspect keadm log file %s: %w", path, err)
	}

	logFile, err := openFileNoFollow(path, os.O_CREATE|os.O_WRONLY|flag, fileMode)
	if err != nil {
		return nil, fmt.Errorf("failed to open keadm log file %s: %w", path, err)
	}
	return logFile, nil
}

// SafeName returns a path-safe filename component suitable for task log names.
func SafeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "._-")
}
