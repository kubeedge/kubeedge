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
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	appsv1alpha1 "github.com/kubeedge/api/apis/apps/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/edgeapplication/utils"
)

type fakeStatusManager struct{}

func (fakeStatusManager) WatchStatus(utils.ResourceInfo) error            { return nil }
func (fakeStatusManager) CancelWatch(utils.ResourceInfo) error            { return nil }
func (fakeStatusManager) SetReconcileTriggerChan(chan event.GenericEvent) {}
func (fakeStatusManager) Start() error                                    { return nil }

func TestSyncEdgeApplicationStaysInNamespace(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := appsv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(corev1.SchemeGroupVersion.WithKind("Service"), meta.RESTScopeNamespace)
	mapper.Add(appsv1.SchemeGroupVersion.WithKind("Deployment"), meta.RESTScopeNamespace)
	mapper.Add(appsv1alpha1.SchemeGroupVersion.WithKind("EdgeApplication"), meta.RESTScopeNamespace)
	mapper.Add(appsv1alpha1.SchemeGroupVersion.WithKind("NodeGroup"), meta.RESTScopeRoot)

	edgeApp := &appsv1alpha1.EdgeApplication{
		TypeMeta:   metav1.TypeMeta{APIVersion: appsv1alpha1.SchemeGroupVersion.String(), Kind: "EdgeApplication"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "tenant", Name: "app", UID: "app-uid"},
	}
	// objects that exist before the EdgeApplication is synced
	kubeDNS := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceSystem, Name: "kube-dns"},
		Spec:       corev1.ServiceSpec{Selector: map[string]string{"k8s-app": "kube-dns"}},
	}
	coreDNS := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceSystem, Name: "coredns"}}
	ownedOld := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Namespace: edgeApp.Namespace,
		Name:      "old",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: edgeApp.APIVersion,
			Kind:       edgeApp.Kind,
			Name:       edgeApp.Name,
			UID:        edgeApp.UID,
			Controller: pointer.Bool(true),
		}},
	}}
	// objects only described in the manifests
	web := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: edgeApp.Namespace, Name: "web"}}
	nodeGroup := &appsv1alpha1.NodeGroup{ObjectMeta: metav1.ObjectMeta{Name: "evil"}}

	manifest := func(obj map[string]interface{}) appsv1alpha1.Manifest {
		raw, err := json.Marshal(obj)
		if err != nil {
			t.Fatal(err)
		}
		return appsv1alpha1.Manifest{RawExtension: runtime.RawExtension{Raw: raw}}
	}
	service := func(svc *corev1.Service, app string) map[string]interface{} {
		return map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata":   map[string]interface{}{"namespace": svc.Namespace, "name": svc.Name},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{"app": app},
				"ports":    []interface{}{map[string]interface{}{"port": int64(53)}},
			},
		}
	}
	edgeApp.Spec.WorkloadTemplate.Manifests = []appsv1alpha1.Manifest{
		manifest(service(web, "web")),
		manifest(service(kubeDNS, "evil")),
		manifest(map[string]interface{}{
			"apiVersion": appsv1alpha1.SchemeGroupVersion.String(),
			"kind":       "NodeGroup",
			"metadata":   map[string]interface{}{"name": nodeGroup.Name},
		}),
	}
	lastContained, err := json.Marshal([]utils.ResourceInfo{
		{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: coreDNS.Namespace, Name: coreDNS.Name},
		{Version: "v1", Kind: "Service", Namespace: ownedOld.Namespace, Name: ownedOld.Name},
	})
	if err != nil {
		t.Fatal(err)
	}
	edgeApp.Annotations = map[string]string{constants.LastContainedResourcesAnnotationKey: string(lastContained)}

	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRESTMapper(mapper).
		WithStatusSubresource(&appsv1alpha1.EdgeApplication{}).
		WithObjects(edgeApp.DeepCopy(), kubeDNS.DeepCopy(), coreDNS.DeepCopy(), ownedOld.DeepCopy()).
		Build()
	c := &Controller{
		Client: cli,
		Serializer: jsonserializer.NewSerializerWithOptions(jsonserializer.DefaultMetaFactory,
			scheme, scheme, jsonserializer.SerializerOptions{Yaml: true}),
		StatusManager: fakeStatusManager{},
	}

	if _, err := c.syncEdgeApplication(ctx, edgeApp); err == nil {
		t.Errorf("expected an error for manifests outside namespace %q", edgeApp.Namespace)
	}

	if err := cli.Get(ctx, client.ObjectKeyFromObject(web), &corev1.Service{}); err != nil {
		t.Errorf("service %s/%s should be created: %v", web.Namespace, web.Name, err)
	}
	gotDNS := &corev1.Service{}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(kubeDNS), gotDNS); err != nil {
		t.Fatalf("failed to get service %s/%s: %v", kubeDNS.Namespace, kubeDNS.Name, err)
	}
	if !reflect.DeepEqual(gotDNS.Spec.Selector, kubeDNS.Spec.Selector) {
		t.Errorf("service %s/%s was modified, selector: %v", kubeDNS.Namespace, kubeDNS.Name, gotDNS.Spec.Selector)
	}
	err = cli.Get(ctx, types.NamespacedName{Name: nodeGroup.Name}, &appsv1alpha1.NodeGroup{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("nodegroup %s should not be created, get err: %v", nodeGroup.Name, err)
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(coreDNS), &appsv1.Deployment{}); err != nil {
		t.Errorf("deployment %s/%s should not be removed: %v", coreDNS.Namespace, coreDNS.Name, err)
	}
	err = cli.Get(ctx, client.ObjectKeyFromObject(ownedOld), &corev1.Service{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("service %s/%s should be removed, get err: %v", ownedOld.Namespace, ownedOld.Name, err)
	}
}
