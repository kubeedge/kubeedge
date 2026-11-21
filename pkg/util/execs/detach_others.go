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
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

var (
	// this exists only when systemd is running an not just installed
	systemdRunDir = "/run/systemd/system"
	lookPath      = exec.LookPath
)

// sends output to the log file
const logWrapperScript = `log="$1"; shift; exec "$@" >"$log" 2>&1`

func systemdAvailable() bool {
	if _, err := os.Stat(systemdRunDir); err != nil {
		return false
	}
	_, err := lookPath("systemd-run")
	return err == nil
}

// this returns the systemd run arguments for the command
func BuildSystemdRunArgs(unitName, logFile, name string, args []string) []string {
	runArgs := []string{
		"--unit=" + unitName,
		"--collect",
		"--quiet",
	}

	if logFile == "" {
		runArgs = append(runArgs, "--", name)
		return append(runArgs, args...)
	}

	runArgs = append(runArgs, "--", "/bin/sh", "-c", logWrapperScript, "sh", logFile, name)
	return append(runArgs, args...)
}

func startInBackground(logFile, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return fmt.Errorf("failed to open log file %s: %v", logFile, err)
		}
		defer f.Close()
		cmd.Stdout = f
		cmd.Stderr = f
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %v", name, err)
	}
	return cmd.Process.Release()
}

// this runs the command in its own systemd unit
func StartDetached(unitName, logFile, name string, args ...string) error {
	if systemdAvailable() {
		cmd := exec.Command("systemd-run", BuildSystemdRunArgs(unitName, logFile, name, args)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to start %s with systemd-run: %v, output: %s", name, err, out)
		}
		return nil
	}
	return startInBackground(logFile, name, args...)
}
