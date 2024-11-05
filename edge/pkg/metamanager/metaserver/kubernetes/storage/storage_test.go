package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"
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
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	fakeclient "github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator/fake"
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

	fakeClient := fakeclient.Client{
		InsertOrUpdatePassThroughObjF: func(ctx context.Context, obj []byte, key string) error {
			if cases.isInsertLocalStorageFailed {
				return fmt.Errorf("insert local storage failed")
			}
			return nil
		},
		GetPassThroughObjF: func(ctx context.Context, key string) ([]byte, error) {
			if !cases.isLocalStored {
				return nil, fmt.Errorf("local does not store it")
			}
			return []byte("test"), nil
		},
	}
	patch := gomonkey.NewPatches()
	defer patch.Reset()
	patch.ApplyFunc(connect.IsConnected, func() bool {
		return !cases.isConnectFailed
	}).ApplyFunc(beehiveContext.SendSync, func(string, model.Message, time.Duration) (model.Message, error) {
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
	}).ApplyGlobalVar(&imitator.DefaultV2Client, fakeClient)

	var tests = []struct {
		name    string
		rest    *REST
		info    request.RequestInfo
		cases   testCase
		want    []byte
		wantErr bool
	}{
		{
			name:    "test isConnectFailed ",
			info:    request.RequestInfo{},
			cases:   testCase{isConnectFailed: true},
			wantErr: true,
		}, {
			name:    "test isSendSyncFailed ",
			info:    request.RequestInfo{},
			cases:   testCase{isSendSyncFailed: true},
			wantErr: true,
		}, {
			name: "test get version from cloud failed, but local stored",
			info: request.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			cases: testCase{isSendSyncFailed: true, isLocalStored: true},
			want:  []byte("test"),
		}, {
			name: "test successfully get the version from the cloud, but insert local storage failed ",
			info: request.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			cases: testCase{isInsertLocalStorageFailed: true},
			want:  []byte("test"),
		}, {
			name: "test successfully get the version from the cloud ",
			info: request.RequestInfo{
				Path: "/versions",
				Verb: "get",
			},
			want: []byte("test"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := request.WithRequestInfo(context.TODO(), &tt.info)
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
	}).ApplyFunc(remote.NewRemoteRuntimeService, func(endpoint string, timeout time.Duration, tracerProvider trace.TracerProvider) (cri.RuntimeService, error) {
		if !cases.isRemoteRuntimeService {
			return nil, fmt.Errorf("err in NewRemoteRuntimeService")
		}
		return &fakeRuntimeService{
			ListContainersF: func(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
				if cases.isListContainersFailed {
					return nil, fmt.Errorf("err in ListContainers")
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
					return nil, nil, fmt.Errorf("err in ExecSync")
				}
				return []byte("stdout"), []byte("stderr"), nil
			},
			ExecF: func(ctx context.Context, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
				if cases.isExecFailed {
					return nil, fmt.Errorf("err in Exec")
				}
				return &runtimeapi.ExecResponse{
					Url: "http://localhost:8080/exec",
				}, nil
			},
		}, nil
	}).ApplyFunc(url.Parse, func(rawURL string) (*url.URL, error) {
		if cases.isParseExecURLFailed {
			return nil, fmt.Errorf("err in Parse")
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
	}).ApplyFunc(remote.NewRemoteRuntimeService, func(endpoint string, timeout time.Duration, tracerProvider trace.TracerProvider) (cri.RuntimeService, error) {
		if !cases.isRemoteRuntimeService {
			return nil, fmt.Errorf("err in NewRemoteRuntimeService")
		}
		return &fakeRuntimeService{
			ListContainersF: func(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
				if cases.isListContainersFailed {
					return nil, fmt.Errorf("err in ListContainers")
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
			return nil, fmt.Errorf("err in RestfulRequest")
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
	return f.VersionF(ctx, apiVersion)
}

func (f *fakeRuntimeService) CreateContainer(ctx context.Context, podSandboxID string, config *runtimeapi.ContainerConfig, sandboxConfig *runtimeapi.PodSandboxConfig) (string, error) {
	return f.CreateContainerF(ctx, podSandboxID, config, sandboxConfig)
}

func (f *fakeRuntimeService) StartContainer(ctx context.Context, containerID string) error {
	return f.StartContainerF(ctx, containerID)
}

func (f *fakeRuntimeService) StopContainer(ctx context.Context, containerID string, timeout int64) error {
	return f.StopContainerF(ctx, containerID, timeout)
}

func (f *fakeRuntimeService) RemoveContainer(ctx context.Context, containerID string) error {
	return f.RemoveContainerF(ctx, containerID)
}

func (f *fakeRuntimeService) ListContainers(ctx context.Context, filter *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
	return f.ListContainersF(ctx, filter)
}

func (f *fakeRuntimeService) ContainerStatus(ctx context.Context, containerID string, verbose bool) (*runtimeapi.ContainerStatusResponse, error) {
	return f.ContainerStatusF(ctx, containerID, verbose)
}

func (f *fakeRuntimeService) UpdateContainerResources(ctx context.Context, containerID string, resources *runtimeapi.ContainerResources) error {
	return f.UpdateContainerResourcesF(ctx, containerID, resources)
}

func (f *fakeRuntimeService) ExecSync(ctx context.Context, containerID string, cmd []string, timeout time.Duration) (stdout []byte, stderr []byte, err error) {
	return f.ExecSyncF(ctx, containerID, cmd, timeout)
}

func (f *fakeRuntimeService) Exec(ctx context.Context, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
	return f.ExecF(ctx, req)
}

func (f *fakeRuntimeService) Attach(ctx context.Context, req *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error) {
	return f.AttachF(ctx, req)
}

func (f *fakeRuntimeService) ReopenContainerLog(ctx context.Context, ContainerID string) error {
	return f.ReopenContainerLogF(ctx, ContainerID)
}

func (f *fakeRuntimeService) CheckpointContainer(ctx context.Context, options *runtimeapi.CheckpointContainerRequest) error {
	return f.CheckpointContainerF(ctx, options)
}

func (f *fakeRuntimeService) GetContainerEvents(containerEventsCh chan *runtimeapi.ContainerEventResponse) error {
	return f.GetContainerEventsF(containerEventsCh)
}

func (f *fakeRuntimeService) RunPodSandbox(ctx context.Context, config *runtimeapi.PodSandboxConfig, runtimeHandler string) (string, error) {
	return f.RunPodSandboxF(ctx, config, runtimeHandler)
}

func (f *fakeRuntimeService) StopPodSandbox(ctx context.Context, podSandboxID string) error {
	return f.StopPodSandboxF(ctx, podSandboxID)
}

func (f *fakeRuntimeService) RemovePodSandbox(ctx context.Context, podSandboxID string) error {
	return f.RemovePodSandboxF(ctx, podSandboxID)
}

func (f *fakeRuntimeService) PodSandboxStatus(ctx context.Context, podSandboxID string, verbose bool) (*runtimeapi.PodSandboxStatusResponse, error) {
	return f.PodSandboxStatusF(ctx, podSandboxID, verbose)
}

func (f *fakeRuntimeService) ListPodSandbox(ctx context.Context, filter *runtimeapi.PodSandboxFilter) ([]*runtimeapi.PodSandbox, error) {
	return f.ListPodSandboxF(ctx, filter)
}

func (f *fakeRuntimeService) PortForward(ctx context.Context, req *runtimeapi.PortForwardRequest) (*runtimeapi.PortForwardResponse, error) {
	return f.PortForwardF(ctx, req)
}

func (f *fakeRuntimeService) ContainerStats(ctx context.Context, containerID string) (*runtimeapi.ContainerStats, error) {
	return f.ContainerStatsF(ctx, containerID)
}

func (f *fakeRuntimeService) ListContainerStats(ctx context.Context, filter *runtimeapi.ContainerStatsFilter) ([]*runtimeapi.ContainerStats, error) {
	return f.ListContainerStatsF(ctx, filter)
}

func (f *fakeRuntimeService) PodSandboxStats(ctx context.Context, podSandboxID string) (*runtimeapi.PodSandboxStats, error) {
	return f.PodSandboxStatsF(ctx, podSandboxID)
}

func (f *fakeRuntimeService) ListPodSandboxStats(ctx context.Context, filter *runtimeapi.PodSandboxStatsFilter) ([]*runtimeapi.PodSandboxStats, error) {
	return f.ListPodSandboxStatsF(ctx, filter)
}

func (f *fakeRuntimeService) ListMetricDescriptors(ctx context.Context) ([]*runtimeapi.MetricDescriptor, error) {
	return f.ListMetricDescriptorsF(ctx)
}

func (f *fakeRuntimeService) ListPodSandboxMetrics(ctx context.Context) ([]*runtimeapi.PodSandboxMetrics, error) {
	return f.ListPodSandboxMetricsF(ctx)
}

func (f *fakeRuntimeService) UpdateRuntimeConfig(ctx context.Context, runtimeConfig *runtimeapi.RuntimeConfig) error {
	return f.UpdateRuntimeConfigF(ctx, runtimeConfig)
}

func (f *fakeRuntimeService) Status(ctx context.Context, verbose bool) (*runtimeapi.StatusResponse, error) {
	return f.StatusF(ctx, verbose)
}

func (f *fakeRuntimeService) RuntimeConfig(ctx context.Context) (*runtimeapi.RuntimeConfigResponse, error) {
	return f.RuntimeConfigF(ctx)
}

func (f *fakeRuntimeService) ImageFsInfo(ctx context.Context) (*runtimeapi.ImageFsInfoResponse, error) {
	return f.ImageFsInfoF(ctx)
}
