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

package execs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildSystemdRunArgs(t *testing.T) {
	tests := []struct {
		name    string
		logFile string
		want    []string
	}{
		{
			name:    "with log file",
			logFile: "/tmp/keadm.log",
			want: []string{
				"--unit=keadm-upgrade-1", "--collect", "--quiet", "--",
				"/bin/sh", "-c", logWrapperScript, "sh", "/tmp/keadm.log",
				"keadm", "edge", "upgrade", "--toVersion", "v1.23.0",
			},
		},
		{
			name:    "without log file",
			logFile: "",
			want: []string{
				"--unit=keadm-upgrade-1", "--collect", "--quiet", "--",
				"keadm", "edge", "upgrade", "--toVersion", "v1.23.0",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSystemdRunArgs("keadm-upgrade-1", tt.logFile, "keadm",
				[]string{"edge", "upgrade", "--toVersion", "v1.23.0"})
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildSystemdRunArgs() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestLogWrapperScript(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "out.log")
	cmd := exec.Command("/bin/sh", "-c", logWrapperScript, "sh", logFile,
		"echo", "a b", "$HOME", ";rm -rf /")
	if err := cmd.Run(); err != nil {
		t.Fatalf("running wrapper: %v", err)
	}
	got, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	if want := "a b $HOME ;rm -rf /\n"; string(got) != want {
		t.Errorf("log content = %q, want %q", got, want)
	}
}

func TestSystemdAvailable(t *testing.T) {
	origDir, origLookPath := systemdRunDir, lookPath
	t.Cleanup(func() { systemdRunDir, lookPath = origDir, origLookPath })

	found := func(string) (string, error) { return "/usr/bin/systemd-run", nil }
	notFound := func(string) (string, error) { return "", errors.New("not found") }

	tests := []struct {
		name     string
		runDir   string
		lookPath func(string) (string, error)
		want     bool
	}{
		{"systemd running and binary present", t.TempDir(), found, true},
		{"systemd not running", filepath.Join(t.TempDir(), "missing"), found, false},
		{"binary missing", t.TempDir(), notFound, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			systemdRunDir, lookPath = tt.runDir, tt.lookPath
			if got := systemdAvailable(); got != tt.want {
				t.Errorf("systemdAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartDetachedFallback(t *testing.T) {
	origDir := systemdRunDir
	t.Cleanup(func() { systemdRunDir = origDir })
	systemdRunDir = filepath.Join(t.TempDir(), "missing")

	logFile := filepath.Join(t.TempDir(), "keadm.log")
	if err := StartDetached("unused", logFile, "echo", "hello"); err != nil {
		t.Fatalf("StartDetached() error = %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		got, _ := os.ReadFile(logFile)
		if strings.TrimSpace(string(got)) == "hello" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("log content = %q, want %q", got, "hello")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestStartDetachedFallbackBadLogFile(t *testing.T) {
	origDir := systemdRunDir
	t.Cleanup(func() { systemdRunDir = origDir })
	systemdRunDir = filepath.Join(t.TempDir(), "missing")

	err := StartDetached("unused", filepath.Join(t.TempDir(), "no", "such", "dir", "x.log"), "echo")
	if err == nil {
		t.Fatal("expected error for unwritable log file but got nil")
	}
}
