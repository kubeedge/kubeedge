package images

import (
	"syscall"

	"github.com/kubeedge/kubeedge/edge/pkg/edged/util"

	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/images"
)

type imageGCManager struct {
	imageGCPath string
}

func (i *imageGCManager) ImageFsStats() (*statsapi.FsStats, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(i.imageGCPath, &fs)
	if err != nil {
		return nil, err
	}
	cap := fs.Blocks * uint64(fs.Bsize)
	ava := fs.Bfree * uint64(fs.Bsize)
	used := (fs.Blocks - fs.Bfree) * uint64(fs.Bsize)
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
