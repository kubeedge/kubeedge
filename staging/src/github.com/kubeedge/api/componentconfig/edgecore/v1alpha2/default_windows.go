//go:build windows

package v1alpha2

const (
	CGroupDriverCGroupFS = "-"
	CGroupDriverSystemd  = ""

	// DataBaseDataSource is edge.db
	DataBaseDataSource = "C:\\var\\lib\\kubeedge\\edgecore.db"

	DefaultCgroupDriver         = ""
	DefaultCgroupsPerQOS        = false
	DefaultResolverConfig       = ""
	DefaultCPUCFSQuota          = false
	DefaultWindowsPriorityClass = "NORMAL_PRIORITY_CLASS"
)

var (
	// TODO: Move these constants to k8s.io/kubelet/config/v1beta1 instead?
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	DefaultNodeAllocatableEnforcement = []string{}
)
