/*
Copyright 2023 The KubeEdge Authors.

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
KubeEdge Authors: To set default tailored kubelet configuration,
This file is derived from K8S Kubelet apis code with reduced set of methods
Changes done are
1. Package edged got some functions from "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"
and made some variant
*/

package v1alpha2

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logsapi "k8s.io/component-base/logs/api/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	configv1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/eviction"
	"k8s.io/kubernetes/pkg/kubelet/qos"
	"k8s.io/utils/ptr"

	"github.com/kubeedge/api/apis/common/constants"
)

// SetDefaultsKubeletConfiguration sets defaults for tailored kubelet configuration
func SetDefaultsKubeletConfiguration(obj *TailoredKubeletConfiguration) {
	obj.StaticPodPath = constants.DefaultManifestsDir
	obj.PodLogsDir = configv1beta1.DefaultPodLogsDir
	obj.SyncFrequency = metav1.Duration{Duration: 1 * time.Minute}
	obj.FileCheckFrequency = metav1.Duration{Duration: 20 * time.Second}
	obj.Address = constants.ServerAddress
	obj.ReadOnlyPort = constants.ServerPort
	obj.ClusterDomain = constants.DefaultClusterDomain
	obj.RegistryPullQPS = ptr.To[int32](5)
	obj.RegistryBurst = 10
	obj.EventRecordQPS = ptr.To[int32](50)
	obj.EventBurst = 100
	obj.EnableDebuggingHandlers = ptr.To(true)
	obj.OOMScoreAdj = ptr.To(int32(qos.KubeletOOMScoreAdj))
	obj.StreamingConnectionIdleTimeout = metav1.Duration{Duration: 4 * time.Hour}
	obj.NodeStatusReportFrequency = metav1.Duration{Duration: 5 * time.Minute}
	obj.NodeStatusUpdateFrequency = metav1.Duration{Duration: 10 * time.Second}
	obj.NodeLeaseDurationSeconds = 40
	obj.ImageMinimumGCAge = metav1.Duration{Duration: 2 * time.Minute}
	// default is below docker's default dm.min_free_space of 90%
	obj.ImageGCHighThresholdPercent = ptr.To[int32](85)
	obj.ImageGCLowThresholdPercent = ptr.To[int32](80)
	obj.VolumeStatsAggPeriod = metav1.Duration{Duration: time.Minute}
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
	obj.PodPidsLimit = ptr.To[int64](-1)
	obj.CPUCFSQuotaPeriod = &metav1.Duration{Duration: 100 * time.Millisecond}
	obj.NodeStatusMaxImages = ptr.To[int32](0)
	obj.MaxOpenFiles = 1000000
	obj.ContentType = "application/json"
	obj.SerializeImagePulls = ptr.To(true)
	obj.EvictionHard = eviction.DefaultEvictionHard
	obj.EvictionPressureTransitionPeriod = metav1.Duration{Duration: 5 * time.Minute}
	obj.EnableControllerAttachDetach = ptr.To(true)
	obj.MakeIPTablesUtilChains = ptr.To(true)
	obj.IPTablesMasqueradeBit = ptr.To[int32](configv1beta1.DefaultIPTablesMasqueradeBit)
	obj.IPTablesDropBit = ptr.To[int32](configv1beta1.DefaultIPTablesDropBit)
	obj.FailSwapOn = ptr.To(false)
	obj.ContainerLogMaxSize = "10Mi"
	obj.ContainerLogMaxFiles = ptr.To[int32](5)
	obj.ContainerLogMonitorInterval = &metav1.Duration{Duration: 10 * time.Second}
	obj.ContainerLogMaxWorkers = ptr.To[int32](1)
	obj.ConfigMapAndSecretChangeDetectionStrategy = kubeletconfigv1beta1.GetChangeDetectionStrategy
	obj.EnforceNodeAllocatable = DefaultNodeAllocatableEnforcement
	obj.VolumePluginDir = constants.DefaultVolumePluginDir
	// Use the Default LoggingConfiguration option
	logsapi.SetRecommendedLoggingConfiguration(&obj.Logging)
	obj.EnableSystemLogHandler = ptr.To(true)
	obj.EnableProfilingHandler = ptr.To(true)
	obj.EnableDebugFlagsHandler = ptr.To(true)
	obj.SeccompDefault = ptr.To(false)
	obj.MemoryThrottlingFactor = ptr.To(configv1beta1.DefaultMemoryThrottlingFactor)
	obj.RegisterNode = ptr.To(true)

	obj.EnforceNodeAllocatable = DefaultNodeAllocatableEnforcement
	obj.CgroupDriver = DefaultCgroupDriver
	obj.CgroupsPerQOS = ptr.To(DefaultCgroupsPerQOS)
	obj.ResolverConfig = ptr.To(DefaultResolverConfig)
	obj.CPUCFSQuota = ptr.To(DefaultCPUCFSQuota)
	obj.LocalStorageCapacityIsolation = ptr.To(true)
	obj.ContainerRuntimeEndpoint = constants.DefaultRemoteRuntimeEndpoint
	obj.ImageServiceEndpoint = constants.DefaultRemoteImageEndpoint
	obj.FailCgroupV1 = ptr.To(false)
}
