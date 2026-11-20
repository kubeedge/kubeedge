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

func TestResourcesOverrider_ApplyOverrides(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name          string
		rawObj        *unstructured.Unstructured
		overriders    OverriderInfo
		expectedObj   map[string]interface{}
		expectedError bool
	}{
		{
			name: "Override Pod resources",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
								"resources": map[string]interface{}{
									"limits": map[string]interface{}{
										"cpu":    "100m",
										"memory": "128Mi",
									},
									"requests": map[string]interface{}{
										"cpu":    "100m",
										"memory": "128Mi",
									},
								},
							},
						},
					},
				},
			},
			overriders: OverriderInfo{
				Overriders: &v1alpha1.Overriders{
					ResourcesOverriders: []v1alpha1.ResourcesOverrider{
						{
							ContainerName: "container1",
							Value: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Pod",
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": "container1",
							"resources": map[string]interface{}{
								"limits": map[string]interface{}{
									"cpu":    "200m",
									"memory": "256Mi",
								},
								"requests": map[string]interface{}{
									"cpu":    "100m",
									"memory": "128Mi",
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Override Deployment resources",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
										"resources": map[string]interface{}{
											"limits": map[string]interface{}{
												"cpu":    "800m",
												"memory": "1024Mi",
											},
											"requests": map[string]interface{}{
												"cpu":    "800m",
												"memory": "512Mi",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			overriders: OverriderInfo{
				Overriders: &v1alpha1.Overriders{
					ResourcesOverriders: []v1alpha1.ResourcesOverrider{
						{
							ContainerName: "container1",
							Value: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("200m"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Deployment",
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "container1",
									"resources": map[string]interface{}{
										"limits": map[string]interface{}{
											"cpu":    "200m",
											"memory": "1024Mi",
										},
										"requests": map[string]interface{}{
											"cpu":    "100m",
											"memory": "512Mi",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "No matching container",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
							},
						},
					},
				},
			},
			overriders: OverriderInfo{
				Overriders: &v1alpha1.Overriders{
					ResourcesOverriders: []v1alpha1.ResourcesOverrider{
						{
							ContainerName: "container2",
							Value: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
			},
			expectedObj: map[string]interface{}{
				"kind": "Pod",
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": "container1",
						},
					},
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrider := &ResourcesOverrider{}
			err := overrider.ApplyOverrides(tt.rawObj, tt.overriders)

			if tt.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tt.expectedObj, tt.rawObj.Object)
			}
		})
	}
}

func TestBuildResourcesPatches(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name               string
		rawObj             *unstructured.Unstructured
		resourcesOverrider *v1alpha1.ResourcesOverrider
		expectedPatches    []overrideOption
		expectedError      bool
	}{
		{
			name: "Build patches for Pod",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
							},
						},
					},
				},
			},
			resourcesOverrider: &v1alpha1.ResourcesOverrider{
				ContainerName: "container1",
				Value: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/containers/0/resources",
					Value: map[string]interface{}{},
				},
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/containers/0/resources/limits",
					Value: map[string]interface{}{},
				},
				{
					Op:    "replace",
					Path:  "/spec/containers/0/resources/limits/cpu",
					Value: "200m",
				},
				{
					Op:    "replace",
					Path:  "/spec/containers/0/resources/limits/memory",
					Value: "256Mi",
				},
			},
			expectedError: false,
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
										"name": "container1",
									},
								},
							},
						},
					},
				},
			},
			resourcesOverrider: &v1alpha1.ResourcesOverrider{
				ContainerName: "container1",
				Value: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/template/spec/containers/0/resources",
					Value: map[string]interface{}{},
				},
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/template/spec/containers/0/resources/limits",
					Value: map[string]interface{}{},
				},
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/resources/limits/cpu",
					Value: "200m",
				},
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/resources/limits/memory",
					Value: "256Mi",
				},
			},
			expectedError: false,
		},
		{
			name: "No matching container",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
							},
						},
					},
				},
			},
			resourcesOverrider: &v1alpha1.ResourcesOverrider{
				ContainerName: "container2",
				Value: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			expectedPatches: []overrideOption{},
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches, err := buildResourcesPatches(tt.rawObj, tt.resourcesOverrider)

			if tt.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.ElementsMatch(tt.expectedPatches, patches)
			}
		})
	}
}

func TestBuildResourcesPatchesWithPath(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name               string
		specContainersPath string
		rawObj             *unstructured.Unstructured
		resourcesOverrider *v1alpha1.ResourcesOverrider
		expectedPatches    []overrideOption
		expectedError      bool
	}{
		{
			name:               "Build patches for Pod",
			specContainersPath: "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
							},
							map[string]interface{}{
								"name": "container2",
							},
						},
					},
				},
			},
			resourcesOverrider: &v1alpha1.ResourcesOverrider{
				ContainerName: "container2",
				Value: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/containers/1/resources",
					Value: map[string]interface{}{},
				},
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/containers/1/resources/limits",
					Value: map[string]interface{}{},
				},
				{
					Op:    "replace",
					Path:  "/spec/containers/1/resources/limits/cpu",
					Value: "200m",
				},
				{
					Op:    "replace",
					Path:  "/spec/containers/1/resources/limits/memory",
					Value: "256Mi",
				},
			},
			expectedError: false,
		},
		{
			name:               "Build patches for Deployment",
			specContainersPath: "spec/template/spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name": "container1",
									},
								},
							},
						},
					},
				},
			},
			resourcesOverrider: &v1alpha1.ResourcesOverrider{
				ContainerName: "container1",
				Value: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/template/spec/containers/0/resources",
					Value: map[string]interface{}{},
				},
				{
					Op:    string(v1alpha1.OverriderOpAdd),
					Path:  "/spec/template/spec/containers/0/resources/limits",
					Value: map[string]interface{}{},
				},
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/resources/limits/cpu",
					Value: "300m",
				},
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/resources/limits/memory",
					Value: "512Mi",
				},
			},
			expectedError: false,
		},
		{
			name:               "No matching container",
			specContainersPath: "spec/containers",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "container1",
							},
						},
					},
				},
			},
			resourcesOverrider: &v1alpha1.ResourcesOverrider{
				ContainerName: "container2",
				Value: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			expectedPatches: []overrideOption{},
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches, err := buildResourcesPatchesWithPath(tt.specContainersPath, tt.rawObj, tt.resourcesOverrider)

			if tt.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.ElementsMatch(tt.expectedPatches, patches)
			}
		})
	}
}

func TestAcquireOverride(t *testing.T) {
	resourcesPath := "/spec/containers/0/resources"
	resourcesValue := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	expectedPatches := []overrideOption{
		{Op: "replace", Path: "/spec/containers/0/resources/requests/cpu", Value: "500m"},
		{Op: "replace", Path: "/spec/containers/0/resources/requests/memory", Value: "128Mi"},
		{Op: "replace", Path: "/spec/containers/0/resources/limits/cpu", Value: "500m"},
		{Op: "replace", Path: "/spec/containers/0/resources/limits/memory", Value: "128Mi"},
	}

	patches, err := acquireOverride(resourcesPath, resourcesValue)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedPatches, patches)
}

func TestResourcesOverrider_ApplyOverridesQualifiedResources(t *testing.T) {
	tests := []struct {
		name             string
		initialRequests  map[string]interface{}
		initialLimits    map[string]interface{}
		overrideRequests corev1.ResourceList
		overrideLimits   corev1.ResourceList
		wantRequests     map[string]interface{}
		wantLimits       map[string]interface{}
	}{
		{
			name:             "GPU request",
			initialRequests:  map[string]interface{}{"nvidia.com/gpu": "1"},
			initialLimits:    map[string]interface{}{"nvidia.com/gpu": "1"},
			overrideRequests: corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")},
			wantRequests:     map[string]interface{}{"nvidia.com/gpu": "2"},
			wantLimits:       map[string]interface{}{"nvidia.com/gpu": "1"},
		},
		{
			name:            "GPU limit",
			initialRequests: map[string]interface{}{"nvidia.com/gpu": "1"},
			initialLimits:   map[string]interface{}{"nvidia.com/gpu": "1"},
			overrideLimits:  corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")},
			wantRequests:    map[string]interface{}{"nvidia.com/gpu": "1"},
			wantLimits:      map[string]interface{}{"nvidia.com/gpu": "2"},
		},
		{
			name:             "GPU request and limit",
			initialRequests:  map[string]interface{}{"nvidia.com/gpu": "1"},
			initialLimits:    map[string]interface{}{"nvidia.com/gpu": "1"},
			overrideRequests: corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")},
			overrideLimits:   corev1.ResourceList{"nvidia.com/gpu": resource.MustParse("2")},
			wantRequests:     map[string]interface{}{"nvidia.com/gpu": "2"},
			wantLimits:       map[string]interface{}{"nvidia.com/gpu": "2"},
		},
		{
			name:             "ordinary CPU and memory",
			initialRequests:  map[string]interface{}{"cpu": "100m", "memory": "128Mi"},
			initialLimits:    map[string]interface{}{"cpu": "100m", "memory": "128Mi"},
			overrideRequests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("200m")},
			overrideLimits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")},
			wantRequests:     map[string]interface{}{"cpu": "200m", "memory": "128Mi"},
			wantLimits:       map[string]interface{}{"cpu": "100m", "memory": "256Mi"},
		},
		{
			name:             "non-NVIDIA extended resource",
			initialRequests:  map[string]interface{}{"example.com/device": "1"},
			initialLimits:    map[string]interface{}{"example.com/device": "1"},
			overrideRequests: corev1.ResourceList{"example.com/device": resource.MustParse("2")},
			overrideLimits:   corev1.ResourceList{"example.com/device": resource.MustParse("2")},
			wantRequests:     map[string]interface{}{"example.com/device": "2"},
			wantLimits:       map[string]interface{}{"example.com/device": "2"},
		},
	}

	deploymentWithResources := func(requests, limits map[string]interface{}) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{
			"kind": "Deployment",
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "inference",
								"resources": map[string]interface{}{
									"requests": requests,
									"limits":   limits,
								},
							},
						},
					},
				},
			},
		}}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := deploymentWithResources(tt.initialRequests, tt.initialLimits)
			overriders := OverriderInfo{Overriders: &v1alpha1.Overriders{
				ResourcesOverriders: []v1alpha1.ResourcesOverrider{{
					ContainerName: "inference",
					Value: corev1.ResourceRequirements{
						Requests: tt.overrideRequests,
						Limits:   tt.overrideLimits,
					},
				}},
			}}
			err := (&ResourcesOverrider{}).ApplyOverrides(obj, overriders)
			assert.NoError(t, err)
			assert.Equal(t, deploymentWithResources(tt.wantRequests, tt.wantLimits).Object, obj.Object)
		})
	}
}

func TestAcquireOverrideEscapesResourceNames(t *testing.T) {
	tests := []struct {
		name       string
		resource   corev1.ResourceName
		wantSuffix string
	}{
		{name: "ordinary CPU", resource: corev1.ResourceCPU, wantSuffix: "cpu"},
		{name: "ordinary memory", resource: corev1.ResourceMemory, wantSuffix: "memory"},
		{name: "ordinary ephemeral storage", resource: corev1.ResourceEphemeralStorage, wantSuffix: "ephemeral-storage"},
		{name: "GPU", resource: "nvidia.com/gpu", wantSuffix: "nvidia.com~1gpu"},
		{name: "other extended resource", resource: "example.com/device", wantSuffix: "example.com~1device"},
		{name: "RFC 6901 tilde and slash", resource: "foo~bar/baz", wantSuffix: "foo~0bar~1baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quantity := resource.MustParse("2")
			patches, err := acquireOverride("/spec/template/spec/containers/0/resources", corev1.ResourceRequirements{
				Requests: corev1.ResourceList{tt.resource: quantity},
				Limits:   corev1.ResourceList{tt.resource: quantity},
			})
			assert.NoError(t, err)
			assert.ElementsMatch(t, []overrideOption{
				{Op: "replace", Path: "/spec/template/spec/containers/0/resources/requests/" + tt.wantSuffix, Value: "2"},
				{Op: "replace", Path: "/spec/template/spec/containers/0/resources/limits/" + tt.wantSuffix, Value: "2"},
			}, patches)
		})
	}
}

func TestResourcesOverrider_ApplyOverridesMissingFields(t *testing.T) {
	cpu := func(q string) corev1.ResourceList {
		return corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(q)}
	}
	newDeployment := func(extra map[string]interface{}) *unstructured.Unstructured {
		container := map[string]interface{}{"name": "inference"}
		for k, v := range extra {
			container[k] = v
		}
		return &unstructured.Unstructured{Object: map[string]interface{}{
			"kind": "Deployment",
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{container},
					},
				},
			},
		}}
	}
	both := map[string]interface{}{
		"requests": map[string]interface{}{"cpu": "100m"},
		"limits":   map[string]interface{}{"cpu": "200m"},
	}

	tests := []struct {
		name      string
		container map[string]interface{}
		override  corev1.ResourceRequirements
		want      map[string]interface{}
	}{
		{
			name:      "container without resources",
			container: nil,
			override:  corev1.ResourceRequirements{Requests: cpu("100m"), Limits: cpu("200m")},
			want:      map[string]interface{}{"resources": both},
		},
		{
			name:      "empty resources",
			container: map[string]interface{}{"resources": map[string]interface{}{}},
			override:  corev1.ResourceRequirements{Requests: cpu("100m"), Limits: cpu("200m")},
			want:      map[string]interface{}{"resources": both},
		},
		{
			name:      "null resources",
			container: map[string]interface{}{"resources": nil},
			override:  corev1.ResourceRequirements{Requests: cpu("100m"), Limits: cpu("200m")},
			want:      map[string]interface{}{"resources": both},
		},
		{
			name: "only limits set",
			container: map[string]interface{}{"resources": map[string]interface{}{
				"limits": map[string]interface{}{"memory": "128Mi"},
			}},
			override: corev1.ResourceRequirements{Requests: cpu("100m"), Limits: cpu("200m")},
			want: map[string]interface{}{"resources": map[string]interface{}{
				"requests": map[string]interface{}{"cpu": "100m"},
				"limits":   map[string]interface{}{"cpu": "200m", "memory": "128Mi"},
			}},
		},
		{
			name: "only requests set",
			container: map[string]interface{}{"resources": map[string]interface{}{
				"requests": map[string]interface{}{"memory": "128Mi"},
			}},
			override: corev1.ResourceRequirements{Requests: cpu("100m"), Limits: cpu("200m")},
			want: map[string]interface{}{"resources": map[string]interface{}{
				"requests": map[string]interface{}{"cpu": "100m", "memory": "128Mi"},
				"limits":   map[string]interface{}{"cpu": "200m"},
			}},
		},
		{
			name: "resource name that the container does not set yet",
			container: map[string]interface{}{"resources": map[string]interface{}{
				"requests": map[string]interface{}{"memory": "128Mi"},
				"limits":   map[string]interface{}{"memory": "128Mi"},
			}},
			override: corev1.ResourceRequirements{Requests: cpu("100m"), Limits: cpu("200m")},
			want: map[string]interface{}{"resources": map[string]interface{}{
				"requests": map[string]interface{}{"cpu": "100m", "memory": "128Mi"},
				"limits":   map[string]interface{}{"cpu": "200m", "memory": "128Mi"},
			}},
		},
		{
			name:      "only requests overridden",
			container: nil,
			override:  corev1.ResourceRequirements{Requests: cpu("100m")},
			want: map[string]interface{}{"resources": map[string]interface{}{
				"requests": map[string]interface{}{"cpu": "100m"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := newDeployment(tt.container)
			overriders := OverriderInfo{Overriders: &v1alpha1.Overriders{
				ResourcesOverriders: []v1alpha1.ResourcesOverrider{{
					ContainerName: "inference",
					Value:         tt.override,
				}},
			}}
			err := (&ResourcesOverrider{}).ApplyOverrides(obj, overriders)
			assert.NoError(t, err)
			assert.Equal(t, newDeployment(tt.want).Object, obj.Object)
		})
	}
}
