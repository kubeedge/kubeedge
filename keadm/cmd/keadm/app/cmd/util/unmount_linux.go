//go:build linux

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

package util

import (
	"os"

	"golang.org/x/sys/unix"
)

// UnmountBelow unmounts everything mounted below dir and reports whether dir itself is a mount point.
func UnmountBelow(dir string) (bool, error) {
	return unmountBelow(dir, func() ([]byte, error) {
		return os.ReadFile("/proc/mounts")
	}, func(path string) error {
		return unix.Unmount(path, unix.UMOUNT_NOFOLLOW)
	})
}
