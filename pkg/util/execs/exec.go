package execs

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Command defines commands to be executed and captures std out and std error
type Command struct {
	Cmd      *exec.Cmd
	StdOut   []byte
	StdErr   []byte
	ExitCode int
}

// Exec run command and exit formatted error, callers can print err directly
// Any running error or non-zero exitcode is consider as error
func (cmd *Command) Exec() error {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Cmd.Stdout = &stdoutBuf
	cmd.Cmd.Stderr = &stderrBuf

	errString := fmt.Sprintf("failed to exec [%s]", cmd.GetCommand())

	if err := cmd.Cmd.Start(); err != nil {
		errString = fmt.Sprintf("%s, err: %v", errString, err)
		return errors.New(errString)
	}

	if err := cmd.Cmd.Wait(); err != nil {
		cmd.StdOut, cmd.StdErr = stdoutBuf.Bytes(), stderrBuf.Bytes()
		errString = cmd.handleWaitError(err, errString)
		return errors.New(errString)
	}

	cmd.StdOut, cmd.StdErr = stdoutBuf.Bytes(), stderrBuf.Bytes()
	return nil
}

// handleWaitError processes the error returned by cmd.Cmd.Wait() and returns
// a formatted error string. Extracted to allow direct unit testing of both
// the *exec.ExitError path and the generic error path.
func (cmd *Command) handleWaitError(err error, errString string) string {
	if exit, ok := err.(*exec.ExitError); ok {
		message := string(cmd.StdErr)
		if message == "" {
			message = string(cmd.StdOut)
		}
		cmd.ExitCode = exit.ExitCode()
		return fmt.Sprintf("%s, err: %s", errString, message)
	}
	cmd.ExitCode = 1
	return fmt.Sprintf("%s, err: %v", errString, err)
}

func (cmd Command) GetCommand() string {
	return strings.Join(cmd.Cmd.Args, " ")
}
