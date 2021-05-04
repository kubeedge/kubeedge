/*
Copyright 2017 The Kubernetes Authors.

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
this file is derived from kubernetes v1.15.3,
and the full file path is k8s.io/kubernetes/pkg/volume/csi/csi_attacher.go
and make some modifications including:
1. empty Attach function
2. empty Detach function.
3. change VolumeAttachments Watch into Get in waitForVolumeAttachment.
4. change VolumeAttachments Watch into Get in waitForVolumeDetachment.
5. remove skipAttach reference.
*/

package csi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	storage "k8s.io/api/storage/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/volume"
)

const (
	persistentVolumeInGlobalPath = "pv"
	globalMountInGlobalPath      = "globalmount"
)

type csiAttacher struct {
	plugin        *csiPlugin
	k8s           kubernetes.Interface
	waitSleepTime time.Duration

	csiClient csiClient
}

// volume.Attacher methods
var _ volume.Attacher = &csiAttacher{}

var _ volume.Detacher = &csiAttacher{}

var _ volume.DeviceMounter = &csiAttacher{}

func (c *csiAttacher) Attach(spec *volume.Spec, nodeName types.NodeName) (string, error) {
	return "", nil
}

func (c *csiAttacher) WaitForAttach(spec *volume.Spec, _ string, pod *v1.Pod, timeout time.Duration) (string, error) {
	source, err := getPVSourceFromSpec(spec)
	if err != nil {
		klog.Error(log("attacher.WaitForAttach failed to extract CSI volume source: %v", err))
		return "", err
	}

	attachID := getAttachmentName(source.VolumeHandle, source.Driver, string(c.plugin.host.GetNodeName()))

	return c.waitForVolumeAttachment(source.VolumeHandle, attachID, timeout)
}

func (c *csiAttacher) waitForVolumeAttachment(volumeHandle, attachID string, timeout time.Duration) (string, error) {
	klog.V(4).Info(log("probing for updates from CSI driver for [attachment.ID=%v]", attachID))

	err := wait.PollImmediate(time.Second*5, time.Minute*5, func() (bool, error) {
		klog.V(4).Info(log("probing VolumeAttachment [id=%v]", attachID))
		attach, err := c.k8s.StorageV1().VolumeAttachments().Get(context.Background(), attachID, meta.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("volume %v has GET error for volume attachment %v: %v", volumeHandle, attachID, err)
		}
		successful, err := verifyAttachmentStatus(attach, volumeHandle)
		return successful, err
	})
	if err != nil {
		return "", err
	}
	return attachID, nil
}

func verifyAttachmentStatus(attachment *storage.VolumeAttachment, volumeHandle string) (bool, error) {
	// if being deleted, fail fast
	if attachment.GetDeletionTimestamp() != nil {
		klog.Error(log("VolumeAttachment [%s] has deletion timestamp, will not continue to wait for attachment", attachment.Name))
		return false, errors.New("volume attachment is being deleted")
	}
	// attachment OK
	if attachment.Status.Attached {
		return true, nil
	}
	// driver reports attach error
	attachErr := attachment.Status.AttachError
	if attachErr != nil {
		klog.Error(log("attachment for %v failed: %v", volumeHandle, attachErr.Message))
		return false, errors.New(attachErr.Message)
	}
	return false, nil
}

func (c *csiAttacher) VolumesAreAttached(specs []*volume.Spec, nodeName types.NodeName) (map[*volume.Spec]bool, error) {
	klog.V(4).Info(log("probing attachment status for %d volume(s) ", len(specs)))

	attached := make(map[*volume.Spec]bool)

	for _, spec := range specs {
		if spec == nil {
			klog.Error(log("attacher.VolumesAreAttached missing volume.Spec"))
			return nil, errors.New("missing spec")
		}
		pvSrc, err := getPVSourceFromSpec(spec)
		if err != nil {
			attached[spec] = false
			klog.Error(log("attacher.VolumesAreAttached failed to get CSIPersistentVolumeSource: %v", err))
			continue
		}
		driverName := pvSrc.Driver
		volumeHandle := pvSrc.VolumeHandle

		attachID := getAttachmentName(volumeHandle, driverName, string(nodeName))
		klog.V(4).Info(log("probing attachment status for VolumeAttachment %v", attachID))
		attach, err := c.k8s.StorageV1().VolumeAttachments().Get(context.Background(), attachID, meta.GetOptions{})
		if err != nil {
			attached[spec] = false
			klog.Error(log("attacher.VolumesAreAttached failed for attach.ID=%v: %v", attachID, err))
			continue
		}
		klog.V(4).Info(log("attacher.VolumesAreAttached attachment [%v] has status.attached=%t", attachID, attach.Status.Attached))
		attached[spec] = attach.Status.Attached
	}

	return attached, nil
}

func (c *csiAttacher) GetDeviceMountPath(spec *volume.Spec) (string, error) {
	klog.V(4).Info(log("attacher.GetDeviceMountPath(%v)", spec))
	deviceMountPath, err := makeDeviceMountPath(c.plugin, spec)
	if err != nil {
		klog.Error(log("attacher.GetDeviceMountPath failed to make device mount path: %v", err))
		return "", err
	}
	klog.V(4).Infof("attacher.GetDeviceMountPath succeeded, deviceMountPath: %s", deviceMountPath)
	return deviceMountPath, nil
}

func (c *csiAttacher) MountDevice(spec *volume.Spec, devicePath string, deviceMountPath string) (err error) {
	klog.V(4).Infof(log("attacher.MountDevice(%s, %s)", devicePath, deviceMountPath))

	if deviceMountPath == "" {
		err = fmt.Errorf("attacher.MountDevice failed, deviceMountPath is empty")
		return err
	}

	mounted, err := isDirMounted(c.plugin, deviceMountPath)
	if err != nil {
		klog.Error(log("attacher.MountDevice failed while checking mount status for dir [%s]", deviceMountPath))
		return err
	}

	if mounted {
		klog.V(4).Info(log("attacher.MountDevice skipping mount, dir already mounted [%s]", deviceMountPath))
		return nil
	}

	// Setup
	if spec == nil {
		return fmt.Errorf("attacher.MountDevice failed, spec is nil")
	}
	csiSource, err := getPVSourceFromSpec(spec)
	if err != nil {
		klog.Error(log("attacher.MountDevice failed to get CSIPersistentVolumeSource: %v", err))
		return err
	}

	// Store volume metadata for UnmountDevice. Keep it around even if the
	// driver does not support NodeStage, UnmountDevice still needs it.
	if err = os.MkdirAll(deviceMountPath, 0750); err != nil {
		klog.Error(log("attacher.MountDevice failed to create dir %#v:  %v", deviceMountPath, err))
		return err
	}
	klog.V(4).Info(log("created target path successfully [%s]", deviceMountPath))
	dataDir := filepath.Dir(deviceMountPath)
	data := map[string]string{
		volDataKey.volHandle:  csiSource.VolumeHandle,
		volDataKey.driverName: csiSource.Driver,
	}
	if err = saveVolumeData(dataDir, volDataFileName, data); err != nil {
		klog.Error(log("failed to save volume info data: %v", err))
		if cleanerr := os.RemoveAll(dataDir); err != nil {
			klog.Error(log("failed to remove dir after error [%s]: %v", dataDir, cleanerr))
		}
		return err
	}
	defer func() {
		if err != nil {
			// clean up metadata
			klog.Errorf(log("attacher.MountDevice failed: %v", err))
			if err := removeMountDir(c.plugin, deviceMountPath); err != nil {
				klog.Error(log("attacher.MountDevice failed to remove mount dir after errir [%s]: %v", deviceMountPath, err))
			}
		}
	}()

	if c.csiClient == nil {
		c.csiClient, err = newCsiDriverClient(csiDriverName(csiSource.Driver))
		if err != nil {
			klog.Errorf(log("attacher.MountDevice failed to create newCsiDriverClient: %v", err))
			return err
		}
	}
	csi := c.csiClient

	ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
	defer cancel()
	// Check whether "STAGE_UNSTAGE_VOLUME" is set
	stageUnstageSet, err := csi.NodeSupportsStageUnstage(ctx)
	if err != nil {
		return err
	}
	if !stageUnstageSet {
		klog.Infof(log("attacher.MountDevice STAGE_UNSTAGE_VOLUME capability not set. Skipping MountDevice..."))
		// defer does *not* remove the metadata file and it's correct - UnmountDevice needs it there.
		return nil
	}

	// Start MountDevice
	nodeName := string(c.plugin.host.GetNodeName())
	publishContext, err := c.plugin.getPublishContext(c.k8s, csiSource.VolumeHandle, csiSource.Driver, nodeName)

	nodeStageSecrets := map[string]string{}
	if csiSource.NodeStageSecretRef != nil {
		nodeStageSecrets, err = getCredentialsFromSecret(c.k8s, csiSource.NodeStageSecretRef)
		if err != nil {
			err = fmt.Errorf("fetching NodeStageSecretRef %s/%s failed: %v",
				csiSource.NodeStageSecretRef.Namespace, csiSource.NodeStageSecretRef.Name, err)
			return err
		}
	}

	//TODO (vladimirvivien) implement better AccessModes mapping between k8s and CSI
	accessMode := v1.ReadWriteOnce
	if spec.PersistentVolume.Spec.AccessModes != nil {
		accessMode = spec.PersistentVolume.Spec.AccessModes[0]
	}

	fsType := csiSource.FSType
	err = csi.NodeStageVolume(ctx,
		csiSource.VolumeHandle,
		publishContext,
		deviceMountPath,
		fsType,
		accessMode,
		nodeStageSecrets,
		csiSource.VolumeAttributes)

	if err != nil {
		return err
	}

	klog.V(4).Infof(log("attacher.MountDevice successfully requested NodeStageVolume [%s]", deviceMountPath))
	return nil
}

var _ volume.Detacher = &csiAttacher{}

var _ volume.DeviceUnmounter = &csiAttacher{}

func (c *csiAttacher) Detach(volumeName string, nodeName types.NodeName) error {
	return nil
}

func (c *csiAttacher) waitForVolumeDetachment(volumeHandle, attachID string) error {
	klog.V(4).Info(log("probing for updates from CSI driver for [attachment.ID=%v]", attachID))

	err := wait.PollImmediate(time.Second*5, c.waitSleepTime*10, func() (bool, error) {
		klog.V(4).Info(log("probing VolumeAttachment [id=%v]", attachID))
		attach, err := c.k8s.StorageV1().VolumeAttachments().Get(context.Background(), attachID, meta.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				//object deleted or never existed, done
				klog.V(4).Info(log("VolumeAttachment object [%v] for volume [%v] not found, object deleted", attachID, volumeHandle))
				return true, nil
			}
			return false, err
		}
		// driver reports attach error
		detachErr := attach.Status.DetachError
		if detachErr != nil {
			return false, errors.New(detachErr.Message)
		}
		return false, nil
	})

	return err
}

func (c *csiAttacher) UnmountDevice(deviceMountPath string) error {
	klog.V(4).Info(log("attacher.UnmountDevice(%s)", deviceMountPath))

	// Setup
	var driverName, volID string
	dataDir := filepath.Dir(deviceMountPath)
	data, err := loadVolumeData(dataDir, volDataFileName)
	if err == nil {
		driverName = data[volDataKey.driverName]
		volID = data[volDataKey.volHandle]
	} else {
		klog.Error(log("UnmountDevice failed to load volume data file [%s]: %v", dataDir, err))

		// The volume might have been mounted by old CSI volume plugin. Fall back to the old behavior: read PV from API server
		driverName, volID, err = getDriverAndVolNameFromDeviceMountPath(c.k8s, deviceMountPath)
		if err != nil {
			klog.Errorf(log("attacher.UnmountDevice failed to get driver and volume name from device mount path: %v", err))
			return err
		}
	}

	if c.csiClient == nil {
		c.csiClient, err = newCsiDriverClient(csiDriverName(driverName))
		if err != nil {
			klog.Errorf(log("attacher.UnmountDevice failed to create newCsiDriverClient: %v", err))
			return err
		}
	}
	csi := c.csiClient

	ctx, cancel := context.WithTimeout(context.Background(), csiTimeout)
	defer cancel()
	// Check whether "STAGE_UNSTAGE_VOLUME" is set
	stageUnstageSet, err := csi.NodeSupportsStageUnstage(ctx)
	if err != nil {
		klog.Errorf(log("attacher.UnmountDevice failed to check whether STAGE_UNSTAGE_VOLUME set: %v", err))
		return err
	}
	if !stageUnstageSet {
		klog.Infof(log("attacher.UnmountDevice STAGE_UNSTAGE_VOLUME capability not set. Skipping UnmountDevice..."))
		// Just	delete the global directory + json file
		if err := removeMountDir(c.plugin, deviceMountPath); err != nil {
			return fmt.Errorf("failed to clean up gloubal mount %s: %s", dataDir, err)
		}

		return nil
	}

	// Start UnmountDevice
	err = csi.NodeUnstageVolume(ctx,
		volID,
		deviceMountPath)

	if err != nil {
		klog.Errorf(log("attacher.UnmountDevice failed: %v", err))
		return err
	}

	// Delete the global directory + json file
	if err := removeMountDir(c.plugin, deviceMountPath); err != nil {
		return fmt.Errorf("failed to clean up gloubal mount %s: %s", dataDir, err)
	}

	klog.V(4).Infof(log("attacher.UnmountDevice successfully requested NodeStageVolume [%s]", deviceMountPath))
	return nil
}

// getAttachmentName returns csi-<sha256(volName,csiDriverName,NodeName)>
func getAttachmentName(volName, csiDriverName, nodeName string) string {
	result := sha256.Sum256([]byte(fmt.Sprintf("%s%s%s", volName, csiDriverName, nodeName)))
	return fmt.Sprintf("csi-%x", result)
}

// isAttachmentName returns true if the string given is of the form of an Attach ID
// and false otherwise
func isAttachmentName(unknownString string) bool {
	// 68 == "csi-" + len(sha256hash)
	if strings.HasPrefix(unknownString, "csi-") && len(unknownString) == 68 {
		return true
	}
	return false
}

func makeDeviceMountPath(plugin *csiPlugin, spec *volume.Spec) (string, error) {
	if spec == nil {
		return "", fmt.Errorf("makeDeviceMountPath failed, spec is nil")
	}

	pvName := spec.PersistentVolume.Name
	if pvName == "" {
		return "", fmt.Errorf("makeDeviceMountPath failed, pv name empty")
	}

	return filepath.Join(plugin.host.GetPluginDir(plugin.GetPluginName()), persistentVolumeInGlobalPath, pvName, globalMountInGlobalPath), nil
}

func getDriverAndVolNameFromDeviceMountPath(k8s kubernetes.Interface, deviceMountPath string) (string, string, error) {
	// deviceMountPath structure: /var/lib/kubelet/plugins/kubernetes.io/csi/pv/{pvname}/globalmount
	dir := filepath.Dir(deviceMountPath)
	if file := filepath.Base(deviceMountPath); file != globalMountInGlobalPath {
		return "", "", fmt.Errorf("getDriverAndVolNameFromDeviceMountPath failed, path did not end in %s", globalMountInGlobalPath)
	}
	// dir is now /var/lib/kubelet/plugins/kubernetes.io/csi/pv/{pvname}
	pvName := filepath.Base(dir)

	// Get PV and check for errors
	pv, err := k8s.CoreV1().PersistentVolumes().Get(context.Background(), pvName, meta.GetOptions{})
	if err != nil {
		return "", "", err
	}
	if pv == nil || pv.Spec.CSI == nil {
		return "", "", fmt.Errorf("getDriverAndVolNameFromDeviceMountPath could not find CSI Persistent Volume Source for pv: %s", pvName)
	}

	// Get VolumeHandle and PluginName from pv
	csiSource := pv.Spec.CSI
	if csiSource.Driver == "" {
		return "", "", fmt.Errorf("getDriverAndVolNameFromDeviceMountPath failed, driver name empty")
	}
	if csiSource.VolumeHandle == "" {
		return "", "", fmt.Errorf("getDriverAndVolNameFromDeviceMountPath failed, VolumeHandle empty")
	}

	return csiSource.Driver, csiSource.VolumeHandle, nil
}
