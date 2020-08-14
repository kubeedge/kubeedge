package sysutil

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

var curShell string

// CurrentShell get current used shell env file. eg "/bin/zsh" "/bin/bash"
func CurrentShell(onlyName bool) (path string) {
	var err error
	if curShell == "" {
		path, err = ShellExec("echo $SHELL")
		if err != nil {
			return ""
		}

		path = strings.TrimSpace(path)
		// cache result
		curShell = path
	} else {
		path = curShell
	}

	if onlyName && len(path) > 0 {
		path = filepath.Base(path)
	}
	return
}

// HasShellEnv has shell env check.
// Usage:
// 	HasShellEnv("sh")
// 	HasShellEnv("bash")
func HasShellEnv(shell string) bool {
	// can also use: "echo $0"
	out, err := ShellExec("echo OK", shell)
	if err != nil {
		return false
	}

	return strings.TrimSpace(out) == "OK"
}

// IsWin system. linux windows darwin
func IsWin() bool {
	return runtime.GOOS == "windows"
}

// IsWindows system. linux windows darwin
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMac system
func IsMac() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux system
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsMSys msys(MINGW64) env，不一定支持颜色
func IsMSys() bool {
	// "MSYSTEM=MINGW64"
	if len(os.Getenv("MSYSTEM")) > 0 {
		return true
	}

	return false
}

// IsConsole check out is in stderr/stdout/stdin
func IsConsole(out io.Writer) bool {
	o, ok := out.(*os.File)
	if !ok {
		return false
	}

	fd := o.Fd()

	// fix: cannot use 'o == os.Stdout' to compare
	return fd == uintptr(syscall.Stdout) || fd == uintptr(syscall.Stdin) || fd == uintptr(syscall.Stderr)
}
