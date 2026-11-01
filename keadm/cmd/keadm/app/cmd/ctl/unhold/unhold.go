/*
Copyright 2025 The KubeEdge Authors.

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

package unhold

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

var (
	edgeUnholdLongDescription = `
Release the upgrade hold on a pod or node, allowing a previously
paused (held) upgrade to resume.

KubeEdge supports a hold/unhold mechanism during upgrades. When an
upgrade task is created with hold enabled, the upgrade process pauses
at a checkpoint and waits for explicit user approval before continuing.

This command sends an unhold signal via the local MetaService API to
release that checkpoint for the specified resource.

Resource types:
  pod   Unhold the upgrade for a specific pod in a namespace.
        Requires the pod name. Uses --namespace (default: "default").
  node  Unhold the upgrade for the entire edge node.
        If no node name is given, reads the node name from the local
        EdgeCore configuration file automatically.

Upgrade workflow with hold/unhold:
  1. Cloud creates an upgrade task with hold=true.
  2. EdgeCore upgrades but pauses at the checkpoint.
  3. User inspects the node and confirms it is safe to proceed.
  4. User runs 'keadm ctl unhold-upgrade node' (or 'pod') to release.
  5. Upgrade resumes and completes.

WARNING: Releasing a hold resumes the upgrade immediately. Ensure
the node and its workloads are in a healthy state before running
this command.

Note: This command must be run directly on the edge node where
EdgeCore is running. It communicates with the local MetaService API.`

	edgeUnholdExample = `
  # Unhold the upgrade for the current edge node (node name read from EdgeCore config)
  keadm ctl unhold-upgrade node

  # Unhold the upgrade for a specific edge node by name
  keadm ctl unhold-upgrade node <node-name>

  # Unhold the upgrade for a specific pod in the default namespace
  keadm ctl unhold-upgrade pod <pod-name>

  # Unhold the upgrade for a pod in a specific namespace
  keadm ctl unhold-upgrade pod <pod-name> --namespace <namespace>`
)

// NewEdgeUnholdUpgrade returns KubeEdge unhold-upgrade command.
func NewEdgeUnholdUpgrade() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unhold-upgrade <resource-type> [<name>] [--namespace namespace]",
		Short:   "Unhold an upgrade for a pod or node, allowing it to resume.",
		Long:    edgeUnholdLongDescription,
		Example: edgeUnholdExample,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resourceType := args[0]
			var resourceName string
			if len(args) > 1 {
				resourceName = args[1]
			}
			namespace, _ := cmd.Flags().GetString("namespace")
			if resourceType == "pod" {
				if resourceName == "" {
					return fmt.Errorf("pod name is required")
				}
				return unholdPodUpgrade(fmt.Sprintf("%s/%s", namespace, resourceName))
			} else if resourceType == "node" {
				if resourceName == "" {
					var err error
					resourceName, err = getCurrentNodeName()
					if err != nil {
						return err
					}
				}
				return unholdNodeUpgrade(resourceName)
			} else {
				return fmt.Errorf("error: unknown resource type: %s", resourceType)
			}
		},
	}
	cmd.Flags().StringP("namespace", "n", "default", "Namespace of the pod (defaults to 'default')")
	return cmd
}

func unholdPodUpgrade(target string) error {
	return unholdResourceUpgrade("pods", target)
}

func unholdNodeUpgrade(target string) error {
	return unholdResourceUpgrade("nodes", target)
}

func unholdResourceUpgrade(resource, target string) error {
	ctx := context.Background()
	clientset, err := metaclient.KubeClient()
	if err != nil {
		return err
	}

	result := clientset.CoreV1().RESTClient().Post().
		Resource(resource).
		SubResource("unhold-upgrade").
		SetHeader("Content-Type", "text/plain").
		Body([]byte(target)).
		Do(ctx)

	if err := result.Error(); err != nil {
		// Do not use the wrapped error when an error http code is returned.
		stErr, ok := err.(*apierrors.StatusError)
		if ok || errors.As(err, &stErr) {
			var msg string
			if dtl := stErr.Status().Details; dtl != nil && len(dtl.Causes) > 0 {
				msg = dtl.Causes[0].Message
			} else {
				msg = stErr.Status().Message
			}
			return fmt.Errorf("failed to unhold %s upgrade, status code: %d, message: %s",
				resource, stErr.Status().Code, msg)
		}
		return fmt.Errorf("failed to send unhold request to MetaService API, err: %v", result.Error())
	}

	return nil
}

func getCurrentNodeName() (string, error) {
	config, err := util.ParseEdgecoreConfig(constants.EdgecoreConfigPath)
	if err != nil {
		return "", fmt.Errorf("get edge config failed with err:%v", err)
	}
	nodeName := config.Modules.Edged.HostnameOverride
	return nodeName, nil
}
