//go:build !windows

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
*/

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	resetLongDescription = `
keadm reset command can be executed in both cloud and edge node
In cloud node it shuts down the cloud processes of KubeEdge
In edge node it shuts down the edge processes of KubeEdge
`
	resetExample = `
For cloud node:
keadm reset

For edge node:
keadm reset
`
)

func newResetOptions() *common.ResetOptions {
	opts := &common.ResetOptions{}
	opts.Kubeconfig = common.DefaultKubeConfig
	opts.RuntimeType = constants.DefaultRuntimeType
	return opts
}

func NewKubeEdgeReset() *cobra.Command {
	isEdgeNode := false
	reset := newResetOptions()

	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns KubeEdge (cloud(helm installed) & edge) component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			whoRunning := util.RunningModuleV2(reset)
			if whoRunning == common.NoneRunning {
				fmt.Println("None of KubeEdge components are running in this host, exit")
				os.Exit(0)
			}

			if whoRunning == common.KubeEdgeEdgeRunning {
				isEdgeNode = true
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !reset.Force {
				fmt.Println("[reset] WARNING: Changes made to this host by 'keadm init' or 'keadm join' will be reverted.")
				fmt.Print("[reset] Are you sure you want to proceed? [y/N]: ")
				s := bufio.NewScanner(os.Stdin)
				s.Scan()
				if err := s.Err(); err != nil {
					return err
				}
				if strings.ToLower(s.Text()) != "y" {
					return fmt.Errorf("aborted reset operation")
				}
			}

			// first cleanup edge node static pod directory to stop static and mirror pod
			if isEdgeNode {
				config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
				if err != nil {
					return err
				}
				dir := config.Modules.Edged.TailoredKubeletConfig.StaticPodPath
				if dir != "" {
					if err := phases.CleanDir(dir); err != nil {
						fmt.Printf("Failed to delete static pod directory %s: %v\n", dir, err)
					} else {
						time.Sleep(1 * time.Second)
						fmt.Printf("Static pod directory has been removed!\n")
					}
				}
			}

			// 1. kill cloudcore/edgecore process.
			// For edgecore, don't delete node from K8S
			if err := TearDownKubeEdge(isEdgeNode, reset.Kubeconfig); err != nil {
				return err
			}

			// 2. Remove containers managed by KubeEdge. Only for edge node.
			if err := RemoveContainers(isEdgeNode, utilsexec.New()); err != nil {
				fmt.Printf("Failed to remove containers: %v\n", err)
			}

			// 3. Clean stateful directories
			if err := cleanDirectories(isEdgeNode); err != nil {
				return err
			}

			// cleanup mqtt container
			if err := RemoveMqttContainer(reset.RuntimeType, reset.Endpoint, ""); err != nil {
				fmt.Printf("Failed to remove MQTT container: %v\n", err)
			}
			//4. TODO: clean status information

			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

func RemoveMqttContainer(runtimeType, endpoint, cgroupDriver string) error {
	runtime, err := util.NewContainerRuntime(runtimeType, endpoint, cgroupDriver)
	if err != nil {
		return fmt.Errorf("failed to new container runtime: %v", err)
	}

	return runtime.RemoveMQTT()
}

// TearDownKubeEdge will bring down either cloud or edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, kubeConfig string) error {
	var ke common.ToolsInstaller
	ke = &helm.KubeCloudHelmInstTool{
		Common: util.Common{
			KubeConfig: kubeConfig,
		},
	}
	if isEdgeNode {
		ke = &util.KubeEdgeInstTool{Common: util.Common{}}
	}

	err := ke.TearDown()
	if err != nil {
		return fmt.Errorf("TearDown failed, err:%v", err)
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

func cleanDirectories(isEdgeNode bool) error {
	var dirToClean = []string{
		util.KubeEdgePath,
		util.KubeEdgeLogPath,
		util.KubeEdgeSocketPath,
		util.EdgeRootDir,
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

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().StringVar(&resetOpts.Kubeconfig, common.KubeConfig, resetOpts.Kubeconfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation")
	cmd.Flags().StringVar(&resetOpts.RuntimeType, common.RuntimeType, resetOpts.RuntimeType,
		"Use this key to set container runtime")
	cmd.Flags().StringVar(&resetOpts.Endpoint, common.RemoteRuntimeEndpoint, resetOpts.Endpoint,
		"Use this key to set container runtime endpoint")
}
