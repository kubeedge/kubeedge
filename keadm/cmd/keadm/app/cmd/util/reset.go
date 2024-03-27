package util

import (
	"fmt"

	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
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
		dirToClean = append(dirToClean, "/var/lib/dockershim", "/var/run/kubernetes", "/var/lib/cni")
	}

	for _, dir := range dirToClean {
		if err := phases.CleanDir(dir); err != nil {
			fmt.Printf("Failed to delete directory %s: %v\n", dir, err)
		}
	}

	return nil
}
