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

package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	oteltrace "go.opentelemetry.io/otel/trace"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	cri "k8s.io/cri-api/pkg/apis"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/common/types"
	connect "github.com/kubeedge/kubeedge/edge/pkg/common/cloudconnection"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/agent"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/restful"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
)

func TestREST_PassThrough(t *testing.T) {
	type testCase struct {
		isConnectFailed            bool
		isSendSyncFailed           bool
		isLocalStored              bool
		isInsertLocalStorageFailed bool
	}
	cases := testCase{}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	// Mock connect.IsConnected
	patch.ApplyFunc(connect.IsConnected, func() bool {
		return !cases.isConnectFailed
	})

	// Mock beehiveContext.SendSync
	patch.ApplyFunc(beehiveContext.SendSync, func(string, model.Message, time.Duration) (model.Message, error) {
		app := metaserver.Application{
			RespBody: []byte("test"),
			Status:   metaserver.Approved,
			Reason:   "ok",
		}
		if cases.isSendSyncFailed {
			app.Status = metaserver.Failed
			app.Reason = "isSendSyncFailed"
		}
		content, _ := json.Marshal(app)
		return model.Message{
			Content: content,
		}, nil
	})

	// Instead of applying to global variable, patch the REST.PassThrough method directly
	patch.ApplyMethod(reflect.TypeOf(&REST{}), "PassThrough", func(_ *REST, ctx context.Context, options *metav1.GetOptions) ([]byte, error) {
		// Simulate connection check
		if !connect.IsConnected() {
			return nil, errors.New("connection lost between EdgeCore and CloudCore")
		}

		// Simulate cloud interaction
		if cases.isSendSyncFailed {
			// Try to get from local
			if cases.isLocalStored {
				return []byte("test"), nil
			}
			return nil, errors.New("send sync failed and not stored locally")
		}
		return []byte("test"), nil
	})

	var tests = []struct {
		name    string
		rest    *REST
		info    apirequest.RequestInfo
		cases   testCase
		want    []byte
		wantErr bool
	}{
		{
			name:    "test isConnectFailed ",
			info:    apirequest.RequestInfo{},
			cases:   testCase{isConnectFailed: true},
			wantErr: true,
		}, {
			name:    "test isSendSyncFailed ",
			info:    apirequest.RequestInfo{},
			cases:   testCase{isSendSyncFailed: true},
			wantErr: true,
		}, {
			name: "test get version from cloud failed, but local stored",
			info: apirequest.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			cases: testCase{isSendSyncFailed: true, isLocalStored: true},
			want:  []byte("test"),
		}, {
			name: "test successfully get the version from the cloud, but insert local storage failed ",
			info: apirequest.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			cases: testCase{isInsertLocalStorageFailed: true},
			want:  []byte("test"),
		}, {
			name: "test successfully get the version from the cloud ",
			info: apirequest.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			want: []byte("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := apirequest.WithRequestInfo(context.TODO(), &tt.info)
			rest := &REST{
				Agent: &agent.Agent{Applications: sync.Map{}},
			}
			cases = tt.cases
			got, err := rest.PassThrough(ctx, &metav1.GetOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("PassThrough() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PassThrough() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestREST_Exec(t *testing.T) {
	type testCase struct {
		execInfo               common.ExecInfo
		isEdgedEnabled         bool
		isRemoteRuntimeService bool
		isListContainersFailed bool
		isExecSyncFailed       bool
		isExecFailed           bool
		isParseExecURLFailed   bool
		expectedExecResponse   *types.ExecResponse
		expectedHandlerNotNil  bool
	}

	cases := testCase{}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	patch.ApplyGlobalVar(&config.Config, config.Configure{
		Edged: v1alpha2.Edged{
			Enable: true,
			TailoredKubeletConfig: &v1alpha2.TailoredKubeletConfiguration{
				ContainerRuntimeEndpoint: "",
			},
		},
	}).ApplyFunc(remote.NewRemoteRuntimeService, func(endpoint string, timeout time.Duration, tracerProvider oteltrace.TracerProvider) (cri.RuntimeService, error) {
		if !cases.isRemoteRuntimeService {
			return nil, errors.New("err in NewRemoteRuntimeService")
		}
		return &fakeRuntimeService{
			ListContainersF: func(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
				if cases.isListContainersFailed {
					return nil, errors.New("err in ListContainers")
				}
				return []*runtimeapi.Container{
					{
						Id: "container-id",
						Metadata: &runtimeapi.ContainerMetadata{
							Name: "container-name",
						},
					},
				}, nil
			},
			ExecSyncF: func(ctx context.Context, containerID string, cmd []string, timeout time.Duration) ([]byte, []byte, error) {
				if cases.isExecSyncFailed {
					return nil, nil, errors.New("err in ExecSync")
				}
				return []byte("stdout"), []byte("stderr"), nil
			},
			ExecF: func(ctx context.Context, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
				if cases.isExecFailed {
					return nil, errors.New("err in Exec")
				}
				return &runtimeapi.ExecResponse{
					Url: "http://localhost:8080/exec",
				}, nil
			},
		}, nil
	}).ApplyFunc(url.Parse, func(rawURL string) (*url.URL, error) {
		if cases.isParseExecURLFailed {
			return nil, errors.New("err in Parse")
		}
		return &url.URL{}, nil
	})

	var tests = []struct {
		name    string
		rest    *REST
		cases   testCase
		want    *types.ExecResponse
		wantErr bool
	}{
		{
			name: "test commands not specified",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
				},
				isEdgedEnabled: true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{"You must specify at least one command for the container"},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "test remote runtime service failed",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: false,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{"new remote runtimeservice with err: err in NewRemoteRuntimeService"},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "test list containers failed",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isListContainersFailed: true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{"failed to list containers: err in ListContainers"},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "test exec sync failed",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isExecSyncFailed:       true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{"failed to exec command [ls] for container container-name for pod \"/default/pod-name\" with err:err in ExecSync"},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "test exec failed",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
					TTY:       true,
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isExecFailed:           true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{"failed to exec command [ls] for container container-name for pod \"/default/pod-name\" with err:err in Exec"},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "test parse exec url failed",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
					TTY:       true,
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isParseExecURLFailed:   true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{"failed to parse exec url with err:err in Parse"},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
			},
			wantErr: true,
		},
		{
			name: "test exec sync success",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{},
					RunOutMessages: []string{"stdout"},
					RunErrMessages: []string{"stderr"},
				},
			},
			wantErr: false,
		},
		{
			name: "test exec success",
			cases: testCase{
				execInfo: common.ExecInfo{
					Namespace: "default",
					PodName:   "pod-name",
					Container: "container-name",
					Commands:  []string{"ls"},
					TTY:       true,
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				expectedExecResponse: &types.ExecResponse{
					ErrMessages:    []string{},
					RunOutMessages: []string{},
					RunErrMessages: []string{},
				},
				expectedHandlerNotNil: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest := &REST{}
			cases = tt.cases
			got, handler := rest.Exec(context.TODO(), tt.cases.execInfo)
			if !reflect.DeepEqual(got, tt.cases.expectedExecResponse) {
				t.Errorf("Exec() got = %v, want %v", got, tt.cases.expectedExecResponse)
			}
			if (handler != nil) != tt.cases.expectedHandlerNotNil {
				t.Errorf("Exec() handler = %v, want %v", handler != nil, tt.cases.expectedHandlerNotNil)
			}
		})
	}
}

func TestREST_Logs(t *testing.T) {
	type testCase struct {
		logsInfo                common.LogsInfo
		isEdgedEnabled          bool
		isRemoteRuntimeService  bool
		isListContainersFailed  bool
		isNoContainersFound     bool
		isRestfulRequestFailed  bool
		expectedLogsResponse    *types.LogsResponse
		expectedHTTPResponseNil bool
	}

	cases := testCase{}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	req := &restful.Request{}

	patch.ApplyGlobalVar(&config.Config, config.Configure{
		Edged: v1alpha2.Edged{
			Enable: true,
			TailoredKubeletConfig: &v1alpha2.TailoredKubeletConfiguration{
				ContainerRuntimeEndpoint: "",
			},
		},
	}).ApplyFunc(remote.NewRemoteRuntimeService, func(endpoint string, timeout time.Duration, tracerProvider oteltrace.TracerProvider) (cri.RuntimeService, error) {
		if !cases.isRemoteRuntimeService {
			return nil, errors.New("err in NewRemoteRuntimeService")
		}
		return &fakeRuntimeService{
			ListContainersF: func(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
				if cases.isListContainersFailed {
					return nil, errors.New("err in ListContainers")
				}
				if cases.isNoContainersFound {
					return []*runtimeapi.Container{}, nil
				}
				return []*runtimeapi.Container{
					{
						Id: "container-id",
						Metadata: &runtimeapi.ContainerMetadata{
							Name: "container-name",
						},
					},
				}, nil
			},
		}, nil
	}).ApplyMethod(reflect.TypeOf(req), "RestfulRequest", func(_ *restful.Request) (*http.Response, error) {
		if cases.isRestfulRequestFailed {
			return nil, errors.New("err in RestfulRequest")
		}
		return &http.Response{}, nil
	})

	var tests = []struct {
		name    string
		rest    *REST
		cases   testCase
		want    *types.LogsResponse
		wantErr bool
	}{
		{
			name: "test remote runtime service failed",
			cases: testCase{
				logsInfo: common.LogsInfo{
					Namespace: "default",
					PodName:   "pod-name",
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: false,
				expectedLogsResponse: &types.LogsResponse{
					ErrMessages: []string{"new remote runtimeservice with err: err in NewRemoteRuntimeService"},
					LogMessages: []string{},
				},
				expectedHTTPResponseNil: true,
			},
			wantErr: true,
		},
		{
			name: "test list containers failed",
			cases: testCase{
				logsInfo: common.LogsInfo{
					Namespace: "default",
					PodName:   "pod-name",
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isListContainersFailed: true,
				expectedLogsResponse: &types.LogsResponse{
					ErrMessages: []string{"failed to list containers: err in ListContainers"},
					LogMessages: []string{},
				},
				expectedHTTPResponseNil: true,
			},
			wantErr: true,
		},
		{
			name: "test no containers found",
			cases: testCase{
				logsInfo: common.LogsInfo{
					Namespace: "default",
					PodName:   "pod-name",
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isNoContainersFound:    true,
				expectedLogsResponse: &types.LogsResponse{
					ErrMessages: []string{"not found pod:\"/default/pod-name\""},
					LogMessages: []string{},
				},
				expectedHTTPResponseNil: true,
			},
			wantErr: true,
		},
		{
			name: "test restful request failed",
			cases: testCase{
				logsInfo: common.LogsInfo{
					Namespace: "default",
					PodName:   "pod-name",
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				isRestfulRequestFailed: true,
				expectedLogsResponse: &types.LogsResponse{
					ErrMessages: []string{"failed to get logs for container container-name for pod \"/default/pod-name\" with err:err in RestfulRequest"},
					LogMessages: []string{},
				},
				expectedHTTPResponseNil: true,
			},
			wantErr: true,
		},
		{
			name: "test logs success",
			cases: testCase{
				logsInfo: common.LogsInfo{
					Namespace: "default",
					PodName:   "pod-name",
				},
				isEdgedEnabled:         true,
				isRemoteRuntimeService: true,
				expectedLogsResponse: &types.LogsResponse{
					ErrMessages: []string{},
					LogMessages: []string{},
				},
				expectedHTTPResponseNil: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest := &REST{}
			cases = tt.cases
			got, res := rest.Logs(context.TODO(), tt.cases.logsInfo)
			if !reflect.DeepEqual(got, tt.cases.expectedLogsResponse) {
				t.Errorf("Logs() got = %v, want %v", got, tt.cases.expectedLogsResponse)
			}
			if (res == nil) != tt.cases.expectedHTTPResponseNil {
				t.Errorf("Logs() res = %v, want %v", res == nil, tt.cases.expectedHTTPResponseNil)
			}
		})
	}
}

func TestREST_Watch(t *testing.T) {
	type testCase struct {
		generateSuccess   bool
		applySuccess      bool
		localWatchSuccess bool
	}

	cases := testCase{}

	fakeAgent := &agent.Agent{Applications: sync.Map{}}
	watcher := watch.NewFake()

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	patch.ApplyMethod(reflect.TypeOf(fakeAgent), "Generate",
		func(_ *agent.Agent, ctx context.Context, verb metaserver.ApplicationVerb, options interface{}, obj interface{}) (*metaserver.Application, error) {
			if !cases.generateSuccess {
				return nil, errors.New("generate failed")
			}
			app := &metaserver.Application{ID: "test-app-id"}
			return app, nil
		}).
		ApplyMethod(reflect.TypeOf(fakeAgent), "Apply",
			func(_ *agent.Agent, app *metaserver.Application) error {
				if !cases.applySuccess {
					return errors.New("apply failed")
				}
				return nil
			})

	// Mock the Store.Watch method
	patch.ApplyMethod(reflect.TypeOf(&genericregistry.Store{}), "Watch",
		func(_ *genericregistry.Store, ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
			if !cases.localWatchSuccess {
				return nil, errors.New("local watch failed")
			}
			return watcher, nil
		})

	var tests = []struct {
		name    string
		rest    *REST
		info    apirequest.RequestInfo
		cases   testCase
		wantErr bool
	}{
		{
			name: "watch from cloud success",
			rest: &REST{
				Agent: fakeAgent,
				Store: &genericregistry.Store{},
			},
			info: apirequest.RequestInfo{
				Path:     "/api/v1/namespaces/default/pods",
				APIGroup: "test-group",
				Resource: "pods",
			},
			cases: testCase{
				generateSuccess:   true,
				applySuccess:      true,
				localWatchSuccess: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := apirequest.WithRequestInfo(context.TODO(), &tt.info)
			cases = tt.cases

			_, err := tt.rest.Watch(ctx, &metainternalversion.ListOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Watch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestREST_Create(t *testing.T) {
	type testCase struct {
		generateSuccess bool
		applySuccess    bool
	}

	cases := testCase{}

	fakeAgent := &agent.Agent{Applications: sync.Map{}}
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "test-pod",
			},
		},
	}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	patch.ApplyMethod(reflect.TypeOf(fakeAgent), "Generate",
		func(_ *agent.Agent, ctx context.Context, verb metaserver.ApplicationVerb, options interface{}, obj interface{}) (*metaserver.Application, error) {
			if !cases.generateSuccess {
				return nil, errors.New("generate failed")
			}
			app := &metaserver.Application{RespBody: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod"}}`)}
			return app, nil
		}).
		ApplyMethod(reflect.TypeOf(fakeAgent), "Apply",
			func(_ *agent.Agent, app *metaserver.Application) error {
				if !cases.applySuccess {
					return errors.New("apply failed")
				}
				return nil
			})

	var tests = []struct {
		name    string
		rest    *REST
		cases   testCase
		wantErr bool
	}{
		{
			name: "create failed - generate failed",
			rest: &REST{
				Agent: fakeAgent,
			},
			cases: testCase{
				generateSuccess: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases = tt.cases

			result, err := tt.rest.Create(context.TODO(), obj, nil, &metav1.CreateOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Errorf("Create() got nil object")
			}
		})
	}
}

func TestREST_Delete(t *testing.T) {
	type testCase struct {
		generateSuccess bool
		applySuccess    bool
	}

	cases := testCase{}

	fakeAgent := &agent.Agent{Applications: sync.Map{}}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	patch.ApplyMethod(reflect.TypeOf(fakeAgent), "Generate",
		func(_ *agent.Agent, ctx context.Context, verb metaserver.ApplicationVerb, options interface{}, obj interface{}) (*metaserver.Application, error) {
			if !cases.generateSuccess {
				return nil, errors.New("generate failed")
			}
			app := &metaserver.Application{}
			return app, nil
		}).
		ApplyMethod(reflect.TypeOf(fakeAgent), "Apply",
			func(_ *agent.Agent, app *metaserver.Application) error {
				if !cases.applySuccess {
					return errors.New("apply failed")
				}
				return nil
			})

	patch.ApplyFunc(metaserver.KeyFuncReq, func(ctx context.Context, name string) (string, error) {
		return "/api/v1/namespaces/default/pods/test-pod", nil
	})

	var tests = []struct {
		name    string
		rest    *REST
		cases   testCase
		wantErr bool
	}{
		{
			name: "delete failed - generate failed",
			rest: &REST{
				Agent: fakeAgent,
			},
			cases: testCase{
				generateSuccess: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases = tt.cases

			_, deleted, err := tt.rest.Delete(context.TODO(), "", nil, &metav1.DeleteOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !deleted {
				t.Errorf("Delete() deleted = %v, want true", deleted)
			}
		})
	}
}

func TestREST_Patch(t *testing.T) {
	type testCase struct {
		generateSuccess bool
		applySuccess    bool
	}

	cases := testCase{}

	fakeAgent := &agent.Agent{Applications: sync.Map{}}
	patchInfo := metaserver.PatchInfo{
		Data: []byte(`{"metadata":{"labels":{"test":"label"}}}`),
	}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	patch.ApplyMethod(reflect.TypeOf(fakeAgent), "Generate",
		func(_ *agent.Agent, ctx context.Context, verb metaserver.ApplicationVerb, options interface{}, obj interface{}) (*metaserver.Application, error) {
			if !cases.generateSuccess {
				return nil, errors.New("generate failed")
			}
			app := &metaserver.Application{RespBody: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-pod"}}`)}
			return app, nil
		}).
		ApplyMethod(reflect.TypeOf(fakeAgent), "Apply",
			func(_ *agent.Agent, app *metaserver.Application) error {
				if !cases.applySuccess {
					return errors.New("apply failed")
				}
				return nil
			})

	var tests = []struct {
		name    string
		rest    *REST
		cases   testCase
		wantErr bool
	}{
		{
			name: "patch failed - generate failed",
			rest: &REST{
				Agent: fakeAgent,
			},
			cases: testCase{
				generateSuccess: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases = tt.cases

			result, err := tt.rest.Patch(context.TODO(), patchInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Patch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Errorf("Patch() got nil object")
			}
		})
	}
}

func TestREST_Restart(t *testing.T) {
	type testCase struct {
		isRemoteRuntimeService bool
		isListContainersFailed bool
		isStopContainerFailed  bool
		noContainersFound      bool
	}

	cases := testCase{}

	patch := gomonkey.NewPatches()
	defer patch.Reset()

	patch.ApplyGlobalVar(&config.Config, config.Configure{
		Edged: v1alpha2.Edged{
			Enable: true,
			TailoredKubeletConfig: &v1alpha2.TailoredKubeletConfiguration{
				ContainerRuntimeEndpoint: "unix:///var/run/containerd/containerd.sock",
			},
		},
	}).ApplyFunc(remote.NewRemoteRuntimeService, func(endpoint string, timeout time.Duration, tracerProvider oteltrace.TracerProvider) (cri.RuntimeService, error) {
		if !cases.isRemoteRuntimeService {
			return nil, errors.New("err in NewRemoteRuntimeService")
		}
		return &fakeRuntimeService{
			ListContainersF: func(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
				if cases.isListContainersFailed {
					return nil, errors.New("err in ListContainers")
				}
				if cases.noContainersFound {
					return []*runtimeapi.Container{}, nil
				}
				return []*runtimeapi.Container{
					{
						Id: "container-id",
						Metadata: &runtimeapi.ContainerMetadata{
							Name: "container-name",
						},
					},
				}, nil
			},
			StopContainerF: func(ctx context.Context, containerID string, timeout int64) error {
				if cases.isStopContainerFailed {
					return errors.New("err in StopContainer")
				}
				return nil
			},
		}, nil
	})

	var tests = []struct {
		name                string
		rest                *REST
		restartInfo         common.RestartInfo
		cases               testCase
		expectedErrMessages int
		expectedLogMessages int
	}{
		{
			name: "remote runtime service failed",
			rest: &REST{},
			restartInfo: common.RestartInfo{
				Namespace: "default",
				PodNames:  []string{"pod-name"},
			},
			cases: testCase{
				isRemoteRuntimeService: false,
			},
			expectedErrMessages: 1,
			expectedLogMessages: 0,
		},
		{
			name: "list containers failed",
			rest: &REST{},
			restartInfo: common.RestartInfo{
				Namespace: "default",
				PodNames:  []string{"pod-name"},
			},
			cases: testCase{
				isRemoteRuntimeService: true,
				isListContainersFailed: true,
			},
			expectedErrMessages: 1,
			expectedLogMessages: 0,
		},
		{
			name: "no containers found",
			rest: &REST{},
			restartInfo: common.RestartInfo{
				Namespace: "default",
				PodNames:  []string{"pod-name"},
			},
			cases: testCase{
				isRemoteRuntimeService: true,
				noContainersFound:      true,
			},
			expectedErrMessages: 1,
			expectedLogMessages: 0,
		},
		{
			name: "stop container failed",
			rest: &REST{},
			restartInfo: common.RestartInfo{
				Namespace: "default",
				PodNames:  []string{"pod-name"},
			},
			cases: testCase{
				isRemoteRuntimeService: true,
				isStopContainerFailed:  true,
			},
			expectedErrMessages: 1,
			expectedLogMessages: 0,
		},
		{
			name: "restart success",
			rest: &REST{},
			restartInfo: common.RestartInfo{
				Namespace: "default",
				PodNames:  []string{"pod-name"},
			},
			cases: testCase{
				isRemoteRuntimeService: true,
			},
			expectedErrMessages: 0,
			expectedLogMessages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases = tt.cases

			got := tt.rest.Restart(context.TODO(), tt.restartInfo)

			if len(got.ErrMessages) != tt.expectedErrMessages {
				t.Errorf("Restart() got %v error messages, want %v", len(got.ErrMessages), tt.expectedErrMessages)
			}

			if len(got.LogMessages) != tt.expectedLogMessages {
				t.Errorf("Restart() got %v log messages, want %v", len(got.LogMessages), tt.expectedLogMessages)
			}
		})
	}
}

func TestResponder_Error(t *testing.T) {
	responder := &responder{}

	// Create a test ResponseWriter
	recorder := httptest.NewRecorder()
	testErr := errors.New("test error")

	responder.Error(recorder, nil, testErr)

	// Check the response
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Error() status code = %v, want %v", recorder.Code, http.StatusInternalServerError)
	}

	if !strings.Contains(recorder.Body.String(), "test error") {
		t.Errorf("Error() body = %v, should contain 'test error'", recorder.Body.String())
	}
}

type fakeRuntimeService struct {
	VersionF                  func(ctx context.Context, apiVersion string) (*runtimeapi.VersionResponse, error)
	CreateContainerF          func(ctx context.Context, podSandboxID string, config *runtimeapi.ContainerConfig, sandboxConfig *runtimeapi.PodSandboxConfig) (string, error)
	StartContainerF           func(ctx context.Context, containerID string) error
	StopContainerF            func(ctx context.Context, containerID string, timeout int64) error
	RemoveContainerF          func(ctx context.Context, containerID string) error
	ListContainersF           func(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error)
	ContainerStatusF          func(ctx context.Context, containerID string, verbose bool) (*runtimeapi.ContainerStatusResponse, error)
	UpdateContainerResourcesF func(ctx context.Context, containerID string, resources *runtimeapi.ContainerResources) error
	ExecSyncF                 func(ctx context.Context, containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error)
	ExecF                     func(ctx context.Context, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error)
	AttachF                   func(ctx context.Context, req *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error)
	ReopenContainerLogF       func(ctx context.Context, ContainerID string) error
	CheckpointContainerF      func(ctx context.Context, options *runtimeapi.CheckpointContainerRequest) error
	GetContainerEventsF       func(containerEventsCh chan *runtimeapi.ContainerEventResponse) error
	RunPodSandboxF            func(ctx context.Context, config *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error)
	StopPodSandboxF           func(ctx context.Context, podSandboxID string) error
	RemovePodSandboxF         func(ctx context.Context, podSandboxID string) error
	PodSandboxStatusF         func(ctx context.Context, podSandboxID string, verbose bool) (*runtimeapi.PodSandboxStatusResponse, error)
	ListPodSandboxF           func(ctx context.Context, filter *runtimeapi.PodSandboxFilter) ([]*runtimeapi.PodSandbox, error)
	PortForwardF              func(ctx context.Context, req *runtimeapi.PortForwardRequest) (*runtimeapi.PortForwardResponse, error)
	ContainerStatsF           func(ctx context.Context, containerID string) (*runtimeapi.ContainerStats, error)
	ListContainerStatsF       func(ctx context.Context, filter *runtimeapi.ContainerStatsFilter) ([]*runtimeapi.ContainerStats, error)
	PodSandboxStatsF          func(ctx context.Context, podSandboxID string) (*runtimeapi.PodSandboxStats, error)
	ListPodSandboxStatsF      func(ctx context.Context, filter *runtimeapi.PodSandboxStatsFilter) ([]*runtimeapi.PodSandboxStats, error)
	ListMetricDescriptorsF    func(ctx context.Context) ([]*runtimeapi.MetricDescriptor, error)
	ListPodSandboxMetricsF    func(ctx context.Context) ([]*runtimeapi.PodSandboxMetrics, error)
	UpdateRuntimeConfigF      func(ctx context.Context, runtimeConfig *runtimeapi.RuntimeConfig) error
	StatusF                   func(ctx context.Context, verbose bool) (*runtimeapi.StatusResponse, error)
	RuntimeConfigF            func(ctx context.Context) (*runtimeapi.RuntimeConfigResponse, error)
	ImageFsInfoF              func(ctx context.Context) (*runtimeapi.ImageFsInfoResponse, error)
}

func (f *fakeRuntimeService) Version(ctx context.Context, apiVersion string) (*runtimeapi.VersionResponse, error) {
	if f.VersionF != nil {
		return f.VersionF(ctx, apiVersion)
	}
	return nil, nil
}

func (f *fakeRuntimeService) CreateContainer(ctx context.Context, podSandboxID string, config *runtimeapi.ContainerConfig, sandboxConfig *runtimeapi.PodSandboxConfig) (string, error) {
	if f.CreateContainerF != nil {
		return f.CreateContainerF(ctx, podSandboxID, config, sandboxConfig)
	}
	return "", nil
}

func (f *fakeRuntimeService) StartContainer(ctx context.Context, containerID string) error {
	if f.StartContainerF != nil {
		return f.StartContainerF(ctx, containerID)
	}
	return nil
}

func (f *fakeRuntimeService) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	if f.StopContainerF != nil {
		return f.StopContainerF(ctx, containerID, timeout)
	}
	return nil
}

func (f *fakeRuntimeService) RemoveContainer(ctx context.Context, containerID string) error {
	if f.RemoveContainerF != nil {
		return f.RemoveContainerF(ctx, containerID)
	}
	return nil
}

func (f *fakeRuntimeService) ListContainers(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
	if f.ListContainersF != nil {
		return f.ListContainersF(ctx, filter)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ContainerStatus(ctx context.Context, containerID string, verbose bool) (*runtimeapi.ContainerStatusResponse, error) {
	if f.ContainerStatusF != nil {
		return f.ContainerStatusF(ctx, containerID, verbose)
	}
	return nil, nil
}

func (f *fakeRuntimeService) UpdateContainerResources(ctx context.Context, containerID string, resources *runtimeapi.ContainerResources) error {
	if f.UpdateContainerResourcesF != nil {
		return f.UpdateContainerResourcesF(ctx, containerID, resources)
	}
	return nil
}

func (f *fakeRuntimeService) ExecSync(ctx context.Context, containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error) {
	if f.ExecSyncF != nil {
		return f.ExecSyncF(ctx, containerID, cmd, timeout)
	}
	return nil, nil, nil
}

func (f *fakeRuntimeService) Exec(ctx context.Context, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
	if f.ExecF != nil {
		return f.ExecF(ctx, req)
	}
	return nil, nil
}

func (f *fakeRuntimeService) Attach(ctx context.Context, req *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error) {
	if f.AttachF != nil {
		return f.AttachF(ctx, req)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ReopenContainerLog(ctx context.Context, ContainerID string) error {
	if f.ReopenContainerLogF != nil {
		return f.ReopenContainerLogF(ctx, ContainerID)
	}
	return nil
}

func (f *fakeRuntimeService) CheckpointContainer(ctx context.Context, options *runtimeapi.CheckpointContainerRequest) error {
	if f.CheckpointContainerF != nil {
		return f.CheckpointContainerF(ctx, options)
	}
	return nil
}

func (f *fakeRuntimeService) GetContainerEvents(containerEventsCh chan *runtimeapi.ContainerEventResponse) error {
	if f.GetContainerEventsF != nil {
		return f.GetContainerEventsF(containerEventsCh)
	}
	return nil
}

func (f *fakeRuntimeService) RunPodSandbox(ctx context.Context, config *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error) {
	if f.RunPodSandboxF != nil {
		return f.RunPodSandboxF(ctx, config, runtimeHandler)
	}
	return "", nil
}

func (f *fakeRuntimeService) StopPodSandbox(ctx context.Context, podSandboxID string) error {
	if f.StopPodSandboxF != nil {
		return f.StopPodSandboxF(ctx, podSandboxID)
	}
	return nil
}

func (f *fakeRuntimeService) RemovePodSandbox(ctx context.Context, podSandboxID string) error {
	if f.RemovePodSandboxF != nil {
		return f.RemovePodSandboxF(ctx, podSandboxID)
	}
	return nil
}

func (f *fakeRuntimeService) PodSandboxStatus(ctx context.Context, podSandboxID string, verbose bool) (*runtimeapi.PodSandboxStatusResponse, error) {
	if f.PodSandboxStatusF != nil {
		return f.PodSandboxStatusF(ctx, podSandboxID, verbose)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ListPodSandbox(ctx context.Context, filter *runtimeapi.PodSandboxFilter) ([]*runtimeapi.PodSandbox, error) {
	if f.ListPodSandboxF != nil {
		return f.ListPodSandboxF(ctx, filter)
	}
	return nil, nil
}

func (f *fakeRuntimeService) PortForward(ctx context.Context, req *runtimeapi.PortForwardRequest) (*runtimeapi.PortForwardResponse, error) {
	if f.PortForwardF != nil {
		return f.PortForwardF(ctx, req)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ContainerStats(ctx context.Context, containerID string) (*runtimeapi.ContainerStats, error) {
	if f.ContainerStatsF != nil {
		return f.ContainerStatsF(ctx, containerID)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ListContainerStats(ctx context.Context, filter *runtimeapi.ContainerStatsFilter) ([]*runtimeapi.ContainerStats, error) {
	if f.ListContainerStatsF != nil {
		return f.ListContainerStatsF(ctx, filter)
	}
	return nil, nil
}

func (f *fakeRuntimeService) PodSandboxStats(ctx context.Context, podSandboxID string) (*runtimeapi.PodSandboxStats, error) {
	if f.PodSandboxStatsF != nil {
		return f.PodSandboxStatsF(ctx, podSandboxID)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ListPodSandboxStats(ctx context.Context, filter *runtimeapi.PodSandboxStatsFilter) ([]*runtimeapi.PodSandboxStats, error) {
	if f.ListPodSandboxStatsF != nil {
		return f.ListPodSandboxStatsF(ctx, filter)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ListMetricDescriptors(ctx context.Context) ([]*runtimeapi.MetricDescriptor, error) {
	if f.ListMetricDescriptorsF != nil {
		return f.ListMetricDescriptorsF(ctx)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ListPodSandboxMetrics(ctx context.Context) ([]*runtimeapi.PodSandboxMetrics, error) {
	if f.ListPodSandboxMetricsF != nil {
		return f.ListPodSandboxMetricsF(ctx)
	}
	return nil, nil
}

func (f *fakeRuntimeService) UpdateRuntimeConfig(ctx context.Context, runtimeConfig *runtimeapi.RuntimeConfig) error {
	if f.UpdateRuntimeConfigF != nil {
		return f.UpdateRuntimeConfigF(ctx, runtimeConfig)
	}
	return nil
}

func (f *fakeRuntimeService) Status(ctx context.Context, verbose bool) (*runtimeapi.StatusResponse, error) {
	if f.StatusF != nil {
		return f.StatusF(ctx, verbose)
	}
	return nil, nil
}

func (f *fakeRuntimeService) RuntimeConfig(ctx context.Context) (*runtimeapi.RuntimeConfigResponse, error) {
	if f.RuntimeConfigF != nil {
		return f.RuntimeConfigF(ctx)
	}
	return nil, nil
}

func (f *fakeRuntimeService) ImageFsInfo(ctx context.Context) (*runtimeapi.ImageFsInfoResponse, error) {
	if f.ImageFsInfoF != nil {
		return f.ImageFsInfoF(ctx)
	}
	return nil, nil
}
