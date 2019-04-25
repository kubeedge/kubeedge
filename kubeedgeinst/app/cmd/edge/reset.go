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
)

var (
	edgeResetLongDescription = `
edge reset command will tear down KubeEdge 
edge component and disconnect with cloud
`
	edgeResetExample = `
kectl cloud reset 
`
)

// NewEdgeReset represents the reset command
func NewEdgeReset(out io.Writer) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "reset",
		Short:   "Teardowns edge component",
		Long:    edgeResetLongDescription,
		Example: edgeResetExample,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Work your own magic here
			fmt.Println("edge reset called")
		},
		Args: cobra.NoArgs,
	}
	return cmd
}
