//go:build !windows

package v1alpha2

import kubetypes "k8s.io/kubernetes/pkg/kubelet/types"

const (
	CGroupDriverCGroupFS = "cgroupfs"
	CGroupDriverSystemd  = "systemd"

	// DataBaseDataSource is edge.db
	DataBaseDataSource = "/var/lib/kubeedge/edgecore.db"

	DefaultCgroupDriver         = "cgroupfs"
	DefaultCgroupsPerQOS        = true
	DefaultResolverConfig       = kubetypes.ResolvConfDefault
	DefaultCPUCFSQuota          = true
	DefaultWindowsPriorityClass = ""
)

var (
	// TODO: Move these constants to k8s.io/kubelet/config/v1beta1 instead?
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	DefaultNodeAllocatableEnforcement = []string{"pods"}
)
