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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

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

type recordingMessageLayer struct {
	MockMessageLayer
	messages []model.Message
}

func (m *recordingMessageLayer) Send(msg model.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

// TestSendObjResourceTypeUsesObjectKind checks that the resource type of the
// message comes from the kind of the object rather than from a guess on the
// resource of the listener, which is wrong for kinds like Database (databases)
// or PodChaos (podchaos).
func TestSendObjResourceTypeUsesObjectKind(t *testing.T) {
	tests := []struct {
		name         string
		gvr          schema.GroupVersionResource
		obj          runtime.Object
		resourceType string
	}{
		{
			name: "CRD with an irregular plural",
			gvr:  schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "databases"},
			obj: &TestObject{
				TypeMeta:   metav1.TypeMeta{APIVersion: "example.com/v1", Kind: "Database"},
				ObjectMeta: metav1.ObjectMeta{Name: "db", Namespace: "default"},
			},
			resourceType: "database",
		},
		{
			name: "CRD whose plural is its singular",
			gvr:  schema.GroupVersionResource{Group: "chaos-mesh.org", Version: "v1alpha1", Resource: "podchaos"},
			obj: &TestObject{
				TypeMeta:   metav1.TypeMeta{APIVersion: "chaos-mesh.org/v1alpha1", Kind: "PodChaos"},
				ObjectMeta: metav1.ObjectMeta{Name: "chaos", Namespace: "default"},
			},
			resourceType: "podchaos",
		},
		{
			name: "typed object without TypeMeta falls back to the RESTMapper",
			gvr:  schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"},
			obj: &TestObject{
				ObjectMeta: metav1.ObjectMeta{Name: "lease", Namespace: "default"},
			},
			resourceType: "lease",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener := NewSelectorListener("id", "node", tt.gvr, NewSelector("", ""))
			ml := &recordingMessageLayer{}
			listener.sendObj(watch.Event{Type: watch.Added, Object: tt.obj}, ml)

			assert.Len(t, ml.messages, 1)
			accessor, err := meta.Accessor(tt.obj)
			assert.NoError(t, err)
			assert.Equal(t, "node/node/default/"+tt.resourceType+"/"+accessor.GetName(), ml.messages[0].GetResource())
		})
	}
}
