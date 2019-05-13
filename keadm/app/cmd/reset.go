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

	"github.com/kubeedge/kubeedge/keadm/app/cmd/options"
	"github.com/kubeedge/kubeedge/keadm/app/cmd/util"
)

var (
	resetLongDescription = `
kubeedge reset command can be executed in both cloud and edge node
In cloud node it shuts down the cloud processes of KubeEdge
In edge node it shuts down the edge processes of KubeEdge
`
	resetExample = `
For cloud node:
kubeedge reset

For edge node:
kubeedge reset --server 10.20.30.40:8080
    - For this command --server option is a Mandatory option
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
			case util.KubeEdgeEdgeRunning:
				IsEdgeNode = true
			case util.NoneRunning:
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
	//2 methods, 1. To get it from the flag option and 2. To read from edge.yaml. TODO: method 2
	cmd.Flags().StringVarP(&K8SAPIServerIPPort, options.K8SAPIServerIPPort, "s", K8SAPIServerIPPort,
		"IP:Port address of cloud components host/VM")
	//cmd.MarkFlagRequired(options.K8SAPIServerIPPort)

	return cmd
}

//TearDownKubeEdge will bring down either cloud or edge components,
//depending upon in which type of node it is executed
func TearDownKubeEdge(isEdgeNode bool, server string) error {
	var ke util.ToolsInstaller
	ke = &util.KubeCloudInstTool{Common: util.Common{}}
	if false != isEdgeNode {
		if server == "" {
			return fmt.Errorf("On KubeEdge Edge node '--server' option is mandatory with 'kubeedge reset' command ")
		}
		ke = &util.KubeEdgeInstTool{Common: util.Common{}, K8SApiServerIP: server}
	}

	ke.TearDown()
	return nil
}
