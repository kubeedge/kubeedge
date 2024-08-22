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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeedge/api/apis/apps/v1alpha1"
)

func TestEnvOverrider_ApplyOverrides(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		rawObj         *unstructured.Unstructured
		overriderInfo  OverriderInfo
		expectedResult *unstructured.Unstructured
		expectError    bool
	}{
		{
			name: "Apply env overrides to Pod",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "test-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "NAME_ONE",
										"value": "value-one",
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &v1alpha1.Overriders{
					EnvOverriders: []v1alpha1.EnvOverrider{
						{
							ContainerName: "test-container",
							Operator:      v1alpha1.OverriderOpAdd,
							Value: []corev1.EnvVar{
								{Name: "NAME_TWO", Value: "value-two"},
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "test-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "NAME_ONE",
										"value": "value-one",
									},
									map[string]interface{}{
										"name":  "NAME_TWO",
										"value": "value-two",
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Apply env override to Deployment",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "test-container",
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &v1alpha1.Overriders{
					EnvOverriders: []v1alpha1.EnvOverrider{
						{
							ContainerName: "test-container",
							Operator:      v1alpha1.OverriderOpAdd,
							Value: []corev1.EnvVar{
								{Name: "NEW_NAME", Value: "new-value"},
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "test-container",
										"env": []interface{}{
											map[string]interface{}{
												"name":  "NEW_NAME",
												"value": "new-value",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Apply remove env override",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "test-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "TEST_NAME",
										"value": "test-value",
									},
									map[string]interface{}{
										"name":  "REMOVE_THIS_NAME",
										"value": "remove-this-value",
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &v1alpha1.Overriders{
					EnvOverriders: []v1alpha1.EnvOverrider{
						{
							ContainerName: "test-container",
							Operator:      v1alpha1.OverriderOpRemove,
							Value: []corev1.EnvVar{
								{Name: "REMOVE_THIS_NAME"},
							},
						},
					},
				},
			},
			expectedResult: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "test-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "TEST_NAME",
										"value": "test-value",
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			overrider := &EnvOverrider{}
			err := overrider.ApplyOverrides(tc.rawObj, tc.overriderInfo)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedResult, tc.rawObj)
			}
		})
	}
}

func TestBuildEnvPatches(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		rawObj         *unstructured.Unstructured
		envOverrider   *v1alpha1.EnvOverrider
		expectedResult []overrideOption
		expectError    bool
	}{
		{
			name: "Build patches for Pod",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "test-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "EXISTING_NAME",
										"value": "existing-value",
									},
								},
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "test-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_NAME", Value: "new-value"},
				},
			},
			expectedResult: []overrideOption{
				{
					Op:   "replace",
					Path: "/spec/containers/0/env",
					Value: []corev1.EnvVar{
						{Name: "EXISTING_NAME", Value: "existing-value"},
						{Name: "NEW_NAME", Value: "new-value"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Build patches for Deployment",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "test-container",
										"env": []interface{}{
											map[string]interface{}{
												"name":  "EXISTING_NAME",
												"value": "existing-value",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "test-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_NAME", Value: "new-value"},
				},
			},
			expectedResult: []overrideOption{
				{
					Op:   "replace",
					Path: "/spec/template/spec/containers/0/env",
					Value: []corev1.EnvVar{
						{Name: "EXISTING_NAME", Value: "existing-value"},
						{Name: "NEW_NAME", Value: "new-value"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Build patches for StatefulSet",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "StatefulSet",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "test-container",
										"env": []interface{}{
											map[string]interface{}{
												"name":  "EXISTING_NAME",
												"value": "existing-value",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "test-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_NAME", Value: "new-value"},
				},
			},
			expectedResult: []overrideOption{
				{
					Op:   "replace",
					Path: "/spec/template/spec/containers/0/env",
					Value: []corev1.EnvVar{
						{Name: "EXISTING_NAME", Value: "existing-value"},
						{Name: "NEW_NAME", Value: "new-value"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Build patches for unsupported resource",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "test-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expectedResult: nil,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildEnvPatches(tc.rawObj, tc.envOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedResult, result)
			}
		})
	}
}
