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

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	devicesv1beta1 "github.com/kubeedge/api/apis/devices/v1beta1"
	"github.com/kubeedge/beehive/pkg/core/model"
)

type MockMessageLayer struct {
	sendCalled          int
	buildResourceCalled int
}

func (m *MockMessageLayer) Send(_ model.Message) error {
	m.sendCalled++
	return nil
}

func (m *MockMessageLayer) Receive() (model.Message, error) {
	return model.Message{}, nil
}

func (m *MockMessageLayer) Response(_ model.Message) error {
	return nil
}

func (m *MockMessageLayer) BuildResource(_, _, _, _ string) (string, error) {
	m.buildResourceCalled++
	return "test-resource", nil
}

func (m *MockMessageLayer) BuildResponse(_, _, _, _ string, _ model.Message) (model.Message, error) {
	return model.Message{}, nil
}

func (m *MockMessageLayer) AssertNotCalled(t *testing.T, methodName string) {
	switch methodName {
	case "BuildResource":
		assert.Equal(t, 0, m.buildResourceCalled, "BuildResource should not have been called")
	case "Send":
		assert.Equal(t, 0, m.sendCalled, "Send should not have been called")
	}
}

func (m *MockMessageLayer) AssertNumberOfCalls(t *testing.T, methodName string, expectedCalls int) {
	switch methodName {
	case "Send":
		assert.Equal(t, expectedCalls, m.sendCalled, "Send was called %d times, expected %d", m.sendCalled, expectedCalls)
	}
}

type TestObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (o *TestObject) GetObjectKind() schema.ObjectKind {
	return &o.TypeMeta
}

func (o *TestObject) DeepCopyObject() runtime.Object {
	return &TestObject{
		TypeMeta:   o.TypeMeta,
		ObjectMeta: *o.ObjectMeta.DeepCopy(),
	}
}

type CustomLabelFieldSelector struct {
	LabelFieldSelector
	ShouldMatch bool
}

func (s CustomLabelFieldSelector) MatchObj(_ runtime.Object) bool {
	return s.ShouldMatch
}

func createTestSelector(shouldMatch bool) LabelFieldSelector {
	labelStr := ""
	fieldStr := ""

	if !shouldMatch {
		labelStr = "app=non-existent-app"
		fieldStr = "metadata.name=non-existent-name"
	}

	return NewSelector(labelStr, fieldStr)
}

func init() {
	scheme := runtime.NewScheme()
	groupVersion := schema.GroupVersion{Group: "test", Version: "v1"}
	scheme.AddKnownTypes(groupVersion, &TestObject{})
}

func TestNewSelectorListener(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	selector := NewSelector("", "")

	listener := NewSelectorListener("test-id", "test-node", gvr, selector)

	assert.Equal(t, "test-id", listener.id)
	assert.Equal(t, "test-node", listener.nodeName)
	assert.Equal(t, gvr, listener.gvr)
	assert.Equal(t, selector, listener.selector)
}

func TestSendObjWithNonMatchingSelector(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	labelSelector, _ := labels.Parse("app=non-existent-app")
	fieldSelector := fields.ParseSelectorOrDie("metadata.name=non-existent-name")

	selector := LabelFieldSelector{
		Label: labelSelector,
		Field: fieldSelector,
	}

	listener := &SelectorListener{
		id:       "test-id",
		nodeName: "test-node",
		gvr:      gvr,
		selector: selector,
	}

	mockML := &MockMessageLayer{}
	testObj := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
	}

	event := watch.Event{
		Type:   watch.Added,
		Object: testObj,
	}

	listener.sendObj(event, mockML)

	mockML.AssertNotCalled(t, "BuildResource")
	mockML.AssertNotCalled(t, "Send")
}

func TestSendObjWithMetaAccessorError(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	selector := NewSelector("", "")

	listener := &SelectorListener{
		id:       "test-id",
		nodeName: "test-node",
		gvr:      gvr,
		selector: selector,
	}

	mockML := &MockMessageLayer{}

	invalidObj := &struct{ runtime.Object }{}

	event := watch.Event{
		Type:   watch.Added,
		Object: invalidObj,
	}

	listener.sendObj(event, mockML)

	mockML.AssertNotCalled(t, "BuildResource")
	mockML.AssertNotCalled(t, "Send")
}

func TestSendAllObjects(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	selector := NewSelector("", "")

	listener := &SelectorListener{
		id:       "test-id",
		nodeName: "test-node",
		gvr:      gvr,
		selector: selector,
	}

	mockML := &MockMessageLayer{}

	obj1 := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-obj-1",
			Namespace: "test-namespace",
		},
	}

	obj2 := &TestObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "test/v1",
			Kind:       "TestObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-obj-2",
			Namespace: "test-namespace",
		},
	}

	objects := []runtime.Object{obj1, obj2}

	handler := &CommonResourceEventHandler{
		messageLayer: mockML,
	}

	listener.sendAllObjects(objects, handler)

	mockML.AssertNumberOfCalls(t, "Send", 2)
}

type capturingMessageLayer struct {
	MockMessageLayer
	messages []model.Message
}

func (m *capturingMessageLayer) Send(msg model.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

// TestSendAllObjectsSetsTypeMeta checks that objects listed from the informer
// cache, which have an empty TypeMeta, are sent with an apiVersion/kind: from
// the client-go or KubeEdge scheme for typed objects, from the listener's GVR
// otherwise. It also checks that the cached objects are not modified.
func TestSendAllObjectsSetsTypeMeta(t *testing.T) {
	tests := []struct {
		name           string
		gvr            schema.GroupVersionResource
		obj            runtime.Object
		wantAPIVersion string
		wantKind       string
	}{
		{
			name: "core endpoints",
			gvr:  schema.GroupVersionResource{Version: "v1", Resource: "endpoints"},
			obj: &corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{Name: "websvc", Namespace: "e2e"},
				Subsets:    []corev1.EndpointSubset{{Addresses: []corev1.EndpointAddress{{IP: "192.0.2.10"}}}},
			},
			wantAPIVersion: "v1",
			wantKind:       "Endpoints",
		},
		{
			name:           "core services",
			gvr:            schema.GroupVersionResource{Version: "v1", Resource: "services"},
			obj:            &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "websvc", Namespace: "e2e"}},
			wantAPIVersion: "v1",
			wantKind:       "Service",
		},
		{
			name:           "core configmaps",
			gvr:            schema.GroupVersionResource{Version: "v1", Resource: "configmaps"},
			obj:            &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: "e2e"}},
			wantAPIVersion: "v1",
			wantKind:       "ConfigMap",
		},
		{
			name:           "apps daemonsets",
			gvr:            schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
			obj:            &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ds", Namespace: "e2e"}},
			wantAPIVersion: "apps/v1",
			wantKind:       "DaemonSet",
		},
		{
			name:           "KubeEdge devicemodels",
			gvr:            devicesv1beta1.SchemeGroupVersion.WithResource("devicemodels"),
			obj:            &devicesv1beta1.DeviceModel{ObjectMeta: metav1.ObjectMeta{Name: "model", Namespace: "e2e"}},
			wantAPIVersion: "devices.kubeedge.io/v1beta1",
			wantKind:       "DeviceModel",
		},
		{
			name: "unstructured without TypeMeta falls back to the GVR",
			gvr:  schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"},
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"metadata": map[string]interface{}{"name": "w", "namespace": "e2e"},
			}},
			wantAPIVersion: "example.com/v1",
			wantKind:       "Widget",
		},
		{
			name: "existing TypeMeta is kept",
			gvr:  schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"},
			obj: &corev1.Endpoints{
				TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Endpoints"},
				ObjectMeta: metav1.ObjectMeta{Name: "websvc", Namespace: "e2e"},
			},
			wantAPIVersion: "v1",
			wantKind:       "Endpoints",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener := NewSelectorListener("test-id", "test-node", tt.gvr, NewSelector("", ""))
			ml := &capturingMessageLayer{}
			handler := &CommonResourceEventHandler{messageLayer: ml}
			cached := tt.obj.DeepCopyObject()

			listener.sendAllObjects([]runtime.Object{tt.obj}, handler)

			if !assert.Len(t, ml.messages, 1) {
				return
			}
			msg := ml.messages[0]
			assert.Equal(t, model.InsertOperation, msg.GetOperation())
			content, ok := msg.GetContent().(*unstructured.Unstructured)
			if !assert.True(t, ok, "content is %T, want *unstructured.Unstructured", msg.GetContent()) {
				return
			}
			assert.Equal(t, tt.wantAPIVersion, content.GetAPIVersion())
			assert.Equal(t, tt.wantKind, content.GetKind())
			// The object is shared with the informer cache: the GVK must be
			// set on the sent copy only, not on it (as util.SetMetaType would).
			assert.Equal(t, cached, tt.obj, "informer cache object was modified")
		})
	}
}
