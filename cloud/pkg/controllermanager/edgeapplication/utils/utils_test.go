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

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
)

func TestGetAllOverriders(t *testing.T) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"region": "hangzhou"}}

	tests := []struct {
		name  string
		scope appsv1alpha1.WorkloadScope
		want  []overridemanager.OverriderInfo
	}{
		{
			name: "different node groups and a label selector",
			scope: appsv1alpha1.WorkloadScope{
				TargetNodeGroups: []appsv1alpha1.TargetNodeGroup{
					{Name: "hangzhou", Overriders: appsv1alpha1.Overriders{Replicas: ptr.To(2)}},
					{Name: "beijing"},
				},
				TargetNodeLabels: []appsv1alpha1.TargetNodeLabel{
					{LabelSelector: labelSelector, Overriders: appsv1alpha1.Overriders{Replicas: ptr.To(3)}},
				},
			},
			want: []overridemanager.OverriderInfo{
				{TargetNodeGroup: "hangzhou", Overriders: &appsv1alpha1.Overriders{Replicas: ptr.To(2)}},
				{TargetNodeGroup: "beijing", Overriders: &appsv1alpha1.Overriders{}},
				{TargetNodeLabelSelector: labelSelector, Overriders: &appsv1alpha1.Overriders{Replicas: ptr.To(3)}},
			},
		},
		{
			name: "a node group targeted more than once only uses the first entry",
			scope: appsv1alpha1.WorkloadScope{
				TargetNodeGroups: []appsv1alpha1.TargetNodeGroup{
					{Name: "hangzhou", Overriders: appsv1alpha1.Overriders{Replicas: ptr.To(2)}},
					{Name: "beijing"},
					{Name: "hangzhou"},
					{Name: "hangzhou", Overriders: appsv1alpha1.Overriders{Replicas: ptr.To(5)}},
				},
				TargetNodeLabels: []appsv1alpha1.TargetNodeLabel{
					{LabelSelector: labelSelector},
				},
			},
			want: []overridemanager.OverriderInfo{
				{TargetNodeGroup: "hangzhou", Overriders: &appsv1alpha1.Overriders{Replicas: ptr.To(2)}},
				{TargetNodeGroup: "beijing", Overriders: &appsv1alpha1.Overriders{}},
				{TargetNodeLabelSelector: labelSelector, Overriders: &appsv1alpha1.Overriders{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edgeApp := &appsv1alpha1.EdgeApplication{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "edgeapp"},
				Spec:       appsv1alpha1.EdgeApplicationSpec{WorkloadScope: tt.scope},
			}
			assert.Equal(t, tt.want, GetAllOverriders(edgeApp))
		})
	}
}
