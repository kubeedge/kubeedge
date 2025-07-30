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

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

// NewEdgeUnholdUpgrade returns KubeEdge unhold-upgrade command.
func NewEdgeUnholdUpgrade() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unhold-upgrade (node <node-name> | pod <pod-name>)",
		Short: "Unhold an upgrade for a node or pod",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("error: resource type and name are required")
			}
			resourceType := args[0] // "node" or "pod"
			resourceName := args[1] // "node1" or "mypod"
			namespace, _ := cmd.Flags().GetString("namespace")

			if resourceType == "node" {
				// TODO:
				return nil
			} else if resourceType == "pod" {
				if namespace == "" {
					return fmt.Errorf("error: --namespace is required for pods")
				}
				return unholdPodUpgrade(fmt.Sprintf("%s/%s", namespace, resourceName))
			} else {
				return fmt.Errorf("error: unknown resource type: %s", resourceType)
			}
		},
	}
	cmd.Flags().StringP("namespace", "n", "", "Namespace of the pod")
	return cmd
}

func unholdPodUpgrade(target string) error {
	ctx := context.Background()
	clientset, err := metaclient.KubeClient()
	if err != nil {
		return err
	}

	result := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		SubResource("unhold-upgrade").
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
			return fmt.Errorf("failed to unhold pod upgrade, status code: %d, message: %s",
				stErr.Status().Code, msg)
		}
		return fmt.Errorf("failed to send unhold request to MetaService API, err: %v", result.Error())
	}

	return nil
}
