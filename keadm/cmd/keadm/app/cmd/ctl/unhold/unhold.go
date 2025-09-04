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
		Use:   "unhold-upgrade pod <pod-name>  [--namespace namespace]",
		Short: "Unhold an upgrade for a pod",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("error: requires exactly two arguments (resource type and name, e.g.,  'pod <pod-name>')")
			}
			resourceType := args[0]
			resourceName := args[1]
			namespace, _ := cmd.Flags().GetString("namespace")

			if resourceType == "pod" {
				return unholdPodUpgrade(fmt.Sprintf("%s/%s", namespace, resourceName))
			} else {
				return fmt.Errorf("error: unknown resource type: %s", resourceType)
			}
		},
	}
	cmd.Flags().StringP("namespace", "n", "default", "Namespace of the pod (defaults to 'default')")
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
			return fmt.Errorf("failed to unhold pod upgrade, status code: %d, message: %s",
				stErr.Status().Code, msg)
		}
		return fmt.Errorf("failed to send unhold request to MetaService API, err: %v", result.Error())
	}

	return nil
}
