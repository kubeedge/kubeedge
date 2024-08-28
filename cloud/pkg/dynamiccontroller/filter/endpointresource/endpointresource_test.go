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

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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

	// Test case 1: UnstructuredList containing an EndpointSlice object
	endpointSliceObj := &unstructured.Unstructured{}
	endpointSliceObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: resourceEpSliceName})
	unstructuredList := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{*endpointSliceObj},
	}

	result := filter.NeedFilter(unstructuredList)
	assert.True(result)

	// Test case 2: UnstructuredList containing a non EndpointSlice object
	nonEndpointSliceObj := &unstructured.Unstructured{}
	nonEndpointSliceObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "Pod"})
	unstructuredList.Items = []unstructured.Unstructured{*nonEndpointSliceObj}

	result = filter.NeedFilter(unstructuredList)
	assert.False(result)

	// Test case 3: Unstructured object of type EndpointSlice
	result = filter.NeedFilter(endpointSliceObj)
	assert.True(result)

	// Test case 4: Unstructured object of type Endpoints
	endpointsObj := &unstructured.Unstructured{}
	endpointsObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: resourceEpName})

	result = filter.NeedFilter(endpointsObj)
	assert.True(result)

	// Test case 5: Unstructured object of another type
	podObj := &unstructured.Unstructured{}
	podObj.SetGroupVersionKind(schema.GroupVersionKind{Kind: "Pod"})

	result = filter.NeedFilter(podObj)
	assert.False(result)
}
