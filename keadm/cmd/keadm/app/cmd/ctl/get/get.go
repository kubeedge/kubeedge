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

package get

import "github.com/spf13/cobra"

var (
	edgeGetShortDescription = `Get resources in edge node`

	edgeGetLongDescription = `
List resources stored in the edge node's local MetaManager database.

This command provides a list of resources (pods, devices) managed on
the current edge node. Unlike 'kubectl get', which queries the cloud
API server, this command reads from the edge node's local MetaManager
database (SQLite-backed).

Supported resource types and aliases:
  pods, po         List all pods on this edge node.
  devices          List all devices managed by this edge node.

Resource filtering:
The list is scoped to the current edge node only. If you specify a
namespace with -n/--namespace, only resources in that namespace are
shown. By default, resources in the 'default' namespace are displayed.

Edge autonomy: This command works even when the cloud connection is
temporarily lost, as it queries local metadata only.

Note: This command must be run directly on the edge node where
EdgeCore is running. The edge node name is read from the local
EdgeCore configuration file automatically.`

	edgeGetExample = `
  # List all pods in the default namespace on this edge node
  keadm ctl get pods

  # List all pods in a specific namespace
  keadm ctl get pods -n <namespace>

  # List all devices in the default namespace
  keadm ctl get devices

  # List all devices in a specific namespace
  keadm ctl get devices -n <namespace>

  # Use the pod alias to list pods
  keadm ctl get po`
)

// NewEdgeGet returns KubeEdge edge resources get command.
func NewEdgeGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   edgeGetShortDescription,
		Long:    edgeGetLongDescription,
		Example: edgeGetExample,
	}

	cmd.AddCommand(NewEdgePodGet())
	cmd.AddCommand(NewEdgeDeviceGet())
	return cmd
}
