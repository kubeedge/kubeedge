/*
Copyright 2019 The Kubeedge Authors.

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

	types "github.com/kubeedge/kubeedge/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/app/cmd/util"
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
keadm reset --k8sserverip 10.20.30.40:8080
`
)

// NewKubeEdgeReset represents the reset command
func NewKubeEdgeReset(out io.Writer) *cobra.Command {
	IsEdgeNode := false
	K8SAPIServerIPPort := ""
	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns KubeEdge (cloud & edge) component",
		Long:    resetLongDescription,
		Example: resetExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			whoRunning, err := util.IsKubeEdgeController()
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
			// 1. Executing kubeadm reset
			// 2. killing edgecontroller process

			// Tear down edge node. It includes
			// 1. Removing edge node from api-server
			// 2. killing edge_core process
			return TearDownKubeEdge(IsEdgeNode, K8SAPIServerIPPort)
		},
	}

	//This command requires to know the api-server address so that node can be removed from api-server
	//possible 2 methods, 1. To get it from the flag option and 2. To read from edge.yaml. TODO: method 2
	cmd.Flags().StringVarP(&K8SAPIServerIPPort, types.K8SAPIServerIPPort, "k", K8SAPIServerIPPort,
		"IP:Port address of cloud components host/VM")

	return cmd
}

//TearDownKubeEdge will bring down either cloud or edge components,
//depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, server string) error {
	var ke types.ToolsInstaller
	ke = &util.KubeCloudInstTool{Common: util.Common{}}
	if false != isEdgeNode {
		ke = &util.KubeEdgeInstTool{Common: util.Common{}, K8SApiServerIP: server}
	}

	ke.TearDown()
	return nil
}
