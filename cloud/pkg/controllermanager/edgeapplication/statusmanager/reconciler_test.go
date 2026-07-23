/*
Copyright 2026 The KubeEdge Authors.

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

package statusmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
)

func TestDeploymentAvailable_IsAvailable(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, appsv1.AddToScheme(scheme))

	tests := []struct {
		name       string
		deploy     *appsv1.Deployment
		wantResult bool
		wantErr    bool
	}{
		{
			name: "nil Spec.Replicas with ReadyReplicas=1 should be available",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					// Replicas is nil, Kubernetes defaults to 1
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			wantResult: true,
			wantErr:    false,
		},
		{
			name: "nil Spec.Replicas with ReadyReplicas=0 should be unavailable",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					// Replicas is nil, Kubernetes defaults to 1
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 0,
				},
			},
			wantResult: false,
			wantErr:    false,
		},
		{
			name: "explicit Spec.Replicas=3 with ReadyReplicas=3 should be available",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To[int32](3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 3,
				},
			},
			wantResult: true,
			wantErr:    false,
		},
		{
			name: "explicit Spec.Replicas=3 with ReadyReplicas=1 should be unavailable",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deploy",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To[int32](3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			wantResult: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.deploy).
				Build()

			info := utils.ResourceInfo{
				Group:     "apps",
				Version:   "v1",
				Kind:      "Deployment",
				Namespace: tt.deploy.Namespace,
				Name:      tt.deploy.Name,
			}

			d := deploymentAvailable{}
			got, err := d.IsAvailable(context.Background(), fakeClient, info)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantResult, got)
		})
	}
}

func TestDeploymentAvailable_ObjectNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, appsv1.AddToScheme(scheme))

	// Build a client with no objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	info := utils.ResourceInfo{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "nonexistent-deploy",
	}

	d := deploymentAvailable{}
	got, err := d.IsAvailable(context.Background(), fakeClient, info)
	assert.NoError(t, err, "getObjAccordingToResourceInfo returns nil object, expected unavailable without error")
	assert.False(t, got)
}

func TestAvailableIfExists_ObjectNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, appsv1.AddToScheme(scheme))

	// Build a client with no objects
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	info := utils.ResourceInfo{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "nonexistent-deploy",
	}

	e := availableIfExists{}
	got, err := e.IsAvailable(context.Background(), fakeClient, info)
	assert.NoError(t, err)
	assert.False(t, got)
}

func TestAvailableIfExists_ObjectExists(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, appsv1.AddToScheme(scheme))

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deploy).
		Build()

	info := utils.ResourceInfo{
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "test-deploy",
	}

	e := availableIfExists{}
	got, err := e.IsAvailable(context.Background(), fakeClient, info)
	assert.NoError(t, err)
	assert.True(t, got)
}
