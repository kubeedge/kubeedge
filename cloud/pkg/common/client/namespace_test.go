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
package client

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func filterActions(actions []k8stesting.Action, verb string) []k8stesting.Action {
	var filtered []k8stesting.Action
	for _, action := range actions {
		if action.GetVerb() == verb {
			filtered = append(filtered, action)
		}
	}
	return filtered
}

func TestCreateNamespaceIfNeeded(t *testing.T) {
	testCases := []struct {
		name          string
		namespace     string
		setupFunc     func(client *fake.Clientset)
		expectedError bool
		validateFunc  func(t *testing.T, client *fake.Clientset)
	}{
		{
			name:      "namespace already exists",
			namespace: "existing-ns",
			setupFunc: func(client *fake.Clientset) {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-ns",
					},
				}
				if _, err := client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{}); err != nil {
					t.Fatalf("Failed to create test namespace: %v", err)
				}
			},
			expectedError: false,
			validateFunc: func(t *testing.T, client *fake.Clientset) {
				actions := client.Actions()
				getActions := filterActions(actions, "get")
				createActions := filterActions(actions, "create")

				if len(getActions) != 1 {
					t.Errorf("Expected 1 get action, got %d", len(getActions))
				}
				if len(createActions) != 0 {
					t.Errorf("Expected 0 create actions, got %d", len(createActions))
				}
			},
		},
		{
			name:          "namespace needs to be created",
			namespace:     "new-ns",
			setupFunc:     nil,
			expectedError: false,
			validateFunc: func(t *testing.T, client *fake.Clientset) {
				actions := client.Actions()
				getActions := filterActions(actions, "get")
				createActions := filterActions(actions, "create")

				if len(getActions) != 1 {
					t.Errorf("Expected 1 get action, got %d", len(getActions))
				}
				if len(createActions) != 1 {
					t.Errorf("Expected 1 create action, got %d", len(createActions))
				}
			},
		},
		{
			name:      "namespace creation conflict",
			namespace: "conflict-ns",
			setupFunc: func(client *fake.Clientset) {
				client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.NewAlreadyExists(
						schema.GroupResource{Resource: "namespaces"},
						"conflict-ns",
					)
				})
			},
			expectedError: false,
			validateFunc: func(t *testing.T, client *fake.Clientset) {
				actions := client.Actions()
				getActions := filterActions(actions, "get")
				createActions := filterActions(actions, "create")

				if len(getActions) != 1 {
					t.Errorf("Expected 1 get action, got %d", len(getActions))
				}
				if len(createActions) != 1 {
					t.Errorf("Expected 1 create action, got %d", len(createActions))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset()

			client.ClearActions()

			if tc.setupFunc != nil {
				tc.setupFunc(client)
			}

			client.ClearActions()

			kubeClient = client

			err := CreateNamespaceIfNeeded(context.TODO(), tc.namespace)

			if tc.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(t, client)
			}
		})
	}
}
