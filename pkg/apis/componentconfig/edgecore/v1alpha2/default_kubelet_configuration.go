/*
Copyright 2022 The KubeEdge Authors.

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
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	configv1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/config/v1beta1"
	"k8s.io/kubernetes/pkg/kubelet/qos"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
	utilpointer "k8s.io/utils/pointer"

	"github.com/kubeedge/kubeedge/common/constants"
)

var (
	zeroDuration = metav1.Duration{}
	// TODO: Move these constants to k8s.io/kubelet/config/v1beta1 instead?
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	DefaultNodeAllocatableEnforcement = []string{"pods"}
)

// SetDefaultsKubeletConfiguration sets defaults for tailored kubelet configuration
func SetDefaultsKubeletConfiguration(obj *TailoredKubeletConfiguration) {
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
	obj.PodPidsLimit = utilpointer.Int64(-1)
	obj.ResolverConfig = utilpointer.String(kubetypes.ResolvConfDefault)
	obj.CPUCFSQuota = utilpointer.BoolPtr(true)
	obj.CPUCFSQuotaPeriod = &metav1.Duration{Duration: 100 * time.Millisecond}
	obj.NodeStatusMaxImages = utilpointer.Int32Ptr(50)
	obj.MaxOpenFiles = 1000000
	obj.ContentType = "application/json"
	obj.SerializeImagePulls = utilpointer.BoolPtr(true)
	obj.EvictionHard = configv1beta1.DefaultEvictionHard
	obj.EvictionPressureTransitionPeriod = metav1.Duration{Duration: 5 * time.Minute}
	obj.EnableControllerAttachDetach = utilpointer.BoolPtr(true)
	obj.MakeIPTablesUtilChains = utilpointer.BoolPtr(true)
	obj.IPTablesMasqueradeBit = utilpointer.Int32Ptr(configv1beta1.DefaultIPTablesMasqueradeBit)
	obj.IPTablesDropBit = utilpointer.Int32Ptr(configv1beta1.DefaultIPTablesDropBit)
	obj.FailSwapOn = utilpointer.BoolPtr(false)
	obj.ContainerLogMaxSize = "10Mi"
	obj.ContainerLogMaxFiles = utilpointer.Int32Ptr(5)
	obj.ConfigMapAndSecretChangeDetectionStrategy = kubeletconfigv1beta1.GetChangeDetectionStrategy
	obj.EnforceNodeAllocatable = DefaultNodeAllocatableEnforcement
	obj.VolumePluginDir = configv1beta1.DefaultVolumePluginDir
	// Use the Default LoggingConfiguration option
	componentbaseconfigv1alpha1.RecommendedLoggingConfiguration(&obj.Logging)
	obj.EnableSystemLogHandler = utilpointer.BoolPtr(true)
	obj.EnableProfilingHandler = utilpointer.BoolPtr(true)
	obj.EnableDebugFlagsHandler = utilpointer.BoolPtr(true)
	obj.SeccompDefault = utilpointer.BoolPtr(false)
	obj.MemoryThrottlingFactor = utilpointer.Float64Ptr(configv1beta1.DefaultMemoryThrottlingFactor)
	obj.RegisterNode = utilpointer.BoolPtr(true)
}
