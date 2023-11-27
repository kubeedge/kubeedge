package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func CleanEdgeNodeDirectories(isEdgeNode bool) error {
	if !isEdgeNode {
		return nil
	}
	config, err := ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return err
	}
	dir := config.Modules.Edged.TailoredKubeletConfig.StaticPodPath
	if dir != "" {
		if err = phases.CleanDir(dir); err != nil {
			fmt.Printf("Failed to delete static pod directory %s: %v\n", dir, err)
		} else {
			time.Sleep(1 * time.Second)
			fmt.Printf("Static pod directory has been removed!\n")
		}
	}
	return nil
}

func UserConfirm(force bool) error {
	if force {
		return nil
	}
	fmt.Println("[reset] WARNING: Changes made to this host by 'keadm init' or 'keadm join' will be reverted.")
	fmt.Print("[reset] Are you sure you want to proceed? [Y/N]: ")
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	if err := s.Err(); err != nil {
		return err
	}
	if strings.ToLower(s.Text()) != "y" {
		return fmt.Errorf("aborted reset operation")
	}
	return nil
}

// RemoveContainers removes all Kubernetes-managed containers
func RemoveContainers(isEdgeNode bool, execer utilsexec.Interface) error {
	if !isEdgeNode {
		return nil
	}
	criSocketPath, err := utilruntime.DetectCRISocket()
	if err != nil {
		return err
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
		dirToClean = append(dirToClean, kubeEdgeDirs...)
	}

	for _, dir := range dirToClean {
		if err := phases.CleanDir(dir); err != nil {
			fmt.Printf("Failed to delete directory %s: %v\n", dir, err)
		}
	}

	return nil
}

func RemoveMqttContainer(runtimeType, endpoint, cgroupDriver string) error {
	runtime, err := NewContainerRuntime(runtimeType, endpoint, cgroupDriver)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}

	return runtime.RemoveMQTT()
}
