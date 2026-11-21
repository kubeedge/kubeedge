/*
Copyright 2025 The KubeEdge Authors.

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

package edge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"k8s.io/klog/v2"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	edgecoreutil "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/edgecore"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/idempotency"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

const (
	unknownEdgeCoreVersion = "unknown"
	stagedFileSuffix       = ".keadm-tmp"
)

var OccupiedError = errors.New("the mutually exclusive command is being executed")

type baseUpgradeExecutor struct {
	currentVersion string

	cfg *cfgv1alpha2.EdgeCoreConfig
}

func (executor *baseUpgradeExecutor) prePreRun(configpath string) error {
	ctx := context.Background()
	// Set the default value of currentVersion to unknown.
	executor.currentVersion = unknownEdgeCoreVersion

	// Determine whether there are mutually exclusive commands running. If not,
	// it occupies the resource and prevents other mutually exclusive commands from
	// running at the same time. Otherwise, return an error.
	occupied, err := idempotency.Occupy()
	if err != nil {
		return fmt.Errorf("failed to occupy command execution, err: %v", err)
	}
	if occupied {
		return OccupiedError
	}

	// Parse the edgecore config file.
	cfg, err := util.ParseEdgecoreConfig(configpath)
	if err != nil {
		return fmt.Errorf("failed to parse the edgecore config file %s, err: %v",
			configpath, err)
	}
	executor.cfg = cfg

	// Get the current version of edgecore.
	if ver := edgecoreutil.GetVersion(ctx, cfg); ver != "" {
		executor.currentVersion = ver
	}
	return nil
}

func (executor *baseUpgradeExecutor) postPreRun(prerunHook string) error {
	// If pre-run script is specified, execute it.
	if prerunHook != "" {
		fmt.Printf("Executing pre-run script: %s\n", prerunHook)
		if err := util.RunScript(prerunHook); err != nil {
			return fmt.Errorf("failed to run the pre-run script %s, err: %v", prerunHook, err)
		}
	}
	return nil
}

func (executor *baseUpgradeExecutor) release() {
	// Release the occupied resource.
	if idempotency.IsOccupied() {
		if err := idempotency.Release(); err != nil {
			klog.Errorf("failed to release the occupied command execution, err: %v", err)
		}
	}
}

func (executor *baseUpgradeExecutor) runPostRunHook(postrunHook string) {
	// If post-run script is specified, execute it.
	if postrunHook != "" {
		fmt.Printf("Executing post-run script: %s\n", postrunHook)
		if err := util.RunScript(postrunHook); err != nil {
			fmt.Printf("Execute post-run script: %s failed: %v\n", postrunHook, err)
		}
	}
}

// replaceFiles writes each source next to its destination before calling stop,
// so a failed write leaves edgecore running.
func replaceFiles(srcToDest map[string]string, stop func() error) error {
	type stagedFile struct{ src, tmp, dest string }
	var staged []stagedFile
	defer func() {
		for _, f := range staged {
			os.Remove(f.tmp)
		}
	}()
	for _, src := range slices.Sorted(maps.Keys(srcToDest)) {
		dest := srcToDest[src]
		if target, err := filepath.EvalSymlinks(dest); err == nil {
			dest = target
		} else if info, lerr := os.Lstat(dest); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("failed to resolve symlink %s, err: %v", dest, err)
		}
		f := stagedFile{src: src, tmp: dest + stagedFileSuffix, dest: dest}
		staged = append(staged, f)
		if err := stageFile(f.src, f.tmp, f.dest); err != nil {
			return fmt.Errorf("failed to copy %s to %s, err: %v", f.src, f.tmp, err)
		}
	}
	if err := stop(); err != nil {
		return fmt.Errorf("failed to stop edgecore, err: %v", err)
	}
	for _, f := range staged {
		err := os.Rename(f.tmp, f.dest)
		if err == nil {
			continue
		}
		// Renaming over a bind-mounted file fails, so copy in place as before.
		klog.Warningf("failed to rename %s to %s, copy %s instead, err: %v", f.tmp, f.dest, f.src, err)
		os.Remove(f.tmp)
		if err := files.FileCopy(f.src, f.dest); err != nil {
			return fmt.Errorf("failed to copy %s to %s, err: %v", f.src, f.dest, err)
		}
	}
	return nil
}

func stageFile(src, tmp, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("source file %s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	mode := srcInfo.Mode()
	destInfo, destErr := os.Stat(dest)
	if destErr == nil {
		mode = destInfo.Mode()
	}
	os.Remove(tmp)
	tmpFile, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	if _, err := io.Copy(tmpFile, source); err != nil {
		return err
	}
	if destErr == nil {
		// OpenFile applies the umask, so set the mode of dest again.
		if err := tmpFile.Chmod(mode); err != nil {
			return err
		}
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	return tmpFile.Close()
}
