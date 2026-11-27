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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNameOverrider_ApplyOverrides(t *testing.T) {
	labels := map[string]string{"app": "nginx", "zone": "edge-a"}

	tests := []struct {
		name       string
		kind       string
		overriders OverriderInfo
		wantName   string
	}{
		{
			name:     "no target keeps the name",
			kind:     DeploymentKind,
			wantName: "nginx",
		},
		{
			name:       "node group is appended to the name",
			kind:       DeploymentKind,
			overriders: OverriderInfo{TargetNodeGroup: "beijing"},
			wantName:   "nginx-beijing",
		},
		{
			name: "label selector suffix is appended to the name",
			kind: DeploymentKind,
			overriders: OverriderInfo{
				TargetNodeLabelSelector: metav1.LabelSelector{MatchLabels: labels},
			},
			wantName: "nginx-" + CreateSuffixFromLabels(labels),
		},
		{
			name: "label selector without match labels keeps the name",
			kind: DeploymentKind,
			overriders: OverriderInfo{
				TargetNodeLabelSelector: metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{Key: "app", Operator: metav1.LabelSelectorOpExists},
					},
				},
			},
			wantName: "nginx",
		},
		{
			name:       "names of other kinds are overridden as well",
			kind:       "Service",
			overriders: OverriderInfo{TargetNodeGroup: "beijing"},
			wantName:   "nginx-beijing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       tt.kind,
					"apiVersion": "apps/v1",
					"metadata": map[string]interface{}{
						"name":      "nginx",
						"namespace": "default",
					},
				},
			}

			overrider := &NameOverrider{}
			assert.NoError(t, overrider.ApplyOverrides(rawObj, tt.overriders))
			assert.Equal(t, tt.wantName, rawObj.GetName())
			assert.Equal(t, "default", rawObj.GetNamespace())
		})
	}
}

func TestCreateSuffixFromLabels(t *testing.T) {
	labels := map[string]string{"app": "nginx", "zone": "edge-a"}

	t.Run("suffix is stable", func(t *testing.T) {
		// The suffix is part of the names of the workloads deployed for a label
		// selector, so changing it renames them.
		assert.Equal(t, "ls-be056d6e", CreateSuffixFromLabels(labels))
	})

	t.Run("suffix has the label selector prefix and 8 hex characters", func(t *testing.T) {
		assert.Regexp(t, regexp.MustCompile(`^ls-[0-9a-f]{8}$`), CreateSuffixFromLabels(labels))
		assert.Regexp(t, regexp.MustCompile(`^ls-[0-9a-f]{8}$`), CreateSuffixFromLabels(map[string]string{}))
	})

	t.Run("suffix does not depend on the order of the labels", func(t *testing.T) {
		reordered := make(map[string]string, len(labels))
		reordered["zone"] = "edge-a"
		reordered["app"] = "nginx"
		for i := 0; i < 10; i++ {
			assert.Equal(t, CreateSuffixFromLabels(labels), CreateSuffixFromLabels(reordered))
		}
	})

	t.Run("different labels give different suffixes", func(t *testing.T) {
		suffix := CreateSuffixFromLabels(labels)
		assert.NotEqual(t, suffix, CreateSuffixFromLabels(map[string]string{"app": "nginx", "zone": "edge-b"}))
		assert.NotEqual(t, suffix, CreateSuffixFromLabels(map[string]string{"app": "nginx", "region": "edge-a"}))
		assert.NotEqual(t, suffix, CreateSuffixFromLabels(map[string]string{"app": "nginx"}))
	})
}
