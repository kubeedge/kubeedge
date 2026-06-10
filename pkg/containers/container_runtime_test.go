package containers

import (
	"context"
	"errors"
	"testing"
	"time"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type mockRuntimeService struct {
	callOrder []string
	execErr   error
	startErr  error
}

func (m *mockRuntimeService) StopContainer(_ context.Context, _ string, _ int64) error {
	m.callOrder = append(m.callOrder, "stop")
	return nil
}
func (m *mockRuntimeService) RemoveContainer(_ context.Context, _ string) error {
	m.callOrder = append(m.callOrder, "remove")
	return nil
}
func (m *mockRuntimeService) CreateContainer(_ context.Context, _ string, _ *runtimeapi.ContainerConfig, _ *runtimeapi.PodSandboxConfig) (string, error) {
	return "fake-container-id", nil
}
func (m *mockRuntimeService) StartContainer(_ context.Context, _ string) error {
	return m.startErr
}
func (m *mockRuntimeService) ExecSync(_ context.Context, _ string, _ []string, _ time.Duration) ([]byte, []byte, error) {
	return nil, nil, m.execErr
}
func (m *mockRuntimeService) RunPodSandbox(_ context.Context, _ *runtimeapi.PodSandboxConfig, _ string) (string, error) {
	return "fake-sandbox-id", nil
}
func (m *mockRuntimeService) StopPodSandbox(_ context.Context, _ string) error   { return nil }
func (m *mockRuntimeService) RemovePodSandbox(_ context.Context, _ string) error { return nil }
func (m *mockRuntimeService) PodSandboxStatus(_ context.Context, _ string, _ bool) (*runtimeapi.PodSandboxStatusResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) ListPodSandbox(_ context.Context, _ *runtimeapi.PodSandboxFilter) ([]*runtimeapi.PodSandbox, error) {
	return nil, nil
}
func (m *mockRuntimeService) PortForward(_ context.Context, _ *runtimeapi.PortForwardRequest) (*runtimeapi.PortForwardResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) ListContainers(_ context.Context, _ *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
	return nil, nil
}
func (m *mockRuntimeService) ContainerStatus(_ context.Context, _ string, _ bool) (*runtimeapi.ContainerStatusResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) UpdateContainerResources(_ context.Context, _ string, _ *runtimeapi.ContainerResources) error {
	return nil
}
func (m *mockRuntimeService) Exec(_ context.Context, _ *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) Attach(_ context.Context, _ *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) ReopenContainerLog(_ context.Context, _ string) error { return nil }
func (m *mockRuntimeService) CheckpointContainer(_ context.Context, _ *runtimeapi.CheckpointContainerRequest) error {
	return nil
}
func (m *mockRuntimeService) GetContainerEvents(_ context.Context, _ chan *runtimeapi.ContainerEventResponse, _ func(runtimeapi.RuntimeService_GetContainerEventsClient)) error {
	return nil
}
func (m *mockRuntimeService) Version(_ context.Context, _ string) (*runtimeapi.VersionResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) ContainerStats(_ context.Context, _ string) (*runtimeapi.ContainerStats, error) {
	return nil, nil
}
func (m *mockRuntimeService) ListContainerStats(_ context.Context, _ *runtimeapi.ContainerStatsFilter) ([]*runtimeapi.ContainerStats, error) {
	return nil, nil
}
func (m *mockRuntimeService) PodSandboxStats(_ context.Context, _ string) (*runtimeapi.PodSandboxStats, error) {
	return nil, nil
}
func (m *mockRuntimeService) ListPodSandboxStats(_ context.Context, _ *runtimeapi.PodSandboxStatsFilter) ([]*runtimeapi.PodSandboxStats, error) {
	return nil, nil
}
func (m *mockRuntimeService) ListMetricDescriptors(_ context.Context) ([]*runtimeapi.MetricDescriptor, error) {
	return nil, nil
}
func (m *mockRuntimeService) ListPodSandboxMetrics(_ context.Context) ([]*runtimeapi.PodSandboxMetrics, error) {
	return nil, nil
}
func (m *mockRuntimeService) UpdateRuntimeConfig(_ context.Context, _ *runtimeapi.RuntimeConfig) error {
	return nil
}
func (m *mockRuntimeService) Status(_ context.Context, _ bool) (*runtimeapi.StatusResponse, error) {
	return nil, nil
}
func (m *mockRuntimeService) RuntimeConfig(_ context.Context) (*runtimeapi.RuntimeConfigResponse, error) {
	return nil, nil
}

func newTestRuntime(mock *mockRuntimeService) *ContainerRuntimeImpl {
	return &ContainerRuntimeImpl{ctrsvc: mock}
}

func TestStopCalledBeforeRemoveOnSuccess(t *testing.T) {
	mock := &mockRuntimeService{}
	rt := newTestRuntime(mock)
	_ = rt.CopyResources(context.Background(), "fake-image", map[string]string{"/usr/bin/test": "/usr/bin/test"})

	if len(mock.callOrder) < 2 {
		t.Fatalf("expected at least stop and remove calls, got: %v", mock.callOrder)
	}
	stopIdx, removeIdx := -1, -1
	for i, c := range mock.callOrder {
		if c == "stop" {
			stopIdx = i
		}
		if c == "remove" {
			removeIdx = i
		}
	}
	if stopIdx == -1 {
		t.Error("StopContainer was never called")
	}
	if removeIdx == -1 {
		t.Error("RemoveContainer was never called")
	}
	if stopIdx > removeIdx {
		t.Errorf("StopContainer (pos %d) must be called before RemoveContainer (pos %d)", stopIdx, removeIdx)
	}
}

func TestStopCalledBeforeRemoveOnExecFailure(t *testing.T) {
	mock := &mockRuntimeService{execErr: errors.New("exec failed")}
	rt := newTestRuntime(mock)
	_ = rt.CopyResources(context.Background(), "fake-image", map[string]string{"/usr/bin/test": "/usr/bin/test"})

	stopIdx, removeIdx := -1, -1
	for i, c := range mock.callOrder {
		if c == "stop" {
			stopIdx = i
		}
		if c == "remove" {
			removeIdx = i
		}
	}
	if stopIdx == -1 {
		t.Error("StopContainer was never called on exec failure")
	}
	if removeIdx == -1 {
		t.Error("RemoveContainer was never called on exec failure")
	}
	if stopIdx > removeIdx {
		t.Errorf("StopContainer (pos %d) must be called before RemoveContainer (pos %d)", stopIdx, removeIdx)
	}
}
