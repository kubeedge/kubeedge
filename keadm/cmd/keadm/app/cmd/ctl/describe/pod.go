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

package describe

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/describe"
	api "k8s.io/kubernetes/pkg/apis/core"
	k8s_v1_api "k8s.io/kubernetes/pkg/apis/core/v1"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var edgeDescribePodShortDescription = `Describe pod in edge node`

type PodDescribeOptions struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	// ShowEvents is true if events should be included in the description. Default is false.
	ShowEvents bool
	// ChunkSize is the number of bytes to include in a chunk. Default is 500.
	ChunkSize int64
	genericiooptions.IOStreams
}

// NewEdgeDescribePod returns KubeEdge describe edge pod command.
func NewEdgeDescribePod() *cobra.Command {
	describePodOptions := NewDescribePodOptions()
	cmd := &cobra.Command{
		Use:   "pod",
		Short: edgeDescribePodShortDescription,
		Long:  edgeDescribePodShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.CheckErr(describePodOptions.describePod(args))
			return nil
		},
		Aliases: []string{"pods", "po"},
	}
	AddDescribePodFlags(cmd, describePodOptions)
	return cmd
}

func NewDescribePodOptions() *PodDescribeOptions {
	return &PodDescribeOptions{
		IOStreams: genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
	}
}

func AddDescribePodFlags(cmd *cobra.Command, options *PodDescribeOptions) {
	cmd.Flags().StringVarP(&options.Namespace, common.FlagNameNamespace, "n", "default", "If present, the namespace scope for this CLI request")
	cmd.Flags().StringVarP(&options.LabelSelector, common.FlagNameLabelSelector, "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")
	cmd.Flags().BoolVarP(&options.AllNamespaces, common.FlagNameAllNamespaces, "A", false, "If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace.")
	cmd.Flags().BoolVar(&options.ShowEvents, common.FlagNameShowEvents, false, "If present, display events related to the described object.")
	cmd.Flags().Int64Var(&options.ChunkSize, common.FlagNameChunkSize, 500, "If non-zero, split the output into chunks where each chunk contains N items")
}

func (o *PodDescribeOptions) describePod(args []string) error {
	config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return fmt.Errorf("get edge config failed with err:%v", err)
	}
	nodeName := config.Modules.Edged.HostnameOverride

	ctx := context.Background()

	var podListFilter *api.PodList

	if len(args) > 0 {
		podListFilter = &api.PodList{
			Items: make([]api.Pod, 0, len(args)),
		}

		var podRequest *client.PodRequest
		for _, podName := range args {
			podRequest = &client.PodRequest{
				Namespace: o.Namespace,
				PodName:   podName,
			}
			pod, err := podRequest.GetPod(ctx)
			if err != nil {
				klog.Error(err.Error())
				continue
			}

			if pod.Spec.NodeName == nodeName {
				var apiPod api.Pod
				if err := k8s_v1_api.Convert_v1_Pod_To_core_Pod(pod, &apiPod, nil); err != nil {
					klog.Errorf("failed to convert pod with err:%v\n", err)
					continue
				}
				podListFilter.Items = append(podListFilter.Items, apiPod)
			} else {
				klog.Errorf("can't to query pod: \"%s\" for node: \"%s\"\n", pod.Name, pod.Spec.NodeName)
			}
		}
	} else {
		podRequest := &client.PodRequest{
			Namespace:     o.Namespace,
			AllNamespaces: o.AllNamespaces,
			LabelSelector: o.LabelSelector,
		}
		podList, err := podRequest.GetPods(ctx)
		if err != nil {
			return err
		}
		podListFilter = &api.PodList{
			Items: make([]api.Pod, 0, len(podList.Items)),
		}

		for _, pod := range podList.Items {
			if pod.Spec.NodeName == nodeName {
				var apiPod api.Pod
				if err := k8s_v1_api.Convert_v1_Pod_To_core_Pod(&pod, &apiPod, nil); err != nil {
					return err
				}
				podListFilter.Items = append(podListFilter.Items, apiPod)
			}
		}
	}

	if len(podListFilter.Items) == 0 {
		if len(args) > 0 {
			return nil
		}
		if o.AllNamespaces {
			klog.Info("No resources found in all namespaces.")
		} else {
			klog.Infof("No resources found in %s namespaces.", o.Namespace)
		}
		return nil
	}

	NamespaceToPodName := make(map[string][]string)

	for _, pod := range podListFilter.Items {
		if _, ok := NamespaceToPodName[pod.Namespace]; !ok {
			NamespaceToPodName[pod.Namespace] = make([]string, 0)
		}
		NamespaceToPodName[pod.Namespace] = append(NamespaceToPodName[pod.Namespace], pod.Name)
	}

	c, err := client.KubeClient()
	if err != nil {
		return err
	}

	d := describe.PodDescriber{Interface: c}

	first := true
	for namespace, podName := range NamespaceToPodName {
		for _, podName := range podName {
			settings := describe.DescriberSettings{
				ShowEvents: o.ShowEvents,
				ChunkSize:  o.ChunkSize,
			}
			s, err := d.Describe(namespace, podName, settings)
			if err != nil {
				return err
			}

			if first {
				first = false
				klog.Info(s)
			} else {
				klog.Infof("\n\n%s", s)
			}
		}
	}
	return nil
}
