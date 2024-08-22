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

package overridemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	apppsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

func TestReplicasOverrider_ApplyOverrides(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name             string
		rawObj           *unstructured.Unstructured
		overriders       OverriderInfo
		expectedReplicas int64
		expectError      bool
	}{
		{
			name: "Deployment",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       DeploymentKind,
					"apiVersion": "apps/v1",
					"metadata": map[string]interface{}{
						"name":      "test-deployment",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"replicas": int64(1),
					},
				},
			},
			overriders: OverriderInfo{
				Overriders: &apppsv1alpha1.Overriders{
					Replicas: intPtr(3),
				},
			},
			expectedReplicas: 3,
			expectError:      false,
		},
		{
			name: "Deployment without replicas field",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       DeploymentKind,
					"apiVersion": "apps/v1",
					"metadata": map[string]interface{}{
						"name":      "test-deployment",
						"namespace": "default",
					},
					"spec": map[string]interface{}{},
				},
			},
			overriders: OverriderInfo{
				Overriders: &apppsv1alpha1.Overriders{
					Replicas: intPtr(3),
				},
			},
			expectedReplicas: 3,
			expectError:      false,
		},
		{
			name: "Deployment with nil replicas override",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       DeploymentKind,
					"apiVersion": "apps/v1",
					"metadata": map[string]interface{}{
						"name":      "test-deployment",
						"namespace": "default",
					},
					"spec": map[string]interface{}{
						"replicas": int64(1),
					},
				},
			},
			overriders: OverriderInfo{
				Overriders: &apppsv1alpha1.Overriders{
					Replicas: nil,
				},
			},
			expectedReplicas: 1,
			expectError:      false,
		},
		{
			name: "Unsupported kind",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Pod",
					"apiVersion": "v1",
					"metadata": map[string]interface{}{
						"name":      "test-pod",
						"namespace": "default",
					},
				},
			},
			overriders: OverriderInfo{
				Overriders: &apppsv1alpha1.Overriders{
					Replicas: intPtr(3),
				},
			},
			expectedReplicas: 0,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrider := &ReplicasOverrider{}
			err := overrider.ApplyOverrides(tt.rawObj, tt.overriders)

			if tt.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				if tt.rawObj.GetKind() == DeploymentKind {
					replicas, found, err := unstructured.NestedInt64(tt.rawObj.Object, "spec", "replicas")
					assert.NoError(err)
					assert.True(found)
					assert.Equal(tt.expectedReplicas, replicas)
				}
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
