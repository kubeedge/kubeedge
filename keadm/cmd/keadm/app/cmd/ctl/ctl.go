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

package ctl

import (
	"github.com/spf13/cobra"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/confirm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/describe"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/edit"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/exec"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/get"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/logs"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/restart"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/unhold"
)

var ctlShortDescription = `Commands operating on data plane at edge`

var ctlLongDescription = `
The ctl command provides utilities for managing edge nodes and workloads
directly from cloud side, enabling operations on resources running at
the edge without requiring direct SSH access to edge nodes.

Available Commands:
  get       - Get resources on edge nodes (pods, devices)
  restart   - Restart edge components (edgecore)
  logs      - Get logs from pods running on edge nodes
  exec      - Execute commands in pods on edge nodes
  describe  - Show detailed information about edge resources
  edit      - Edit resources on edge nodes
  confirm   - Confirm edge node operations
  unhold    - Release held upgrade operations

Common Usage Examples:

  # List all pods on a specific edge node
  keadm ctl get pods --node edge-node-1

  # Get detailed information about a pod on edge node
  keadm ctl describe pod my-pod --node edge-node-1

  # View real-time logs from a pod
  keadm ctl logs my-pod --node edge-node-1 -f

  # Execute a command inside a pod on edge node
  keadm ctl exec my-pod --node edge-node-1 -- /bin/sh

  # Restart edgecore service on an edge node
  keadm ctl restart edgecore --node edge-node-1

Note: These commands communicate through the cloud-edge channel and
require that EdgeHub is connected and functioning properly.

For detailed help on any command, use:
  keadm ctl [command] --help
`

var ctlExample = `
  # Get pods on edge node
  keadm ctl get pods --node edge-node-1

  # Stream logs from a pod
  keadm ctl logs nginx-pod --node edge-node-1 -f

  # Execute interactive shell in pod
  keadm exec -it nginx-pod --node edge-node-1 -- /bin/bash
`

// NewCtl returns KubeEdge edge pod command.
func NewCtl() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ctl",
		Short:   ctlShortDescription,
		Long:    ctlLongDescription,
		Example: ctlExample,
	}

	cmd.AddCommand(get.NewEdgeGet())
	cmd.AddCommand(restart.NewEdgeRestart())
	cmd.AddCommand(confirm.NewEdgeConfirm())
	cmd.AddCommand(unhold.NewEdgeUnholdUpgrade())
	cmd.AddCommand(logs.NewEdgePodLogs())
	cmd.AddCommand(exec.NewEdgePodExec())
	cmd.AddCommand(describe.NewEdgeDescribe())
	cmd.AddCommand(edit.NewEdgeEdit())
	return cmd
}
