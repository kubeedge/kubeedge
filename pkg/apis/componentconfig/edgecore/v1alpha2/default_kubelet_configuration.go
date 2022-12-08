package v1alpha2

import (
	"github.com/kubeedge/kubeedge/common/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/qos"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
	utilpointer "k8s.io/utils/pointer"
	"time"
)

const (
	// TODO: Move these constants to k8s.io/kubelet/config/v1beta1 instead?
	DefaultIPTablesMasqueradeBit = 14
	DefaultIPTablesDropBit       = 15
	DefaultVolumePluginDir       = "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/"

	// See https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2570-memory-qos
	DefaultMemoryThrottlingFactor = 0.8
)

var (
	zeroDuration = metav1.Duration{}
	// TODO: Move these constants to k8s.io/kubelet/config/v1beta1 instead?
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	DefaultNodeAllocatableEnforcement = []string{"pods"}
)

// DefaultEvictionHard includes default options for hard eviction.
var DefaultEvictionHard = map[string]string{
	"memory.available":  "100Mi",
	"nodefs.available":  "10%",
	"nodefs.inodesFree": "5%",
	"imagefs.available": "15%",
}

func SetDefaults_KubeletConfiguration(obj *TailoredKubeletConfiguration) {
	obj.SyncFrequency = metav1.Duration{Duration: 1 * time.Minute}
	obj.Address = constants.ServerAddress
	obj.ReadOnlyPort = constants.ServerPort
	obj.ClusterDomain = constants.DefaultClusterDomain
	obj.RegistryPullQPS = utilpointer.Int32Ptr(5)
	obj.RegistryBurst = 10
	obj.EnableDebuggingHandlers = utilpointer.BoolPtr(true)
	obj.OOMScoreAdj = utilpointer.Int32Ptr(int32(qos.KubeletOOMScoreAdj))
	obj.StreamingConnectionIdleTimeout = metav1.Duration{Duration: 4 * time.Hour}
	obj.NodeStatusReportFrequency = metav1.Duration{Duration: 5 * time.Minute}
	obj.NodeStatusUpdateFrequency = metav1.Duration{Duration: 10 * time.Second}
	obj.NodeLeaseDurationSeconds = 40
	obj.ImageMinimumGCAge = metav1.Duration{Duration: 2 * time.Minute}
	// default is below docker's default dm.min_free_space of 90%
	obj.ImageGCHighThresholdPercent = utilpointer.Int32Ptr(constants.DefaultImageGCHighThreshold)
	obj.ImageGCLowThresholdPercent = utilpointer.Int32Ptr(constants.DefaultImageGCLowThreshold)
	obj.VolumeStatsAggPeriod = metav1.Duration{Duration: time.Minute}
	obj.CgroupsPerQOS = utilpointer.BoolPtr(true)
	obj.CgroupDriver = "cgroupfs"
	obj.CPUManagerPolicy = "none"
	// Keep the same as default NodeStatusUpdateFrequency
	obj.CPUManagerReconcilePeriod = metav1.Duration{Duration: 10 * time.Second}
	obj.MemoryManagerPolicy = kubeletconfigv1beta1.NoneMemoryManagerPolicy
	obj.TopologyManagerPolicy = kubeletconfigv1beta1.NoneTopologyManagerPolicy
	obj.TopologyManagerScope = kubeletconfigv1beta1.ContainerTopologyManagerScope
	obj.RuntimeRequestTimeout = metav1.Duration{Duration: 2 * time.Minute}
	obj.HairpinMode = kubeletconfigv1beta1.PromiscuousBridge
	obj.MaxPods = 110
	// default nil or negative value to -1 (implies node allocatable pid limit)
	temp := int64(-1)
	obj.PodPidsLimit = &temp
	obj.ResolverConfig = kubetypes.ResolvConfDefault
	obj.CPUCFSQuota = utilpointer.BoolPtr(true)
	obj.CPUCFSQuotaPeriod = &metav1.Duration{Duration: 100 * time.Millisecond}
	obj.NodeStatusMaxImages = utilpointer.Int32Ptr(50)
	obj.MaxOpenFiles = 1000000
	obj.ContentType = "application/json"
	obj.SerializeImagePulls = utilpointer.BoolPtr(true)
	obj.EvictionHard = DefaultEvictionHard
	obj.EvictionPressureTransitionPeriod = metav1.Duration{Duration: 5 * time.Minute}
	obj.EnableControllerAttachDetach = utilpointer.BoolPtr(true)
	obj.MakeIPTablesUtilChains = utilpointer.BoolPtr(true)
	obj.IPTablesMasqueradeBit = utilpointer.Int32Ptr(DefaultIPTablesMasqueradeBit)
	obj.IPTablesDropBit = utilpointer.Int32Ptr(DefaultIPTablesDropBit)
	obj.FailSwapOn = utilpointer.BoolPtr(false)
	obj.ContainerLogMaxSize = "10Mi"
	obj.ContainerLogMaxFiles = utilpointer.Int32Ptr(5)
	obj.ConfigMapAndSecretChangeDetectionStrategy = kubeletconfigv1beta1.GetChangeDetectionStrategy
	obj.EnforceNodeAllocatable = DefaultNodeAllocatableEnforcement
	obj.VolumePluginDir = DefaultVolumePluginDir
	// Use the Default LoggingConfiguration option
	componentbaseconfigv1alpha1.RecommendedLoggingConfiguration(&obj.Logging)
	obj.EnableSystemLogHandler = utilpointer.BoolPtr(true)
	obj.EnableProfilingHandler = utilpointer.BoolPtr(true)
	obj.EnableDebugFlagsHandler = utilpointer.BoolPtr(true)
	obj.SeccompDefault = utilpointer.BoolPtr(false)
	obj.MemoryThrottlingFactor = utilpointer.Float64Ptr(DefaultMemoryThrottlingFactor)
}
