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
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/overridemanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/statusmanager"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
)

type noopStatusWatcher struct {
	statusmanager.StatusManager
}

func (noopStatusWatcher) WatchStatus(utils.ResourceInfo) error { return nil }

func TestReconcileManifestOrder(t *testing.T) {
	ctx := t.Context()
	scheme := runtime.NewScheme()
	for _, addToScheme := range []func(*runtime.Scheme) error{appsv1.AddToScheme, corev1.AddToScheme, appsv1alpha1.AddToScheme} {
		if err := addToScheme(scheme); err != nil {
			t.Fatal(err)
		}
	}

	deployment := appsv1alpha1.Manifest{RawExtension: runtime.RawExtension{Raw: []byte(`{
		"apiVersion": "apps/v1",
		"kind": "Deployment",
		"metadata": {"namespace": "default", "name": "workload"},
		"spec": {
			"selector": {"matchLabels": {"app": "workload"}},
			"template": {
				"metadata": {"labels": {"app": "workload"}},
				"spec": {"containers": [{"name": "workload", "image": "nginx"}]}
			}
		}
	}`)}}
	configMap := appsv1alpha1.Manifest{RawExtension: runtime.RawExtension{Raw: []byte(`{
		"apiVersion": "v1",
		"kind": "ConfigMap",
		"metadata": {"namespace": "default", "name": "settings"}
	}`)}}
	edgeApp := &appsv1alpha1.EdgeApplication{
		TypeMeta:   metav1.TypeMeta{APIVersion: appsv1alpha1.SchemeGroupVersion.String(), Kind: "EdgeApplication"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "application", UID: "edge-app"},
		Spec: appsv1alpha1.EdgeApplicationSpec{
			WorkloadScope: appsv1alpha1.WorkloadScope{
				TargetNodeGroups: []appsv1alpha1.TargetNodeGroup{{Name: "edge"}},
			},
		},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(edgeApp).WithObjects(edgeApp).Build()
	controller := &Controller{
		Client:        cli,
		Serializer:    unstructured.UnstructuredJSONScheme,
		Overrider:     &overridemanager.OverrideManager{},
		StatusManager: noopStatusWatcher{},
	}
	req := controllerruntime.Request{NamespacedName: client.ObjectKeyFromObject(edgeApp)}
	workloadKey := client.ObjectKey{Namespace: "default", Name: "workload"}

	for _, step := range []struct {
		name        string
		manifests   []appsv1alpha1.Manifest
		wantOrdinal int
	}{
		{name: "create", manifests: []appsv1alpha1.Manifest{deployment}, wantOrdinal: 0},
		{name: "insert", manifests: []appsv1alpha1.Manifest{configMap, deployment}, wantOrdinal: 1},
		{name: "reorder", manifests: []appsv1alpha1.Manifest{deployment, configMap}, wantOrdinal: 0},
		{name: "remove", manifests: []appsv1alpha1.Manifest{configMap}, wantOrdinal: -1},
	} {
		if err := cli.Get(ctx, req.NamespacedName, edgeApp); err != nil {
			t.Fatalf("%s: get EdgeApplication: %v", step.name, err)
		}
		edgeApp.Spec.WorkloadTemplate.Manifests = step.manifests
		if err := cli.Update(ctx, edgeApp); err != nil {
			t.Fatalf("%s: update EdgeApplication: %v", step.name, err)
		}
		if _, err := controller.Reconcile(ctx, req); err != nil {
			t.Fatalf("%s: reconcile EdgeApplication: %v", step.name, err)
		}

		err := cli.Get(ctx, workloadKey, &appsv1.Deployment{})
		if step.wantOrdinal == -1 {
			if !apierrors.IsNotFound(err) {
				t.Fatalf("%s: expected Deployment to be deleted, got %v", step.name, err)
			}
		} else if err != nil {
			t.Fatalf("%s: expected Deployment to exist, got %v", step.name, err)
		}

		if err := cli.Get(ctx, req.NamespacedName, edgeApp); err != nil {
			t.Fatalf("%s: get workload status: %v", step.name, err)
		}
		var workloadStatus *appsv1alpha1.ManifestStatus
		for i := range edgeApp.Status.WorkloadStatus {
			status := &edgeApp.Status.WorkloadStatus[i]
			if status.Identifier.Name == workloadKey.Name {
				if workloadStatus != nil {
					t.Fatalf("%s: found duplicate Deployment status", step.name)
				}
				workloadStatus = status
			}
		}
		if step.wantOrdinal == -1 {
			if workloadStatus != nil {
				t.Fatalf("%s: removed Deployment still has status: %+v", step.name, workloadStatus)
			}
			continue
		}
		if workloadStatus == nil || workloadStatus.Identifier.Ordinal != step.wantOrdinal ||
			workloadStatus.Identifier.Version != "v1" {
			t.Fatalf("%s: expected Deployment status with ordinal %d and version v1, got %+v", step.name, step.wantOrdinal, workloadStatus)
		}
		if step.name == "create" {
			// A child event would normally set the status to Available after creation.
			workloadStatus.Condition = appsv1alpha1.EdgeAppAvailable
			if err := cli.Status().Update(ctx, edgeApp); err != nil {
				t.Fatalf("set Available status: %v", err)
			}
		} else if workloadStatus.Condition != appsv1alpha1.EdgeAppAvailable {
			t.Fatalf("%s: expected Deployment to remain Available, got %q", step.name, workloadStatus.Condition)
		}
	}
}

func TestReconcileRemovalAfterUnservedVersion(t *testing.T) {
	ctx := t.Context()
	scheme := runtime.NewScheme()
	for _, addToScheme := range []func(*runtime.Scheme) error{corev1.AddToScheme, appsv1alpha1.AddToScheme} {
		if err := addToScheme(scheme); err != nil {
			t.Fatal(err)
		}
	}
	edgeApp := &appsv1alpha1.EdgeApplication{
		TypeMeta:   metav1.TypeMeta{APIVersion: appsv1alpha1.SchemeGroupVersion.String(), Kind: "EdgeApplication"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "application", UID: "edge-app"},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(edgeApp).WithObjects(edgeApp).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, cli client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if obj.GetObjectKind().GroupVersionKind().Version == "v2" {
					return &meta.NoKindMatchError{GroupKind: obj.GetObjectKind().GroupVersionKind().GroupKind(), SearchedVersions: []string{"v2"}}
				}
				return cli.Get(ctx, key, obj, opts...)
			},
		}).Build()
	controller := &Controller{
		Client:        cli,
		Serializer:    unstructured.UnstructuredJSONScheme,
		Overrider:     &overridemanager.OverrideManager{},
		StatusManager: noopStatusWatcher{},
	}
	req := controllerruntime.Request{NamespacedName: client.ObjectKeyFromObject(edgeApp)}
	reconcile := func(manifests ...appsv1alpha1.Manifest) error {
		t.Helper()
		if err := cli.Get(ctx, req.NamespacedName, edgeApp); err != nil {
			t.Fatal(err)
		}
		edgeApp.Spec.WorkloadTemplate.Manifests = manifests
		if err := cli.Update(ctx, edgeApp); err != nil {
			t.Fatal(err)
		}
		_, err := controller.Reconcile(ctx, req)
		return err
	}
	configMap := func(version string) appsv1alpha1.Manifest {
		return appsv1alpha1.Manifest{RawExtension: runtime.RawExtension{
			Raw: []byte(fmt.Sprintf(`{"apiVersion":%q,"kind":"ConfigMap","metadata":{"namespace":"default","name":"settings"}}`, version)),
		}}
	}
	if err := reconcile(configMap("v1")); err != nil {
		t.Fatal(err)
	}
	if err := reconcile(configMap("v2")); err == nil {
		t.Fatal("expected reconciliation to fail for the unserved API version")
	}
	key := client.ObjectKey{Namespace: "default", Name: "settings"}
	if err := cli.Get(ctx, key, &corev1.ConfigMap{}); err != nil {
		t.Fatalf("expected original ConfigMap after failed apply: %v", err)
	}
	if err := reconcile(); err != nil {
		t.Fatalf("remove ConfigMap manifest: %v", err)
	}
	if err := cli.Get(ctx, key, &corev1.ConfigMap{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected original ConfigMap to be deleted, got %v", err)
	}
}
