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
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/edge"
)

var (
	keadmLongDescription = `
    +----------------------------------------------------------+
    | KEADM                                                    |
    | Easily bootstrap a KubeEdge cluster                      |
    |                                                          |
    | Please give us feedback at:                              |
    | https://github.com/kubeedge/kubeedge/issues              |
    +----------------------------------------------------------+

    Create a cluster with cloud node
    (which controls the edge node cluster), and edge nodes
    (where native containerized application, in the form of
    pods and deployments run), connects to devices.

`
	keadmExample = `
    +-----------------------------------------------------------------+
    | On the edge machine(current dont support cloudcore on windows): |                                    |
    +-----------------------------------------------------------------+
    | worker node (at the edge)# keadm join <flags>                   |
    +-----------------------------------------------------------------+

    You can then repeat the second step on, as many other machines as you like.
`
)

// NewKubeedgeCommand returns cobra.Command to run keadm commands
func NewKubeedgeCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:     "keadm",
		Short:   "keadm: Bootstrap KubeEdge cluster",
		Long:    keadmLongDescription,
		Example: keadmExample,
	}

	cmds.ResetFlags()

	cmds.AddCommand(NewCmdVersion())

	// recommended cmds
	cmds.AddCommand(edge.NewEdgeJoin())
	cmds.AddCommand(NewKubeEdgeReset())

	// beta cmds
	//cmds.AddCommand(edge.NewEdgeUpgrade())

	return cmds
}
