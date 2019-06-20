/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

@CHANGELOG
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with reduced set of methods
Changes done are
1. Package edged got some functions from "k8s.io/kubernetes/pkg/kubelet/kubelet_getters.go"
and made some variant
*/

package edged

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/kubeedge/beehive/pkg/common/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/config"
	utilfile "k8s.io/kubernetes/pkg/util/file"
	"k8s.io/kubernetes/pkg/volume/util"
)

//constants for Kubelet
const (
	DefaultKubeletPluginsDirName = "plugins"
	DefaultKubeletVolumesDirName = "volumes"
	DefaultKubeletPodsDirName    = "pods"
)

// getRootDir returns the full path to the directory under which kubelet can
// store data.  These functions are useful to pass interfaces to other modules
// that may need to know where to write data without getting a whole kubelet
// instance.
func (e *edged) getRootDir() string {
	return e.rootDirectory
}

// getPluginsDir returns the full path to the directory under which plugin
// directories are created.  Plugins can use these directories for data that
// they need to persist.  Plugins should create subdirectories under this named
// after their own names.
func (e *edged) getPluginsDir() string {
	return path.Join(e.getRootDir(), DefaultKubeletPluginsDirName)
}

// getPluginDir returns a data directory name for a given plugin name.
// Plugins can use these directories to store data that they need to persist.
// For per-pod plugin data, see getPodPluginDir.
func (e *edged) getPluginDir(pluginName string) string {
	return path.Join(e.getPluginsDir(), pluginName)
}

// getPodsDir returns the full path to the directory under which pod
// directories are created.
func (e *edged) getPodsDir() string {
	return path.Join(e.getRootDir(), DefaultKubeletPodsDirName)
}

// getPodDir returns the full path to the per-pod directory for the pod with
// the given UID.
func (e *edged) getPodDir(podUID types.UID) string {
	// Backwards compat.  The "old" stuff should be removed before 1.0
	// release.  The thinking here is this:
	//     !old && !new = use new
	//     !old && new  = use new
	//     old && !new  = use old
	//     old && new   = use new (but warn)
	oldPath := path.Join(e.getRootDir(), string(podUID))
	oldExists := dirExists(oldPath)
	newPath := path.Join(e.getPodsDir(), string(podUID))
	newExists := dirExists(newPath)
	if oldExists && !newExists {
		return oldPath
	}
	if oldExists {
		log.LOGGER.Warnf("Data dir for pod %q exists in both old and new form, using new", podUID)
	}
	return newPath
}

// getPodVolumesDir returns the full path to the per-pod data directory under
// which volumes are created for the specified pod.  This directory may not
// exist if the pod does not exist.
func (e *edged) getPodVolumesDir(podUID types.UID) string {
	return path.Join(e.getPodDir(podUID), DefaultKubeletVolumesDirName)
}

// getPodVolumeDir returns the full path to the directory which represents the
// named volume under the named plugin for specified pod.  This directory may not
// exist if the pod does not exist.
func (e *edged) getPodVolumeDir(podUID types.UID, pluginName string, volumeName string) string {
	return path.Join(e.getPodVolumesDir(podUID), pluginName, volumeName)
}

// dirExists returns true if the path exists and represents a directory.
func dirExists(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// getPodPluginDir returns a data directory name for a given plugin name for a
// given pod UID.  Plugins can use these directories to store data that they
// need to persist.  For non-per-pod plugin data, see getPluginDir.
func (e *edged) getPodPluginDir(podUID types.UID, pluginName string) string {
	return path.Join(e.getPodPluginsDir(podUID), pluginName)
}

// getPodPluginsDir returns the full path to the per-pod data directory under
// which plugins may store data for the specified pod.  This directory may not
// exist if the pod does not exist.
func (e *edged) getPodPluginsDir(podUID types.UID) string {
	return path.Join(e.getPodDir(podUID), DefaultKubeletPluginsDirName)
}

// getPodVolumePathListFromDisk returns a list of the volume paths by reading the
// volume directories for the given pod from the disk.
func (e *edged) getPodVolumePathListFromDisk(podUID types.UID) ([]string, error) {
	volumes := []string{}
	podVolDir := e.getPodVolumesDir(podUID)

	if pathExists, pathErr := util.PathExists(podVolDir); pathErr != nil {
		return volumes, fmt.Errorf("Error checking if path %q exists: %v", podVolDir, pathErr)
	} else if !pathExists {
		log.LOGGER.Warnf("Path %q does not exist", podVolDir)
		return volumes, nil
	}

	volumePluginDirs, err := ioutil.ReadDir(podVolDir)
	if err != nil {
		log.LOGGER.Errorf("Could not read directory %s: %v", podVolDir, err)
		return volumes, err
	}
	for _, volumePluginDir := range volumePluginDirs {
		volumePluginName := volumePluginDir.Name()
		volumePluginPath := filepath.Join(podVolDir, volumePluginName)
		volumeDirs, err := utilfile.ReadDirNoStat(volumePluginPath)
		if err != nil {
			return volumes, fmt.Errorf("Could not read directory %s: %v", volumePluginPath, err)
		}
		for _, volumeDir := range volumeDirs {
			volumes = append(volumes, filepath.Join(volumePluginPath, volumeDir))
		}
	}
	return volumes, nil
}

// GetPodDir returns the full path to the per-pod data directory for the
// specified pod. This directory may not exist if the pod does not exist.
func (e *edged) GetPodDir(podUID types.UID) string {
	return e.getPodDir(podUID)
}

// GetExtraSupplementalGroupsForPod returns a list of the extra
// supplemental groups for the Pod. These extra supplemental groups come
// from annotations on persistent volumes that the pod depends on.
func (e *edged) GetExtraSupplementalGroupsForPod(pod *v1.Pod) []int64 {
	return e.volumeManager.GetExtraSupplementalGroupsForPod(pod)
}

func (e *edged) getPodContainerDir(podUID types.UID, ctrName string) string {
	return filepath.Join(e.getPodDir(podUID), config.DefaultKubeletContainersDirName, ctrName)
}
