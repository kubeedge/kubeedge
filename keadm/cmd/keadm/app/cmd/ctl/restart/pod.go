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
	"fmt"
	"time"

	"github.com/spf13/cobra"
	oteltrace "go.opentelemetry.io/otel/trace"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/restful"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
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
	for _, podName := range podNames {
		podRequest := &restful.PodRequest{
			Namespace: o.Namespace,
			PodName:   podName,
		}
		pod, err := podRequest.GetPod()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
		if err != nil {
			fmt.Printf("get edge config failed with err:%v\n", err)
			continue
		}
		nodeName := config.Modules.Edged.HostnameOverride
		if nodeName != pod.Spec.NodeName {
			fmt.Printf("can't to restart pod: \"%s\" for node: \"%s\"\n", pod.Name, pod.Spec.NodeName)
			continue
		}
		endpoint := config.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
		remoteRuntimeService, err := remote.NewRemoteRuntimeService(endpoint, time.Second*10, oteltrace.NewNoopTracerProvider())

		var labelSelector = map[string]string{
			"io.kubernetes.pod.name":      pod.Name,
			"io.kubernetes.pod.namespace": pod.Namespace,
		}

		filter := &runtimeapi.ContainerFilter{
			LabelSelector: labelSelector,
		}
		containers, err := remoteRuntimeService.ListContainers(context.TODO(), filter)
		if err != nil {
			return err
		}

		for _, container := range containers {
			containerID := container.Id
			err := remoteRuntimeService.StopContainer(context.TODO(), containerID, 3)
			if err != nil {
				fmt.Printf("stop containerID:%s with err:%v\n", containerID, err)
			} else {
				fmt.Println(containerID)
			}
		}
	}
	return nil
}
