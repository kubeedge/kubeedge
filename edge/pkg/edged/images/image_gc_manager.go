package images

import (
	"fmt"
	"time"

	cadvisorfs "github.com/google/cadvisor/fs"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	internalapi "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog"
	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/cadvisor"
	"k8s.io/kubernetes/pkg/kubelet/images"
)

type imageGCManager struct {
	cadvisor     cadvisor.Interface
	imageService internalapi.ImageManagerService
}

// getFsInfo returns the information of the filesystem with the specified
// fsID. If any error occurs, this function logs the error and returns
// nil.
func (i *imageGCManager) getFsInfo(fsID *runtimeapi.FilesystemIdentifier) *cadvisorapiv2.FsInfo {
	if fsID == nil {
		klog.V(2).Infof("Failed to get filesystem info: fsID is nil.")
		return nil
	}
	mountpoint := fsID.GetMountpoint()
	fsInfo, err := i.cadvisor.GetDirFsInfo(mountpoint)
	if err != nil {
		msg := fmt.Sprintf("Failed to get the info of the filesystem with mountpoint %q: %v.", mountpoint, err)
		if err == cadvisorfs.ErrNoSuchDevice {
			klog.V(2).Info(msg)
		} else {
			klog.Error(msg)
		}
		return nil
	}
	return &fsInfo
}

// ImageFsStats returns the stats of the image filesystem.
func (i *imageGCManager) ImageFsStats() (*statsapi.FsStats, error) {
	resp, err := i.imageService.ImageFsInfo()
	if err != nil {
		return nil, err
	}

	// CRI may return the stats of multiple image filesystems but we only
	// return the first one.
	if len(resp) == 0 {
		return nil, fmt.Errorf("imageFs information is unavailable")
	}
	fs := resp[0]
	s := &statsapi.FsStats{
		Time:      metav1.NewTime(time.Unix(0, fs.Timestamp)),
		UsedBytes: &fs.UsedBytes.Value,
	}
	if fs.InodesUsed != nil {
		s.InodesUsed = &fs.InodesUsed.Value
	}
	imageFsInfo := i.getFsInfo(fs.GetFsId())
	if imageFsInfo != nil {
		// The image filesystem id is unknown to the local node or there's
		// an error on retrieving the stats. In these cases, we omit those
		// stats and return the best-effort partial result. See
		// https://github.com/kubernetes/heapster/issues/1793.
		s.AvailableBytes = &imageFsInfo.Available
		s.CapacityBytes = &imageFsInfo.Capacity
		s.InodesFree = imageFsInfo.InodesFree
		s.Inodes = imageFsInfo.Inodes
	}
	return s, nil
}

//NewStatsProvider returns image status provider
func NewStatsProvider(cadvisorInterface cadvisor.Interface, imageService internalapi.ImageManagerService) images.StatsProvider {
	return &imageGCManager{
		cadvisor:     cadvisorInterface,
		imageService: imageService,
	}
}
