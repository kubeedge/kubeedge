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

	node "github.com/kubeedge/kubeedge/kubeedgeinst/app/cmd/node"
)

var (
	nodeLongDescription = `
node commands help in operating with KubeEdge's edge component.
`
	nodeExample = `
kubeedge node join <arguments> 
kubeedge node reset
`
)

// NewCmdNode represents the node command
func NewCmdNode(out io.Writer) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "node",
		Short:   "Edge component command option for KubeEdge",
		Long:    nodeLongDescription,
		Example: nodeExample,
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(node.NewNodeJoin(out, nil))
	cmd.AddCommand(node.NewNodeReset(out))
	return cmd
}

// func init() {
// }
