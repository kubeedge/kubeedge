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

package edgecore

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/metaclient"
)

func GetVersion(ctx context.Context, config *v1alpha2.EdgeCoreConfig) string {
	kubecli, err := metaclient.KubeClientWithConfig(config)
	if err != nil {
		klog.Warningf("failed to get kube client, uses default version: %s, err: %v", config.EdgeCoreVersion, err)
		return config.EdgeCoreVersion
	}
	nodeName := config.Modules.Edged.HostnameOverride
	node, err := kubecli.CoreV1().Nodes().
		Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("failed to get node %s, uses default version: %s, err: %v", nodeName, config.EdgeCoreVersion, err)
		return config.EdgeCoreVersion
	}
	if kver := node.Status.NodeInfo.KubeletVersion; kver != "" {
		arr := strings.SplitN(kver, "-", 3)
		return arr[len(arr)-1]
	}
	return ""
}
