//go:build linux
// +build linux

/*
Copyright 2018 The Kubernetes Authors.

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

package kuberuntime

import (
	"math"
	"os"
	"strconv"
	"time"

	libcontainercgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
	v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
	kubefeatures "k8s.io/kubernetes/pkg/features"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/qos"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
)

var defaultPageSize = int64(os.Getpagesize())

// applyPlatformSpecificContainerConfig applies platform specific configurations to runtimeapi.ContainerConfig.
func (m *kubeGenericRuntimeManager) applyPlatformSpecificContainerConfig(config *runtimeapi.ContainerConfig, container *v1.Container, pod *v1.Pod, uid *int64, username string, nsTarget *kubecontainer.ContainerID) error {
	enforceMemoryQoS := false
	// Set memory.min and memory.high if MemoryQoS enabled with cgroups v2
	if utilfeature.DefaultFeatureGate.Enabled(kubefeatures.MemoryQoS) &&
		libcontainercgroups.IsCgroup2UnifiedMode() {
		enforceMemoryQoS = true
	}
	cl, err := m.generateLinuxContainerConfig(container, pod, uid, username, nsTarget, enforceMemoryQoS)
	if err != nil {
		return err
	}
	config.Linux = cl

	if utilfeature.DefaultFeatureGate.Enabled(kubefeatures.UserNamespacesStatelessPodsSupport) {
		if cl.SecurityContext.NamespaceOptions.UsernsOptions != nil {
			for _, mount := range config.Mounts {
				mount.UidMappings = cl.SecurityContext.NamespaceOptions.UsernsOptions.Uids
				mount.GidMappings = cl.SecurityContext.NamespaceOptions.UsernsOptions.Gids
			}
		}
	}
	return nil
}

// generateLinuxContainerConfig generates linux container config for kubelet runtime v1.
func (m *kubeGenericRuntimeManager) generateLinuxContainerConfig(container *v1.Container, pod *v1.Pod, uid *int64, username string, nsTarget *kubecontainer.ContainerID, enforceMemoryQoS bool) (*runtimeapi.LinuxContainerConfig, error) {
	sc, err := m.determineEffectiveSecurityContext(pod, container, uid, username)
	if err != nil {
		return nil, err
	}
	lc := &runtimeapi.LinuxContainerConfig{
		Resources:       m.generateLinuxContainerResources(pod, container, enforceMemoryQoS),
		SecurityContext: sc,
	}

	if nsTarget != nil && lc.SecurityContext.NamespaceOptions.Pid == runtimeapi.NamespaceMode_CONTAINER {
		lc.SecurityContext.NamespaceOptions.Pid = runtimeapi.NamespaceMode_TARGET
		lc.SecurityContext.NamespaceOptions.TargetId = nsTarget.ID
	}

	return lc, nil
}

// generateLinuxContainerResources generates linux container resources config for runtime
func (m *kubeGenericRuntimeManager) generateLinuxContainerResources(pod *v1.Pod, container *v1.Container, enforceMemoryQoS bool) *runtimeapi.LinuxContainerResources {
	// set linux container resources
	var cpuRequest *resource.Quantity
	if _, cpuRequestExists := container.Resources.Requests[v1.ResourceCPU]; cpuRequestExists {
		cpuRequest = container.Resources.Requests.Cpu()
	}
	lcr := m.calculateLinuxResources(cpuRequest, container.Resources.Limits.Cpu(), container.Resources.Limits.Memory())

	lcr.OomScoreAdj = int64(qos.GetContainerOOMScoreAdjust(pod, container,
		int64(m.machineInfo.MemoryCapacity)))

	lcr.HugepageLimits = GetHugepageLimitsFromResources(container.Resources)

	if utilfeature.DefaultFeatureGate.Enabled(kubefeatures.NodeSwap) {
		// NOTE(ehashman): Behaviour is defined in the opencontainers runtime spec:
		// https://github.com/opencontainers/runtime-spec/blob/1c3f411f041711bbeecf35ff7e93461ea6789220/config-linux.md#memory
		switch m.memorySwapBehavior {
		case kubelettypes.UnlimitedSwap:
			// -1 = unlimited swap
			lcr.MemorySwapLimitInBytes = -1
		case kubelettypes.LimitedSwap:
			fallthrough
		default:
			// memorySwapLimit = total permitted memory+swap; if equal to memory limit, => 0 swap above memory limit
			// Some swapping is still possible.
			// Note that if memory limit is 0, memory swap limit is ignored.
			lcr.MemorySwapLimitInBytes = lcr.MemoryLimitInBytes
		}
	}

	// Set memory.min and memory.high to enforce MemoryQoS
	if enforceMemoryQoS {
		unified := map[string]string{}
		memoryRequest := container.Resources.Requests.Memory().Value()
		memoryLimit := container.Resources.Limits.Memory().Value()
		if memoryRequest != 0 {
			unified[cm.MemoryMin] = strconv.FormatInt(memoryRequest, 10)
		}

		// Guaranteed pods by their QoS definition requires that memory request equals memory limit and cpu request must equal cpu limit.
		// Here, we only check from memory perspective. Hence MemoryQoS feature is disabled on those QoS pods by not setting memory.high.
		if memoryRequest != memoryLimit {
			// The formula for memory.high for container cgroup is modified in Alpha stage of the feature in K8s v1.27.
			// It will be set based on formula:
			// `memory.high=floor[(requests.memory + memory throttling factor * (limits.memory or node allocatable memory - requests.memory))/pageSize] * pageSize`
			// where default value of memory throttling factor is set to 0.9
			// More info: https://git.k8s.io/enhancements/keps/sig-node/2570-memory-qos
			memoryHigh := int64(0)
			if memoryLimit != 0 {
				memoryHigh = int64(math.Floor(
					float64(memoryRequest)+
						(float64(memoryLimit)-float64(memoryRequest))*float64(m.memoryThrottlingFactor))/float64(defaultPageSize)) * defaultPageSize
			} else {
				allocatable := m.getNodeAllocatable()
				allocatableMemory, ok := allocatable[v1.ResourceMemory]
				if ok && allocatableMemory.Value() > 0 {
					memoryHigh = int64(math.Floor(
						float64(memoryRequest)+
							(float64(allocatableMemory.Value())-float64(memoryRequest))*float64(m.memoryThrottlingFactor))/float64(defaultPageSize)) * defaultPageSize
				}
			}
			if memoryHigh != 0 && memoryHigh > memoryRequest {
				unified[cm.MemoryHigh] = strconv.FormatInt(memoryHigh, 10)
			}
		}
		if len(unified) > 0 {
			if lcr.Unified == nil {
				lcr.Unified = unified
			} else {
				for k, v := range unified {
					lcr.Unified[k] = v
				}
			}
			klog.V(4).InfoS("MemoryQoS config for container", "pod", klog.KObj(pod), "containerName", container.Name, "unified", unified)
		}
	}

	return lcr
}

// generateContainerResources generates platform specific (linux) container resources config for runtime
func (m *kubeGenericRuntimeManager) generateContainerResources(pod *v1.Pod, container *v1.Container) *runtimeapi.ContainerResources {
	enforceMemoryQoS := false
	// Set memory.min and memory.high if MemoryQoS enabled with cgroups v2
	if utilfeature.DefaultFeatureGate.Enabled(kubefeatures.MemoryQoS) &&
		libcontainercgroups.IsCgroup2UnifiedMode() {
		enforceMemoryQoS = true
	}
	return &runtimeapi.ContainerResources{
		Linux: m.generateLinuxContainerResources(pod, container, enforceMemoryQoS),
	}
}

// calculateLinuxResources will create the linuxContainerResources type based on the provided CPU and memory resource requests, limits
func (m *kubeGenericRuntimeManager) calculateLinuxResources(cpuRequest, cpuLimit, memoryLimit *resource.Quantity) *runtimeapi.LinuxContainerResources {
	resources := runtimeapi.LinuxContainerResources{}
	var cpuShares int64

	memLimit := memoryLimit.Value()

	// If request is not specified, but limit is, we want request to default to limit.
	// API server does this for new containers, but we repeat this logic in Kubelet
	// for containers running on existing Kubernetes clusters.
	if cpuRequest == nil && cpuLimit != nil {
		cpuShares = int64(cm.MilliCPUToShares(cpuLimit.MilliValue()))
	} else {
		// if cpuRequest.Amount is nil, then MilliCPUToShares will return the minimal number
		// of CPU shares.
		cpuShares = int64(cm.MilliCPUToShares(cpuRequest.MilliValue()))
	}
	resources.CpuShares = cpuShares
	if memLimit != 0 {
		resources.MemoryLimitInBytes = memLimit
	}

	if m.cpuCFSQuota {
		// if cpuLimit.Amount is nil, then the appropriate default value is returned
		// to allow full usage of cpu resource.
		cpuPeriod := int64(quotaPeriod)
		if utilfeature.DefaultFeatureGate.Enabled(kubefeatures.CPUCFSQuotaPeriod) {
			// kubeGenericRuntimeManager.cpuCFSQuotaPeriod is provided in time.Duration,
			// but we need to convert it to number of microseconds which is used by kernel.
			cpuPeriod = int64(m.cpuCFSQuotaPeriod.Duration / time.Microsecond)
		}
		cpuQuota := milliCPUToQuota(cpuLimit.MilliValue(), cpuPeriod)
		resources.CpuQuota = cpuQuota
		resources.CpuPeriod = cpuPeriod
	}

	return &resources
}

// GetHugepageLimitsFromResources returns limits of each hugepages from resources.
func GetHugepageLimitsFromResources(resources v1.ResourceRequirements) []*runtimeapi.HugepageLimit {
	var hugepageLimits []*runtimeapi.HugepageLimit

	// For each page size, limit to 0.
	for _, pageSize := range libcontainercgroups.HugePageSizes() {
		hugepageLimits = append(hugepageLimits, &runtimeapi.HugepageLimit{
			PageSize: pageSize,
			Limit:    uint64(0),
		})
	}

	requiredHugepageLimits := map[string]uint64{}
	for resourceObj, amountObj := range resources.Limits {
		if !v1helper.IsHugePageResourceName(resourceObj) {
			continue
		}

		pageSize, err := v1helper.HugePageSizeFromResourceName(resourceObj)
		if err != nil {
			klog.InfoS("Failed to get hugepage size from resource", "object", resourceObj, "err", err)
			continue
		}

		sizeString, err := v1helper.HugePageUnitSizeFromByteSize(pageSize.Value())
		if err != nil {
			klog.InfoS("Size is invalid", "object", resourceObj, "err", err)
			continue
		}
		requiredHugepageLimits[sizeString] = uint64(amountObj.Value())
	}

	for _, hugepageLimit := range hugepageLimits {
		if limit, exists := requiredHugepageLimits[hugepageLimit.PageSize]; exists {
			hugepageLimit.Limit = limit
		}
	}

	return hugepageLimits
}

func toKubeContainerResources(statusResources *runtimeapi.ContainerResources) *kubecontainer.ContainerResources {
	var cStatusResources *kubecontainer.ContainerResources
	runtimeStatusResources := statusResources.GetLinux()
	if runtimeStatusResources != nil {
		var cpuLimit, memLimit, cpuRequest *resource.Quantity
		if runtimeStatusResources.CpuPeriod > 0 {
			milliCPU := quotaToMilliCPU(runtimeStatusResources.CpuQuota, runtimeStatusResources.CpuPeriod)
			if milliCPU > 0 {
				cpuLimit = resource.NewMilliQuantity(milliCPU, resource.DecimalSI)
			}
		}
		if runtimeStatusResources.CpuShares > 0 {
			milliCPU := sharesToMilliCPU(runtimeStatusResources.CpuShares)
			if milliCPU > 0 {
				cpuRequest = resource.NewMilliQuantity(milliCPU, resource.DecimalSI)
			}
		}
		if runtimeStatusResources.MemoryLimitInBytes > 0 {
			memLimit = resource.NewQuantity(runtimeStatusResources.MemoryLimitInBytes, resource.BinarySI)
		}
		if cpuLimit != nil || memLimit != nil || cpuRequest != nil {
			cStatusResources = &kubecontainer.ContainerResources{
				CPULimit:    cpuLimit,
				CPURequest:  cpuRequest,
				MemoryLimit: memLimit,
			}
		}
	}
	return cStatusResources
}
