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

package edgeapplication

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
)

func TestUpdateStatusMergesDuplicatedEntries(t *testing.T) {
	ctx := context.TODO()
	identifier := appsv1alpha1.ResourceIdentifier{
		Ordinal:   0,
		Group:     "apps",
		Version:   "v1",
		Kind:      "Deployment",
		Namespace: "default",
		Name:      "nginx-hangzhou",
	}
	// The status of an EdgeApplication that targeted a node group more than once
	// before the CRD rejected it.
	edgeApp := &appsv1alpha1.EdgeApplication{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "nginx"},
		Spec: appsv1alpha1.EdgeApplicationSpec{
			WorkloadTemplate: appsv1alpha1.ResourceTemplate{
				Manifests: []appsv1alpha1.Manifest{
					{RawExtension: runtime.RawExtension{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"namespace":"default","name":"nginx"}}`)}},
				},
			},
		},
		Status: appsv1alpha1.EdgeApplicationStatus{
			WorkloadStatus: []appsv1alpha1.ManifestStatus{
				{Identifier: identifier, Condition: appsv1alpha1.EdgeAppAvailable},
				{Identifier: identifier, Condition: appsv1alpha1.EdgeAppProcessing},
				{Identifier: identifier, Condition: appsv1alpha1.EdgeAppProcessing},
			},
		},
	}
	tmpl := &unstructured.Unstructured{}
	tmpl.SetAPIVersion("apps/v1")
	tmpl.SetKind("Deployment")
	tmpl.SetNamespace("default")
	tmpl.SetName("nginx-hangzhou")

	scheme := runtime.NewScheme()
	require.NoError(t, appsv1alpha1.AddToScheme(scheme))
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(edgeApp).
		WithStatusSubresource(&appsv1alpha1.EdgeApplication{}).
		Build()
	c := &Controller{Client: cli}

	got := &appsv1alpha1.EdgeApplication{}
	require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(edgeApp), got))
	require.NoError(t, c.updateStatus(ctx, got, []*utils.TemplateInfo{{Ordinal: 0, Template: tmpl}}))

	require.NoError(t, cli.Get(ctx, client.ObjectKeyFromObject(edgeApp), got))
	assert.Equal(t, []appsv1alpha1.ManifestStatus{
		{Identifier: identifier, Condition: appsv1alpha1.EdgeAppAvailable},
	}, got.Status.WorkloadStatus)
}
