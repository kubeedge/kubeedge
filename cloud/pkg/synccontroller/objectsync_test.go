/*
Copyright 2022 The KubeEdge Authors.

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

package synccontroller

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"github.com/kubeedge/api/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/beehive/pkg/common"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	tf "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/testing"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
)

func TestCompareResourceVersion(t *testing.T) {
	tests := []struct {
		name       string
		testNumber []string
		want       int
	}{
		{
			name:       "test greater than",
			testNumber: []string{"123", "124"},
			want:       -1,
		},
		{
			name:       "test less than",
			testNumber: []string{"124", "123"},
			want:       1,
		},
		{
			name:       "test equal",
			testNumber: []string{"123", "123"},
			want:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareResourceVersion(tt.testNumber[0], tt.testNumber[1])
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("CompareResourceVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetObjectResourceVersion(t *testing.T) {
	tests := []struct {
		name string
		obj  *v1.ObjectMeta
		want string
	}{
		{
			name: "new full object",
			obj:  newK8sObjectMeta("test", "test", "123"),
			want: "123",
		},
		{
			name: "new nil objcet",
			obj:  nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if tt.obj == nil {
				got = GetObjectResourceVersion(nil)
			} else {
				got = GetObjectResourceVersion(tt.obj)
			}
			if !reflect.DeepEqual(tt.want, got) {
				t.Errorf("GetObjectResourceVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newK8sObjectMeta(name, namespace, resourceVersion string) *v1.ObjectMeta {
	return &v1.ObjectMeta{
		Name:            name,
		Namespace:       namespace,
		ResourceVersion: resourceVersion,
	}
}

func TestGCOrphanedObjectSync(t *testing.T) {
	resource, _ := messagelayer.BuildResource(tf.TestNodeID, tf.TestNamespace, "pod", tf.TestPodName)
	tests := []struct {
		name             string
		ExpectedResource string
		ObjectSyncs      *v1alpha1.ObjectSync
	}{
		{
			name:             "test gcOrphanedObjectSyncs",
			ObjectSyncs:      tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			ExpectedResource: resource,
		},
	}
	cloudHub := &common.ModuleInfo{
		ModuleName: modules.CloudHubModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(cloudHub)
	beehiveContext.AddModuleGroup(modules.CloudHubModuleName, modules.CloudHubModuleGroup)
	client.DefaultGetRestMapper = func() (mapper meta.RESTMapper, err error) { return nil, nil }

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testController := newSyncController(true)
			go testController.gcOrphanedObjectSync(tt.ObjectSyncs)
			message, _ := beehiveContext.Receive(modules.CloudHubModuleName)
			if !reflect.DeepEqual(message.GetResource(), tt.ExpectedResource) {
				t.Errorf("gcOrphanedObjectSync() = %v, want %v", message.GetResource(), tt.ExpectedResource)
			}
		})
	}
}

func TestSendEvents(t *testing.T) {
	tests := []struct {
		name              string
		ExpectedOperation string
		ObjectSyncs       *v1alpha1.ObjectSync
	}{
		{
			name:              "test sendEvents",
			ObjectSyncs:       tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			ExpectedOperation: model.UpdateOperation,
		},
	}
	cloudHub := &common.ModuleInfo{
		ModuleName: modules.CloudHubModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(cloudHub)
	beehiveContext.AddModuleGroup(modules.CloudHubModuleName, modules.CloudHubModuleGroup)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := tt.ObjectSyncs.DeepCopy()
			go sendEvents(tf.TestNodeID, tt.ObjectSyncs, "pod", "2", tmp)
			message, _ := beehiveContext.Receive(modules.CloudHubModuleName)
			if !reflect.DeepEqual(message.GetOperation(), tt.ExpectedOperation) {
				t.Errorf("sendEvents() = %v, want %v", message.GetResource(), tt.ExpectedOperation)
			}
		})
	}
}
func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}

var podGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

func TestManageObjectSync(t *testing.T) {
	tests := []struct {
		name              string
		ExpectedOperation string
		ObjectSyncs       *v1alpha1.ObjectSync
	}{
		{
			name:              "test manageObjectSyncs delete",
			ObjectSyncs:       tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "2"), "Pod"),
			ExpectedOperation: model.DeleteOperation,
		},
		{
			name:              "test manageObjectSyncs update",
			ObjectSyncs:       tf.NewObjectSync(tf.NewTestPodResource(tf.TestPodName, tf.TestPodUID, "1"), "Pod"),
			ExpectedOperation: model.UpdateOperation,
		},
	}

	cloudHub := &common.ModuleInfo{
		ModuleName: modules.CloudHubModuleName,
		ModuleType: common.MsgCtxTypeChannel,
	}
	beehiveContext.InitContext([]string{common.MsgCtxTypeChannel})
	beehiveContext.AddModule(cloudHub)
	beehiveContext.AddModuleGroup(modules.CloudHubModuleName, modules.CloudHubModuleGroup)

	client.DefaultGetRestMapper = func() (mapper meta.RESTMapper, err error) { return nil, nil }

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testController := newSyncController(true)
			testController.informerManager = informers.NewFakeInformerManager()
			newDynamicClient := fake.NewSimpleDynamicClient(runtime.NewScheme())
			testController.kubeclient = newDynamicClient
			err := testController.informerManager.GetKubeEdgeInformerFactory().Reliablesyncs().V1alpha1().ObjectSyncs().Informer().GetIndexer().Add(tt.ObjectSyncs)
			if err != nil {
				t.Errorf("add objectSync failed: %v", err)
			}
			testController.objectSyncLister = testController.informerManager.GetKubeEdgeInformerFactory().Reliablesyncs().V1alpha1().ObjectSyncs().Lister()

			if tt.ExpectedOperation == model.UpdateOperation {
				testPod := &corev1.Pod{
					TypeMeta:   v1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
					ObjectMeta: v1.ObjectMeta{Name: tf.TestPodName, Namespace: tf.TestNamespace, UID: tf.TestPodUID, ResourceVersion: "2"},
				}

				podInformerPair, _ := testController.informerManager.GetInformerPair(podGVR)
				err := podInformerPair.Informer.GetStore().Add(testPod)
				if err != nil {
					t.Errorf("create pod failed: %v", err)
				}
				t.Logf("create pod success: %v", testPod)
			}

			go testController.reconcileObjectSyncs()
			message, _ := beehiveContext.Receive(modules.CloudHubModuleName)
			if !reflect.DeepEqual(message.GetOperation(), tt.ExpectedOperation) {
				t.Errorf("manageObjectSyncs() = %v, want %v", message.GetOperation(), tt.ExpectedOperation)
			}
		})
	}
}

// keepNodeFilter keeps only the EndpointSlice endpoints on the target node.
type keepNodeFilter struct{}

func (keepNodeFilter) Name() string { return "synccontrollerTestKeepNode" }

func (keepNodeFilter) NeedFilter(content interface{}) bool {
	u, ok := content.(*unstructured.Unstructured)
	return ok && u.GetKind() == "EndpointSlice"
}

func (keepNodeFilter) FilterResource(targetNode string, obj runtime.Object) {
	u := obj.(*unstructured.Unstructured)
	endpoints, _, _ := unstructured.NestedSlice(u.Object, "endpoints")
	var kept []interface{}
	for _, ep := range endpoints {
		if node, _, _ := unstructured.NestedString(ep.(map[string]interface{}), "nodeName"); node == targetNode {
			kept = append(kept, ep)
		}
	}
	_ = unstructured.SetNestedSlice(u.Object, kept, "endpoints")
}

func TestFilterForNode(t *testing.T) {
	filter.Register(keepNodeFilter{})
	defer filter.Unregister(keepNodeFilter{}.Name())

	node1, node2 := "node1", "node2"
	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: v1.ObjectMeta{Name: "svc-abc", Namespace: "default", ResourceVersion: "7"},
		Endpoints: []discoveryv1.Endpoint{
			{Addresses: []string{"10.0.0.1"}, NodeName: &node1},
			{Addresses: []string{"10.0.0.2"}, NodeName: &node2},
		},
	}
	gvk := discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice")

	got := filterForNode(slice, gvk, node1)
	if got == nil {
		t.Fatal("expected a filtered copy of the EndpointSlice")
	}
	if got.GroupVersionKind() != gvk {
		t.Errorf("kind = %v, want %v", got.GroupVersionKind(), gvk)
	}
	if got.GetResourceVersion() != "7" {
		t.Errorf("resourceVersion = %q, want 7", got.GetResourceVersion())
	}
	endpoints, _, _ := unstructured.NestedSlice(got.Object, "endpoints")
	if len(endpoints) != 1 {
		t.Errorf("got %d endpoints, want 1", len(endpoints))
	}
	if len(slice.Endpoints) != 2 {
		t.Error("the cached object must not change")
	}

	cm := &corev1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: "cm", Namespace: "default"}}
	if got := filterForNode(cm, corev1.SchemeGroupVersion.WithKind("ConfigMap"), node1); got != nil {
		t.Errorf("expected no copy for an object no filter applies to, got %v", got)
	}
}

// TestFilterForNodeWhileFiltersRegister models cloudcore startup: beehive
// starts every module in its own goroutine, so dynamiccontroller registers its
// filters while synccontroller may already be filtering re-sent objects.
func TestFilterForNodeWhileFiltersRegister(t *testing.T) {
	defer filter.Unregister(keepNodeFilter{}.Name())

	node1 := "node1"
	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: v1.ObjectMeta{Name: "svc-abc", Namespace: "default", ResourceVersion: "7"},
		Endpoints:  []discoveryv1.Endpoint{{Addresses: []string{"10.0.0.1"}, NodeName: &node1}},
	}
	gvk := discoveryv1.SchemeGroupVersion.WithKind("EndpointSlice")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			filter.Register(keepNodeFilter{})
		}
	}()
	for i := 0; i < 1000; i++ {
		filterForNode(slice, gvk, node1)
	}
	<-done
}
