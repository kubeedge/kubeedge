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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	genericinformers "github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
)

type MockInformer struct {
	addEventHandlerFunc func(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error)
}

func (m *MockInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	if m.addEventHandlerFunc != nil {
		return m.addEventHandlerFunc(handler)
	}
	return &MockResourceEventHandlerRegistration{}, nil
}

func (m *MockInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, _ time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	return m.AddEventHandler(handler)
}

func (m *MockInformer) GetStore() cache.Store                                { return nil }
func (m *MockInformer) GetController() cache.Controller                      { return nil }
func (m *MockInformer) Run(_ <-chan struct{})                                {}
func (m *MockInformer) HasSynced() bool                                      { return true }
func (m *MockInformer) LastSyncResourceVersion() string                      { return "" }
func (m *MockInformer) SetWatchErrorHandler(_ cache.WatchErrorHandler) error { return nil }
func (m *MockInformer) SetTransform(_ cache.TransformFunc) error             { return nil }

type MockResourceEventHandlerRegistration struct{}

func (m *MockResourceEventHandlerRegistration) HasSynced() bool { return true }
func (m *MockResourceEventHandlerRegistration) Key() string     { return "mock-key" }

type MockMessageLayer struct{}

func (m *MockMessageLayer) Send(_ interface{}) error                  { return nil }
func (m *MockMessageLayer) Response(_ interface{}) error              { return nil }
func (m *MockMessageLayer) Request(_ interface{}) error               { return nil }
func (m *MockMessageLayer) Receive() (interface{}, error)             { return nil, nil }
func (m *MockMessageLayer) ReceiveResourceMsg() (interface{}, error)  { return nil, nil }
func (m *MockMessageLayer) ReceiveResourceResp() (interface{}, error) { return nil, nil }

type MockGenericLister struct {
	listFunc func(selector labels.Selector) ([]runtime.Object, error)
	getFunc  func(name string) (runtime.Object, error)
}

func (m *MockGenericLister) List(selector labels.Selector) ([]runtime.Object, error) {
	if m.listFunc != nil {
		return m.listFunc(selector)
	}
	return []runtime.Object{}, nil
}

func (m *MockGenericLister) Get(name string) (runtime.Object, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, fmt.Errorf("not found")
}

func (m *MockGenericLister) ByNamespace(_ string) cache.GenericNamespaceLister {
	return &MockGenericNamespaceLister{}
}

type MockGenericNamespaceLister struct {
	listFunc func(selector labels.Selector) ([]runtime.Object, error)
	getFunc  func(name string) (runtime.Object, error)
}

func (m *MockGenericNamespaceLister) List(selector labels.Selector) ([]runtime.Object, error) {
	if m.listFunc != nil {
		return m.listFunc(selector)
	}
	return []runtime.Object{}, nil
}

func (m *MockGenericNamespaceLister) Get(name string) (runtime.Object, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, fmt.Errorf("not found")
}

func createTestObject(name string, labels map[string]string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": "default",
				"labels":    labels,
			},
			"spec":   map[string]interface{}{},
			"status": map[string]interface{}{},
		},
	}
}

func TestHandlerCenter(t *testing.T) {
	t.Run("GetListenersForNode", func(t *testing.T) {
		center := &handlerCenter{
			listenerManager: newListenerManager(),
			handlers:        make(map[schema.GroupVersionResource]*CommonResourceEventHandler),
		}

		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		selector := LabelFieldSelector{Label: labels.Everything(), Field: fields.Everything()}

		listener := &SelectorListener{
			gvr:      gvr,
			nodeName: "test-node",
			id:       "test-listener",
			selector: selector,
		}

		center.listenerManager.AddListener(listener)

		listeners := center.GetListenersForNode("test-node")
		assert.NotNil(t, listeners)
		assert.Len(t, listeners, 1)
		assert.Contains(t, listeners, "test-listener")

		assert.Nil(t, center.GetListenersForNode("non-existent-node"))
	})

	t.Run("HandlersMap", func(t *testing.T) {
		center := &handlerCenter{
			listenerManager: newListenerManager(),
			handlers:        make(map[schema.GroupVersionResource]*CommonResourceEventHandler),
		}

		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

		assert.Empty(t, center.handlers)

		handler := &CommonResourceEventHandler{
			listenerManager: center.listenerManager,
			gvr:             gvr,
			events:          make(chan watch.Event, 10),
		}

		center.handlers[gvr] = handler

		assert.Len(t, center.handlers, 1)
		assert.Contains(t, center.handlers, gvr)
		assert.Equal(t, handler, center.handlers[gvr])
	})

	t.Run("DeleteListener", func(t *testing.T) {
		lm := newListenerManager()
		center := &handlerCenter{
			listenerManager: lm,
			handlers:        make(map[schema.GroupVersionResource]*CommonResourceEventHandler),
		}

		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

		handler := &CommonResourceEventHandler{
			listenerManager: lm,
			gvr:             gvr,
			events:          make(chan watch.Event, 10),
		}

		center.handlers[gvr] = handler

		selector := LabelFieldSelector{Label: labels.Everything(), Field: fields.Everything()}
		listener := &SelectorListener{
			gvr:      gvr,
			nodeName: "test-node",
			id:       "test-listener",
			selector: selector,
		}

		lm.AddListener(listener)

		listeners := lm.GetListenersForNode("test-node")
		assert.NotNil(t, listeners)
		assert.Contains(t, listeners, "test-listener")

		center.DeleteListener(listener)

		assert.Nil(t, lm.GetListenersForNode("test-node"))
	})

	t.Run("ForResource", func(t *testing.T) {
		center := &handlerCenter{
			listenerManager: newListenerManager(),
			handlers:        make(map[schema.GroupVersionResource]*CommonResourceEventHandler),
		}

		gvr := schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "tests"}

		mockHandler := &CommonResourceEventHandler{
			gvr:    gvr,
			events: make(chan watch.Event, 10),
		}

		center.handlers[gvr] = mockHandler

		handler := center.ForResource(gvr)

		assert.Equal(t, mockHandler, handler)
	})

	t.Run("FactoryFunctionReplacement", func(t *testing.T) {
		originalFunc := newCommonResourceEventHandlerFunc

		callCount := 0
		newCommonResourceEventHandlerFunc = func(
			gvr schema.GroupVersionResource,
			listenerManager *listenerManager,
			layer messagelayer.MessageLayer) *CommonResourceEventHandler {
			callCount++
			return &CommonResourceEventHandler{
				listenerManager: listenerManager,
				events:          make(chan watch.Event, 10),
				messageLayer:    layer,
				gvr:             gvr,
			}
		}

		gvr := schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "tests"}

		center := &handlerCenter{
			listenerManager: newListenerManager(),
			handlers:        make(map[schema.GroupVersionResource]*CommonResourceEventHandler),
		}

		handler := center.ForResource(gvr)

		assert.Equal(t, 1, callCount)
		assert.NotNil(t, handler)
		assert.Equal(t, gvr, handler.gvr)

		newCommonResourceEventHandlerFunc = originalFunc
	})
}

func TestCommonResourceEventHandler(t *testing.T) {
	t.Run("Event handling", func(t *testing.T) {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		handler := &CommonResourceEventHandler{
			gvr:    gvr,
			events: make(chan watch.Event, 10),
		}

		obj := createTestObject("test-deployment", map[string]string{"app": "nginx"})

		eventTypes := []watch.EventType{watch.Added, watch.Modified, watch.Deleted}
		eventHandlers := []func(interface{}){
			handler.handleAddEvent,
			func(o interface{}) { handler.handleUpdateEvent(nil, o) },
			handler.handleDeleteEvent,
		}

		for i, eventType := range eventTypes {
			eventHandlers[i](obj)

			select {
			case event := <-handler.events:
				assert.Equal(t, eventType, event.Type)
				assert.Equal(t, obj, event.Object)
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("No event received for event type %v", eventType)
			}
		}

		handler.objToEvent(watch.Added, "not-a-runtime-object")

		select {
		case <-handler.events:
			t.Fatal("Unexpected event received for non-runtime.Object")
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("AddListener Error", func(t *testing.T) {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		lm := newListenerManager()
		handler := &CommonResourceEventHandler{
			listenerManager: lm,
			gvr:             gvr,
			events:          make(chan watch.Event, 10),
		}

		mockLister := &MockGenericLister{
			listFunc: func(selector labels.Selector) ([]runtime.Object, error) {
				return nil, fmt.Errorf("test error")
			},
		}

		handler.informer = &genericinformers.InformerPair{
			Lister: mockLister,
		}

		selector := LabelFieldSelector{Label: labels.Everything(), Field: fields.Everything()}
		listener := &SelectorListener{
			gvr:      gvr,
			nodeName: "test-node",
			id:       "test-listener",
			selector: selector,
		}

		err := handler.AddListener(listener)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "Failed to list")
	})

	t.Run("DeleteListener", func(t *testing.T) {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		lm := newListenerManager()
		handler := &CommonResourceEventHandler{
			listenerManager: lm,
			gvr:             gvr,
			events:          make(chan watch.Event, 10),
		}

		selector := LabelFieldSelector{Label: labels.Everything(), Field: fields.Everything()}
		listener := &SelectorListener{
			gvr:      gvr,
			nodeName: "test-node-handler-delete",
			id:       "test-listener-handler-delete",
			selector: selector,
		}

		lm.AddListener(listener)

		listeners := lm.GetListenersForNode("test-node-handler-delete")
		assert.NotNil(t, listeners)
		assert.Contains(t, listeners, "test-listener-handler-delete")

		handler.DeleteListener(listener)

		assert.Nil(t, lm.GetListenersForNode("test-node-handler-delete"))
	})
}

func TestSelectorAndListenerFunctionality(t *testing.T) {
	t.Run("NewSelectorListener", func(t *testing.T) {
		id := "test-id"
		nodeName := "test-node"
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		selector := LabelFieldSelector{
			Label: labels.Everything(),
			Field: fields.Everything(),
		}

		listener := NewSelectorListener(id, nodeName, gvr, selector)

		assert.Equal(t, id, listener.id)
		assert.Equal(t, nodeName, listener.nodeName)
		assert.Equal(t, gvr, listener.gvr)
		assert.Equal(t, selector, listener.selector)
	})

	t.Run("MatchObj function", func(t *testing.T) {
		labelSelector := labels.SelectorFromSet(labels.Set{"app": "nginx"})
		fieldSelector := fields.Everything()
		selector := LabelFieldSelector{
			Label: labelSelector,
			Field: fieldSelector,
		}

		matchingObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "nginx-deployment",
					"namespace": "default",
					"labels": map[string]interface{}{
						"app": "nginx",
					},
				},
			},
		}

		nonMatchingObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "apache-deployment",
					"namespace": "default",
					"labels": map[string]interface{}{
						"app": "apache",
					},
				},
			},
		}

		assert.True(t, selector.MatchObj(matchingObj))
		assert.False(t, selector.MatchObj(nonMatchingObj))
	})

	t.Run("MatchObjWithComplexSelectors", func(t *testing.T) {
		labelSelector := labels.SelectorFromSet(labels.Set{
			"app":     "nginx",
			"version": "v1",
		})

		fieldSelector := fields.SelectorFromSet(fields.Set{
			"metadata.name":      "test-deployment",
			"metadata.namespace": "default",
		})

		selector := LabelFieldSelector{
			Label: labelSelector,
			Field: fieldSelector,
		}

		fullyMatchingObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "test-deployment",
					"namespace": "default",
					"labels": map[string]interface{}{
						"app":     "nginx",
						"version": "v1",
					},
				},
			},
		}

		labelMatchingOnlyObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "wrong-name",
					"namespace": "default",
					"labels": map[string]interface{}{
						"app":     "nginx",
						"version": "v1",
					},
				},
			},
		}

		fieldMatchingOnlyObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "test-deployment",
					"namespace": "default",
					"labels": map[string]interface{}{
						"app":     "apache",
						"version": "v2",
					},
				},
			},
		}

		assert.True(t, selector.MatchObj(fullyMatchingObj), "Should match fully matching object")
		assert.False(t, selector.MatchObj(labelMatchingOnlyObj), "Should not match labels-only matching object")
		assert.False(t, selector.MatchObj(fieldMatchingOnlyObj), "Should not match fields-only matching object")
	})
}

func TestListenerManager(t *testing.T) {
	t.Run("Operations", func(t *testing.T) {
		lm := newListenerManager()

		gvr1 := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		gvr2 := schema.GroupVersionResource{Group: "core", Version: "v1", Resource: "pods"}

		selector := LabelFieldSelector{Label: labels.Everything(), Field: fields.Everything()}

		listener1 := &SelectorListener{gvr: gvr1, nodeName: "node1", id: "listener1", selector: selector}
		lm.AddListener(listener1)

		byNode := lm.GetListenersForNode("node1")
		assert.NotNil(t, byNode)
		assert.Contains(t, byNode, "listener1")

		byGVR := lm.GetListenersForGVR(gvr1)
		assert.NotNil(t, byGVR)
		assert.Contains(t, byGVR, "listener1")

		listener2 := &SelectorListener{gvr: gvr2, nodeName: "node1", id: "listener2", selector: selector}
		lm.AddListener(listener2)

		byNode = lm.GetListenersForNode("node1")
		assert.Len(t, byNode, 2)
		assert.Contains(t, byNode, "listener1")
		assert.Contains(t, byNode, "listener2")

		byGVR = lm.GetListenersForGVR(gvr1)
		assert.Len(t, byGVR, 1)
		assert.Contains(t, byGVR, "listener1")

		byGVR = lm.GetListenersForGVR(gvr2)
		assert.Len(t, byGVR, 1)
		assert.Contains(t, byGVR, "listener2")

		lm.DeleteListener(listener1)

		byNode = lm.GetListenersForNode("node1")
		assert.Len(t, byNode, 1)
		assert.NotContains(t, byNode, "listener1")
		assert.Contains(t, byNode, "listener2")

		assert.Nil(t, lm.GetListenersForGVR(gvr1))

		lm.DeleteListener(listener2)

		assert.Nil(t, lm.GetListenersForNode("node1"))
		assert.Nil(t, lm.GetListenersForGVR(gvr2))
	})
}

func TestEventHandlerObjectConversion(t *testing.T) {
	handler := &CommonResourceEventHandler{
		events: make(chan watch.Event, 10),
	}

	testObj := createTestObject("test-deployment", map[string]string{"app": "nginx"})

	t.Run("Nil object", func(t *testing.T) {
		handler.handleAddEvent(nil)
		select {
		case <-handler.events:
			t.Fatal("Unexpected event received for nil object")
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("Custom runtime.Object", func(t *testing.T) {
		handler.handleAddEvent(struct{ runtime.Object }{})

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Valid unstructured object", func(t *testing.T) {
		freshHandler := &CommonResourceEventHandler{
			events: make(chan watch.Event, 10),
		}

		freshHandler.objToEvent(watch.Added, testObj)

		select {
		case event := <-freshHandler.events:
			assert.Equal(t, watch.Added, event.Type)
			actualObj, ok := event.Object.(*unstructured.Unstructured)
			if assert.True(t, ok, "Expected *unstructured.Unstructured, got %T", event.Object) {
				assert.Equal(t, testObj.GetName(), actualObj.GetName())
				assert.Equal(t, testObj.GetNamespace(), actualObj.GetNamespace())
				assert.Equal(t, testObj.GetLabels(), actualObj.GetLabels())
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("No event received for valid object")
		}
	})
}
