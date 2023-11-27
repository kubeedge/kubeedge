//go:build windows

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
*/

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	kubetypes "k8s.io/kubernetes/pkg/kubelet/types"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	resetLongDescription = `
keadm reset command in windows can only be executed in edge node
It shut down the edge processes of KubeEdge
`
	resetExample = `
keadm reset
`
)

func newResetOptions() *common.ResetOptions {
	opts := &common.ResetOptions{}
	opts.Kubeconfig = common.DefaultKubeConfig
	opts.RuntimeType = kubetypes.RemoteContainerRuntime
	return opts
}

func NewKubeEdgeReset() *cobra.Command {
	reset := newResetOptions()

	//currently keadm only supports edge node management
	isEdgeNode := true

	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns KubeEdge edge component in windows server",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !util.IsNSSMInstalled() {
				fmt.Println("Seems like you haven't exec 'keadm join' in this host, because nssm not found in system path (auto installed by 'keadm join'), exit")
				os.Exit(0)
			}
			whoRunning := util.RunningModuleV2(reset)
			if whoRunning == common.NoneRunning && !reset.Force {
				fmt.Println("Edgecore service installed by nssm not found in this host, exit. If you want to clean the related files, using flag --force")
				os.Exit(0)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// when reset.Force is not true, user must confirm again
			if err := util.UserConfirm(reset.Force); err != nil {
				return err
			}

			// 1. kill edgecore process.
			// For edgecore, don't delete node from K8S
			if err := util.TearDownKubeEdge(true, reset.Kubeconfig); err != nil {
				err = fmt.Errorf("err when stop and remove edgecore using nssm: %s", err.Error())
				fmt.Print("[reset] No edgecore running now, do you want to clean all the related directories? [y/N]: ")
				s := bufio.NewScanner(os.Stdin)
				s.Scan()
				if err := s.Err(); err != nil {
					return err
				}
				if strings.ToLower(s.Text()) != "y" {
					return fmt.Errorf("aborted reset operation")
				}
				return util.CleanDirectories(isEdgeNode)
			}

			// 2. Remove containers managed by KubeEdge
			if err := util.RemoveContainers(isEdgeNode, utilsexec.New()); err != nil {
				fmt.Printf("Failed to remove containers: %v\n", err)
			}

			// 3. Clean stateful directories
			if err := util.CleanDirectories(isEdgeNode); err != nil {
				return err
			}

			fmt.Println("Reset Complete")

			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

func addResetFlags(cmd *cobra.Command, resetOpts *common.ResetOptions) {
	//cmd.Flags().StringVar(&resetOpts.Kubeconfig, common.KubeConfig, resetOpts.Kubeconfig,
	//	"Use this key to set kube-config path, eg: $HOME/.kube/config")
	cmd.Flags().BoolVar(&resetOpts.Force, "force", resetOpts.Force,
		"Reset the node without prompting for confirmation, and continue even if running edgecore not found")
	cmd.Flags().StringVar(&resetOpts.Endpoint, common.RemoteRuntimeEndpoint, resetOpts.Endpoint,
		"Use this key to set container runtime endpoint")
}
