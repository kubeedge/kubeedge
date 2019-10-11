package images

import (
	"syscall"
	"unsafe"

	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/images"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/util"
)

type imageGCManager struct {
	imageGCPath string
}

func (i *imageGCManager) ImageFsStats() (*statsapi.FsStats, error) {
	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	var lpFreeBytesAvailableToCaller int64
	var lpTotalNumberOfBytes int64
	var lpTotalNumberOfFreeBytes int64

	_, _, err := c.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(i.imageGCPath))),
		uintptr(unsafe.Pointer(&lpFreeBytesAvailableToCaller)), uintptr(unsafe.Pointer(&lpTotalNumberOfBytes)), uintptr(unsafe.Pointer(&lpTotalNumberOfFreeBytes)))

	if err != nil {
		return nil, err
	}
	cap := uint64(lpTotalNumberOfBytes)
	ava := uint64(lpFreeBytesAvailableToCaller)
	used := uint64(lpTotalNumberOfBytes - lpFreeBytesAvailableToCaller)
	return &statsapi.FsStats{
		CapacityBytes:  &cap,
		AvailableBytes: &ava,
		UsedBytes:      &used,
	}, nil
}

//NewStatsProvider returns image status provider
func NewStatsProvider() images.StatsProvider {
	return &imageGCManager{
		util.GetCurPath(),
	}
}
