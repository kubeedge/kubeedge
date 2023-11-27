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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	utilsexec "k8s.io/utils/exec"

	"github.com/kubeedge/kubeedge/common/constants"
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

			isEdgeNode = whoRunning == common.KubeEdgeEdgeRunning
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// when reset.Force is not true, user must confirm again
			if err := util.UserConfirm(reset.Force); err != nil {
				return err
			}

			// first cleanup edge node static pod directory to stop static and mirror pod
			if err := util.CleanEdgeNodeDirectories(isEdgeNode); err != nil {
				return err
			}

			// 1. kill cloudcore/edgecore process.
			// For edgecore, don't delete node from K8S
			if err := util.TearDownKubeEdge(isEdgeNode, reset.Kubeconfig); err != nil {
				return err
			}

			// 2. Remove containers managed by KubeEdge. Only for edge node.
			if err := util.RemoveContainers(isEdgeNode, utilsexec.New()); err != nil {
				fmt.Printf("Failed to remove containers: %v\n", err)
			}

			// 3. Clean stateful directories
			if err := util.CleanDirectories(isEdgeNode); err != nil {
				return err
			}

			// cleanup mqtt container
			if err := util.RemoveMqttContainer(reset.RuntimeType, reset.Endpoint, ""); err != nil {
				fmt.Printf("Failed to remove MQTT container: %v\n", err)
			}
			//4. TODO: clean status information

			return nil
		},
	}

	addResetFlags(cmd, reset)
	return cmd
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
