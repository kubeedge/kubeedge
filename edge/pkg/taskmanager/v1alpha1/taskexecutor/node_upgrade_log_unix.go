//go:build !windows

/*
Copyright 2026 The KubeEdge Authors.

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

package taskexecutor

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func openAndValidateUpgradeLogFile(logPath string) (*os.File, error) {
	fd, err := unix.Open(logPath, unix.O_CREAT|unix.O_WRONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(keadmUpgradeLogFilePerm))
	if err != nil {
		return nil, fmt.Errorf("open upgrade log file %s failed: %w", logPath, err)
	}
	logFile := os.NewFile(uintptr(fd), logPath)
	if logFile == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open upgrade log file %s failed: create file handle", logPath)
	}
	if err := ensureRegularOwnedFile(logFile); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("upgrade log file %s %w", logPath, err)
	}
	return logFile, nil
}

func ensureRegularOwnedFile(logFile *os.File) error {
	info, err := logFile.Stat()
	if err != nil {
		return fmt.Errorf("inspect file failed: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("must be a regular file")
	}
	return ensureOwnedByCurrentUser(info.Sys())
}

func ensureOwnedByCurrentUser(stat any) error {
	statT, ok := stat.(*syscall.Stat_t)
	if !ok {
		return errors.New("owner could not be determined")
	}
	if statT.Uid != uint32(os.Geteuid()) {
		return fmt.Errorf("must be owned by uid %d", os.Geteuid())
	}
	return nil
}

func infoOwner(path string) any {
	info, err := os.Lstat(path)
	if err != nil {
		return nil
	}
	return info.Sys()
}
