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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubeedge/api/apis/apps/v1alpha1"
)

func TestReplaceEnv(t *testing.T) {
	tests := []struct {
		name          string
		curEnv        []corev1.EnvVar
		replaceValues []corev1.EnvVar
		expected      []corev1.EnvVar
	}{
		{
			name: "Replace existing environment variable",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
			replaceValues: []corev1.EnvVar{
				{Name: "VAR1", Value: "new-value1"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new-value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Add new environment variable",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			replaceValues: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name:   "Add to empty env list",
			curEnv: []corev1.EnvVar{},
			replaceValues: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name: "Replace multiple env vars",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "VAR3", Value: "value3"},
			},
			replaceValues: []corev1.EnvVar{
				{Name: "VAR1", Value: "new-value1"},
				{Name: "VAR3", Value: "new-value3"},
				{Name: "VAR4", Value: "value4"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new-value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "VAR3", Value: "new-value3"},
				{Name: "VAR4", Value: "value4"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceEnv(tt.curEnv, tt.replaceValues)

			assert.Equal(t, len(tt.expected), len(result), "Expected same number of env vars")

			resultMap := make(map[string]string)
			for _, env := range result {
				resultMap[env.Name] = env.Value
			}

			for _, expected := range tt.expected {
				val, exists := resultMap[expected.Name]
				assert.True(t, exists, "Expected env var %s not found", expected.Name)
				assert.Equal(t, expected.Value, val, "Value mismatch for env var %s", expected.Name)
			}
		})
	}
}

func TestEnvRemove(t *testing.T) {
	tests := []struct {
		name         string
		curEnv       []corev1.EnvVar
		removeValues []corev1.EnvVar
		expected     []corev1.EnvVar
	}{
		{
			name: "Remove single env var",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
			removeValues: []corev1.EnvVar{
				{Name: "VAR1"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Remove multiple env vars",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
				{Name: "VAR3", Value: "value3"},
			},
			removeValues: []corev1.EnvVar{
				{Name: "VAR1"},
				{Name: "VAR3"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Remove non-existent env var",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			removeValues: []corev1.EnvVar{
				{Name: "VAR2"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name:   "Remove from empty env list",
			curEnv: []corev1.EnvVar{},
			removeValues: []corev1.EnvVar{
				{Name: "VAR1"},
			},
			expected: []corev1.EnvVar{},
		},
		{
			name: "Remove all env vars",
			curEnv: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
			removeValues: []corev1.EnvVar{
				{Name: "VAR1"},
				{Name: "VAR2"},
			},
			expected: []corev1.EnvVar{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := envRemove(tt.curEnv, tt.removeValues)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOverrideEnv(t *testing.T) {
	tests := []struct {
		name         string
		curEnv       []corev1.EnvVar
		envOverrider *v1alpha1.EnvOverrider
		expected     []corev1.EnvVar
		expectError  bool
	}{
		{
			name: "Add operation",
			curEnv: []corev1.EnvVar{
				{Name: "EXISTING", Value: "existing-value"},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW", Value: "new-value"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "EXISTING", Value: "existing-value"},
				{Name: "NEW", Value: "new-value"},
			},
			expectError: false,
		},
		{
			name: "Remove operation",
			curEnv: []corev1.EnvVar{
				{Name: "KEEP", Value: "keep-value"},
				{Name: "REMOVE", Value: "remove-value"},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpRemove,
				Value: []corev1.EnvVar{
					{Name: "REMOVE"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "KEEP", Value: "keep-value"},
			},
			expectError: false,
		},
		{
			name: "Replace operation",
			curEnv: []corev1.EnvVar{
				{Name: "REPLACE", Value: "old-value"},
				{Name: "KEEP", Value: "keep-value"},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpReplace,
				Value: []corev1.EnvVar{
					{Name: "REPLACE", Value: "new-value"},
					{Name: "NEW", Value: "added-value"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "REPLACE", Value: "new-value"},
				{Name: "KEEP", Value: "keep-value"},
				{Name: "NEW", Value: "added-value"},
			},
			expectError: false,
		},
		{
			name: "Unsupported operation",
			curEnv: []corev1.EnvVar{
				{Name: "TEST", Value: "test-value"},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: "UnsupportedOp",
				Value: []corev1.EnvVar{
					{Name: "NEW", Value: "new-value"},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "TEST", Value: "test-value"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := overrideEnv(tt.curEnv, tt.envOverrider)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.envOverrider.Operator == v1alpha1.OverriderOpReplace {
					assert.Equal(t, len(tt.expected), len(result))

					expectedMap := make(map[string]string)
					for _, env := range tt.expected {
						expectedMap[env.Name] = env.Value
					}

					resultMap := make(map[string]string)
					for _, env := range result {
						resultMap[env.Name] = env.Value
					}

					assert.Equal(t, expectedMap, resultMap)
				} else {
					assert.Equal(t, tt.expected, result)
				}
			}
		})
	}
}

func TestConvertToEnvVar(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    *corev1.EnvVar
		expectError bool
	}{
		{
			name: "Simple value",
			input: map[string]interface{}{
				"name":  "SIMPLE_VAR",
				"value": "simple-value",
			},
			expected: &corev1.EnvVar{
				Name:  "SIMPLE_VAR",
				Value: "simple-value",
			},
			expectError: false,
		},
		{
			name: "With field ref",
			input: map[string]interface{}{
				"name": "POD_NAME",
				"valueFrom": map[string]interface{}{
					"fieldRef": map[string]interface{}{
						"fieldPath":  "metadata.name",
						"apiVersion": "v1",
					},
				},
			},
			expected: &corev1.EnvVar{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath:  "metadata.name",
						APIVersion: "v1",
					},
				},
			},
			expectError: false,
		},
		{
			name:        "Not a map",
			input:       "not-a-map",
			expected:    nil,
			expectError: true,
		},
		{
			name: "No name",
			input: map[string]interface{}{
				"value": "value-without-name",
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToEnvVar(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConvertToEnvVarSource(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		expected    corev1.EnvVarSource
		expectError bool
	}{
		{
			name: "Field ref",
			input: map[string]interface{}{
				"fieldRef": map[string]interface{}{
					"fieldPath":  "metadata.name",
					"apiVersion": "v1",
				},
			},
			expected: corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath:  "metadata.name",
					APIVersion: "v1",
				},
			},
			expectError: false,
		},
		{
			name: "Resource field ref",
			input: map[string]interface{}{
				"resourceFieldRef": map[string]interface{}{
					"resource":      "limits.cpu",
					"containerName": "test-container",
					"divisor":       "1m",
				},
			},
			expected: corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					Resource:      "limits.cpu",
					ContainerName: "test-container",
					Divisor:       resource.MustParse("1m"),
				},
			},
			expectError: false,
		},
		{
			name: "ConfigMap key ref",
			input: map[string]interface{}{
				"configMapKeyRef": map[string]interface{}{
					"name": "my-config",
					"key":  "my-key",
				},
			},
			expected: corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "my-config",
					},
					Key: "my-key",
				},
			},
			expectError: false,
		},
		{
			name: "Secret key ref",
			input: map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name": "my-secret",
					"key":  "my-key",
				},
			},
			expected: corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Key: "my-key",
				},
			},
			expectError: false,
		},
		{
			name:        "Empty map",
			input:       map[string]interface{}{},
			expected:    corev1.EnvVarSource{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertToEnvVarSource(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expected.FieldRef != nil {
					assert.Equal(t, tt.expected.FieldRef, result.FieldRef)
				}

				if tt.expected.ResourceFieldRef != nil {
					assert.Equal(t, tt.expected.ResourceFieldRef.Resource, result.ResourceFieldRef.Resource)
					assert.Equal(t, tt.expected.ResourceFieldRef.ContainerName, result.ResourceFieldRef.ContainerName)
					assert.Equal(t, tt.expected.ResourceFieldRef.Divisor.String(), result.ResourceFieldRef.Divisor.String())
				}

				if tt.expected.ConfigMapKeyRef != nil {
					assert.Equal(t, tt.expected.ConfigMapKeyRef, result.ConfigMapKeyRef)
				}

				if tt.expected.SecretKeyRef != nil {
					assert.Equal(t, tt.expected.SecretKeyRef, result.SecretKeyRef)
				}
			}
		})
	}
}

func TestProcessFieldRef(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]interface{}
		expected    *corev1.ObjectFieldSelector
		expectError bool
	}{
		{
			name: "Valid field ref",
			input: map[string]interface{}{
				"fieldRef": map[string]interface{}{
					"fieldPath":  "metadata.name",
					"apiVersion": "v1",
				},
			},
			expected: &corev1.ObjectFieldSelector{
				FieldPath:  "metadata.name",
				APIVersion: "v1",
			},
			expectError: false,
		},
		{
			name: "Missing fieldPath",
			input: map[string]interface{}{
				"fieldRef": map[string]interface{}{
					"apiVersion": "v1",
				},
			},
			expected:    nil,
			expectError: false,
		},
		{
			name: "Missing apiVersion",
			input: map[string]interface{}{
				"fieldRef": map[string]interface{}{
					"fieldPath": "metadata.name",
				},
			},
			expected:    nil,
			expectError: false,
		},
		{
			name:        "No fieldRef",
			input:       map[string]interface{}{},
			expected:    nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processFieldRef(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBuildEnvPatchesWithPath(t *testing.T) {
	tests := []struct {
		name               string
		specContainersPath string
		rawObj             *unstructured.Unstructured
		envOverrider       *v1alpha1.EnvOverrider
		expectedPatches    []overrideOption
		expectError        bool
	}{
		{
			name:               "Container with existing env",
			specContainersPath: "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "target-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "EXISTING_VAR",
										"value": "existing-value",
									},
								},
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "target-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:   "replace",
					Path: "/spec/containers/0/env",
					Value: []corev1.EnvVar{
						{Name: "EXISTING_VAR", Value: "existing-value"},
						{Name: "NEW_VAR", Value: "new-value"},
					},
				},
			},
			expectError: false,
		},
		{
			name:               "Container without env",
			specContainersPath: "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "target-container",
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "target-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:   "add",
					Path: "/spec/containers/0/env",
					Value: []corev1.EnvVar{
						{Name: "NEW_VAR", Value: "new-value"},
					},
				},
			},
			expectError: false,
		},
		{
			name:               "Multiple containers",
			specContainersPath: "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "first-container",
							},
							map[string]interface{}{
								"name": "target-container",
								"env": []interface{}{
									map[string]interface{}{
										"name":  "EXISTING_VAR",
										"value": "existing-value",
									},
								},
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "target-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:   "replace",
					Path: "/spec/containers/1/env",
					Value: []corev1.EnvVar{
						{Name: "EXISTING_VAR", Value: "existing-value"},
						{Name: "NEW_VAR", Value: "new-value"},
					},
				},
			},
			expectError: false,
		},
		{
			name:               "Non-existent container",
			specContainersPath: "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "other-container",
							},
						},
					},
				},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				ContainerName: "target-container",
				Operator:      v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expectedPatches: []overrideOption{},
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches, err := buildEnvPatchesWithPath(tt.specContainersPath, tt.rawObj, tt.envOverrider)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if len(tt.expectedPatches) == 0 {
					assert.Empty(t, patches)
					return
				}

				assert.Equal(t, len(tt.expectedPatches), len(patches))

				for i, expectedPatch := range tt.expectedPatches {
					assert.Equal(t, expectedPatch.Op, patches[i].Op)
					assert.Equal(t, expectedPatch.Path, patches[i].Path)

					expectedEnvVars, ok := expectedPatch.Value.([]corev1.EnvVar)
					if !ok {
						t.Fatalf("Expected Value to be []corev1.EnvVar")
					}

					actualEnvVars, ok := patches[i].Value.([]corev1.EnvVar)
					if !ok {
						t.Fatalf("Patch Value is not []corev1.EnvVar")
					}

					expectedMap := make(map[string]string)
					for _, env := range expectedEnvVars {
						expectedMap[env.Name] = env.Value
					}

					actualMap := make(map[string]string)
					for _, env := range actualEnvVars {
						actualMap[env.Name] = env.Value
					}

					assert.Equal(t, expectedMap, actualMap)
				}
			}
		})
	}
}

func TestAcquireAddEnvOverrideOption(t *testing.T) {
	tests := []struct {
		name         string
		envPath      string
		envOverrider *v1alpha1.EnvOverrider
		expected     overrideOption
		expectError  bool
	}{
		{
			name:    "Valid add override",
			envPath: "/spec/containers/0/env",
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "TEST_VAR", Value: "test-value"},
				},
			},
			expected: overrideOption{
				Op:   string(v1alpha1.OverriderOpAdd),
				Path: "/spec/containers/0/env",
				Value: []corev1.EnvVar{
					{Name: "TEST_VAR", Value: "test-value"},
				},
			},
			expectError: false,
		},
		{
			name:    "Invalid path (no leading slash)",
			envPath: "spec/containers/0/env",
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "TEST_VAR", Value: "test-value"},
				},
			},
			expected:    overrideOption{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := acquireAddEnvOverrideOption(tt.envPath, tt.envOverrider)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Op, result.Op)
				assert.Equal(t, tt.expected.Path, result.Path)

				expectedEnvVars, ok := tt.expected.Value.([]corev1.EnvVar)
				if !ok {
					t.Fatalf("Expected Value to be []corev1.EnvVar")
				}

				resultEnvVars, ok := result.Value.([]corev1.EnvVar)
				if !ok {
					t.Fatalf("Result Value is not []corev1.EnvVar")
				}

				assert.Equal(t, expectedEnvVars, resultEnvVars)
			}
		})
	}
}

func TestAcquireReplaceEnvOverrideOption(t *testing.T) {
	tests := []struct {
		name         string
		envPath      string
		envValue     []corev1.EnvVar
		envOverrider *v1alpha1.EnvOverrider
		expected     overrideOption
		expectError  bool
	}{
		{
			name:    "Valid replace override",
			envPath: "/spec/containers/0/env",
			envValue: []corev1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing-value"},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expected: overrideOption{
				Op:   string(v1alpha1.OverriderOpReplace),
				Path: "/spec/containers/0/env",
				Value: []corev1.EnvVar{
					{Name: "EXISTING_VAR", Value: "existing-value"},
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expectError: false,
		},
		{
			name:    "Invalid path (no leading slash)",
			envPath: "spec/containers/0/env",
			envValue: []corev1.EnvVar{
				{Name: "EXISTING_VAR", Value: "existing-value"},
			},
			envOverrider: &v1alpha1.EnvOverrider{
				Operator: v1alpha1.OverriderOpAdd,
				Value: []corev1.EnvVar{
					{Name: "NEW_VAR", Value: "new-value"},
				},
			},
			expected:    overrideOption{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := acquireReplaceEnvOverrideOption(tt.envPath, tt.envValue, tt.envOverrider)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Op, result.Op)
				assert.Equal(t, tt.expected.Path, result.Path)

				expectedEnvVars, ok := tt.expected.Value.([]corev1.EnvVar)
				if !ok {
					t.Fatalf("Expected Value to be []corev1.EnvVar")
				}

				resultEnvVars, ok := result.Value.([]corev1.EnvVar)
				if !ok {
					t.Fatalf("Result Value is not []corev1.EnvVar")
				}

				expectedMap := make(map[string]string)
				for _, env := range expectedEnvVars {
					expectedMap[env.Name] = env.Value
				}

				resultMap := make(map[string]string)
				for _, env := range resultEnvVars {
					resultMap[env.Name] = env.Value
				}

				assert.Equal(t, expectedMap, resultMap)
			}
		})
	}
}
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
