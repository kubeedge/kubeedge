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

package scope

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewRequestScope(t *testing.T) {
	assert := assert.New(t)

	requestScope := NewRequestScope()

	assert.NotNil(requestScope, "NewRequestScope should return a non-nil value")

	assert.NotNil(requestScope.Namer)
	assert.NotNil(requestScope.Serializer)
	assert.NotNil(requestScope.ParameterCodec)
	assert.Empty(requestScope.StandardSerializers)
	assert.NotNil(requestScope.Creater)
	assert.NotNil(requestScope.Convertor)
	assert.NotNil(requestScope.Defaulter)
	assert.NotNil(requestScope.Typer)
	assert.NotNil(requestScope.UnsafeConvertor)
	assert.NotNil(requestScope.Authorizer)
	assert.NotNil(requestScope.EquivalentResourceMapper)
	assert.NotNil(requestScope.TableConvertor)
	assert.NotNil(requestScope.FieldManager)
	assert.Nil(requestScope.AcceptsGroupVersionDelegate)

	assert.Equal(schema.GroupVersionResource{}, requestScope.Resource)
	assert.Equal(schema.GroupVersionKind{}, requestScope.Kind)
	assert.Equal("", requestScope.Subresource)
	assert.Equal(metav1.SchemeGroupVersion, requestScope.MetaGroupVersion)
	assert.Equal(schema.GroupVersion{}, requestScope.HubGroupVersion)
	assert.Equal(int64(3*1024*1024), requestScope.MaxRequestBodyBytes)
}
