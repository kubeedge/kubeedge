package sysutil

import (
	"bytes"
	"os/exec"

	"github.com/gookit/goutil/strutil"
)

// QuickExec quick exec an simple command line
// Usage:
//	QuickExec("git status")
func QuickExec(cmdLine string, workDir ...string) (string, error) {
	ss := strutil.Split(cmdLine, " ")

	return ExecCmd(ss[0], ss[1:], workDir...)
}

// ExecCmd an command and return output.
// Usage:
// 	ExecCmd("ls", []string{"-al"})
func ExecCmd(binName string, args []string, workDir ...string) (string, error) {
	// create a new Cmd instance
	cmd := exec.Command(binName, args...)
	if len(workDir) > 0 {
		cmd.Dir = workDir[0]
	}

	bs, err := cmd.Output()
	return string(bs), err
}

// ShellExec exec command by shell
// cmdStr eg. "ls -al"
func ShellExec(cmdStr string, shells ...string) (string, error) {
	// shell := "/bin/sh"
	shell := "sh"
	if len(shells) > 0 {
		shell = shells[0]
	}

	var out bytes.Buffer

	cmd := exec.Command(shell, "-c", cmdStr)
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return out.String(), nil
}
