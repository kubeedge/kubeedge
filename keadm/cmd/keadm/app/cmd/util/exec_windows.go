//go:build windows

package util

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// Command defines commands to be executed and captures std out and std error
type Command struct {
	Cmd      *exec.Cmd
	StdOut   []byte
	StdErr   []byte
	ExitCode int
}

func NewCommand(command string) *Command {
	return &Command{
		Cmd: exec.Command("powershell", "-c", command),
	}
}

// Exec run command and exit formatted error, callers can print err directly
// Any running error or non-zero exitcode is consider as error
func (cmd *Command) Exec() error {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Cmd.Stdout = &stdoutBuf
	cmd.Cmd.Stderr = &stderrBuf

	errString := fmt.Sprintf("failed to exec '%s'", cmd.GetCommand())

	err := cmd.Cmd.Start()
	if err != nil {
		errString = fmt.Sprintf("%s, err: %v", errString, err)
		return errors.New(errString)
	}

	err = cmd.Cmd.Wait()
	if err != nil {
		cmd.StdErr = stderrBuf.Bytes()

		if exit, ok := err.(*exec.ExitError); ok {
			cmd.ExitCode = exit.Sys().(syscall.WaitStatus).ExitStatus()
			errString = fmt.Sprintf("%s, err: %s", errString, stderrBuf.Bytes())
		} else {
			cmd.ExitCode = 1
		}

		errString = fmt.Sprintf("%s, err: %v", errString, err)

		return errors.New(errString)
	}

	cmd.StdOut, cmd.StdErr = stdoutBuf.Bytes(), stderrBuf.Bytes()
	return nil
}

func (cmd Command) GetCommand() string {
	return strings.Join(cmd.Cmd.Args, " ")
}

func (cmd Command) GetStdOut() string {
	if len(cmd.StdOut) != 0 {
		return strings.TrimSuffix(ConvertByte2String(cmd.StdOut, "GB18030"), "\n")
	}
	return ""
}

func (cmd Command) GetStdErr() string {
	if len(cmd.StdErr) != 0 {
		return strings.TrimSuffix(ConvertByte2String(cmd.StdErr, "GB18030"), "\n")
	}
	return ""
}

func ConvertByte2String(byte []byte, charset string) string {
	var str string
	switch charset {
	case "GB18030":
		var decodeBytes, _ = simplifiedchinese.GB18030.NewDecoder().Bytes(byte)
		str = string(decodeBytes)
	case "UTF8":
		fallthrough
	default:
		str = string(byte)
	}
	return str
}
