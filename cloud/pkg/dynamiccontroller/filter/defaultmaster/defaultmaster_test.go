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

package defaultmaster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	discovery "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewDefaultMasterFilter(t *testing.T) {
	assert := assert.New(t)

	filter := newDefaultMasterFilter()

	assert.NotNil(filter)
	assert.Equal(defaultMetaServerIP, filter.hostIP)
	assert.Equal(int32(defaultMetaServerPort), filter.port)
}

func TestNeedFilter(t *testing.T) {
	assert := assert.New(t)
	filter := newDefaultMasterFilter()

	// Case 1: UnstructuredList with EndpointSlice objects
	objList := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]interface{}{
					"kind": "EndpointSlice",
				},
			},
		},
	}
	assert.True(filter.NeedFilter(objList))

	// Case 2: Unstructured with matching EndpointSlice object
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":     "EndpointSlice",
			"metadata": map[string]interface{}{"name": defaultEndpointSliceName, "namespace": defaultEndpointSliceNameSpace},
		},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "EndpointSlice"})
	assert.True(filter.NeedFilter(obj))

	// Case 3: Unstructured with non-matching EndpointSlice object (wrong name)
	nonMatchingObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":     "EndpointSlice",
			"metadata": map[string]interface{}{"name": "nonMatchingName", "namespace": defaultEndpointSliceNameSpace},
		},
	}
	nonMatchingObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "EndpointSlice"})
	assert.False(filter.NeedFilter(nonMatchingObj))

	// Case 4: Unstructured with a non-EndpointSlice object
	nonEndpointSliceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Pod",
		},
	}
	assert.False(filter.NeedFilter(nonEndpointSliceObj))

	// Case 5: Invalid input (nil object)
	assert.False(filter.NeedFilter(nil))
}

func TestFilterResource(t *testing.T) {
	assert := assert.New(t)
	filter := newDefaultMasterFilter()

	// EndpointSlice object with multiple endpoints
	endpointSlice := &discovery.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultEndpointSliceName,
			Namespace: defaultEndpointSliceNameSpace,
		},
		Endpoints: []discovery.Endpoint{
			{
				Addresses: []string{"192.168.1.1", "192.168.1.2"},
			},
		},
		Ports: []discovery.EndpointPort{
			{
				Name: func(s string) *string { return &s }(defaultEndpointSlicePortName),
				Port: func(i int32) *int32 { return &i }(8080),
			},
		},
	}

	// Convert the EndpointSlice to unstructured format
	unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(endpointSlice)
	assert.NoError(err, "Failed to convert EndpointSlice to unstructured")
	unstructuredObj := &unstructured.Unstructured{Object: unstr}
	unstructuredObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "EndpointSlice"})

	filter.FilterResource("", unstructuredObj)

	var filteredEndpointSlice discovery.EndpointSlice
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &filteredEndpointSlice)
	assert.NoError(err, "Failed to convert unstructured back to EndpointSlice")

	assert.Len(filteredEndpointSlice.Endpoints, 1)
	assert.Len(filteredEndpointSlice.Endpoints[0].Addresses, 1)
	assert.Equal(defaultMetaServerIP, filteredEndpointSlice.Endpoints[0].Addresses[0])
	assert.Equal(int32(defaultMetaServerPort), *filteredEndpointSlice.Ports[0].Port)
	assert.Equal(defaultEndpointSlicePortName, *filteredEndpointSlice.Ports[0].Name)
}

func TestFilterResourceOptionalEndpointPortFields(t *testing.T) {
	tests := []struct {
		name      string
		ports     []discovery.EndpointPort
		wantPorts []*int32
	}{
		{
			name: "unnamed port is ignored",
			ports: []discovery.EndpointPort{{
				Port: func(i int32) *int32 { return &i }(443),
			}},
			wantPorts: []*int32{func(i int32) *int32 { return &i }(443)},
		},
		{
			name: "https port without number is rewritten",
			ports: []discovery.EndpointPort{{
				Name: func(s string) *string { return &s }(defaultEndpointSlicePortName),
			}},
			wantPorts: []*int32{func(i int32) *int32 { return &i }(defaultMetaServerPort)},
		},
		{
			name: "unnamed port does not block https port rewrite",
			ports: []discovery.EndpointPort{
				{Port: func(i int32) *int32 { return &i }(443)},
				{Name: func(s string) *string { return &s }(defaultEndpointSlicePortName)},
			},
			wantPorts: []*int32{
				func(i int32) *int32 { return &i }(443),
				func(i int32) *int32 { return &i }(defaultMetaServerPort),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpointSlice := &discovery.EndpointSlice{
				Endpoints: []discovery.Endpoint{{Addresses: []string{"192.168.1.1"}}},
				Ports:     tt.ports,
			}
			unstr, err := runtime.DefaultUnstructuredConverter.ToUnstructured(endpointSlice)
			assert.NoError(t, err)
			unstructuredObj := &unstructured.Unstructured{Object: unstr}

			newDefaultMasterFilter().FilterResource("", unstructuredObj)

			var filteredEndpointSlice discovery.EndpointSlice
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &filteredEndpointSlice)
			assert.NoError(t, err)
			assert.Equal(t, defaultMetaServerIP, filteredEndpointSlice.Endpoints[0].Addresses[0])
			for i := range tt.wantPorts {
				assert.Equal(t, tt.wantPorts[i], filteredEndpointSlice.Ports[i].Port)
			}
		})
	}
}
