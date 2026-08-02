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
	"os"
	"path/filepath"
	"testing"
)

func TestOpenKeadmLogAtCreatesPrivateLog(t *testing.T) {
	dir := t.TempDir()
	logFile, err := OpenKeadmLogAt(dir, "keadm-upgrade-job-1.log", os.O_TRUNC)
	if err != nil {
		t.Fatalf("OpenKeadmLogAt() error = %v", err)
	}
	if _, err := logFile.WriteString("test"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "keadm-upgrade-job-1.log"))
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != fileMode {
		t.Fatalf("log mode = %o, want %o", got, fileMode)
	}
}

func TestOpenKeadmLogAtTightensExistingPermissions(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0777); err != nil {
		t.Fatalf("Chmod(dir) error = %v", err)
	}
	name := "keadm-upgrade-job-1.log"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("stale"), 0666); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	logFile, err := OpenKeadmLogAt(dir, name, os.O_APPEND)
	if err != nil {
		t.Fatalf("OpenKeadmLogAt() error = %v", err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(dir) error = %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != dirMode {
		t.Fatalf("existing dir mode = %o, want %o", got, dirMode)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(file) error = %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != fileMode {
		t.Fatalf("existing file mode = %o, want %o", got, fileMode)
	}
}

func TestOpenKeadmLogAtRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	link := filepath.Join(dir, "keadm-upgrade-job-1.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	logFile, err := OpenKeadmLogAt(dir, filepath.Base(link), os.O_APPEND)
	if err == nil {
		_ = logFile.Close()
		t.Fatal("OpenKeadmLogAt() succeeded for a symlink")
	}
}

func TestOpenKeadmLogAtRejectsNonRegularFile(t *testing.T) {
	dir := t.TempDir()
	name := "keadm-upgrade-job-1.log"
	subdir := filepath.Join(dir, name)
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	logFile, err := OpenKeadmLogAt(dir, name, os.O_APPEND)
	if err == nil {
		_ = logFile.Close()
		t.Fatal("OpenKeadmLogAt() succeeded for a directory")
	}
}

func TestSafeName(t *testing.T) {
	got := SafeName("../job/upgrade 1:$history.log")
	want := "job_upgrade_1__history.log"
	if got != want {
		t.Fatalf("SafeName() = %q, want %q", got, want)
	}
}
