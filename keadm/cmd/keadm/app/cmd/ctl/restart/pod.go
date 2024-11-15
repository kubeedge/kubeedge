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

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
)

type PodRestartOptions struct {
	Namespace string
}

var (
	edgePodRestartShortDescription = `Restart pods in edge node`
)

// NewEdgePodRestart returns KubeEdge delete edge pod command.
func NewEdgePodRestart() *cobra.Command {
	deleteOpts := NewRestartPodOpts()
	cmd := &cobra.Command{
		Use:   "pod",
		Short: edgePodRestartShortDescription,
		Long:  edgePodRestartShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 0 {
				return fmt.Errorf("no pod specified for reboot")
			}
			cmdutil.CheckErr(deleteOpts.restartPod(args))
			return nil
		},
		Aliases: []string{"pods", "po"},
	}
	AddRestartPodFlags(cmd, deleteOpts)
	return cmd
}

func NewRestartPodOpts() *PodRestartOptions {
	podDeleteOptions := &PodRestartOptions{}
	podDeleteOptions.Namespace = "default"
	return podDeleteOptions
}

func AddRestartPodFlags(cmd *cobra.Command, RestartPodOptions *PodRestartOptions) {
	cmd.Flags().StringVarP(&RestartPodOptions.Namespace, common.FlagNameNamespace, "n", RestartPodOptions.Namespace,
		"Specify a namespace")
}

func (o *PodRestartOptions) restartPod(podNames []string) error {
	kubeClient, err := client.KubeClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	restartResponse, err := podRestart(ctx, kubeClient, o.Namespace, podNames)
	if err != nil {
		return err
	}

	for _, logMsg := range restartResponse.LogMessages {
		fmt.Println(logMsg)
	}

	for _, errMsg := range restartResponse.ErrMessages {
		fmt.Println(errMsg)
	}
	return nil
}

func podRestart(ctx context.Context, clientSet *kubernetes.Clientset, namespace string, podNames []string) (*types.RestartResponse, error) {
	bodyBytes, err := json.Marshal(podNames)
	if err != nil {
		return nil, err
	}
	result := clientSet.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		SubResource("restart").
		Body(bodyBytes).
		Do(ctx)

	if result.Error() != nil {
		return nil, result.Error()
	}

	statusCode := -1
	result.StatusCode(&statusCode)
	if statusCode != 200 {
		return nil, fmt.Errorf("pod restart failed with status code: %d", statusCode)
	}

	body, err := result.Raw()
	if err != nil {
		return nil, err
	}

	var restartResponse types.RestartResponse
	err = json.Unmarshal(body, &restartResponse)
	if err != nil {
		return nil, err
	}
	return &restartResponse, nil
}
