/*
Copyright 2019 The KubeEdge Authors.

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
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	phases "k8s.io/kubernetes/cmd/kubeadm/app/cmd/phases/reset"
	utilruntime "k8s.io/kubernetes/cmd/kubeadm/app/util/runtime"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
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
	return opts
}

// NewKubeEdgeReset represents the reset command
func NewKubeEdgeReset(out io.Writer, reset *common.ResetOptions) *cobra.Command {
	IsEdgeNode := false
	if reset == nil {
		reset = newResetOptions()
	}

	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns KubeEdge (cloud & edge) component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			whoRunning, err := util.IsCloudCore()
			if err != nil {
				return err
			}
			switch whoRunning {
			case common.KubeEdgeEdgeRunning:
				IsEdgeNode = true
			case common.NoneRunning:
				return fmt.Errorf("None of KubeEdge components are running in this host")
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
			// 1. kill cloudcore/edgecore process.
			// For edgecore, don't delete node from K8S
			if err := TearDownKubeEdge(IsEdgeNode, reset.Kubeconfig); err != nil {
				return err
			}

			// 2. Remove containers managed by KubeEdge. Only for edge node.
			if err := RemoveContainers(IsEdgeNode, utilsexec.New()); err != nil {
				klog.Warningf("Failed to remove containers: %v\n", err)
			}

			// 3. Clean stateful directories
			if err := cleanDirectories(IsEdgeNode); err != nil {
				return err
			}

			//4. TODO: clean status information

			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

// TearDownKubeEdge will bring down either cloud or edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, kubeConfig string) error {
	var ke common.ToolsInstaller
	ke = &util.KubeCloudInstTool{Common: util.Common{KubeConfig: kubeConfig}}
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
	var dirToClean = []string{"/var/lib/edged", "/etc/kubeedge"}

	if isEdgeNode {
		dirToClean = append(dirToClean, "/var/lib/dockershim", "/var/run/kubernetes", "/var/lib/cni")
	}

	for _, dir := range dirToClean {
		if err := phases.CleanDir(dir); err != nil {
			klog.Warningf("Failed to delete directory %s: %v", dir, err)
		}
	}

	return nil
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().StringVar(&resetOpts.Kubeconfig, common.KubeConfig, resetOpts.Kubeconfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation")
}
