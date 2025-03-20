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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
)

func TestNodeSelectorOverrider_ApplyOverrides(t *testing.T) {
	testCases := []struct {
		name          string
		inputObj      *unstructured.Unstructured
		overriderInfo OverriderInfo
		expectedError bool
		validateFunc  func(t *testing.T, obj *unstructured.Unstructured)
	}{
		{
			name: "Apply Override with Node Group",
			inputObj: func() *unstructured.Unstructured {
				deployment := createTestDeployment()
				unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
				obj := &unstructured.Unstructured{Object: unstructuredObj}
				obj.SetKind("Deployment")
				obj.SetAPIVersion("apps/v1")
				return obj
			}(),
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
			},
			expectedError: false,
			validateFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				deployment, err := ConvertToDeployment(obj)
				if err != nil {
					t.Fatalf("Failed to convert unstructured to deployment: %v", err)
				}

				nodeSelector := deployment.Spec.Template.Spec.NodeSelector
				if nodeSelector == nil {
					t.Fatal("NodeSelector should not be nil")
				}

				expectedLabel := map[string]string{
					nodegroup.LabelBelongingTo: "test-node-group",
				}

				if !mapEqual(nodeSelector, expectedLabel) {
					t.Errorf("Unexpected node selector. Got %v, want %v", nodeSelector, expectedLabel)
				}
			},
		},
		{
			name: "Apply Override with Node Label Selector",
			inputObj: func() *unstructured.Unstructured {
				deployment := createTestDeployment()
				unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(deployment)
				obj := &unstructured.Unstructured{Object: unstructuredObj}
				obj.SetKind("Deployment")
				obj.SetAPIVersion("apps/v1")
				return obj
			}(),
			overriderInfo: OverriderInfo{
				TargetNodeLabelSelector: v1.LabelSelector{
					MatchLabels: map[string]string{
						"environment": "production",
						"tier":        "backend",
					},
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				deployment, err := ConvertToDeployment(obj)
				if err != nil {
					t.Fatalf("Failed to convert unstructured to deployment: %v", err)
				}

				if deployment.Spec.Template.Spec.Affinity == nil {
					t.Fatal("Affinity should not be nil")
				}

				nodeAffinity := deployment.Spec.Template.Spec.Affinity.NodeAffinity
				if nodeAffinity == nil {
					t.Fatal("NodeAffinity should not be nil")
				}

				terms := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
				if len(terms) != 1 {
					t.Fatalf("Expected 1 node selector term, got %d", len(terms))
				}

				matchExpressions := terms[0].MatchExpressions
				if len(matchExpressions) != 2 {
					t.Fatalf("Expected 2 match expressions, got %d", len(matchExpressions))
				}

				expectedExpressions := []corev1.NodeSelectorRequirement{
					{
						Key:      "environment",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"production"},
					},
					{
						Key:      "tier",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"backend"},
					},
				}

				// Create a map to check expressions more flexibly
				expressionMap := make(map[string]corev1.NodeSelectorRequirement)
				for _, expr := range matchExpressions {
					expressionMap[expr.Key] = expr
				}

				for _, expected := range expectedExpressions {
					actual, exists := expressionMap[expected.Key]
					if !exists {
						t.Errorf("Expected expression for key %s not found", expected.Key)
						continue
					}
					if actual.Operator != expected.Operator ||
						actual.Values[0] != expected.Values[0] {
						t.Errorf("Unexpected match expression for key %s. Got %v, want %v",
							expected.Key, actual, expected)
					}
				}
			},
		},
		{
			name: "Error for Unsupported Object Kind",
			inputObj: func() *unstructured.Unstructured {
				pod := &corev1.Pod{}
				unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
				obj := &unstructured.Unstructured{Object: unstructuredObj}
				obj.SetKind("Pod")
				obj.SetAPIVersion("v1")
				return obj
			}(),
			overriderInfo: OverriderInfo{
				TargetNodeGroup: "test-node-group",
			},
			expectedError: true,
			validateFunc:  nil,
		},
	}

	overrider := &NodeSelectorOverrider{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := overrider.ApplyOverrides(tc.inputObj, tc.overriderInfo)

			if (err != nil) != tc.expectedError {
				t.Errorf("Expected error: %v, got: %v", tc.expectedError, err)
			}

			if !tc.expectedError && tc.validateFunc != nil {
				tc.validateFunc(t, tc.inputObj)
			}
		})
	}
}

// Helper function to create a test deployment
func createTestDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
}

// Helper function to compare maps
func mapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}

	return true
}
