/*
Copyright 2024 The KubeEdge Authors.

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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	klog "k8s.io/klog/v2"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func NewResetOptions() *common.ResetOptions {
	opts := &common.ResetOptions{}
	return opts
}

// RemoveContainers removes all Kubernetes-managed containers
func RemoveContainers(criSocketPath string) error {
	if criSocketPath == "" {
		var err error
		criSocketPath, err = utilruntime.DetectCRISocket()
		if err != nil {
			return fmt.Errorf("failed to get crisocket with err:%v", err)
		}
	}

	containerRuntime := utilruntime.NewContainerRuntime(criSocketPath)
	if err := containerRuntime.Connect(); err != nil {
		return err
	}
	// TODO: add defer after upgrade to specific k8s version
	// see https://github.com/kubeedge/kubeedge/pull/6536#discussion_r2508874149
	// defer containerRuntime.Close()

	containers, err := containerRuntime.ListKubeContainers()
	if err != nil {
		return err
	}

	return containerRuntime.RemoveContainers(containers)
}

func CleanDirectories(isEdgeNode bool) error {
	var dirToClean = []string{
		constants.KubeEdgePath,
		constants.KubeEdgeLogPath,
		common.KubeEdgeSocketPath,
		common.EdgeRootDir,
	}

	if isEdgeNode {
		if err := WaitForEdgeCoreToStop(); err != nil {
			return fmt.Errorf("skip cleaning up dirs: %w", err)
		}
		dirToClean = append(dirToClean, constants.DefaultRootDir)
	}

	for _, dir := range dirToClean {
		klog.V(2).Infof("remove dir %s", dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		remove := os.RemoveAll
		if isEdgeNode {
			dirMounted, err := UnmountBelow(dir)
			if err != nil {
				klog.Warningf("skip deleting dir %s, err: %v", dir, err)
				continue
			}
			if dirMounted {
				remove = phases.CleanDir
			}
		}
		if err := remove(dir); err != nil {
			klog.Warningf("failed to delete dir %s, err: %v", dir, err)
		}
	}
	return nil
}

var edgeCoreStopTimeout = 10 * time.Second

// WaitForEdgeCoreToStop waits for edgecore to exit, so that it cannot mount volumes while its dirs are cleaned up.
func WaitForEdgeCoreToStop() error {
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, edgeCoreStopTimeout, true,
		func(context.Context) (bool, error) {
			running, err := IsKubeEdgeProcessRunning(constants.KubeEdgeBinaryName)
			if err != nil {
				return false, fmt.Errorf("failed to check whether edgecore is running: %w", err)
			}
			return !running, nil
		})
	if wait.Interrupted(err) {
		return fmt.Errorf("edgecore is still running after %v", edgeCoreStopTimeout)
	}
	return err
}

func unmountBelow(dir string, readMounts func() ([]byte, error), unmount func(string) error) (bool, error) {
	info, err := os.Lstat(filepath.Clean(dir))
	if os.IsNotExist(err) || (err == nil && info.Mode()&os.ModeSymlink != 0) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if dir, err = filepath.EvalSymlinks(dir); err != nil {
		return false, err
	}
	if dir == "/" {
		return false, errors.New("refusing to clean the filesystem root")
	}

	for {
		procMounts, err := readMounts()
		if err != nil {
			return false, err
		}
		var dirMounted bool
		var below []string
		for _, mp := range parseMountPoints(string(procMounts)) {
			if mp == dir {
				dirMounted = true
			} else if strings.HasPrefix(mp, dir+"/") {
				below = append(below, mp)
			}
		}
		if len(below) == 0 {
			return dirMounted, nil
		}
		sort.SliceStable(below, func(i, j int) bool { return len(below[i]) > len(below[j]) })

		var errs []error
		for _, mp := range below {
			parent := filepath.Dir(mp)
			resolved, err := filepath.EvalSymlinks(parent)
			if err == nil && resolved != parent {
				err = fmt.Errorf("%s resolves to %s", parent, resolved)
			}
			if err == nil {
				err = unmount(mp)
			}
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to unmount %s: %w", mp, err))
			}
		}
		if len(errs) == len(below) {
			return dirMounted, errors.Join(errs...)
		}
	}
}

func parseMountPoints(procMounts string) []string {
	var mountPoints []string
	for _, line := range strings.Split(procMounts, "\n") {
		if fields := strings.Split(line, " "); len(fields) > 1 {
			mountPoints = append(mountPoints, unescapeOctal(fields[1]))
		}
	}
	return mountPoints
}

func unescapeOctal(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if c, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(c))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
