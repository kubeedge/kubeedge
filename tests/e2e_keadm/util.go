/*
Copyright 2023 The KubeEdge Authors.

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

package keadm

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/tests/e2e/constants"
)

func newPod(podName, imgURL string) *v1.Pod {
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"app":                 podName,
				constants.E2ELabelKey: constants.E2ELabelValue,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  podName,
					Image: imgURL,
				},
			},
			NodeSelector: map[string]string{
				"node-role.kubernetes.io/edge": "",
			},
			Tolerations: []v1.Toleration{
				{
					Key:      v1.TaintNodeDiskPressure,
					Operator: v1.TolerationOpExists,
				},
			},
		},
	}
	return &pod
}
