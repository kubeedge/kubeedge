/*
Copyright 2019 The KubeEdge Authors.

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

package images

import (
	"syscall"

	statsapi "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
	"k8s.io/kubernetes/pkg/kubelet/images"

	"github.com/kubeedge/kubeedge/pkg/util"
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
