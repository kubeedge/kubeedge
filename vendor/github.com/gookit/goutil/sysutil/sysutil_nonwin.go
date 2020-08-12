// +build !windows

package sysutil

import "syscall"

// Kill process by pid
func Kill(pid int, signal syscall.Signal) error {
	return syscall.Kill(pid, signal)
}

// ProcessExists check process exists by pid
func ProcessExists(pid int) bool {
	return nil == syscall.Kill(pid, 0)
}
