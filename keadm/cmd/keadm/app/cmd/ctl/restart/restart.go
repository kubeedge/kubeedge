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

package restart

import "github.com/spf13/cobra"

var (
	edgeRestartShortDescription = `Restart resources in edge node`

	edgeRestartLongDescription = `
Restart resources (such as pods) running on an edge node.

This command communicates with the local MetaService API on the
edge node to trigger a restart of the specified resource. It does
NOT restart EdgeCore itself or affect the cloud-side state.

Restart behaviour for pods:
  - The pod is deleted and recreated by Edged (the edge-side kubelet).
  - If the pod is managed by a Deployment or DaemonSet, a new pod
    is scheduled automatically after deletion.
  - Workloads continue running on the edge even if the cloud connection
    is temporarily lost (edge autonomy is preserved).

Subcommands:
  pod    Restart one or more pods by name in the edge node.

Note: This command must be run directly on the edge node.`

	edgeRestartExample = `
  # Restart a specific pod in the default namespace
  keadm ctl restart pod <pod-name>

  # Restart a pod in a specific namespace
  keadm ctl restart pod <pod-name> -n <namespace>

  # Restart multiple pods at once
  keadm ctl restart pod <pod1> <pod2> <pod3>`
)

// NewEdgeRestart returns KubeEdge restart edge resources command.
func NewEdgeRestart() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restart",
		Short:   edgeRestartShortDescription,
		Long:    edgeRestartLongDescription,
		Example: edgeRestartExample,
	}
	cmd.AddCommand(NewEdgePodRestart())
	return cmd
}
