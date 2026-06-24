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

var (
	edgeDescribeShortDescription = `Show details of a specific resource`

	edgeDescribeLongDescription = `
Show detailed information about resources stored in the edge node's
local MetaManager database.

This command provides detailed descriptions of edge resources (pods
and devices) stored locally on the edge node. Unlike 'kubectl describe',
which queries the cloud API server, this command reads from the edge
node's local MetaManager database (SQLite-backed).

Supported resource types:
  pod      Show details of a pod running on this edge node.
  device   Show details of a device managed by this edge node.

The output includes resource metadata, status, and system annotations
stored in the local MetaManager.

Edge autonomy: This command works even when the cloud connection is
temporarily lost, as it queries local metadata only.

Note: This command must be run directly on the edge node where
EdgeCore is running.`

	edgeDescribeExample = `
  # Describe a pod in the default namespace
  keadm ctl describe pod <pod-name>

  # Describe a pod in a specific namespace
  keadm ctl describe pod <pod-name> -n <namespace>

  # Describe a device in the default namespace
  keadm ctl describe device <device-name>

  # Describe a device in a specific namespace
  keadm ctl describe device <device-name> -n <namespace>`
)

// NewEdgeDescribe returns KubeEdge edge resources describe command.
func NewEdgeDescribe() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "describe",
		Short:   edgeDescribeShortDescription,
		Long:    edgeDescribeLongDescription,
		Example: edgeDescribeExample,
	}

	cmd.AddCommand(NewEdgeDescribePod())
	cmd.AddCommand(NewEdgeDescribeDevice())
	return cmd
}
