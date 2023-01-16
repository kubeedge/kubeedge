package config

import (
	"sync"
	"unsafe"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/component-base/config/v1alpha1"
	"k8s.io/component-base/featuregate"
	kubeletoptions "k8s.io/kubernetes/cmd/kubelet/app/options"
	"k8s.io/kubernetes/pkg/kubelet"
	kubeletconfig "k8s.io/kubernetes/pkg/kubelet/apis/config"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

var Config Configure
var once sync.Once

type Configure struct {
	v1alpha2.Edged
	*kubelet.Dependencies
	featuregate.FeatureGate
}

func InitConfigure(e *v1alpha2.Edged) {
	once.Do(func() {
		Config = Configure{
			Edged: *e,
		}
	})
}

func ConvertEdgedKubeletConfigurationToConfigKubeletConfiguration(in *v1alpha2.TailoredKubeletConfiguration, out *kubeletconfig.KubeletConfiguration, s conversion.Scope) error {
	out.SyncFrequency = in.SyncFrequency
	out.Address = in.Address
	out.ReadOnlyPort = in.ReadOnlyPort
	if err := v1.Convert_Pointer_int32_To_int32(&in.RegistryPullQPS, &out.RegistryPullQPS, s); err != nil {
		return err
	}
	out.RegistryBurst = in.RegistryBurst
	if err := v1.Convert_Pointer_int32_To_int32(&in.EventRecordQPS, &out.EventRecordQPS, s); err != nil {
		return err
	}
	out.EventBurst = in.EventBurst
	if err := v1.Convert_Pointer_bool_To_bool(&in.EnableDebuggingHandlers, &out.EnableDebuggingHandlers, s); err != nil {
		return err
	}
	out.EnableContentionProfiling = in.EnableContentionProfiling
	if err := v1.Convert_Pointer_int32_To_int32(&in.OOMScoreAdj, &out.OOMScoreAdj, s); err != nil {
		return err
	}
	out.ClusterDomain = in.ClusterDomain
	out.ClusterDNS = *(*[]string)(unsafe.Pointer(&in.ClusterDNS))
	out.StreamingConnectionIdleTimeout = in.StreamingConnectionIdleTimeout
	out.NodeStatusUpdateFrequency = in.NodeStatusUpdateFrequency
	out.NodeStatusReportFrequency = in.NodeStatusReportFrequency
	out.NodeLeaseDurationSeconds = in.NodeLeaseDurationSeconds
	out.ImageMinimumGCAge = in.ImageMinimumGCAge
	if err := v1.Convert_Pointer_int32_To_int32(&in.ImageGCHighThresholdPercent, &out.ImageGCHighThresholdPercent, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_int32_To_int32(&in.ImageGCLowThresholdPercent, &out.ImageGCLowThresholdPercent, s); err != nil {
		return err
	}
	out.VolumeStatsAggPeriod = in.VolumeStatsAggPeriod
	out.KubeletCgroups = in.KubeletCgroups
	out.SystemCgroups = in.SystemCgroups
	out.CgroupRoot = in.CgroupRoot
	if err := v1.Convert_Pointer_bool_To_bool(&in.CgroupsPerQOS, &out.CgroupsPerQOS, s); err != nil {
		return err
	}
	out.CgroupDriver = in.CgroupDriver
	out.CPUManagerPolicy = in.CPUManagerPolicy
	out.CPUManagerPolicyOptions = *(*map[string]string)(unsafe.Pointer(&in.CPUManagerPolicyOptions))
	out.CPUManagerReconcilePeriod = in.CPUManagerReconcilePeriod
	out.MemoryManagerPolicy = in.MemoryManagerPolicy
	out.TopologyManagerPolicy = in.TopologyManagerPolicy
	out.TopologyManagerScope = in.TopologyManagerScope
	out.QOSReserved = *(*map[string]string)(unsafe.Pointer(&in.QOSReserved))
	out.RuntimeRequestTimeout = in.RuntimeRequestTimeout
	out.HairpinMode = in.HairpinMode
	out.MaxPods = in.MaxPods
	out.PodCIDR = in.PodCIDR
	if err := v1.Convert_Pointer_int64_To_int64(&in.PodPidsLimit, &out.PodPidsLimit, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_string_To_string(&in.ResolverConfig, &out.ResolverConfig, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_bool_To_bool(&in.CPUCFSQuota, &out.CPUCFSQuota, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_v1_Duration_To_v1_Duration(&in.CPUCFSQuotaPeriod, &out.CPUCFSQuotaPeriod, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_int32_To_int32(&in.NodeStatusMaxImages, &out.NodeStatusMaxImages, s); err != nil {
		return err
	}
	out.MaxOpenFiles = in.MaxOpenFiles
	out.ContentType = in.ContentType
	if err := v1.Convert_Pointer_bool_To_bool(&in.SerializeImagePulls, &out.SerializeImagePulls, s); err != nil {
		return err
	}
	out.EvictionHard = *(*map[string]string)(unsafe.Pointer(&in.EvictionHard))
	out.EvictionSoft = *(*map[string]string)(unsafe.Pointer(&in.EvictionSoft))
	out.EvictionSoftGracePeriod = *(*map[string]string)(unsafe.Pointer(&in.EvictionSoftGracePeriod))
	out.EvictionPressureTransitionPeriod = in.EvictionPressureTransitionPeriod
	out.EvictionMaxPodGracePeriod = in.EvictionMaxPodGracePeriod
	out.EvictionMinimumReclaim = *(*map[string]string)(unsafe.Pointer(&in.EvictionMinimumReclaim))
	out.PodsPerCore = in.PodsPerCore
	if err := v1.Convert_Pointer_bool_To_bool(&in.EnableControllerAttachDetach, &out.EnableControllerAttachDetach, s); err != nil {
		return err
	}
	out.ProtectKernelDefaults = in.ProtectKernelDefaults
	if err := v1.Convert_Pointer_bool_To_bool(&in.MakeIPTablesUtilChains, &out.MakeIPTablesUtilChains, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_int32_To_int32(&in.IPTablesMasqueradeBit, &out.IPTablesMasqueradeBit, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_int32_To_int32(&in.IPTablesDropBit, &out.IPTablesDropBit, s); err != nil {
		return err
	}
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	if err := v1.Convert_Pointer_bool_To_bool(&in.FailSwapOn, &out.FailSwapOn, s); err != nil {
		return err
	}
	out.MemorySwap.SwapBehavior = in.MemorySwap.SwapBehavior
	out.ContainerLogMaxSize = in.ContainerLogMaxSize
	if err := v1.Convert_Pointer_int32_To_int32(&in.ContainerLogMaxFiles, &out.ContainerLogMaxFiles, s); err != nil {
		return err
	}
	out.ConfigMapAndSecretChangeDetectionStrategy = kubeletconfig.ResourceChangeDetectionStrategy(in.ConfigMapAndSecretChangeDetectionStrategy)
	out.SystemReserved = *(*map[string]string)(unsafe.Pointer(&in.SystemReserved))
	out.KubeReserved = *(*map[string]string)(unsafe.Pointer(&in.KubeReserved))
	out.ReservedSystemCPUs = in.ReservedSystemCPUs
	out.ShowHiddenMetricsForVersion = in.ShowHiddenMetricsForVersion
	out.SystemReservedCgroup = in.SystemReservedCgroup
	out.KubeReservedCgroup = in.KubeReservedCgroup
	out.EnforceNodeAllocatable = *(*[]string)(unsafe.Pointer(&in.EnforceNodeAllocatable))
	out.AllowedUnsafeSysctls = *(*[]string)(unsafe.Pointer(&in.AllowedUnsafeSysctls))
	out.VolumePluginDir = in.VolumePluginDir
	out.KernelMemcgNotification = in.KernelMemcgNotification
	if err := v1alpha1.Convert_v1alpha1_LoggingConfiguration_To_config_LoggingConfiguration(&in.Logging, &out.Logging, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_bool_To_bool(&in.EnableSystemLogHandler, &out.EnableSystemLogHandler, s); err != nil {
		return err
	}
	out.ShutdownGracePeriod = in.ShutdownGracePeriod
	out.ShutdownGracePeriodCriticalPods = in.ShutdownGracePeriodCriticalPods
	out.ShutdownGracePeriodByPodPriority = *(*[]kubeletconfig.ShutdownGracePeriodByPodPriority)(unsafe.Pointer(&in.ShutdownGracePeriodByPodPriority))
	out.ReservedMemory = *(*[]kubeletconfig.MemoryReservation)(unsafe.Pointer(&in.ReservedMemory))
	if err := v1.Convert_Pointer_bool_To_bool(&in.EnableProfilingHandler, &out.EnableProfilingHandler, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_bool_To_bool(&in.EnableDebugFlagsHandler, &out.EnableDebugFlagsHandler, s); err != nil {
		return err
	}
	if err := v1.Convert_Pointer_bool_To_bool(&in.SeccompDefault, &out.SeccompDefault, s); err != nil {
		return err
	}
	out.MemoryThrottlingFactor = (*float64)(unsafe.Pointer(in.MemoryThrottlingFactor))
	out.RegisterWithTaints = *(*[]corev1.Taint)(unsafe.Pointer(&in.RegisterWithTaints))
	if err := v1.Convert_Pointer_bool_To_bool(&in.RegisterNode, &out.RegisterNode, s); err != nil {
		return err
	}
	return nil
}

func ConvertConfigEdgedFlagToConfigKubeletFlag(in *v1alpha2.TailoredKubeletFlag, out *kubeletoptions.KubeletFlags) {
	out.KubeConfig = in.KubeConfig
	out.HostnameOverride = in.HostnameOverride
	out.NodeIP = in.NodeIP
	out.RootDirectory = in.RootDirectory
	out.RemoteRuntimeEndpoint = in.RemoteRuntimeEndpoint
	out.RemoteImageEndpoint = in.RemoteImageEndpoint
	out.ExperimentalMounterPath = in.ExperimentalMounterPath
	out.ExperimentalCheckNodeCapabilitiesBeforeMount = in.ExperimentalCheckNodeCapabilitiesBeforeMount
	out.ExperimentalNodeAllocatableIgnoreEvictionThreshold = in.ExperimentalNodeAllocatableIgnoreEvictionThreshold
	out.NodeLabels = in.NodeLabels
	out.MinimumGCAge = in.MinimumGCAge
	out.MaxPerPodContainerCount = in.MaxPerPodContainerCount
	out.MaxContainerCount = in.MaxContainerCount
	out.MasterServiceNamespace = in.MasterServiceNamespace
	out.RegisterSchedulable = in.RegisterSchedulable
	out.NonMasqueradeCIDR = in.NonMasqueradeCIDR
	out.KeepTerminatedPodVolumes = in.KeepTerminatedPodVolumes
	out.SeccompDefault = in.SeccompDefault

	// container-runtime-specific options
	out.ContainerRuntime = in.ContainerRuntime
	out.RuntimeCgroups = in.RuntimeCgroups
	out.DockershimRootDirectory = in.DockershimRootDirectory
	out.PodSandboxImage = in.PodSandboxImage
	out.DockerEndpoint = in.DockerEndpoint
	out.ImagePullProgressDeadline = in.ImagePullProgressDeadline
	out.NetworkPluginName = in.NetworkPluginName
	out.NetworkPluginMTU = in.NetworkPluginMTU
	out.CNIConfDir = in.CNIConfDir
	out.CNIBinDir = in.CNIBinDir
	out.CNICacheDir = in.CNICacheDir
}
