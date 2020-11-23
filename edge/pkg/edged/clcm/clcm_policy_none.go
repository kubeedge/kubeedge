/*
Copyright 2016 The Kubernetes Authors.

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
KubeEdge Authors: To create mini-kubelet for edge deployment scenario,
This file is derived from K8S Kubelet code with reduced set of methods
*/

package clcm

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	internalapi "k8s.io/cri-api/pkg/apis"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cm/containermap"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpumanager"
	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"
	"k8s.io/kubernetes/pkg/kubelet/config"
	"k8s.io/kubernetes/pkg/kubelet/status"
)

const (
	//PolicyNone define no policy
	PolicyNone      = "none"
	reconcilePeriod = 1 * time.Second
)

type containerLifecycleManagerImpl struct {
	cpuManager cpumanager.Manager
}

var _ ContainerLifecycleManager = &containerLifecycleManagerImpl{}

// NewContainerLifecycleManager create new object for container lifecycle manager
func NewContainerLifecycleManager(kubeletRootDir string) (ContainerLifecycleManager, error) {
	var err error
	clcm := &containerLifecycleManagerImpl{}
	result := make(v1.ResourceList)
	clcm.cpuManager, err = cpumanager.NewManager(
		PolicyNone,
		reconcilePeriod,
		nil,
		nil,
		cpuset.NewCPUSet(),
		result,
		kubeletRootDir,
		nil,
	)
	if err != nil {
		klog.Errorf("failed to initialize cpu manager: %v", err)
		return nil, err
	}
	return clcm, nil
}

func (clcm *containerLifecycleManagerImpl) InternalContainerLifecycle() InternalContainerLifecycle {
	return &internalContainerLifecycleImpl{clcm.cpuManager}
}

func (clcm *containerLifecycleManagerImpl) StartCPUManager(activePods cpumanager.ActivePodsFunc,
	sourcesReady config.SourcesReady,
	podStatusProvider status.PodStatusProvider,
	runtimeService internalapi.RuntimeService) error {
	containerMap, err := buildContainerMapFromRuntime(runtimeService)
	if err != nil {
		klog.Errorf("Error when starting the CPU manager in Container Lifecycle Manager: [%v]", err)
		return err
	}
	clcm.cpuManager.Start(activePods, sourcesReady, podStatusProvider, runtimeService, containerMap)
	return nil
}

// This is introduced from Kubernetes 1.18 container_manager_linux.go
func buildContainerMapFromRuntime(runtimeService internalapi.RuntimeService) (containermap.ContainerMap, error) {
	podSandboxMap := make(map[string]string)
	podSandboxList, _ := runtimeService.ListPodSandbox(nil)
	for _, p := range podSandboxList {
		podSandboxMap[p.Id] = p.Metadata.Uid
	}

	containerMap := containermap.NewContainerMap()
	containerList, _ := runtimeService.ListContainers(nil)
	for _, c := range containerList {
		if _, exists := podSandboxMap[c.PodSandboxId]; !exists {
			return nil, fmt.Errorf("no PodsandBox found with Id '%s'", c.PodSandboxId)
		}
		containerMap.Add(podSandboxMap[c.PodSandboxId], c.Metadata.Name, c.Id)
	}

	return containerMap, nil
}
