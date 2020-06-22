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
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
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
func NewKubeEdgeReset(out io.Writer, reset *types.ResetOptions) *cobra.Command {
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
			case types.KubeEdgeEdgeRunning:
				IsEdgeNode = true
			case types.NoneRunning:
				return fmt.Errorf("None of KubeEdge components are running in this host")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Tear down cloud node. It includes
			// 1. killing cloudcore process

			// Tear down edge node. It includes
			// 1. killing edgecore process, but don't delete node from K8s
			return TearDownKubeEdge(IsEdgeNode, reset.Kubeconfig)
		},
	}

	addResetFlags(cmd, reset)
	return cmd
}

// TearDownKubeEdge will bring down either cloud or edge components,
// depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, kubeConfig string) error {
	var ke types.ToolsInstaller
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

func addResetFlags(cmd *cobra.Command, resetOpts *types.ResetOptions) {
	cmd.Flags().StringVar(&resetOpts.Kubeconfig, common.KubeConfig, resetOpts.Kubeconfig,
		"Use this key to set kube-config path, eg: $HOME/.kube/config")
}
