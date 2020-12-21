package ingress

import (
	"k8s.io/klog"
	"os/exec"
	"syscall"
)

// IsRespawnIfRequired checks if error type is exec.ExitError or not
func IsRespawnIfRequired(err error) bool {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}

	waitStatus := exitError.Sys().(syscall.WaitStatus)
	klog.Warningf(`
-------------------------------------------------------------------------------
NGINX master process died (%v): %v
-------------------------------------------------------------------------------
`, waitStatus.ExitStatus(), err)
	return true
}

