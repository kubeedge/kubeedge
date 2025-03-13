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

package overridemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNameOverrider_ApplyOverrides(t *testing.T) {
	tests := []struct {
		name          string
		rawObj        *unstructured.Unstructured
		overriderInfo OverriderInfo
		expectedName  string
		expectedError bool
	}{
		{
			name: "Modify name with TargetNodeGroup",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup:         "edge-group",
				TargetNodeLabelSelector: metav1.LabelSelector{},
				Overriders:              nil,
			},
			expectedName:  "test-deployment-edge-group",
			expectedError: false,
		},
		{
			name: "Modify name with TargetNodeLabelSelector",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "",
				TargetNodeLabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"zone": "edge",
						"env":  "prod",
					},
				},
				Overriders: nil,
			},
			// We can't predict the exact suffix due to hash, so we'll check if it has the prefix
			expectedName:  "test-deployment-ls-",
			expectedError: false,
		},
		{
			name: "No modification with empty TargetNodeGroup and empty MatchLabels",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "",
				TargetNodeLabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{},
				},
				Overriders: nil,
			},
			expectedName:  "test-deployment",
			expectedError: false,
		},
		{
			name: "Label selector overrides node group",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "edge-group",
				TargetNodeLabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"zone": "edge",
					},
				},
				Overriders: nil,
			},
			expectedName:  "test-deployment-ls-", // Will check prefix
			expectedError: false,
		},
		{
			name: "Empty object name",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "",
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup:         "edge-group",
				TargetNodeLabelSelector: metav1.LabelSelector{},
				Overriders:              nil,
			},
			expectedName:  "-edge-group", // Empty name + suffix
			expectedError: false,
		},
		{
			name: "Name with special characters",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment-123!@#",
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup:         "edge-group",
				TargetNodeLabelSelector: metav1.LabelSelector{},
				Overriders:              nil,
			},
			expectedName:  "test-deployment-123!@#-edge-group",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nameOverrider := &NameOverrider{}

			err := nameOverrider.ApplyOverrides(tt.rawObj, tt.overriderInfo)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				actualName := tt.rawObj.GetName()

				if tt.overriderInfo.TargetNodeLabelSelector.MatchLabels != nil &&
					len(tt.overriderInfo.TargetNodeLabelSelector.MatchLabels) > 0 {
					assert.True(t, len(actualName) > len(tt.expectedName),
						"Expected name with suffix longer than %s, got %s", tt.expectedName, actualName)
					assert.True(t, actualName[:len(tt.expectedName)] == tt.expectedName,
						"Expected name to start with %s, got %s", tt.expectedName, actualName)
				} else {
					assert.Equal(t, tt.expectedName, actualName)
				}
			}
		})
	}
}

func TestCreateSuffixFromLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
	}{
		{
			name:   "Empty labels",
			labels: map[string]string{},
		},
		{
			name: "Single label",
			labels: map[string]string{
				"app": "nginx",
			},
		},
		{
			name: "Multiple labels",
			labels: map[string]string{
				"app":     "nginx",
				"version": "1.0",
				"tier":    "frontend",
			},
		},
		{
			name: "Labels with special characters",
			labels: map[string]string{
				"app":     "nginx-123!@#",
				"version": "1.0.0-beta.1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suffix := CreateSuffixFromLabels(tt.labels)

			assert.True(t, len(suffix) >= 3, "Expected suffix to be at least 3 characters long")
			assert.Equal(t, "ls-", suffix[:3])

			suffix2 := CreateSuffixFromLabels(tt.labels)
			assert.Equal(t, suffix, suffix2, "Expected deterministic suffixes for the same input")

			if len(tt.labels) > 1 {
				reorderedLabels := make(map[string]string)
				keys := make([]string, 0, len(tt.labels))
				for k := range tt.labels {
					keys = append(keys, k)
				}
				for i := len(keys) - 1; i >= 0; i-- {
					reorderedLabels[keys[i]] = tt.labels[keys[i]]
				}

				suffix3 := CreateSuffixFromLabels(reorderedLabels)
				assert.Equal(t, suffix, suffix3, "Expected same suffix regardless of map iteration order")
			}
		})
	}
}

func TestCreateSuffixFromLabels_Uniqueness(t *testing.T) {
	labelSets := []map[string]string{
		{"app": "nginx"},
		{"app": "apache"},
		{"service": "nginx"},
		{"app": "nginx", "version": "1.0"},
		{"app": "nginx", "version": "2.0"},
		{"app": "nginx", "tier": "frontend"},
		{"version": "1.0", "app": "nginx"},
	}

	suffixes := make(map[string]struct{})
	for _, labels := range labelSets {
		suffix := CreateSuffixFromLabels(labels)

		if (len(labels) == 2 && labels["app"] == "nginx" && labels["version"] == "1.0") ||
			(len(labels) == 2 && labels["version"] == "1.0" && labels["app"] == "nginx") {
			continue
		}

		_, exists := suffixes[suffix]
		assert.False(t, exists, "Expected unique suffix for different label sets, got duplicate: %s", suffix)
		suffixes[suffix] = struct{}{}
	}

	// Verify that equivalent label sets produce the same suffix
	suffix1 := CreateSuffixFromLabels(map[string]string{"app": "nginx", "version": "1.0"})
	suffix2 := CreateSuffixFromLabels(map[string]string{"version": "1.0", "app": "nginx"})
	assert.Equal(t, suffix1, suffix2, "Expected same suffix for equivalent label sets")
}
