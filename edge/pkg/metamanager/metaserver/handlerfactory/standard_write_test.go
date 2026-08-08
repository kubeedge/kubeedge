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

package handlerfactory

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
)

func TestFactoryCreateImplementation_Success(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})

	// DecodeAndConvert should return a typed object based on the input body
	mockObj := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "nginx-deployment"},
	}
	patches.ApplyFunc(storage.DecodeAndConvert, func(_ []byte, _ string) (runtime.Object, error) {
		return mockObj, nil
	})

	// Patch the storage Create method to return the object
	patches.ApplyMethod((*storage.REST)(nil), "Create",
		func(_ *storage.REST, _ context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
			return mockObj, nil
		})

	factory := NewFactory()

	reqInfo := &request.RequestInfo{
		APIGroup:   "apps",
		APIVersion: "v1",
		Resource:   "deployments",
	}
	handler := factory.Create(reqInfo)

	deployData := `{"kind":"Deployment","metadata":{"name":"nginx-deployment"}}`
	req := httptest.NewRequest("POST", "/apis/apps/v1/namespaces/default/deployments", strings.NewReader(deployData))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "nginx-deployment")
}

func TestFactoryCreateImplementation_StorageError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})

	mockObj := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "nginx-deployment"},
	}
	patches.ApplyFunc(storage.DecodeAndConvert, func(_ []byte, _ string) (runtime.Object, error) {
		return mockObj, nil
	})

	patches.ApplyMethod((*storage.REST)(nil), "Create",
		func(_ *storage.REST, _ context.Context, _ runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
			return nil, errors.New("create error")
		})

	factory := NewFactory()

	reqInfo := &request.RequestInfo{
		APIGroup:   "apps",
		APIVersion: "v1",
		Resource:   "deployments",
	}
	handler := factory.Create(reqInfo)

	deployData := `{"kind":"Deployment","metadata":{"name":"nginx-deployment"}}`
	req := httptest.NewRequest("POST", "/apis/apps/v1/namespaces/default/deployments", strings.NewReader(deployData))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "create error")
}

func TestFactoryUpdate_Implementation_Success(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})

	mockObj := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "nginx-deployment"},
	}
	patches.ApplyFunc(storage.DecodeAndConvert, func(_ []byte, _ string) (runtime.Object, error) {
		return mockObj, nil
	})

	patches.ApplyMethod((*storage.REST)(nil), "Update",
		func(_ *storage.REST, _ context.Context, _ string, _ rest.UpdatedObjectInfo, _ rest.ValidateObjectFunc, _ rest.ValidateObjectUpdateFunc, _ bool, _ *metav1.UpdateOptions) (runtime.Object, bool, error) {
			return mockObj, false, nil
		})

	factory := NewFactory()
	reqInfo := &request.RequestInfo{
		APIGroup:   "apps",
		APIVersion: "v1",
		Resource:   "deployments",
	}

	handler := factory.Update(reqInfo)

	deployData := `{"kind":"Deployment","metadata":{"name":"nginx-deployment"},"spec":{"replicas":3}}`
	req := httptest.NewRequest("PUT", "/apis/apps/v1/namespaces/default/deployments/nginx-deployment", strings.NewReader(deployData))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "nginx-deployment")
}

func TestFactoryUpdate_Implementation_StorageError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})

	mockObj := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "nginx-deployment"},
	}
	patches.ApplyFunc(storage.DecodeAndConvert, func(_ []byte, _ string) (runtime.Object, error) {
		return mockObj, nil
	})

	patches.ApplyMethod((*storage.REST)(nil), "Update",
		func(_ *storage.REST, _ context.Context, _ string, _ rest.UpdatedObjectInfo, _ rest.ValidateObjectFunc, _ rest.ValidateObjectUpdateFunc, _ bool, _ *metav1.UpdateOptions) (runtime.Object, bool, error) {
			return nil, false, errors.New("update error")
		})

	factory := NewFactory()
	reqInfo := &request.RequestInfo{
		APIGroup:   "apps",
		APIVersion: "v1",
		Resource:   "deployments",
	}

	handler := factory.Update(reqInfo)

	deployData := `{"kind":"Deployment","metadata":{"name":"nginx-deployment"},"spec":{"replicas":3}}`
	req := httptest.NewRequest("PUT", "/apis/apps/v1/namespaces/default/deployments/nginx-deployment", strings.NewReader(deployData))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "update error")
}
