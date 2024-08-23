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

func TestCommandOverrider_ApplyOverrides(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		rawObj         *unstructured.Unstructured
		overriderInfo  OverriderInfo
		expectedResult *unstructured.Unstructured
		expectError    bool
	}{
		{
			name: "Apply command overrides to Pod",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &appsv1alpha1.Overriders{
					CommandOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "test-container",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"three"},
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
								"name":    "test-container",
								"command": []interface{}{"one", "two", "three"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Apply command overrides to Deployment",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":    "test-container",
										"command": []interface{}{"one", "two"},
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &appsv1alpha1.Overriders{
					CommandOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "test-container",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"three"},
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
										"name":    "test-container",
										"command": []interface{}{"one", "two", "three"},
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
			name: "Apply remove command override",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "test-container",
								"command": []interface{}{"one", "two", "three"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &appsv1alpha1.Overriders{
					CommandOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "test-container",
							Operator:      appsv1alpha1.OverriderOpRemove,
							Value:         []string{"three"},
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
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Apply command override to non-existent container",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &appsv1alpha1.Overriders{
					CommandOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "non-existent-container",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"three"},
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
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Apply command override to container without existing command",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "test-container",
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
				Overriders: &appsv1alpha1.Overriders{
					CommandOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "test-container",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"one", "two"},
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
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
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
			overrider := &CommandOverrider{}
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

func TestBuildCommandArgsPatches(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		target         string
		rawObj         *unstructured.Unstructured
		overrider      *appsv1alpha1.CommandArgsOverrider
		expectedResult []overrideOption
		expectError    bool
	}{
		{
			name:   "Pod",
			target: CommandString,
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/command",
					Value: []string{"one", "two", "three"},
				},
			},
			expectError: false,
		},
		{
			name:   "Deployment",
			target: CommandString,
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":    "test-container",
										"command": []interface{}{"one", "two"},
									},
								},
							},
						},
					},
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/command",
					Value: []string{"one", "two", "three"},
				},
			},
			expectError: false,
		},
		{
			name:   "StatefulSet",
			target: CommandString,
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "StatefulSet",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":    "test-container",
										"command": []interface{}{"one", "two"},
									},
								},
							},
						},
					},
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/command",
					Value: []string{"one", "two", "three"},
				},
			},
			expectError: false,
		},
		{
			name:   "ConfigMap (unsupported)",
			target: CommandString,
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ConfigMap",
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: nil,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildCommandArgsPatches(tc.target, tc.rawObj, tc.overrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedResult, result)
			}
		})
	}
}

func TestBuildCommandArgsPatchesWithPath(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		target         string
		path           string
		rawObj         *unstructured.Unstructured
		overrider      *appsv1alpha1.CommandArgsOverrider
		expectedResult []overrideOption
		expectError    bool
	}{
		{
			name:   "Deployment",
			target: CommandString,
			path:   "spec/template/spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":    "test-container",
										"command": []interface{}{"one", "two"},
									},
								},
							},
						},
					},
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/command",
					Value: []string{"one", "two", "three"},
				},
			},
			expectError: false,
		},
		{
			name:   "Pod",
			target: CommandString,
			path:   "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/command",
					Value: []string{"one", "two", "three"},
				},
			},
			expectError: false,
		},
		{
			name:   "Invalid path",
			target: CommandString,
			path:   "invalid/path",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":    "test-container",
								"command": []interface{}{"one", "two"},
							},
						},
					},
				},
			},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: nil,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := buildCommandArgsPatchesWithPath(tc.target, tc.path, tc.rawObj, tc.overrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}

			assert.Equal(tc.expectedResult, result)
		})
	}
}

func TestAcquireAddOverrideOption(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		commandArgsPath string
		overrider       *appsv1alpha1.CommandArgsOverrider
		expectedResult  overrideOption
		expectError     bool
	}{
		{
			name:            "Valid value and path",
			commandArgsPath: "/spec/containers/0/command",
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"one", "two"},
			},
			expectedResult: overrideOption{
				Op:    "add",
				Path:  "/spec/containers/0/command",
				Value: []string{"one", "two"},
			},
			expectError: false,
		},
		{
			name:            "Empty value",
			commandArgsPath: "/spec/containers/0/command",
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{},
			},
			expectedResult: overrideOption{
				Op:    "add",
				Path:  "/spec/containers/0/command",
				Value: []string{},
			},
			expectError: false,
		},
		{
			name:            "Invalid path",
			commandArgsPath: "invalid-path",
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"one", "two"},
			},
			expectedResult: overrideOption{},
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := acquireAddOverrideOption(tc.commandArgsPath, tc.overrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedResult, result)
			}
		})
	}
}

func TestAcquireReplaceOverrideOption(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name             string
		commandArgsPath  string
		commandArgsValue []string
		overrider        *appsv1alpha1.CommandArgsOverrider
		expectedResult   overrideOption
		expectError      bool
	}{
		{
			name:             "Valid path",
			commandArgsPath:  "/spec/containers/0/command",
			commandArgsValue: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: overrideOption{
				Op:    "replace",
				Path:  "/spec/containers/0/command",
				Value: []string{"one", "two", "three"},
			},
			expectError: false,
		},
		{
			name:             "Remove operation with valid path",
			commandArgsPath:  "/spec/containers/0/command",
			commandArgsValue: []string{"one", "two", "three"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpRemove,
				Value:         []string{"three"},
			},
			expectedResult: overrideOption{
				Op:    "replace",
				Path:  "/spec/containers/0/command",
				Value: []string{"one", "two"},
			},
			expectError: false,
		},
		{
			name:             "Invalid path",
			commandArgsPath:  "invalid-path",
			commandArgsValue: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				ContainerName: "test-container",
				Operator:      appsv1alpha1.OverriderOpAdd,
				Value:         []string{"three"},
			},
			expectedResult: overrideOption{},
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := acquireReplaceOverrideOption(tc.commandArgsPath, tc.commandArgsValue, tc.overrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedResult, result)
			}
		})
	}
}

func TestOverrideCommandArgs(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		curCommandArgs []string
		overrider      *appsv1alpha1.CommandArgsOverrider
		expectedResult []string
	}{
		{
			name:           "Add command args",
			curCommandArgs: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpAdd,
				Value:    []string{"three"},
			},
			expectedResult: []string{"one", "two", "three"},
		},
		{
			name:           "Remove command args",
			curCommandArgs: []string{"one", "two", "three"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpRemove,
				Value:    []string{"three"},
			},
			expectedResult: []string{"one", "two"},
		},
		{
			name:           "Add multiple command args",
			curCommandArgs: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpAdd,
				Value:    []string{"three", "four"},
			},
			expectedResult: []string{"one", "two", "three", "four"},
		},
		{
			name:           "Remove multiple command args",
			curCommandArgs: []string{"one", "two", "three", "four"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpRemove,
				Value:    []string{"two", "three"},
			},
			expectedResult: []string{"one", "four"},
		},
		{
			name:           "Add command args to empty slice",
			curCommandArgs: []string{},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpAdd,
				Value:    []string{"one", "two"},
			},
			expectedResult: []string{"one", "two"},
		},
		{
			name:           "Remove all command args",
			curCommandArgs: []string{"one", "two", "three"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpRemove,
				Value:    []string{"one", "two", "three"},
			},
			expectedResult: []string{},
		},
		{
			name:           "Remove non-existent command args",
			curCommandArgs: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpRemove,
				Value:    []string{"three"},
			},
			expectedResult: []string{"one", "two"},
		},
		{
			name:           "Add duplicate command args",
			curCommandArgs: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: appsv1alpha1.OverriderOpAdd,
				Value:    []string{"two", "three"},
			},
			expectedResult: []string{"one", "two", "two", "three"},
		},
		{
			name:           "Unsupported operator",
			curCommandArgs: []string{"one", "two"},
			overrider: &appsv1alpha1.CommandArgsOverrider{
				Operator: "UnsupportedOp",
				Value:    []string{"three"},
			},
			expectedResult: []string{"one", "two"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := overrideCommandArgs(tc.curCommandArgs, tc.overrider)
			assert.Equal(tc.expectedResult, result, "Test case: %s", tc.name)
		})
	}
}

func TestCommandArgsRemove(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		curCommandArgs []string
		removeValues   []string
		expectedResult []string
	}{
		{
			name:           "Remove single value",
			curCommandArgs: []string{"one", "two", "three"},
			removeValues:   []string{"three"},
			expectedResult: []string{"one", "two"},
		},
		{
			name:           "Remove multiple values",
			curCommandArgs: []string{"one", "two", "three", "four"},
			removeValues:   []string{"three", "four"},
			expectedResult: []string{"one", "two"},
		},
		{
			name:           "Remove non-existent value",
			curCommandArgs: []string{"one", "two", "three"},
			removeValues:   []string{"four"},
			expectedResult: []string{"one", "two", "three"},
		},
		{
			name:           "Remove all values",
			curCommandArgs: []string{"one", "two", "three"},
			removeValues:   []string{"one", "two", "three"},
			expectedResult: []string{},
		},
		{
			name:           "Remove from empty slice",
			curCommandArgs: []string{},
			removeValues:   []string{"one", "two"},
			expectedResult: []string{},
		},
		{
			name:           "Remove with empty removeValues",
			curCommandArgs: []string{"one", "two", "three"},
			removeValues:   []string{},
			expectedResult: []string{"one", "two", "three"},
		},
		{
			name:           "Remove duplicate values",
			curCommandArgs: []string{"one", "two", "two", "three"},
			removeValues:   []string{"two"},
			expectedResult: []string{"one", "three"},
		},
		{
			name:           "Remove subset of values",
			curCommandArgs: []string{"one", "two", "three", "four"},
			removeValues:   []string{"two", "three", "four"},
			expectedResult: []string{"one"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := commandArgsRemove(tc.curCommandArgs, tc.removeValues)
			assert.Equal(tc.expectedResult, result, "Test case: %s", tc.name)
		})
	}
}
