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

package confirm

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

var (
	edgeConfirmLongDescription = `
Send a confirmation signal to the MetaService API to acknowledge
that the current edge node upgrade has been verified and can proceed.

This command is used as part of the KubeEdge upgrade workflow.
When an edge node upgrade is held (paused at a checkpoint), the
user must manually inspect the node and run 'keadm ctl confirm'
to signal that the upgrade can continue.

Typical workflow:
  1. Cloud side triggers a node upgrade task.
  2. EdgeCore pauses at the upgrade checkpoint (hold state).
  3. User verifies node state, application health, and readiness.
  4. User runs 'keadm ctl confirm' to release the hold.
  5. EdgeCore continues and completes the upgrade.

Note: This command must be run directly on the edge node where
EdgeCore is running. It communicates with the local MetaService
API (not the cloud).`

	edgeConfirmExample = `
  # Confirm the pending upgrade on this edge node
  keadm ctl confirm`
)

// NewEdgeConfirm returns KubeEdge confirm command.
func NewEdgeConfirm() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "confirm",
		Short:   "Confirm a held upgrade on this edge node.",
		Long:    edgeConfirmLongDescription,
		Example: edgeConfirmExample,
		RunE: func(_cmd *cobra.Command, _args []string) error {
			ctx := context.Background()
			clientset, err := metaclient.KubeClient()
			if err != nil {
				return err
			}
			return confirmNodeUpgrade(ctx, clientset)
		},
	}
	return cmd
}

func confirmNodeUpgrade(ctx context.Context, clientSet kubernetes.Interface) error {
	result := clientSet.CoreV1().RESTClient().Post().
		Resource("taskupgrade").
		SubResource("confirm-upgrade").
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
			return fmt.Errorf("failed to confirm node upgrade, status code: %d, message: %s",
				stErr.Status().Code, msg)
		}
		return fmt.Errorf("failed to send confirm request to MetaService API, err: %v", result.Error())
	}
	return nil
}
