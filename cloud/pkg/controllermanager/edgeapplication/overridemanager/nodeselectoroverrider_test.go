/*
Copyright 2026 The KubeEdge Authors.

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

package overridemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

func TestNodeSelectorOverriderApplyOverrides(t *testing.T) {
	tests := []struct {
		name            string
		targetNodeGroup string
		nodeSelector    map[string]string
		want            map[string]string
	}{
		{
			name:            "preserves existing selectors",
			targetNodeGroup: "group-a",
			nodeSelector: map[string]string{
				"kubernetes.io/arch": "arm64",
				"accelerator":        "nvidia",
			},
			want: map[string]string{
				"kubernetes.io/arch":       "arm64",
				"accelerator":              "nvidia",
				nodegroup.LabelBelongingTo: "group-a",
			},
		},
		{
			name:            "initializes a nil selector map",
			targetNodeGroup: "group-a",
			want: map[string]string{
				nodegroup.LabelBelongingTo: "group-a",
			},
		},
		{
			name:            "replaces only the stale node group",
			targetNodeGroup: "group-a",
			nodeSelector: map[string]string{
				nodegroup.LabelBelongingTo: "group-b",
				"disk":                     "ssd",
			},
			want: map[string]string{
				nodegroup.LabelBelongingTo: "group-a",
				"disk":                     "ssd",
			},
		},
		{
			name: "leaves selectors unchanged without a target group",
			nodeSelector: map[string]string{
				"disk": "ssd",
			},
			want: map[string]string{
				"disk": "ssd",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       DeploymentKind,
				},
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
						Spec: corev1.PodSpec{
							NodeSelector: tt.nodeSelector,
							Containers:   []corev1.Container{{Name: "test", Image: "nginx:latest"}},
						},
					},
				},
			}

			raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
			require.NoError(t, err)
			rawObj := &unstructured.Unstructured{Object: raw}

			err = (&NodeSelectorOverrider{}).ApplyOverrides(rawObj, OverriderInfo{
				TargetNodeGroup: tt.targetNodeGroup,
			})
			require.NoError(t, err)

			got, err := ConvertToDeployment(rawObj)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.Spec.Template.Spec.NodeSelector)
			assert.Equal(t, deployment.Spec.Template.Labels, got.Spec.Template.Labels)
			assert.Equal(t, deployment.Spec.Template.Spec.Containers, got.Spec.Template.Spec.Containers)
		})
	}
}
