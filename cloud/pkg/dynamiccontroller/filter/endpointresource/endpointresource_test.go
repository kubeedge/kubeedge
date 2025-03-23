/*
Copyright 2024 The KubeEdge Authors.

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

package endpointresource

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/controllermanager/nodegroup"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/filter"
)

const (
	nodeName1 = "node1"
	nodeName2 = "node2"
)

type mockGenericInformer struct {
	mockObj runtime.Object
}

func (m *mockGenericInformer) Informer() cache.SharedIndexInformer {
	return nil
}

func (m *mockGenericInformer) Lister() cache.GenericLister {
	return &mockGenericLister{
		mockObj: m.mockObj,
	}
}

type mockGenericLister struct {
	mockObj runtime.Object
}

func (m *mockGenericLister) List(selector labels.Selector) ([]runtime.Object, error) {
	return []runtime.Object{m.mockObj}, nil
}

func (m *mockGenericLister) Get(name string) (runtime.Object, error) {
	return m.mockObj, nil
}

func (m *mockGenericLister) ByNamespace(namespace string) cache.GenericNamespaceLister {
	return &mockGenericNamespaceLister{
		mockObj: m.mockObj,
	}
}

type mockGenericNamespaceLister struct {
	mockObj runtime.Object
}

func (m *mockGenericNamespaceLister) List(selector labels.Selector) ([]runtime.Object, error) {
	return []runtime.Object{m.mockObj}, nil
}

func (m *mockGenericNamespaceLister) Get(name string) (runtime.Object, error) {
	return m.mockObj, nil
}

func setupPatches() *gomonkey.Patches {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			Annotations: map[string]string{
				nodegroup.ServiceTopologyAnnotation: nodegroup.ServiceTopologyRangeNodegroup,
			},
		},
	}

	mockInformer := &mockGenericInformer{
		mockObj: svc,
	}

	patches := gomonkey.NewPatches()

	patches.ApplyFuncReturn(filter.GetDynamicResourceInformer, mockInformer)

	patches.ApplyFunc(meta.Accessor, func(obj interface{}) (metav1.Object, error) {
		return svc.ObjectMeta.DeepCopy(), nil
	})

	patches.ApplyFunc(filter.IsBelongToSameGroup, func(node1, node2 string) bool {
		return node2 == nodeName1
	})

	return patches
}

func TestNewEndpointsliceFilter(t *testing.T) {
	assert := assert.New(t)
	fi := newEndpointsliceFilter()
	assert.NotNil(fi)
}

func TestName(t *testing.T) {
	assert := assert.New(t)
	fi := newEndpointsliceFilter()
	name := fi.Name()
	assert.Equal(name, filterName)
}

func TestNeedFilter(t *testing.T) {
	assert := assert.New(t)
	filter := newEndpointsliceFilter()

	endpointSliceObj := &unstructured.Unstructured{}
	endpointSliceObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: resourceEpSliceName})
	unstructuredList := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{*endpointSliceObj},
	}

	result := filter.NeedFilter(unstructuredList)
	assert.True(result, "Should need filtering for UnstructuredList with EndpointSlice")

	nonEndpointSliceObj := &unstructured.Unstructured{}
	nonEndpointSliceObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "Pod"})
	unstructuredList.Items = []unstructured.Unstructured{*nonEndpointSliceObj}

	result = filter.NeedFilter(unstructuredList)
	assert.False(result, "Should not need filtering for UnstructuredList with non-EndpointSlice")

	result = filter.NeedFilter(endpointSliceObj)
	assert.True(result, "Should need filtering for EndpointSlice")

	endpointsObj := &unstructured.Unstructured{}
	endpointsObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: resourceEpName})

	result = filter.NeedFilter(endpointsObj)
	assert.True(result, "Should need filtering for Endpoints")

	podObj := &unstructured.Unstructured{}
	podObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "Pod"})

	result = filter.NeedFilter(podObj)
	assert.False(result, "Should not need filtering for non-EndpointSlice/Endpoints")

	result = filter.NeedFilter("string")
	assert.False(result, "Should not need filtering for non-Unstructured objects")

	emptyList := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{},
	}
	result = filter.NeedFilter(emptyList)
	assert.False(result, "Should not need filtering for empty list")

	result = filter.NeedFilter(nil)
	assert.False(result, "Should not need filtering for nil")
}

func TestFilterEndpointSlice(t *testing.T) {
	patches := setupPatches()
	defer patches.Reset()

	n1 := nodeName1
	n2 := nodeName2
	epSlice := &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-epslice",
			Namespace: "default",
			Labels: map[string]string{
				discovery.LabelServiceName: "test-service",
			},
		},
		Endpoints: []discovery.Endpoint{
			{
				Addresses: []string{"192.168.1.1"},
				NodeName:  &n1,
			},
			{
				Addresses: []string{"192.168.1.2"},
				NodeName:  &n2,
			},
		},
	}

	unstructEpSlice, err := runtime.DefaultUnstructuredConverter.ToUnstructured(epSlice)
	assert.NoError(t, err)
	epSliceUnstructured := &unstructured.Unstructured{Object: unstructEpSlice}

	filterEndpointSlice("target-node", epSliceUnstructured)

	var result discovery.EndpointSlice
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(epSliceUnstructured.Object, &result)
	assert.NoError(t, err)

	assert.Len(t, result.Endpoints, 1, "Should only include endpoints from node1")
	assert.Equal(t, nodeName1, *result.Endpoints[0].NodeName)
}

func TestFilterEndpoints(t *testing.T) {
	patches := setupPatches()
	defer patches.Reset()

	n1 := nodeName1
	n2 := nodeName2
	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "192.168.1.1",
						NodeName: &n1,
					},
					{
						IP:       "192.168.1.2",
						NodeName: &n2,
					},
				},
				NotReadyAddresses: []v1.EndpointAddress{
					{
						IP:       "192.168.1.3",
						NodeName: &n1,
					},
					{
						IP:       "192.168.1.4",
						NodeName: &n2,
					},
				},
			},
		},
	}

	unstructEp, err := runtime.DefaultUnstructuredConverter.ToUnstructured(endpoints)
	assert.NoError(t, err)
	epUnstructured := &unstructured.Unstructured{Object: unstructEp}

	filterEndpoints("target-node", epUnstructured)

	var result v1.Endpoints
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(epUnstructured.Object, &result)
	assert.NoError(t, err)

	assert.Len(t, result.Subsets[0].Addresses, 1, "Should only include addresses from node1")
	assert.Equal(t, nodeName1, *result.Subsets[0].Addresses[0].NodeName)

	assert.Len(t, result.Subsets[0].NotReadyAddresses, 1, "Should only include not-ready addresses from node1")
	assert.Equal(t, nodeName1, *result.Subsets[0].NotReadyAddresses[0].NodeName)
}

func TestFilterEndpointsAddress(t *testing.T) {
	patches := setupPatches()
	defer patches.Reset()

	n1 := nodeName1
	n2 := nodeName2
	addresses := []v1.EndpointAddress{
		{
			IP:       "192.168.1.1",
			NodeName: &n1,
		},
		{
			IP:       "192.168.1.2",
			NodeName: &n2,
		},
		{
			IP:       "192.168.1.3",
			NodeName: nil,
		},
	}

	result := filterEndpointsAddress("target-node", addresses)

	assert.Len(t, result, 1, "Should only include addresses from node1")
	assert.Equal(t, "192.168.1.1", result[0].IP)
	assert.Equal(t, nodeName1, *result[0].NodeName)
}

func TestFilterResource(t *testing.T) {
	patches := setupPatches()
	defer patches.Reset()

	filter := newEndpointsliceFilter()

	n1 := nodeName1
	n2 := nodeName2
	epSlice := &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-epslice",
			Namespace: "default",
			Labels: map[string]string{
				discovery.LabelServiceName: "test-service",
			},
		},
		Endpoints: []discovery.Endpoint{
			{
				Addresses: []string{"192.168.1.1"},
				NodeName:  &n1,
			},
			{
				Addresses: []string{"192.168.1.2"},
				NodeName:  &n2,
			},
		},
	}

	unstructEpSlice, err := runtime.DefaultUnstructuredConverter.ToUnstructured(epSlice)
	assert.NoError(t, err)
	epSliceUnstructured := &unstructured.Unstructured{Object: unstructEpSlice}
	epSliceUnstructured.SetGroupVersionKind(schema.GroupVersionKind{Kind: resourceEpSliceName})

	filter.FilterResource("target-node", epSliceUnstructured)

	var resultEpSlice discovery.EndpointSlice
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(epSliceUnstructured.Object, &resultEpSlice)
	assert.NoError(t, err)
	assert.Len(t, resultEpSlice.Endpoints, 1, "Should only include endpoints from node1")
	assert.Equal(t, nodeName1, *resultEpSlice.Endpoints[0].NodeName)

	n1 = nodeName1
	n2 = nodeName2
	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "192.168.1.1",
						NodeName: &n1,
					},
					{
						IP:       "192.168.1.2",
						NodeName: &n2,
					},
				},
			},
		},
	}

	unstructEp, err := runtime.DefaultUnstructuredConverter.ToUnstructured(endpoints)
	assert.NoError(t, err)
	epUnstructured := &unstructured.Unstructured{Object: unstructEp}
	epUnstructured.SetGroupVersionKind(schema.GroupVersionKind{Kind: resourceEpName})

	filter.FilterResource("target-node", epUnstructured)

	var resultEp v1.Endpoints
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(epUnstructured.Object, &resultEp)
	assert.NoError(t, err)
	assert.Len(t, resultEp.Subsets[0].Addresses, 1, "Should only include addresses from node1")
	assert.Equal(t, nodeName1, *resultEp.Subsets[0].Addresses[0].NodeName)

	podObj := &unstructured.Unstructured{}
	podObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "Pod"})

	filter.FilterResource("target-node", podObj)
}

func TestFilterEndpointsNoTopology(t *testing.T) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
	}

	mockInformer := &mockGenericInformer{
		mockObj: svc,
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFuncReturn(filter.GetDynamicResourceInformer, mockInformer)

	patches.ApplyFunc(meta.Accessor, func(obj interface{}) (metav1.Object, error) {
		return svc.ObjectMeta.DeepCopy(), nil
	})

	n1 := nodeName1
	endpoints := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "192.168.1.1",
						NodeName: &n1,
					},
				},
			},
		},
	}

	unstructEp, err := runtime.DefaultUnstructuredConverter.ToUnstructured(endpoints)
	assert.NoError(t, err)
	epUnstructured := &unstructured.Unstructured{Object: unstructEp}

	origAddresses := endpoints.Subsets[0].Addresses

	filterEndpoints("target-node", epUnstructured)

	var result v1.Endpoints
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(epUnstructured.Object, &result)
	assert.NoError(t, err)

	assert.Equal(t, len(origAddresses), len(result.Subsets[0].Addresses),
		"Addresses should not be filtered when no topology annotation is present")
}

func TestFilterEndpointSliceConversionError(t *testing.T) {
	invalid := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       resourceEpSliceName,
			"apiVersion": "discovery.k8s.io/v1",
			"metadata": map[string]interface{}{
				"name":      "test-epslice",
				"namespace": "default",
			},
		},
	}

	filterEndpointSlice("target-node", invalid)
}
