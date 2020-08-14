// +build !windows,!darwin

package process

import (
	"os"
	"path/filepath"
	"strconv"
)

// Exists check process running by given pid
func Exists(pid int) bool {
	if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err == nil {
		return true
	}
	return false
}
