// Package cliutil provides some util functions for CLI
package cliutil

import (
	"github.com/gookit/goutil/sysutil"
)

// QuickExec quick exec an simple command line
func QuickExec(cmdLine string, workDir ...string) (string, error) {
	return sysutil.QuickExec(cmdLine, workDir...)
}

// ExecCmd a CLI bin file and return output.
// Usage:
// 	ExecCmd("ls", []string{"-al"})
func ExecCmd(binName string, args []string, workDir ...string) (string, error) {
	return sysutil.ExecCmd(binName, args, workDir...)
}

// ExecCommand alias of the ExecCmd()
func ExecCommand(binName string, args []string, workDir ...string) (string, error) {
	return sysutil.ExecCmd(binName, args, workDir...)
}

// ShellExec exec command by shell
// cmdStr eg. "ls -al"
func ShellExec(cmdLine string, shells ...string) (string, error) {
	return sysutil.ShellExec(cmdLine, shells...)
}

// CurrentShell get current used shell env file. eg "/bin/zsh" "/bin/bash"
func CurrentShell(onlyName bool) (path string) {
	return sysutil.CurrentShell(onlyName)
}

// HasShellEnv has shell env check.
// Usage:
// 	HasShellEnv("sh")
// 	HasShellEnv("bash")
func HasShellEnv(shell string) bool {
	return sysutil.HasShellEnv(shell)
}
