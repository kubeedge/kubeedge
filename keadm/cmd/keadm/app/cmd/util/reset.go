/*
Copyright 2024 The KubeEdge Authors.

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

package util

import (
	"fmt"
	"os"

	"k8s.io/klog/v2"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func NewResetOptions() *common.ResetOptions {
	opts := &common.ResetOptions{}
	return opts
}

func RemoveMqttContainer(endpoint, cgroupDriver string) error {
	runtime, err := NewContainerRuntime(endpoint, cgroupDriver)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}

	return runtime.RemoveMQTT()
}

// RemoveContainers removes all Kubernetes-managed containers
func RemoveContainers(criSocketPath string, execer utilsexec.Interface) error {
	if criSocketPath == "" {
		var err error
		criSocketPath, err = utilruntime.DetectCRISocket()
		if err != nil {
			return fmt.Errorf("failed to get crisocket with err:%v", err)
		}
	}

	containerRuntime, err := utilruntime.NewContainerRuntime(execer, criSocketPath)
	if err != nil {
		return err
	}

	containers, err := containerRuntime.ListKubeContainers()
	if err != nil {
		return err
	}

	return containerRuntime.RemoveContainers(containers)
}

func CleanDirectories(isEdgeNode bool) error {
	var dirToClean = []string{
		KubeEdgePath,
		KubeEdgeLogPath,
		KubeEdgeSocketPath,
		EdgeRootDir,
	}

	if isEdgeNode {
		dirToClean = append(dirToClean, EdgeKubeletDir)
	}

	for _, dir := range dirToClean {
		klog.V(2).Infof("remove dir %s", dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return nil
		}
		if err := os.RemoveAll(dir); err != nil {
			klog.Warningf("failed to delete dir %s, err: %v", dir, err)
		}
	}
	return nil
}
