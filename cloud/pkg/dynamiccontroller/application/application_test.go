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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	fakerest "k8s.io/client-go/rest/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/dynamiccontroller/config"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	edgemsg "github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

func TestOptionTo(t *testing.T) {
	testCases := []struct {
		name        string
		optionJSON  string
		expectError bool
	}{
		{
			name:        "valid options",
			optionJSON:  `{"labelSelector":"app=nginx","fieldSelector":"metadata.name=test"}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			optionJSON:  `{invalid json}`,
			expectError: true,
		},
		{
			name:        "empty options",
			optionJSON:  `{}`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := &metaserver.Application{
				Option: []byte(tc.optionJSON),
			}

			var listOptions metav1.ListOptions
			err := app.OptionTo(&listOptions)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.optionJSON != `{}` {
					if tc.optionJSON == `{"labelSelector":"app=nginx","fieldSelector":"metadata.name=test"}` {
						assert.Equal(t, "app=nginx", listOptions.LabelSelector)
						assert.Equal(t, "metadata.name=test", listOptions.FieldSelector)
					}
				}
			}
		})
	}
}

func TestApplicationString(t *testing.T) {
	app := &metaserver.Application{
		ID:       "test-id",
		Nodename: "test-node",
		Key:      "apps/v1/deployments/default/nginx",
		Verb:     metaserver.Get,
		Status:   metaserver.Approved,
	}

	str := app.String()

	assert.Contains(t, str, "test-node")
	assert.Contains(t, str, "apps/v1/deployments/default/nginx")
	assert.Contains(t, str, string(metaserver.Get))
	assert.Contains(t, str, string(metaserver.Approved))
}

func TestApplicationToListener(t *testing.T) {
	tests := []struct {
		name    string
		app     *metaserver.Application
		wantErr bool
	}{
		{
			name: "invalid option",
			app: &metaserver.Application{
				ID:       "test-id",
				Nodename: "test-node",
				Key:      "apps/v1/deployments/default/nginx",
				Verb:     metaserver.Watch,
				Option:   []byte(`invalid json`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := applicationToListener(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("applicationToListener() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReqBodyTo(t *testing.T) {
	obj := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "nginx",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
			},
		},
	}

	objData, err := json.Marshal(obj.Object)
	assert.NoError(t, err)

	app := &metaserver.Application{
		ReqBody: objData,
	}

	var result unstructured.Unstructured
	err = app.ReqBodyTo(&result)
	assert.NoError(t, err)

	assert.Equal(t, obj.GetName(), result.GetName())
	assert.Equal(t, obj.GetNamespace(), result.GetNamespace())

	app.ReqBody = []byte(`{invalid json}`)
	err = app.ReqBodyTo(&result)
	assert.Error(t, err)
}

func TestCenter_passThroughRequest(t *testing.T) {
	failureResp := &http.Response{
		Status:     "500 Internal Error",
		StatusCode: http.StatusInternalServerError,
	}
	successResp := &http.Response{
		Status:     "200 ok",
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("{version: 1.27}")),
	}
	getVersions := func(key, verb string) *fakerest.RESTClient {
		if key == "/version" && verb == "get" {
			return &fakerest.RESTClient{
				Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
					return successResp, nil
				}),
				NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
			}
		}
		return &fakerest.RESTClient{
			Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
				return failureResp, nil
			}),
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		}
	}

	tests := []struct {
		name    string
		app     *metaserver.Application
		want    interface{}
		wantErr bool
	}{
		{
			name: "get version success",
			app: &metaserver.Application{
				Key:  "/version",
				Verb: "get",
			},
			want:    []byte("{version: 1.27}"),
			wantErr: false,
		}, {
			name: "pass through failed",
			app: &metaserver.Application{
				Key:  "/healthz",
				Verb: "get",
			},
			want:    []byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			center := &Center{
				kubeClient: &kubernetes.Clientset{
					DiscoveryClient: discovery.NewDiscoveryClient(getVersions(tt.app.Key, string(tt.app.Verb))),
				},
			}
			got, err := center.passThroughRequest(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("passThroughRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("passThroughRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckNodePermission(t *testing.T) {
	originalEnableAuthorization := config.Config.EnableAuthorization
	config.Config.EnableAuthorization = true
	defer func() {
		config.Config.EnableAuthorization = originalEnableAuthorization
	}()

	tests := []struct {
		name    string
		app     *metaserver.Application
		allowed bool
		err     error
		wantErr bool
	}{
		{
			name: "get version success",
			app: &metaserver.Application{
				Verb:        "get",
				Key:         "/version",
				Subresource: "",
				Nodename:    "test-node",
			},
			allowed: true,
			wantErr: false,
		}, {
			name: "get version with error",
			app: &metaserver.Application{
				Verb:        "get",
				Key:         "/version",
				Subresource: "",
				Nodename:    "test-node",
			},
			allowed: true,
			err:     errors.New("permission denied"),
			wantErr: true,
		}, {
			name: "get configmap failed",
			app: &metaserver.Application{
				Verb:        "get",
				Key:         "/core/v1/configmaps/ns/test-cm",
				Subresource: "",
				Nodename:    "test-node",
			},
			allowed: false,
			wantErr: true,
		},
	}

	fakeClientSet := fake.NewSimpleClientset()
	center := &Center{kubeClient: fakeClientSet}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClientSet.PrependReactor("create", "subjectaccessreviews", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &v1.SubjectAccessReview{Status: v1.SubjectAccessReviewStatus{Allowed: tt.allowed}}, tt.err
			})

			err := center.checkNodePermission(tt.app)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkNodePermission() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResponse(t *testing.T) {
	app := &metaserver.Application{
		ID:       "test-id",
		Nodename: "test-node",
		Key:      "apps/v1/deployments/default/nginx",
		Verb:     metaserver.Get,
	}

	center := &Center{
		messageLayer: &mockMessageLayer{},
	}

	t.Run("success status", func(t *testing.T) {
		center.Response(app, "parent-id", metaserver.Approved, nil, nil)
		assert.Equal(t, metaserver.Approved, app.Status)
		assert.Empty(t, app.Reason)
		assert.Nil(t, app.RespBody)
	})

	t.Run("error status", func(t *testing.T) {
		testErr := errors.New("test error")
		center.Response(app, "parent-id", metaserver.Rejected, testErr, nil)
		assert.Equal(t, metaserver.Rejected, app.Status)
		assert.Equal(t, "test error", app.Reason)
	})

	t.Run("with response content", func(t *testing.T) {
		content := map[string]interface{}{
			"test": "value",
		}
		center.Response(app, "parent-id", metaserver.Approved, nil, content)
		assert.Equal(t, metaserver.Approved, app.Status)
		assert.NotNil(t, app.RespBody)
	})
}

func TestResponseErrorHandling(t *testing.T) {
	center := &Center{
		messageLayer: &mockMessageLayer{},
	}

	app := &metaserver.Application{
		ID:       "test-id",
		Nodename: "test-node",
		Key:      "apps/v1/deployments/default/nginx",
		Verb:     metaserver.Get,
	}

	t.Run("API status error", func(t *testing.T) {
		statusErr := apierrors.NewNotFound(
			schema.GroupResource{Group: "apps", Resource: "deployments"},
			"nginx",
		)

		var temp interface{} = statusErr
		_, isAPIStatus := temp.(apierrors.APIStatus)
		assert.True(t, isAPIStatus)

		center.Response(app, "parent-id", metaserver.Rejected, statusErr, nil)

		assert.Equal(t, metaserver.Rejected, app.Status)
		assert.NotEmpty(t, app.Error)
	})
}

func TestGetWatchDiff(t *testing.T) {
	center := &Center{
		HandlerCenter: &mockHandlerCenter{
			listenersForNode: map[string]map[string]*SelectorListener{
				"node1": {
					"listener1": &SelectorListener{id: "listener1", nodeName: "node1"},
					"listener2": &SelectorListener{id: "listener2", nodeName: "node1"},
				},
			},
		},
	}

	allWatchAppInEdge := map[string]metaserver.Application{
		"listener1": {ID: "listener1", Nodename: "node1"},
		"listener3": {ID: "listener3", Nodename: "node1"},
	}

	added, removed := center.getWatchDiff(allWatchAppInEdge, "node1")

	assert.Len(t, added, 1)
	assert.Equal(t, "listener3", added[0].ID)

	assert.Len(t, removed, 1)
	assert.Equal(t, "listener2", removed[0].id)

	added, removed = center.getWatchDiff(map[string]metaserver.Application{}, "node1")
	assert.Empty(t, added)
	assert.Len(t, removed, 2)

	added, removed = center.getWatchDiff(allWatchAppInEdge, "node2")
	assert.Len(t, added, 2)
	assert.Empty(t, removed)
}
func TestProcessWatchSync(t *testing.T) {
	center := &Center{
		messageLayer: &mockMessageLayer{},
		HandlerCenter: &mockHandlerCenter{
			listenersForNode: make(map[string]map[string]*SelectorListener),
		},
	}

	msg := model.NewMessage("test-id").
		BuildRouter(modules.DynamicControllerModuleName, edgemsg.ResourceGroupName, "test-resource", "test-op")

	err := center.ProcessWatchSync(*msg)
	assert.Error(t, err)
}

func TestProcess(t *testing.T) {
	scheme := runtime.NewScheme()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	fakeClientSet := fake.NewSimpleClientset()

	center := &Center{
		dynamicClient: dynamicClient,
		kubeClient:    fakeClientSet,
		messageLayer:  &mockMessageLayer{},
		HandlerCenter: &mockHandlerCenter{},
	}

	t.Run("message with WatchAppSync resource", func(t *testing.T) {
		msg := model.NewMessage("test-id").
			BuildRouter(modules.DynamicControllerModuleName, message.ResourceGroupName,
				"node1/default/pods/"+metaserver.WatchAppSync, "test-op")

		center.Process(*msg)
	})

	originalEnableAuthorization := config.Config.EnableAuthorization
	config.Config.EnableAuthorization = false
	defer func() {
		config.Config.EnableAuthorization = originalEnableAuthorization
	}()

	t.Run("message with valid application", func(t *testing.T) {
		app := &metaserver.Application{
			ID:       "test-id",
			Nodename: "node1",
			Key:      "apps/v1/deployments/default/nginx",
			Verb:     metaserver.Get,
			Option:   []byte(`{}`),
		}

		appBytes, _ := json.Marshal(app)

		msg := model.NewMessage("test-id").
			BuildRouter(modules.DynamicControllerModuleName, message.ResourceGroupName,
				"node1/default/pods", "test-op").
			FillBody(appBytes)

		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic: %v", r)
			}
		}()

		center.Process(*msg)
	})
}

func TestProcessApplication(t *testing.T) {
	scheme := runtime.NewScheme()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	fakeClientSet := fake.NewSimpleClientset()

	center := &Center{
		dynamicClient: dynamicClient,
		kubeClient:    fakeClientSet,
		messageLayer:  &mockMessageLayer{},
		HandlerCenter: &mockHandlerCenter{},
	}

	deployGVR := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "nginx",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
			},
		},
	}

	_, err := dynamicClient.Resource(deployGVR).Namespace("default").Create(context.TODO(), deployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	t.Run("WATCH verb", func(t *testing.T) {
		originalEnableAuthorization := config.Config.EnableAuthorization
		config.Config.EnableAuthorization = false
		defer func() {
			config.Config.EnableAuthorization = originalEnableAuthorization
		}()

		app := &metaserver.Application{
			ID:       "watch-id",
			Nodename: "node1",
			Key:      "apps/v1/deployments/default",
			Verb:     metaserver.Watch,
			Option:   []byte(`{"labelSelector":"app=nginx"}`),
		}

		resp, err := center.ProcessApplication(app)
		assert.NoError(t, err)
		assert.Nil(t, resp)
	})

	t.Run("UpdateStatus verb", func(t *testing.T) {
		dynamicClient.PrependReactor("update", "deployments/status", func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, deployment.DeepCopy(), nil
		})

		deployment.Object["status"] = map[string]interface{}{
			"replicas":      int64(1),
			"readyReplicas": int64(1),
		}
		deploymentBytes, _ := json.Marshal(deployment)

		app := &metaserver.Application{
			Nodename: "node1",
			Key:      "apps/v1/deployments/default/nginx",
			Verb:     metaserver.UpdateStatus,
			Option:   []byte(`{}`),
			ReqBody:  deploymentBytes,
		}

		resp, err := center.ProcessApplication(app)
		if err != nil {
			t.Logf("UpdateStatus test returned error: %v", err)
		} else {
			assert.NotNil(t, resp)
		}
	})

	t.Run("Patch verb", func(t *testing.T) {
		dynamicClient.PrependReactor("patch", "deployments", func(action clienttesting.Action) (bool, runtime.Object, error) {
			return true, deployment.DeepCopy(), nil
		})

		patchInfo := &metaserver.PatchInfo{
			Name:      "nginx",
			PatchType: "application/merge-patch+json",
			Data:      []byte(`{"spec":{"replicas":2}}`),
			Options:   metav1.PatchOptions{},
		}

		patchInfoBytes, _ := json.Marshal(patchInfo)

		app := &metaserver.Application{
			Nodename: "node1",
			Key:      "apps/v1/deployments/default/nginx",
			Verb:     metaserver.Patch,
			Option:   patchInfoBytes,
		}

		_, err := center.ProcessApplication(app)
		if err != nil {
			t.Logf("Patch test returned error: %v", err)
		}
	})

	t.Run("Invalid verb", func(t *testing.T) {
		app := &metaserver.Application{
			Nodename: "node1",
			Key:      "apps/v1/deployments/default/nginx",
			Verb:     "INVALID",
			Option:   []byte(`{}`),
		}

		_, err := center.ProcessApplication(app)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported Application Verb type")
	})
}

func TestProcessWatchApp(t *testing.T) {
	center := &Center{
		HandlerCenter: &mockHandlerCenter{},
	}

	t.Run("valid watch app", func(t *testing.T) {
		app := &metaserver.Application{
			ID:       "watch-id",
			Nodename: "node1",
			Key:      "apps/v1/deployments/default",
			Verb:     metaserver.Watch,
			Option:   []byte(`{"labelSelector":"app=nginx"}`),
		}

		err := center.processWatchApp(app)
		assert.NoError(t, err)
		assert.Equal(t, metaserver.InProcessing, app.Status)
	})

	t.Run("watch app with invalid option", func(t *testing.T) {
		app := &metaserver.Application{
			ID:       "watch-id",
			Nodename: "node1",
			Key:      "apps/v1/deployments/default",
			Verb:     metaserver.Watch,
			Option:   []byte(`invalid json`),
		}

		err := center.processWatchApp(app)
		assert.Error(t, err)
	})
}

func TestNewApplicationCenter(t *testing.T) {
	center := NewApplicationCenter(nil)

	assert.NotNil(t, center)
}

type mockMessageLayer struct{}

func (m *mockMessageLayer) Send(_ model.Message) error {
	return nil
}

func (m *mockMessageLayer) Receive() (model.Message, error) {
	return model.Message{}, nil
}

func (m *mockMessageLayer) Response(_ model.Message) error {
	return nil
}

func (m *mockMessageLayer) BuildResource(nodeName, namespace, resource, resourceType string) (string, error) {
	return nodeName + "/" + namespace + "/" + resource + "/" + resourceType, nil
}

type mockHandlerCenter struct {
	listenersForNode map[string]map[string]*SelectorListener
}

func (m *mockHandlerCenter) AddListener(listener *SelectorListener) error {
	if m.listenersForNode == nil {
		m.listenersForNode = make(map[string]map[string]*SelectorListener)
	}
	if m.listenersForNode[listener.nodeName] == nil {
		m.listenersForNode[listener.nodeName] = make(map[string]*SelectorListener)
	}
	m.listenersForNode[listener.nodeName][listener.id] = listener
	return nil
}

func (m *mockHandlerCenter) DeleteListener(listener *SelectorListener) {
	if listeners, ok := m.listenersForNode[listener.nodeName]; ok {
		delete(listeners, listener.id)
	}
}

func (m *mockHandlerCenter) ForResource(_ schema.GroupVersionResource) *CommonResourceEventHandler {
	return nil
}

func (m *mockHandlerCenter) GetListenersForNode(nodeName string) map[string]*SelectorListener {
	if listeners, ok := m.listenersForNode[nodeName]; ok {
		return listeners
	}
	return make(map[string]*SelectorListener)
}
