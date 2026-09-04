/*
Copyright 2024 The KubeEdge Authors.

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

package describe

import "github.com/spf13/cobra"

var edgeDescribeShortDescription = `Show details of a specific resource`

var edgeDescribeLongDescription = `Show details of a specific resource.

This command is intended to be run on the edge node where EdgeCore is running. It queries the local MetaServer API to retrieve the details of a resource.
Please ensure that the MetaServer module is enabled in your edgecore.yaml configuration.

Currently, only 'pod' and 'device' resources are supported by this command.`

// NewEdgeDescribe returns KubeEdge edge resources describe command.
func NewEdgeDescribe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: edgeDescribeShortDescription,
		Long:  edgeDescribeLongDescription,
	}

	cmd.AddCommand(NewEdgeDescribePod())
	cmd.AddCommand(NewEdgeDescribeDevice())
	return cmd
}
