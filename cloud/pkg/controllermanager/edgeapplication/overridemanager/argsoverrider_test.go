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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

const (
	testPodKind         = "Pod"
	testDeploymentKind  = "Deployment"
	testReplicaSetKind  = "ReplicaSet"
	testDaemonSetKind   = "DaemonSet"
	testJobKind         = "Job"
	testStatefulSetKind = "StatefulSet"
)

func TestArgsOverrider_ApplyOverrides(t *testing.T) {
	tests := []struct {
		name            string
		rawObj          *unstructured.Unstructured
		overriderInfo   OverriderInfo
		expectedError   bool
		validateResults func(*testing.T, *unstructured.Unstructured)
	}{
		{
			name: "Apply single args overrider successfully",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
								"args": []interface{}{"arg1", "arg2"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3", "arg4"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)
				assert.NotEmpty(t, containers)

				container := containers[0].(map[string]interface{})
				assert.Equal(t, "container1", container["name"])

				args := container["args"].([]interface{})
				assert.Len(t, args, 4)
				assert.Equal(t, "arg1", args[0])
				assert.Equal(t, "arg2", args[1])
				assert.Equal(t, "arg3", args[2])
				assert.Equal(t, "arg4", args[3])
			},
		},
		{
			name: "Apply multiple args overriders successfully",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
								"args": []interface{}{"arg1", "arg2"},
							},
							map[string]interface{}{
								"name": "container2",
								"args": []interface{}{"arg1", "arg2"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
						{
							ContainerName: "container2",
							Operator:      appsv1alpha1.OverriderOpRemove,
							Value:         []string{"arg2"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)
				assert.Len(t, containers, 2)

				container1 := containers[0].(map[string]interface{})
				assert.Equal(t, "container1", container1["name"])
				args1 := container1["args"].([]interface{})
				assert.Len(t, args1, 3)
				assert.Equal(t, "arg1", args1[0])
				assert.Equal(t, "arg2", args1[1])
				assert.Equal(t, "arg3", args1[2])

				container2 := containers[1].(map[string]interface{})
				assert.Equal(t, "container2", container2["name"])
				args2 := container2["args"].([]interface{})
				assert.Len(t, args2, 1)
				assert.Equal(t, "arg1", args2[0])
			},
		},
		{
			name: "Empty overriders list",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
			},
		},
		{
			name: "No containers path",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg1"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
			},
		},
		{
			name: "Add args to container that has no existing args",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg1", "arg2"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				assert.Equal(t, "container1", container["name"])

				assert.Contains(t, container, "args")
				args := container["args"].([]interface{})
				assert.Len(t, args, 2)
				assert.Equal(t, "arg1", args[0].(string))
				assert.Equal(t, "arg2", args[1].(string))
			},
		},
		{
			name: "Apply args overrider to Deployment",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       testDeploymentKind,
					"metadata": map[string]interface{}{
						"name": "test-deployment",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
										"args": []interface{}{"arg1", "arg2"},
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(
					obj.Object, "spec", "template", "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				assert.Equal(t, "container1", container["name"])

				args := container["args"].([]interface{})
				assert.Len(t, args, 3)
				assert.Equal(t, "arg1", args[0].(string))
				assert.Equal(t, "arg2", args[1].(string))
				assert.Equal(t, "arg3", args[2].(string))
			},
		},
		{
			name: "Apply args overrider to ReplicaSet",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       testReplicaSetKind,
					"metadata": map[string]interface{}{
						"name": "test-replicaset",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
										"args": []interface{}{"arg1", "arg2"},
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(
					obj.Object, "spec", "template", "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				assert.Equal(t, "container1", container["name"])

				args := container["args"].([]interface{})
				assert.Len(t, args, 3)
				assert.Equal(t, "arg3", args[2].(string))
			},
		},
		{
			name: "Apply args overrider to DaemonSet",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       testDaemonSetKind,
					"metadata": map[string]interface{}{
						"name": "test-daemonset",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
										"args": []interface{}{"arg1", "arg2"},
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(
					obj.Object, "spec", "template", "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				args := container["args"].([]interface{})
				assert.Len(t, args, 3)
			},
		},
		{
			name: "Apply args overrider to Job",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "batch/v1",
					"kind":       testJobKind,
					"metadata": map[string]interface{}{
						"name": "test-job",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
										"args": []interface{}{"arg1", "arg2"},
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(
					obj.Object, "spec", "template", "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				args := container["args"].([]interface{})
				assert.Len(t, args, 3)
			},
		},
		{
			name: "Apply args overrider to StatefulSet",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       testStatefulSetKind,
					"metadata": map[string]interface{}{
						"name": "test-statefulset",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
										"args": []interface{}{"arg1", "arg2"},
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(
					obj.Object, "spec", "template", "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				args := container["args"].([]interface{})
				assert.Len(t, args, 3)
			},
		},
		{
			name: "Container not found in resource",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "other-container",
								"args": []interface{}{"arg1", "arg2"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg3"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				assert.Equal(t, "other-container", container["name"])

				args := container["args"].([]interface{})
				assert.Len(t, args, 2)
				assert.Equal(t, "arg1", args[0])
				assert.Equal(t, "arg2", args[1])
			},
		},
		{
			name: "Unsupported resource kind",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "custom.io/v1",
					"kind":       "CustomResource",
					"metadata": map[string]interface{}{
						"name": "test-custom",
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpAdd,
							Value:         []string{"arg1"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				assert.Equal(t, "CustomResource", obj.GetKind())
			},
		},
		{
			name: "Remove args from container",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       testPodKind,
					"metadata": map[string]interface{}{
						"name": "test-pod",
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
								"args": []interface{}{"keep1", "remove1", "keep2"},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ArgsOverriders: []appsv1alpha1.CommandArgsOverrider{
						{
							ContainerName: "container1",
							Operator:      appsv1alpha1.OverriderOpRemove,
							Value:         []string{"remove1"},
						},
					},
				},
			},
			expectedError: false,
			validateResults: func(t *testing.T, obj *unstructured.Unstructured) {
				containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "containers")
				assert.True(t, found)
				assert.NoError(t, err)

				container := containers[0].(map[string]interface{})
				assert.Equal(t, "container1", container["name"])

				args := container["args"].([]interface{})
				assert.Len(t, args, 2)
				assert.Equal(t, "keep1", args[0])
				assert.Equal(t, "keep2", args[1])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsOverrider := &ArgsOverrider{}

			err := argsOverrider.ApplyOverrides(tt.rawObj, tt.overriderInfo)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validateResults(t, tt.rawObj)
			}
		})
	}
}

func TestArgsOverrider_NilOverriders(t *testing.T) {
	rawObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
		},
	}

	overriderInfo := OverriderInfo{
		Overriders: nil,
	}

	argsOverrider := &ArgsOverrider{}

	err := argsOverrider.ApplyOverrides(rawObj, overriderInfo)

	assert.Error(t, err, "Expected error when Overriders is nil")
	assert.Contains(t, err.Error(), "overriders.Overriders is nil", "Error message should mention nil Overriders")
}
