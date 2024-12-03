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

package get

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/cmd/get"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	api "k8s.io/kubernetes/pkg/apis/core"
	k8s_v1_api "k8s.io/kubernetes/pkg/apis/core/v1"
	k8sprinters "k8s.io/kubernetes/pkg/printers"
	printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"
	"k8s.io/kubernetes/pkg/printers/storage"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/client"
	ctlcommon "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/ctl/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

var (
	edgePodGetShortDescription = `Get pods in edge node`
)

type PodGetOptions struct {
	Namespace     string
	LabelSelector string
	AllNamespaces bool
	Output        string
	ctlcommon.ExtPrintFlags
}

// NewEdgePodGet returns KubeEdge edge pod command.
func NewEdgePodGet() *cobra.Command {
	podGetOptions := NewGetOpts()
	cmd := &cobra.Command{
		Use:   "pod",
		Short: edgePodGetShortDescription,
		Long:  edgePodGetShortDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.CheckErr(podGetOptions.getPods(args))
			return nil
		},
		Aliases: []string{"pods", "po"},
	}
	AddGetPodFlags(cmd, podGetOptions)
	return cmd
}

func (o *PodGetOptions) getPods(args []string) error {
	config, err := util.ParseEdgecoreConfig(common.EdgecoreConfigPath)
	if err != nil {
		return fmt.Errorf("get edge config failed with err:%v", err)
	}
	nodeName := config.Modules.Edged.HostnameOverride

	ctx := context.Background()
	var podList *v1.PodList
	if len(args) > 0 {
		podList = &v1.PodList{
			Items: make([]v1.Pod, 0, len(args)),
		}
		var podRequest *client.PodRequest
		for _, podName := range args {
			podRequest = &client.PodRequest{
				Namespace: o.Namespace,
				PodName:   podName,
			}
			pod, err := podRequest.GetPod(ctx)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}

			if pod.Spec.NodeName == nodeName {
				podList.Items = append(podList.Items, *pod)
			} else {
				fmt.Printf("can't to query pod: \"%s\" for node: \"%s\"\n", pod.Name, pod.Spec.NodeName)
			}
		}
	} else {
		podRequest := &client.PodRequest{
			Namespace:     o.Namespace,
			AllNamespaces: o.AllNamespaces,
			LabelSelector: o.LabelSelector,
		}
		podList, err = podRequest.GetPods(ctx)
		if err != nil {
			return err
		}
	}

	if len(podList.Items) == 0 {
		if len(args) > 0 {
			return nil
		}
		if o.AllNamespaces {
			fmt.Println("No resources found in all namespace.")
		} else {
			fmt.Printf("No resources found in %s namespace.\n", o.Namespace)
		}
		return nil
	}

	if *o.PrintFlags.OutputFormat == "" || *o.PrintFlags.OutputFormat == "wide" {
		podListFilter := &api.PodList{
			Items: make([]api.Pod, 0, len(podList.Items)),
		}
		for _, pod := range podList.Items {
			if pod.Spec.NodeName == nodeName {
				var apiPod api.Pod
				if err := k8s_v1_api.Convert_v1_Pod_To_core_Pod(&pod, &apiPod, nil); err != nil {
					fmt.Printf("pod revert to apiPod with err:%v\n", err)
					continue
				}
				podListFilter.Items = append(podListFilter.Items, apiPod)
			}
		}
		table, err := ConvertDataToTable(podListFilter)
		if err != nil {
			return err
		}
		return o.PrintToTable(table, o.AllNamespaces, os.Stdout)
	}

	runtimeObjects := make([]runtime.Object, 0, len(podList.Items))
	for _, pod := range podList.Items {
		runtimeObjects = append(runtimeObjects, &pod)
	}
	return o.PrintToJSONYaml(runtimeObjects)
}

func NewGetOpts() *PodGetOptions {
	podGetOptions := &PodGetOptions{}
	podGetOptions.Namespace = "default"
	podGetOptions.PrintFlags = get.NewGetPrintFlags()
	podGetOptions.PrintFlags.OutputFormat = &podGetOptions.Output
	return podGetOptions
}

func AddGetPodFlags(cmd *cobra.Command, getOptions *PodGetOptions) {
	cmd.Flags().StringVarP(&getOptions.Namespace, common.FlagNameNamespace, "n", getOptions.Namespace,
		"Specify a namespace")

	cmd.Flags().StringVarP(&getOptions.LabelSelector, common.FlagNameLabelSelector, "l", getOptions.LabelSelector,
		"Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)")

	cmd.Flags().StringVarP(&getOptions.Output, common.FlagNameOutput, "o", getOptions.Output,
		"Indicate the output format. Currently supports formats such as yaml|json|wide")

	cmd.Flags().BoolVarP(&getOptions.AllNamespaces, common.FlagNameAllNamespaces, "A", getOptions.AllNamespaces,
		"If present, list the requested object(s) across all namespaces. Namespace in current context is ignored even if specified with --namespace")
}

func ConvertDataToTable(obj runtime.Object) (runtime.Object, error) {
	to := metav1.TableOptions{}
	tc := storage.TableConvertor{TableGenerator: k8sprinters.NewTableGenerator().With(printersinternal.AddHandlers)}

	return tc.ConvertToTable(context.TODO(), obj, &to)
}
