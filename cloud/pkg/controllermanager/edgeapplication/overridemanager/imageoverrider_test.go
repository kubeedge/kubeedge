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

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
)

func TestImageOverrider_ApplyOverrides(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		rawObj         *unstructured.Unstructured
		overriderInfo  OverriderInfo
		expectedResult *unstructured.Unstructured
		expectError    bool
	}{
		{
			name: "Apply image overrides on Pod",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ImageOverriders: []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Tag,
							Operator:  appsv1alpha1.OverriderOpReplace,
							Value:     "1.15.0",
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
								"name":  "test-container",
								"image": "nginx:1.15.0",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "Apply image overrides on Deployment",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "test-container",
										"image": "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ImageOverriders: []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Registry,
							Operator:  appsv1alpha1.OverriderOpReplace,
							Value:     "test-registry.com",
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
										"name":  "test-container",
										"image": "test-registry.com/nginx:1.14.2",
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
			name: "Apply image override with predicate",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test-container",
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			overriderInfo: OverriderInfo{
				Overriders: &appsv1alpha1.Overriders{
					ImageOverriders: []appsv1alpha1.ImageOverrider{
						{
							Component: appsv1alpha1.Repository,
							Operator:  appsv1alpha1.OverriderOpReplace,
							Value:     "test-nginx",
							Predicate: &appsv1alpha1.ImagePredicate{
								Path: "/spec/containers/0/image",
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
								"name":  "test-container",
								"image": "test-nginx:1.14.2",
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
			overrider := &ImageOverrider{}
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

func TestBuildPatches(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		rawObj          *unstructured.Unstructured
		imageOverrider  *appsv1alpha1.ImageOverrider
		expectedPatches []overrideOption
		expectError     bool
	}{
		{
			name: "Build patches for Pod with empty predicate",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "container1",
								"image": "nginx:1.14.2",
							},
							map[string]interface{}{
								"name":  "container2",
								"image": "redis:6.0.9",
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "nginx:latest",
				},
				{
					Op:    "replace",
					Path:  "/spec/containers/1/image",
					Value: "redis:latest",
				},
			},
			expectError: false,
		},
		{
			name: "Build patches for Deployment with empty predicate",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Deployment",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "container1",
										"image": "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Registry,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-registry.com",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "test-registry.com/nginx:1.14.2",
				},
			},
			expectError: false,
		},
		{
			name: "Build patches with predicate",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "container1",
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Repository,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-nginx",
				Predicate: &appsv1alpha1.ImagePredicate{
					Path: "/spec/containers/0/image",
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "test-nginx:1.14.2",
				},
			},
			expectError: false,
		},
		{
			name: "Error case: unsupported kind",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "UnsupportedKind",
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedPatches: nil,
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches, err := buildPatches(tc.rawObj, tc.imageOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedPatches, patches)
			}
		})
	}
}

func TestBuildPatchesWithEmptyPredicate(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		rawObj          *unstructured.Unstructured
		imageOverrider  *appsv1alpha1.ImageOverrider
		expectedPatches []overrideOption
		expectError     bool
	}{
		{
			name: "Build patches for Pod",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "Pod",
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "container1",
								"image": "nginx:1.14.2",
							},
							map[string]interface{}{
								"name":  "container2",
								"image": "redis:6.0.9",
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "nginx:latest",
				},
				{
					Op:    "replace",
					Path:  "/spec/containers/1/image",
					Value: "redis:latest",
				},
			},
			expectError: false,
		},
		{
			name: "Build patches for ReplicaSet",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "ReplicaSet",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "container1",
										"image": "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Registry,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-registry.com",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "test-registry.com/nginx:1.14.2",
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
										"name":  "container1",
										"image": "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Repository,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-nginx",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "test-nginx:1.14.2",
				},
			},
			expectError: false,
		},
		{
			name: "Build patches for DaemonSet",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "DaemonSet",
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "container1",
										"image": "nginx:1.14.2",
									},
								},
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "1.15.0",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "nginx:1.15.0",
				},
			},
			expectError: false,
		},
		{
			name: "Unsupported kind",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind": "UnsupportedKind",
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedPatches: nil,
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches, err := buildPatchesWithEmptyPredicate(tc.rawObj, tc.imageOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedPatches, patches)
			}
		})
	}
}

func TestExtractPatchesBy(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		podSpec         corev1.PodSpec
		prefixPath      string
		imageOverrider  *appsv1alpha1.ImageOverrider
		expectedPatches []overrideOption
		expectError     bool
	}{
		{
			name: "Extract patches for single container",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "nginx:1.14.2",
					},
				},
			},
			prefixPath: "/spec",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "nginx:latest",
				},
			},
			expectError: false,
		},
		{
			name: "Extract patches for multiple containers",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "nginx:1.14.2",
					},
					{
						Name:  "container2",
						Image: "redis:6.0.9",
					},
				},
			},
			prefixPath: "/spec/template",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Registry,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-registry.com",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/containers/0/image",
					Value: "test-registry.com/nginx:1.14.2",
				},
				{
					Op:    "replace",
					Path:  "/spec/template/containers/1/image",
					Value: "test-registry.com/redis:6.0.9",
				},
			},
			expectError: false,
		},
		{
			name: "Extract patches with repository override",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "nginx:1.14.2",
					},
				},
			},
			prefixPath: "/spec",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Repository,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-nginx",
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "test-nginx:1.14.2",
				},
			},
			expectError: false,
		},
		{
			name: "Extract patches with remove operation",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container1",
						Image: "nginx:1.14.2",
					},
				},
			},
			prefixPath: "/spec",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpRemove,
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "nginx",
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches, err := extractPatchesBy(tc.podSpec, tc.prefixPath, tc.imageOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedPatches, patches)
			}
		})
	}
}

func TestSpliceImagePath(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		prefixPath     string
		containerIndex int
		expectedPath   string
	}{
		{
			name:           "first container",
			prefixPath:     "/spec",
			containerIndex: 0,
			expectedPath:   "/spec/containers/0/image",
		},
		{
			name:           "template prefix",
			prefixPath:     "/spec/template",
			containerIndex: 0,
			expectedPath:   "/spec/template/containers/0/image",
		},
		{
			name:           "empty prefix",
			prefixPath:     "",
			containerIndex: 2,
			expectedPath:   "/containers/2/image",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := spliceImagePath(tc.prefixPath, tc.containerIndex)
			assert.Equal(tc.expectedPath, path)
		})
	}
}

func TestBuildPatchesWithPredicate(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name            string
		rawObj          *unstructured.Unstructured
		imageOverrider  *appsv1alpha1.ImageOverrider
		expectedPatches []overrideOption
		expectError     bool
	}{
		{
			name: "Build patches with predicate",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "container1",
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
				Predicate: &appsv1alpha1.ImagePredicate{
					Path: "/spec/containers/0/image",
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/containers/0/image",
					Value: "nginx:latest",
				},
			},
			expectError: false,
		},
		{
			name: "Build patches with nested predicate",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "container1",
										"image": "redis:6.0.9",
									},
								},
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Registry,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-registry.com",
				Predicate: &appsv1alpha1.ImagePredicate{
					Path: "/spec/template/spec/containers/0/image",
				},
			},
			expectedPatches: []overrideOption{
				{
					Op:    "replace",
					Path:  "/spec/template/spec/containers/0/image",
					Value: "test-registry.com/redis:6.0.9",
				},
			},
			expectError: false,
		},
		{
			name: "invalid predicate path",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "container1",
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
				Predicate: &appsv1alpha1.ImagePredicate{
					Path: "/spec/non-existent/path",
				},
			},
			expectedPatches: nil,
			expectError:     true,
		},
		{
			name: "predicate path doesn't point to string",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "container1",
								"image": 12345, // Invalid type for image
							},
						},
					},
				},
			},
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
				Predicate: &appsv1alpha1.ImagePredicate{
					Path: "/spec/containers/0/image",
				},
			},
			expectedPatches: nil,
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patches, err := buildPatchesWithPredicate(tc.rawObj, tc.imageOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedPatches, patches)
			}
		})
	}
}

func TestObtainImageValue(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name          string
		rawObj        *unstructured.Unstructured
		predicatePath string
		expectedImage string
		expectError   bool
	}{
		{
			name: "Obtain image value",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			predicatePath: "/spec/containers/0/image",
			expectedImage: "nginx:1.14.2",
			expectError:   false,
		},
		{
			name: "Obtain image value from nested path",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "redis:6.0.9",
									},
								},
							},
						},
					},
				},
			},
			predicatePath: "/spec/template/spec/containers/0/image",
			expectedImage: "redis:6.0.9",
			expectError:   false,
		},
		{
			name: "invalid path",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			predicatePath: "/spec/non-existent/path",
			expectedImage: "",
			expectError:   true,
		},
		{
			name: "path doesn't point to string",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": 12345, // Invalid type for image
							},
						},
					},
				},
			},
			predicatePath: "/spec/containers/0/image",
			expectedImage: "",
			expectError:   true,
		},
		{
			name: "invalid array index",
			rawObj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.14.2",
							},
						},
					},
				},
			},
			predicatePath: "/spec/containers/invalid/image",
			expectedImage: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			imageValue, err := obtainImageValue(tc.rawObj, tc.predicatePath)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedImage, imageValue)
			}
		})
	}
}

func TestAcquireOverrideOption(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		imagePath      string
		curImage       string
		imageOverrider *appsv1alpha1.ImageOverrider
		expectedOption overrideOption
		expectError    bool
	}{
		{
			name:      "Replace tag",
			imagePath: "/spec/containers/0/image",
			curImage:  "nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedOption: overrideOption{
				Op:    "replace",
				Path:  "/spec/containers/0/image",
				Value: "nginx:latest",
			},
			expectError: false,
		},
		{
			name:      "Replace registry",
			imagePath: "/spec/containers/0/image",
			curImage:  "docker.io/library/nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Registry,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-registry.com",
			},
			expectedOption: overrideOption{
				Op:    "replace",
				Path:  "/spec/containers/0/image",
				Value: "test-registry.com/library/nginx:1.14.2",
			},
			expectError: false,
		},
		{
			name:      "invalid image path",
			imagePath: "invalid/path",
			curImage:  "nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedOption: overrideOption{},
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			option, err := acquireOverrideOption(tc.imagePath, tc.curImage, tc.imageOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedOption, option)
			}
		})
	}
}

func TestOverrideImage(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name           string
		curImage       string
		imageOverrider *appsv1alpha1.ImageOverrider
		expectedImage  string
		expectError    bool
	}{
		{
			name:     "Replace tag",
			curImage: "nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedImage: "nginx:latest",
			expectError:   false,
		},
		{
			name:     "Replace registry",
			curImage: "docker.io/library/nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Registry,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "test-registry.com",
			},
			expectedImage: "test-registry.com/library/nginx:1.14.2",
			expectError:   false,
		},
		{
			name:     "Add to repository",
			curImage: "nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Repository,
				Operator:  appsv1alpha1.OverriderOpAdd,
				Value:     "-custom",
			},
			expectedImage: "nginx-custom:1.14.2",
			expectError:   false,
		},
		{
			name:     "Remove tag",
			curImage: "nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpRemove,
			},
			expectedImage: "nginx",
			expectError:   false,
		},
		{
			name:     "invalid image",
			curImage: "invalid:image:format",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: appsv1alpha1.Tag,
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedImage: "",
			expectError:   true,
		},
		{
			name:     "unsupported component",
			curImage: "nginx:1.14.2",
			imageOverrider: &appsv1alpha1.ImageOverrider{
				Component: "unsupported",
				Operator:  appsv1alpha1.OverriderOpReplace,
				Value:     "latest",
			},
			expectedImage: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			newImage, err := overrideImage(tc.curImage, tc.imageOverrider)

			if tc.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.Equal(tc.expectedImage, newImage)
			}
		})
	}
}
