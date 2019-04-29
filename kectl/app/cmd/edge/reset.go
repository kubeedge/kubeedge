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
	"io"

	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/kectl/app/cmd/options"
	"github.com/kubeedge/kubeedge/kectl/app/cmd/util"
)

var (
	edgeResetLongDescription = `
edge reset command will remove the edge from api-server and then tear down KubeEdge 
edge component
`
	edgeResetExample = `
kectl cloud reset --server 10.20.30.40:8080
    - For this command --server option is a Mandatory option
`
)

// NewEdgeReset represents the reset command
func NewEdgeReset(out io.Writer) *cobra.Command {

	K8SAPIServerIPPort := ""
	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns edge component",
		Long:    edgeResetLongDescription,
		Example: edgeResetExample,
		Run: func(cmd *cobra.Command, args []string) {
			// Tear down edge node. It includes
			// 1. Removing edge node from api-server
			// 2. killing edge_core process
			TearDownEdgeNode(K8SAPIServerIPPort)
		},
	}

	//This command requires to know the api-server address so that node can be removed from api-server
	//2 methods, 1. To get it from the flag option and 2. To read from edge.yaml. TODO: method 2
	cmd.Flags().StringVarP(&K8SAPIServerIPPort, options.K8SAPIServerIPPort, "s", K8SAPIServerIPPort,
		"IP:Port address of cloud components host/VM")
	cmd.MarkFlagRequired(options.K8SAPIServerIPPort)

	return cmd
}

func TearDownEdgeNode(server string) {
	edge := &util.KubeEdgeInstTool{Common: util.Common{}, K8SApiServerIP: server}
	edge.TearDown()
}
