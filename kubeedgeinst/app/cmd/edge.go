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

	edge "github.com/kubeedge/kubeedge/kubeedgeinst/app/cmd/edge"
)

var (
	edgeLongDescription = `
'edge' commands help in operating with KubeEdge's edge component.
`
	edgeExample = `
kectl edge join <options> 
kectl edge reset
`
)

// NewCmdEdge represents the Edge command
func NewCmdEdge(out io.Writer) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "edge",
		Short:   "Edge component command option for KubeEdge",
		Long:    edgeLongDescription,
		Example: edgeExample,
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(edge.NewEdgeJoin(out, nil))
	cmd.AddCommand(edge.NewEdgeReset(out))
	return cmd
}

// func init() {
// }
