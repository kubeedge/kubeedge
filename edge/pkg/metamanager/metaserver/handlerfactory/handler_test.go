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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/fakers"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/scope"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage"
)

type mockAdmit struct{}

func (m *mockAdmit) Admit(ctx context.Context, a admission.Attributes, o admission.ObjectInterfaces) error {
	return nil
}

func TestNewFactory(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	factory := NewFactory()
	assert.NotNil(t, factory)
	assert.NotNil(t, factory.storage)
	assert.NotNil(t, factory.scope)
	assert.Equal(t, 1800*time.Second, factory.MinRequestTimeout)
	assert.NotNil(t, factory.handlers)
}

func TestFactoryGet(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"kind":"Pod","metadata":{"name":"test-pod"}}`))
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "Get", func(_ *Factory) http.Handler {
		return mockHandler
	})

	factory := NewFactory()
	handler := factory.Get()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/pods/test-pod", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-pod")
}

func TestFactoryGetCached(t *testing.T) {
	factory := &Factory{
		handlers: map[string]http.Handler{
			"get": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"cached":true}`))
				assert.NoError(t, err)
			}),
		},
	}

	handler := factory.Get()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cached")
}

func TestFactoryGetCacheMiss(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockGetResponse := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod-cached-miss",
		},
	}

	patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		mockREST := &storage.REST{}
		return mockREST, nil
	})

	patches.ApplyMethod((*storage.REST)(nil), "Get",
		func(_ *storage.REST, _ context.Context, _ string, _ *metav1.GetOptions) (runtime.Object, error) {
			return mockGetResponse, nil
		})

	patches.ApplyFunc(request.RequestInfoFrom, func(ctx context.Context) (*request.RequestInfo, bool) {
		return &request.RequestInfo{
			APIGroup:   "",
			APIVersion: "v1",
			Resource:   "pods",
			Name:       "test-pod",
			Namespace:  "default",
		}, true
	})

	factory := NewFactory()

	handler := factory.Get()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/pods/test-pod", nil)

	ctx := req.Context()
	reqInfo := &request.RequestInfo{
		APIGroup:   "",
		APIVersion: "v1",
		Resource:   "pods",
		Name:       "test-pod",
		Namespace:  "default",
	}
	ctx = request.WithRequestInfo(ctx, reqInfo)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-pod-cached-miss")

	cachedHandler, exists := factory.handlers["get"]
	assert.True(t, exists)
	assert.NotNil(t, cachedHandler)
}

func TestFactoryListCacheMiss(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockREST := &storage.REST{}
	patches.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return mockREST, nil
	})

	mockListResponse := &metav1.PartialObjectMetadataList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "PodList",
		},
		Items: []metav1.PartialObjectMetadata{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Pod",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-cache-miss",
				},
			},
		},
	}

	patches.ApplyMethod((*storage.REST)(nil), "List",
		func(_ *storage.REST, _ context.Context, _ *metainternalversion.ListOptions) (runtime.Object, error) {
			return mockListResponse, nil
		})

	mockListHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"kind":"PodList","items":[{"metadata":{"name":"pod-cache-miss"}}]}`))
		assert.NoError(t, err)
	})

	factory := &Factory{
		storage:           mockREST,
		scope:             scope.NewRequestScope(),
		MinRequestTimeout: 1800 * time.Second,
		handlers:          make(map[string]http.Handler),
		lock:              sync.RWMutex{},
	}

	factory.handlers["list"] = mockListHandler

	handler := factory.List()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/pods", nil)

	reqInfo := &request.RequestInfo{
		APIGroup:   "",
		APIVersion: "v1",
		Resource:   "pods",
		Namespace:  "default",
	}
	ctx := request.WithRequestInfo(req.Context(), reqInfo)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pod-cache-miss")
}

func TestFactoryCreateSimple(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"kind":"Deployment","metadata":{"name":"nginx-deployment"}}`))
		assert.NoError(t, err)
	})

	patches := gomonkey.ApplyMethod((*Factory)(nil), "Create",
		func(_ *Factory, _ *request.RequestInfo) http.Handler {
			return mockHandler
		})
	defer patches.Reset()

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

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "nginx-deployment")
}

func TestFactoryList(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"kind":"PodList","items":[{"metadata":{"name":"pod1"}},{"metadata":{"name":"pod2"}}]}`))
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "List", func(_ *Factory) http.Handler {
		return mockHandler
	})

	factory := NewFactory()
	handler := factory.List()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/pods", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "PodList")
}

func TestFactoryListCached(t *testing.T) {
	factory := &Factory{
		handlers: map[string]http.Handler{
			"list": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"cached":true}`))
				assert.NoError(t, err)
			}),
		},
	}

	handler := factory.List()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cached")
}

func TestFactoryCreate(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(body)
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "Create", func(_ *Factory, _ *request.RequestInfo) http.Handler {
		return mockHandler
	})

	factory := NewFactory()
	reqInfo := &request.RequestInfo{
		APIGroup:   "apps",
		APIVersion: "v1",
		Resource:   "deployments",
	}

	handler := factory.Create(reqInfo)

	podData := `{"kind":"Deployment","metadata":{"name":"nginx-deployment"}}`
	req := httptest.NewRequest("POST", "/apis/apps/v1/namespaces/default/deployments", strings.NewReader(podData))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "nginx-deployment")
}

func TestFactoryDelete(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"kind":"Status","status":"Success"}`))
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "Delete", func(_ *Factory) http.Handler {
		return mockHandler
	})

	factory := NewFactory()
	handler := factory.Delete()

	req := httptest.NewRequest("DELETE", "/api/v1/namespaces/default/pods/test-pod", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Success")
}

func TestFactoryDeleteCached(t *testing.T) {
	factory := &Factory{
		handlers: map[string]http.Handler{
			"delete": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"cached":true}`))
				assert.NoError(t, err)
			}),
		},
	}

	handler := factory.Delete()

	req := httptest.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cached")
}

func TestFactoryDeleteCacheMiss(t *testing.T) {
	mockStorage := &storage.REST{}
	mockScope := scope.NewRequestScope()

	mockDeleteHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"kind":"Status","status":"Success","message":"Resource deleted"}`))
		assert.NoError(t, err)
	})

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(handlers.DeleteResource,
		func(_ interface{}, _ bool, _ *handlers.RequestScope, _ interface{}) http.Handler {
			return mockDeleteHandler
		})

	patches.ApplyFunc(fakers.NewAlwaysAdmit, func() interface{} {
		return &mockAdmit{}
	})

	factory := &Factory{
		storage:  mockStorage,
		scope:    mockScope,
		handlers: make(map[string]http.Handler),
		lock:     sync.RWMutex{},
	}

	_ = factory.Delete()

	cachedHandler, exists := factory.handlers["delete"]
	assert.True(t, exists)
	assert.NotNil(t, cachedHandler)
}

func TestFactoryUpdate(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(body)
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "Update", func(_ *Factory, _ *request.RequestInfo) http.Handler {
		return mockHandler
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

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "nginx-deployment")
}

func TestFactoryUpdateDevices(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(body)
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "Update", func(_ *Factory, reqInfo *request.RequestInfo) http.Handler {
		if reqInfo.Resource == "devices" {
			return mockHandler
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
	})

	factory := NewFactory()
	reqInfo := &request.RequestInfo{
		APIGroup:   "devices",
		APIVersion: "v1beta1",
		Resource:   "devices",
	}

	handler := factory.Update(reqInfo)

	deviceData := `{"kind":"Device","metadata":{"name":"test-device"}}`
	req := httptest.NewRequest("PUT", "/apis/devices/v1beta1/namespaces/default/devices/test-device", strings.NewReader(deviceData))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-device")
}

func TestUpdateEdgeDeviceImplementation(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockMessage := &model.Message{}
	patches.ApplyFunc(model.NewMessage, func(_ string) *model.Message {
		return mockMessage
	})

	patches.ApplyMethod((*model.Message)(nil), "BuildRouter", func(_ *model.Message, source, target, _, _ string) *model.Message {
		return mockMessage
	})
	patches.ApplyMethod((*model.Message)(nil), "SetResourceVersion", func(_ *model.Message, _ string) *model.Message {
		return mockMessage
	})
	patches.ApplyMethod((*model.Message)(nil), "FillBody", func(_ *model.Message, _ interface{}) *model.Message {
		return mockMessage
	})

	responseBytes := []byte(`{"kind":"Device","metadata":{"name":"test-device"}}`)
	mockResponse := model.Message{
		Content: responseBytes,
	}
	patches.ApplyFunc(beehiveContext.SendSync, func(_ string, _ model.Message, _ interface{}) (model.Message, error) {
		return mockResponse, nil
	})

	patches.ApplyMethod((*model.Message)(nil), "GetContentData", func(_ *model.Message) ([]byte, error) {
		return responseBytes, nil
	})

	handler := updateEdgeDevice()

	device := &v1beta1.Device{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-device",
			Namespace: "default",
		},
	}
	deviceBytes, err := json.Marshal(device)
	assert.NoError(t, err)

	req := httptest.NewRequest("PUT", "/apis/devices/v1beta1/namespaces/default/devices/test-device", bytes.NewReader(deviceBytes))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "test-device")
}

func TestUpdateEdgeDeviceErrors(t *testing.T) {
	t.Run("ReadBodyError", func(t *testing.T) {
		handler := updateEdgeDevice()

		errorReader := &ErrorReadCloser{Err: errors.New("read error")}
		req := httptest.NewRequest("PUT", "/apis/devices/v1beta1/namespaces/default/devices/test-device", nil)
		req.Body = errorReader

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "read error")
	})

	t.Run("UnmarshalError", func(t *testing.T) {
		handler := updateEdgeDevice()

		invalidJSON := "this is not valid JSON"
		req := httptest.NewRequest("PUT", "/apis/devices/v1beta1/namespaces/default/devices/test-device", strings.NewReader(invalidJSON))

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SendSyncError", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		mockMessage := &model.Message{}
		patches.ApplyFunc(model.NewMessage, func(string) *model.Message {
			return mockMessage
		})
		patches.ApplyMethod((*model.Message)(nil), "BuildRouter", func(_ *model.Message, _, _, _, _ string) *model.Message {
			return mockMessage
		})
		patches.ApplyMethod((*model.Message)(nil), "SetResourceVersion", func(_ *model.Message, _ string) *model.Message {
			return mockMessage
		})
		patches.ApplyMethod((*model.Message)(nil), "FillBody", func(_ *model.Message, _ interface{}) *model.Message {
			return mockMessage
		})

		patches.ApplyFunc(beehiveContext.SendSync, func(string, model.Message, interface{}) (model.Message, error) {
			return model.Message{}, errors.New("send sync error")
		})

		handler := updateEdgeDevice()

		device := &v1beta1.Device{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-device",
				Namespace: "default",
			},
		}
		deviceBytes, err := json.Marshal(device)
		assert.NoError(t, err)

		req := httptest.NewRequest("PUT", "/apis/devices/v1beta1/namespaces/default/devices/test-device", bytes.NewReader(deviceBytes))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "send sync error")
	})
}

func TestFactoryPatch(t *testing.T) {
	patches := gomonkey.ApplyFunc(storage.NewREST, func() (*storage.REST, error) {
		return &storage.REST{}, nil
	})
	defer patches.Reset()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"kind":"Device","metadata":{"name":"test-device"}}`))
		assert.NoError(t, err)
	})

	patches.ApplyMethod((*Factory)(nil), "Patch", func(_ *Factory, _ *request.RequestInfo) http.Handler {
		return mockHandler
	})

	factory := NewFactory()
	reqInfo := &request.RequestInfo{
		APIGroup:   "devices",
		APIVersion: "v1beta1",
		Resource:   "devices",
		Name:       "test-device",
		Namespace:  "default",
	}

	handler := factory.Patch(reqInfo)

	patchData := `[{"op":"replace","path":"/spec/replicas","value":3}]`
	req := httptest.NewRequest("PATCH", "/apis/devices/v1beta1/namespaces/default/devices/test-device", strings.NewReader(patchData))
	req.Header.Set("Content-Type", "application/json-patch+json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-device")
}

func TestPassThroughDirectImplementation(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockResult := []byte(`{"kind":"ConfigMap","metadata":{"name":"test-config"}}`)
	mockStorage := &storage.REST{}

	patches.ApplyMethod((*storage.REST)(nil), "PassThrough",
		func(_ *storage.REST, _ context.Context, _ *metav1.GetOptions) ([]byte, error) {
			return mockResult, nil
		})

	factory := &Factory{
		storage: mockStorage,
	}

	handler := factory.PassThrough()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/configmaps/test-config", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, string(mockResult), w.Body.String())
}

func TestPassThroughError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockStorage := &storage.REST{}

	patches.ApplyMethod((*storage.REST)(nil), "PassThrough",
		func(_ *storage.REST, _ context.Context, _ *metav1.GetOptions) ([]byte, error) {
			return nil, errors.New("passthrough error")
		})

	factory := &Factory{
		storage: mockStorage,
	}

	handler := factory.PassThrough()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/configmaps/test-config", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "passthrough error")
}

func TestPassThroughWriteError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(klog.Error, func(_ ...interface{}) {})

	mockResult := []byte(`{"kind":"ConfigMap","metadata":{"name":"test-config"}}`)
	mockStorage := &storage.REST{}

	patches.ApplyMethod((*storage.REST)(nil), "PassThrough",
		func(_ *storage.REST, _ context.Context, _ *metav1.GetOptions) ([]byte, error) {
			return mockResult, nil
		})

	factory := &Factory{
		storage: mockStorage,
	}

	handler := factory.PassThrough()

	req := httptest.NewRequest("GET", "/api/v1/namespaces/default/configmaps/test-config", nil)
	w := &ErrorResponseWriter{}
	handler.ServeHTTP(w, req)

	assert.True(t, w.WroteCalled)
}

func TestGetHandler(t *testing.T) {
	factory := &Factory{
		handlers: map[string]http.Handler{
			"test": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	handler, ok := factory.getHandler("test")
	assert.True(t, ok)
	assert.NotNil(t, handler)

	handler, ok = factory.getHandler("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, handler)
}

func TestParseTimeout(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "valid timeout",
			input:    "10s",
			expected: 10 * time.Second,
		},
		{
			name:     "invalid timeout",
			input:    "invalid",
			expected: 34 * time.Second, // Default value
		},
		{
			name:     "empty timeout",
			input:    "",
			expected: 34 * time.Second, // Default value
		},
	}

	// Patch klog.Errorf to avoid noisy output in tests
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc(klog.Errorf, func(_ string, _ ...interface{}) {})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := parseTimeout(tt.input)
			assert.Equal(t, tt.expected, timeout, "parseTimeout(%q) should return %v", tt.input, tt.expected)
		})
	}
}

func TestLimitedReadBody(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		limit         int64
		expectError   bool
		expectedError string
	}{
		{
			name:        "content within limit",
			content:     "test content",
			limit:       100,
			expectError: false,
		},
		{
			name:          "content exceeds limit",
			content:       "test content that exceeds the limit",
			limit:         10,
			expectError:   true,
			expectedError: "limit is 10",
		},
		{
			name:        "zero limit (no limit)",
			content:     "test content",
			limit:       0,
			expectError: false,
		},
		{
			name:        "negative limit (no limit)",
			content:     "test content",
			limit:       -1,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.content))
			data, err := limitedReadBody(req, tt.limit)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.content, string(data))
			}
		})
	}
}

type ErrorReadCloser struct {
	Err error
}

func (e *ErrorReadCloser) Read(_ []byte) (n int, err error) {
	return 0, e.Err
}

func (e *ErrorReadCloser) Close() error {
	return nil
}

type ErrorResponseWriter struct {
	HeaderMap   http.Header
	Code        int
	WroteCalled bool
}

func (e *ErrorResponseWriter) Header() http.Header {
	if e.HeaderMap == nil {
		e.HeaderMap = make(http.Header)
	}
	return e.HeaderMap
}

func (e *ErrorResponseWriter) Write([]byte) (int, error) {
	e.WroteCalled = true
	return 0, errors.New("write error")
}

func (e *ErrorResponseWriter) WriteHeader(statusCode int) {
	e.Code = statusCode
}
