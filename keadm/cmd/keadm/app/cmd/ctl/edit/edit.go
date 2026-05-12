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
package edit

import "github.com/spf13/cobra"

var (
	edgeEditShortDescription = `Edit a specific resource`

	edgeEditLongDescription = `
Edit a resource stored in the local edge node MetaManager database.

This command opens the resource definition in your default terminal
editor (controlled by the EDITOR or KUBE_EDITOR environment variable).
After you save and close the editor, the updated definition is applied
to the edge node via the MetaService API.

Currently supported resources:
  device   Edit a Device resource by name. Device properties such as
           desired state and twin metadata can be modified.

How edit works:
  1. The current resource definition is fetched from the edge node's
     local MetaManager (SQLite-backed, not from the cloud API server).
  2. The definition is written to a temp file and opened in your editor.
  3. After saving, the changed fields are validated and patched back
     to the local MetaManager.
  4. Changes are then synced to the cloud via EdgeHub when the
     cloud-edge connection is available.

WARNING: Editing resources directly on the edge node bypasses normal
Kubernetes admission webhooks and validation. Only modify fields you
understand. Malformed YAML will be rejected but some invalid values
may not be caught until runtime.

Note: This command must be run directly on the edge node where
EdgeCore is running.`

	edgeEditExample = `
  # Edit a device by name in the default namespace
  keadm ctl edit device <device-name>

  # Edit a device in a specific namespace
  keadm ctl edit device <device-name> -n <namespace>

  # Use a specific editor (overrides EDITOR env variable)
  KUBE_EDITOR=nano keadm ctl edit device <device-name>`
)

// NewEdgeEdit returns KubeEdge edit edge resources command.
func NewEdgeEdit() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "edit",
		Short:   edgeEditShortDescription,
		Long:    edgeEditLongDescription,
		Example: edgeEditExample,
	}
	cmd.AddCommand(NewEdgeEditDevice())
	return cmd
}
