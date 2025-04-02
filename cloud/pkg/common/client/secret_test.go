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
	"errors"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func setupFakeClient(objects ...runtime.Object) kubernetes.Interface {
	return fake.NewSimpleClientset(objects...)
}

func mockGetKubeClient(client kubernetes.Interface) func() {
	originalClient := kubeClient
	kubeClient = client
	return func() {
		kubeClient = originalClient
	}
}

func TestGetSecret(t *testing.T) {
	testCases := []struct {
		name        string
		secretName  string
		namespace   string
		existingObj []runtime.Object
		expectErr   bool
		expectNil   bool
	}{
		{
			name:       "Secret exists",
			secretName: "test-secret",
			namespace:  "test-ns",
			existingObj: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "test-ns",
					},
					Data: map[string][]byte{
						"key": []byte("value"),
					},
				},
			},
			expectErr: false,
			expectNil: false,
		},
		{
			name:        "Secret does not exist",
			secretName:  "non-existent",
			namespace:   "test-ns",
			existingObj: []runtime.Object{},
			expectErr:   true,
			expectNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupFakeClient(tc.existingObj...)
			cleanup := mockGetKubeClient(client)
			defer cleanup()

			result, err := GetSecret(context.Background(), tc.secretName, tc.namespace)

			if tc.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tc.expectNil && result != nil {
				t.Errorf("Expected nil result but got: %v", result)
			}
			if !tc.expectNil && result == nil {
				t.Error("Expected non-nil result but got nil")
			}
		})
	}
}

func TestSaveSecret(t *testing.T) {
	testCases := []struct {
		name         string
		secret       *corev1.Secret
		namespace    string
		existingObjs []runtime.Object
		reactors     []k8stesting.Reactor
		expectErr    bool
	}{
		{
			name: "Create new secret successfully",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-secret",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			namespace:    "test-ns",
			existingObjs: []runtime.Object{},
			expectErr:    false,
		},
		{
			name: "Update existing secret successfully",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-secret",
				},
				Data: map[string][]byte{
					"key": []byte("updated-value"),
				},
			},
			namespace: "test-ns",
			existingObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-secret",
						Namespace: "test-ns",
					},
					Data: map[string][]byte{
						"key": []byte("original-value"),
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Fail to create namespace",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "error-secret",
				},
			},
			namespace:    "error-ns",
			existingObjs: []runtime.Object{},
			reactors: []k8stesting.Reactor{
				&k8stesting.SimpleReactor{
					Verb:     "create",
					Resource: "namespaces",
					Reaction: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("namespace creation error")
					},
				},
			},
			expectErr: true,
		},
		{
			name: "Fail to create secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "error-secret",
				},
			},
			namespace: "test-ns",
			existingObjs: []runtime.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			reactors: []k8stesting.Reactor{
				&k8stesting.SimpleReactor{
					Verb:     "create",
					Resource: "secrets",
					Reaction: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("secret creation error")
					},
				},
			},
			expectErr: true,
		},
		{
			name: "Fail to update secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "update-error-secret",
				},
			},
			namespace: "test-ns",
			existingObjs: []runtime.Object{
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "update-error-secret",
						Namespace: "test-ns",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-ns",
					},
				},
			},
			reactors: []k8stesting.Reactor{
				&k8stesting.SimpleReactor{
					Verb:     "create",
					Resource: "secrets",
					Reaction: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, k8serrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, "update-error-secret")
					},
				},
				&k8stesting.SimpleReactor{
					Verb:     "update",
					Resource: "secrets",
					Reaction: func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
						return true, nil, errors.New("secret update error")
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := setupFakeClient(tc.existingObjs...)

			for _, reactor := range tc.reactors {
				client.(*fake.Clientset).Fake.PrependReactor(
					reactor.(*k8stesting.SimpleReactor).Verb,
					reactor.(*k8stesting.SimpleReactor).Resource,
					reactor.(*k8stesting.SimpleReactor).Reaction,
				)
			}

			cleanup := mockGetKubeClient(client)
			defer cleanup()

			err := SaveSecret(context.Background(), tc.secret, tc.namespace)

			if tc.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tc.expectErr {
				result, err := client.CoreV1().Secrets(tc.namespace).Get(context.Background(), tc.secret.Name, metav1.GetOptions{})
				if err != nil {
					t.Errorf("Failed to get secret after save: %v", err)
				}
				if !reflect.DeepEqual(result.Data, tc.secret.Data) {
					t.Errorf("Secret data mismatch. Expected %v, got %v", tc.secret.Data, result.Data)
				}
			}
		})
	}
}
