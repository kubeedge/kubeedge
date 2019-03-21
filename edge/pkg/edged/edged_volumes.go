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
1. Most functions in this file is come from "k8s.io/kubernetes/pkg/kubelet/kubelet_volumes.go"
   and made some variant.
*/

package edged

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/kubelet/container"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	utilio "k8s.io/kubernetes/pkg/util/io"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/util/removeall"
	"k8s.io/kubernetes/pkg/volume"
	volumetypes "k8s.io/kubernetes/pkg/volume/util/types"

	"github.com/kubeedge/beehive/pkg/common/log"
)

// newVolumeMounterFromPlugins attempts to find a plugin by volume spec, pod
// and volume options and then creates a Mounter.
// Returns a valid Unmounter or an error.
func (e *edged) newVolumeMounterFromPlugins(spec *volume.Spec, pod *api.Pod, opts volume.VolumeOptions) (volume.Mounter, error) {
	plugin, err := e.volumePluginMgr.FindPluginBySpec(spec)
	if err != nil {
		return nil, fmt.Errorf("can't use volume plugins for %s: %v", spec.Name(), err)
	}

	physicalMounter, err := plugin.NewMounter(spec, pod, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate mounter for volume: %s using plugin: %s with a root cause: %v", spec.Name(), plugin.GetPluginName(), err)
	}
	log.LOGGER.Infof("Using volume plugin %q to mount %s", plugin.GetPluginName(), spec.Name())
	return physicalMounter, nil
}

// cleanupOrphanedPodDirs removes the volumes of pods that should not be
// running and that have no containers running.
func (e *edged) cleanupOrphanedPodDirs(pods []*api.Pod, containerRunningPods []*container.Pod) error {

	allPods := sets.NewString()
	for _, pod := range pods {
		allPods.Insert(string(pod.UID))
	}

	for _, pod := range containerRunningPods {
		allPods.Insert(string(pod.ID))
	}

	found, err := e.listPodsFromDisk()
	if err != nil {
		return err
	}
	orphanRemovalErrors := []error{}
	orphanVolumeErrors := []error{}

	for _, uid := range found {
		if allPods.Has(string(uid)) {
			continue
		}
		// If volumes have not been unmounted/detached, do not delete directory.
		// Doing so may result in corruption of data.
		if podVolumesExist := e.podVolumesExist(uid); podVolumesExist {
			log.LOGGER.Infof("Orphaned pod %q found, but volumes are not cleaned up", uid)
			continue
		}
		// If there are still volume directories, do not delete directory
		volumePaths, err := e.getPodVolumePathListFromDisk(uid)
		if err != nil {
			orphanVolumeErrors = append(orphanVolumeErrors, fmt.Errorf("Orphaned pod %q found, but error %v occurred during reading volume dir from disk", uid, err))
			continue
		}
		if len(volumePaths) > 0 {
			orphanVolumeErrors = append(orphanVolumeErrors, fmt.Errorf("Orphaned pod %q found, but volume paths are still present on disk", uid))
			continue
		}

		// TODO: by paas group
		// k8s has no cleanupMoutPoints, what the purpose of this method?
		// can be contributed?
		log.LOGGER.Infof("Clearing up volume directories of orphaned pod %q.", uid)
		if err := e.cleanupMountPoints(e.getPodVolumesDir(uid)); err != nil {
			log.LOGGER.Warnf("Failed to clearing up volume mount points of pod %q: %s.", uid, err)
		}
		log.LOGGER.Infof("Orphaned pod %q found, removing", uid)
		if err := removeall.RemoveAllOneFilesystem(e.mounter, e.getPodDir(uid)); err != nil {
			log.LOGGER.Errorf("Failed to remove orphaned pod %q dir; err: %v", uid, err)
			orphanRemovalErrors = append(orphanRemovalErrors, err)
		}
	}

	logSpew := func(errs []error) {
		if len(errs) > 0 {
			log.LOGGER.Errorf("%v : There were a total of %v errors similar to this. Turn up verbosity to see them.", errs[0], len(errs))
			for _, err := range errs {
				log.LOGGER.Infof("Orphan pod: %v", err)
			}
		}
	}

	logSpew(orphanVolumeErrors)
	logSpew(orphanRemovalErrors)
	return utilerrors.NewAggregate(orphanRemovalErrors)
}

// cleanMountPoints traverses the volume directory to find out all mount points and umount them
func (e *edged) cleanupMountPoints(volumeDir string) error {
	mounter := mount.New("")
	mountPoints, err := listProcMounts(volumeDir)
	if err != nil {
		return err
	}
	errlist := []error{}
	for _, mp := range mountPoints {
		if err := mounter.Unmount(mp.Path); err != nil {
			errlist = append(errlist, err)
		}
	}
	if len(errlist) != 0 {
		return utilerrors.NewAggregate(errlist)
	}
	return nil
}

// podVolumesExist checks with the volume manager and returns true any of the
// pods for the specified volume are mounted.
func (e *edged) podVolumesExist(podUID types.UID) bool {
	if mountedVolumes :=
		e.volumeManager.GetMountedVolumesForPod(
			volumetypes.UniquePodName(podUID)); len(mountedVolumes) > 0 {
		return true
	}

	return false
}

// listProcMounts is come from k8s.io\kubernetes\pkg\util\mount.listProcMounts
func listProcMounts(mountFilePath string) ([]mount.MountPoint, error) {
	listTryTime, expectedNumFieldsPerLine := 3, 6
	content, err := utilio.ConsistentRead(mountFilePath, listTryTime)
	if err != nil {
		return nil, err
	}

	out := []mount.MountPoint{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			// the last split() item is empty string following the last \n
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != expectedNumFieldsPerLine {
			return nil, fmt.Errorf("wrong number of fields (expected %d, got %d): %s", expectedNumFieldsPerLine, len(fields), line)
		}

		mp := mount.MountPoint{
			Device: fields[0],
			Path:   fields[1],
			Type:   fields[2],
			Opts:   strings.Split(fields[3], ","),
		}

		freq, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}
		mp.Freq = freq

		pass, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, err
		}
		mp.Pass = pass

		out = append(out, mp)
	}
	return out, nil
}
