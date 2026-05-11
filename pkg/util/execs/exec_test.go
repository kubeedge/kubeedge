//go:build !windows

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

package execs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_HELPER_PROCESS") == "1" {
		switch os.Getenv("GO_HELPER_ACTION") {
		case "exit1_stderr":
			_, _ = fmt.Fprintln(os.Stderr, "helper stderr message")
			os.Exit(1)
		case "exit1_stdout_only":
			_, _ = fmt.Fprintln(os.Stdout, "helper stdout message")

			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// helperCmd returns a *Command that re-invokes the test binary as a helper
// subprocess that performs the given action.
func helperCmd(t *testing.T, action string) *Command {
	t.Helper()
	cmd := &Command{
		Cmd: exec.Command(os.Args[0], "-test.run=^TestMain$"),
	}
	cmd.Cmd.Env = append(os.Environ(),
		"GO_HELPER_PROCESS=1",
		"GO_HELPER_ACTION="+action,
	)
	return cmd
}

func TestCommand_GetStdOut(t *testing.T) {
	cases := []struct {
		name     string
		cmd      Command
		expected string
	}{
		{
			name:     "case1",
			cmd:      Command{StdOut: []byte("success\n")},
			expected: "success",
		},
		{
			name:     "case2",
			cmd:      Command{StdOut: nil},
			expected: "",
		},
	}
	for _, c := range cases {
		out := c.cmd.GetStdOut()
		if out != c.expected {
			t.Errorf("expected %v, got %v", c.expected, out)
		}
	}
}

func TestCommand_GetStdErr(t *testing.T) {
	cases := []struct {
		name     string
		cmd      Command
		expected string
	}{
		{
			name:     "case1",
			cmd:      Command{StdErr: []byte("failed\n")},
			expected: "failed",
		},
		{
			name:     "case2",
			cmd:      Command{StdErr: nil},
			expected: "",
		},
	}
	for _, c := range cases {
		out := c.cmd.GetStdErr()
		if out != c.expected {
			t.Errorf("%v: expected %v, got %v", c.name, c.expected, out)
		}
	}
}

func TestExecutorNoArgs(t *testing.T) {
	cmd := NewCommand("true")
	err := cmd.Exec()
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
	if len(cmd.StdOut) != 0 || len(cmd.StdErr) != 0 {
		t.Errorf("expected no output, got stdout: %q, stderr: %q", string(cmd.StdOut), string(cmd.StdErr))
	}

	cmd = NewCommand("false")
	err = cmd.Exec()
	if err == nil {
		t.Errorf("expected failure, got nil error")
	}
	if len(cmd.StdOut) != 0 || len(cmd.StdErr) != 0 {
		t.Errorf("expected no output, got stdout: %q, stderr: %q", string(cmd.StdOut), string(cmd.StdErr))
	}
	if cmd.ExitCode != 1 {
		t.Errorf("expected exit status 1, got %d", cmd.ExitCode)
	}
}

func TestExecutorWithArgs(t *testing.T) {
	cmd := NewCommand("echo stdout")
	if err := cmd.Exec(); err != nil {
		t.Errorf("expected success, got %+v", err)
	}
	if string(cmd.StdOut) != "stdout\n" {
		t.Errorf("unexpected output: %q", string(cmd.StdOut))
	}

	cmd = NewCommand("echo stderr > /dev/stderr")
	if err := cmd.Exec(); err != nil {
		t.Errorf("expected success, got %+v", err)
	}
	if string(cmd.StdErr) != "stderr\n" {
		t.Errorf("unexpected output: %q", string(cmd.StdErr))
	}
}

func TestExecutableNotFound(t *testing.T) {
	cmd := NewCommand("fake_executable_name")
	err := cmd.Exec()
	if err == nil {
		t.Errorf("expected err, got nil")
	}
}

func TestGetCommand(t *testing.T) {
	cmd := NewCommand("echo hello world")
	got := cmd.GetCommand()
	want := "bash -c echo hello world"
	if got != want {
		t.Errorf("GetCommand() = %q, want %q", got, want)
	}
}

// TestExec_ExitError_StderrNonEmpty covers the *exec.ExitError branch where
// StdErr is non-empty and is used as the error message (not StdOut).
func TestExec_ExitError_StderrNonEmpty(t *testing.T) {
	cmd := helperCmd(t, "exit1_stderr")
	err := cmd.Exec()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if cmd.ExitCode == 0 {
		t.Errorf("expected non-zero ExitCode, got 0")
	}
	if len(cmd.StdErr) == 0 {
		t.Errorf("expected StdErr to be populated")
	}
	if !strings.Contains(err.Error(), "failed to exec") {
		t.Errorf("unexpected error prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "helper stderr message") {
		t.Errorf("expected stderr content in error message, got: %v", err)
	}
}

// TestExec_ExitError_StderrEmpty_FallsBackToStdout covers the *exec.ExitError
// branch where StdErr is empty so StdOut is used as the error message.
func TestExec_ExitError_StderrEmpty_FallsBackToStdout(t *testing.T) {
	cmd := helperCmd(t, "exit1_stdout_only")
	err := cmd.Exec()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if cmd.ExitCode == 0 {
		t.Errorf("expected non-zero ExitCode, got 0")
	}
	if len(cmd.StdErr) != 0 {
		t.Errorf("expected StdErr to be empty, got: %q", string(cmd.StdErr))
	}
	if !strings.Contains(err.Error(), "helper stdout message") {
		t.Errorf("expected StdOut content in error message, got: %v", err)
	}
}

func TestHandleWaitError_NonExitError(t *testing.T) {
	cmd := &Command{
		StdOut: []byte("some stdout"),
		StdErr: []byte{}, // empty so we can confirm ExitCode is set to 1
	}

	plainErr := errors.New("some non-exit-error")
	errString := "failed to exec [test]"

	result := cmd.handleWaitError(plainErr, errString)

	if cmd.ExitCode != 1 {
		t.Errorf("expected ExitCode=1 for non-ExitError, got %d", cmd.ExitCode)
	}
	if !strings.Contains(result, "some non-exit-error") {
		t.Errorf("expected plain error text in result, got: %s", result)
	}
	if !strings.Contains(result, errString) {
		t.Errorf("expected original errString preserved, got: %s", result)
	}
}

func TestExec_StartError(t *testing.T) {
	cmd := &Command{
		Cmd: exec.Command("/this/binary/does/not/exist/kubeedge"),
	}
	err := cmd.Exec()
	if err == nil {
		t.Fatal("expected error from Start(), got nil")
	}
	if !strings.Contains(err.Error(), "failed to exec") {
		t.Errorf("unexpected error format: %v", err)
	}
}
