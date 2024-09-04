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

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

func TestOverrideManager_ApplyOverrides(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name             string
		overriders       []Overrider
		rawObj           *unstructured.Unstructured
		overrideInfo     OverriderInfo
		expectedError    bool
		expectedReplicas int64
	}{
		{
			name:       "No overriders",
			overriders: []Overrider{},
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"replicas": int64(1),
					},
				},
			},
			overrideInfo: OverriderInfo{
				TargetNodeGroup: "node-group-1",
				Overriders:      &appsv1alpha1.Overriders{},
			},
			expectedError:    false,
			expectedReplicas: 1,
		},
		{
			name: "Apply ReplicasOverrider",
			overriders: []Overrider{
				&ReplicasOverrider{},
			},
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"replicas": int64(1),
					},
				},
			},
			overrideInfo: OverriderInfo{
				TargetNodeGroup: "node-group-1",
				Overriders: &appsv1alpha1.Overriders{
					Replicas: intPtr(3),
				},
			},
			expectedError:    false,
			expectedReplicas: 3,
		},
		{
			name: "Unsupported kind",
			overriders: []Overrider{
				&ReplicasOverrider{},
			},
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
				},
			},
			overrideInfo: OverriderInfo{
				TargetNodeGroup: "node-group-1",
				Overriders: &appsv1alpha1.Overriders{
					Replicas: intPtr(3),
				},
			},
			expectedError:    true,
			expectedReplicas: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrideManager := &OverrideManager{
				Overriders: tt.overriders,
			}

			err := overrideManager.ApplyOverrides(tt.rawObj, tt.overrideInfo)

			if tt.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				if tt.rawObj.GetKind() == "Deployment" {
					replicas, found, err := unstructured.NestedInt64(tt.rawObj.Object, "spec", "replicas")
					assert.NoError(err)
					assert.True(found)
					assert.Equal(tt.expectedReplicas, replicas)
				}
			}
		})
	}
}

func TestApplyJSONPatch(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name          string
		obj           *unstructured.Unstructured
		overrides     []overrideOption
		expectedObj   map[string]interface{}
		expectedError bool
	}{
		{
			name: "Add a new field",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
					"spec": map[string]interface{}{},
				},
			},
			overrides: []overrideOption{
				{
					Op:    "add",
					Path:  "/spec/replicas",
					Value: int64(3),
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Deployment",
				"metadata": map[string]interface{}{
					"name": "test-deployment",
				},
				"spec": map[string]interface{}{
					"replicas": int64(3),
				},
			},
			expectedError: false,
		},
		{
			name: "Replace an existing field",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
					"spec": map[string]interface{}{
						"replicas": int64(1),
					},
				},
			},
			overrides: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/replicas",
					Value: int64(5),
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Deployment",
				"metadata": map[string]interface{}{
					"name": "test-deployment",
				},
				"spec": map[string]interface{}{
					"replicas": int64(5),
				},
			},
			expectedError: false,
		},
		{
			name: "Remove a field",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"metadata": map[string]interface{}{
						"name": "test-deployment",
						"labels": map[string]interface{}{
							"app": "myapp",
						},
					},
				},
			},
			overrides: []overrideOption{
				{
					Op:   "remove",
					Path: "/metadata/labels/app",
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Deployment",
				"metadata": map[string]interface{}{
					"name":   "test-deployment",
					"labels": map[string]interface{}{},
				},
			},
			expectedError: false,
		},
		{
			name: "Invalid patch",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
				},
			},
			overrides: []overrideOption{
				{
					Op:    "invalid",
					Path:  "/spec/replicas",
					Value: int64(3),
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Deployment",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyJSONPatch(tt.obj, tt.overrides)

			if tt.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.expectedObj, tt.obj.Object)
			}
		})
	}
}
