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
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/cloud"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/edge"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	resetLongDescription = `
'keadm reset' command can be executed in both cloud and edge node.
In cloud node it shuts down the cloud processes of KubeEdge.
In edge node it shuts down the edge processes of KubeEdge.
'keadm reset' is no longer supported after version v1.22. 
You must use the third-level command 'keadm reset cloud' or 'keadm reset edge'.
`
	resetExample = `
For cloud node:
keadm reset

For edge node:
keadm reset edge
`
)

func NewKubeEdgeReset() *cobra.Command {
	isEdgeNode := false
	reset := util.NewResetOptions()

	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns KubeEdge (cloud(helm installed) & edge) component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			// FIXME: remove thie hint after version v1.22
			fmt.Println("WARNING: 'keadm reset' is no longer supported after version v1.22.")
			fmt.Println("You must use the third-level command 'keadm reset cloud' or 'keadm reset edge'.")

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
		RunE: func(_ *cobra.Command, _ []string) error {
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

			if isEdgeNode {
				staticPodPath := ""
				config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
				if err != nil {
					fmt.Printf("failed to get edgecore's config with err:%v\n", err)
				} else {
					if reset.Endpoint == "" {
						reset.Endpoint = config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
					}
					staticPodPath = config.Modules.Edged.TailoredKubeletConfig.StaticPodPath
				}
				// first cleanup edge node static pod directory to stop static and mirror pod
				if staticPodPath != "" {
					if err := phases.CleanDir(staticPodPath); err != nil {
						fmt.Printf("Failed to delete static pod directory %s: %v\n", staticPodPath, err)
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
			if isEdgeNode {
				if err := util.RemoveContainers(reset.Endpoint, utilsexec.New()); err != nil {
					fmt.Printf("Failed to remove containers: %v\n", err)
				}
			}

			// 3. Clean stateful directories
			if err := util.CleanDirectories(isEdgeNode); err != nil {
				return err
			}

			//4. TODO: clean status information

			return nil
		},
	}
	edgeCmd := edge.NewOtherEdgeReset()
	cloudCmd := cloud.NewCloudReset()
	cmd.AddCommand(edgeCmd)
	cmd.AddCommand(cloudCmd)
	addResetFlags(cmd, reset)
	return cmd
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	cmd.Flags().StringVar(&resetOpts.Kubeconfig, common.FlagNameKubeConfig, common.DefaultKubeConfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation")
	cmd.Flags().StringVar(&resetOpts.Endpoint, "remote-runtime-endpoint", resetOpts.Endpoint,
		"Use this key to set container runtime endpoint")
}

// TearDownKubeEdge will bring down either cloud or edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, kubeConfig string) error {
	var ke common.ToolsInstaller
	if isEdgeNode {
		ke = &util.KubeEdgeInstTool{Common: util.Common{}}
		err := ke.TearDown()
		if err != nil {
			return fmt.Errorf("TearDown failed, err:%v", err)
		}
	}

	ke = &helm.KubeCloudHelmInstTool{
		Common: util.Common{
			KubeConfig: kubeConfig,
		},
	}
	err := ke.TearDown()
	if err != nil {
		return fmt.Errorf("TearDown failed, err:%v", err)
	}
	return nil
}
