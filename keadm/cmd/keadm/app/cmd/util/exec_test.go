package util

import (
	"testing"
)

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
