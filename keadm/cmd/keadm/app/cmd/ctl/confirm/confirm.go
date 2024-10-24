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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
)

var (
	edgeConfirmShortDescription = `Send a confirmation signal to the MetaService API`
)

// NewEdgeConfirm returns KubeEdge confirm command.
func NewEdgeConfirm() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confirm",
		Short: edgeConfirmShortDescription,
		Long:  edgeConfirmShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return errors.New("no specified node name for confirm upgrade")
			}
			cmdutil.CheckErr(confirmNode())
			return nil
		},
	}
	return cmd
}
func confirmNode() error {
	kubeClient, err := client.KubeClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	confirmResponse, err := nodeConfirm(ctx, kubeClient)
	if err != nil {
		return err
	}

	for _, logMsg := range confirmResponse.LogMessages {
		klog.Info(logMsg)
	}

	for _, errMsg := range confirmResponse.ErrMessages {
		klog.Error(errMsg)
	}
	return nil
}

func nodeConfirm(ctx context.Context, clientSet *kubernetes.Clientset) (*types.NodeUpgradeConfirmResponse, error) {
	result := clientSet.CoreV1().RESTClient().Post().
		Resource("taskupgrade").
		SubResource("confirm-upgrade").
		Do(ctx)

	if result.Error() != nil {
		return nil, result.Error()
	}

	statusCode := -1
	result.StatusCode(&statusCode)
	if statusCode != 200 {
		return nil, fmt.Errorf("node upgrade confirm failed with status code: %d", statusCode)
	}

	body, err := result.Raw()
	if err != nil {
		return nil, err
	}

	var nodeUpgradeConfirmResponse types.NodeUpgradeConfirmResponse
	err = json.Unmarshal(body, &nodeUpgradeConfirmResponse)
	if err != nil {
		return nil, err
	}
	return &nodeUpgradeConfirmResponse, nil
}
