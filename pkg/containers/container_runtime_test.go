/*
Copyright 2022 The KubeEdge Authors.

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

package containers

import (
	"context"
	"errors"
	"testing"
	"time"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type mockRuntimeService struct {
	calls               []string
	errRunPodSandbox    error
	errCreateContainer  error
	errStartContainer   error
	errExecSync         error
	errRemoveContainer  error
	errRemovePodSandbox error
	stopTimeout         int64
	stopCalled          bool
}

func (m *mockRuntimeService) Version(_ context.Context, _ string) (*runtimeapi.VersionResponse, error) {
	panic("unexpected call to Version")
}

func (m *mockRuntimeService) CreateContainer(_ context.Context, _ string, _ *runtimeapi.ContainerConfig, _ *runtimeapi.PodSandboxConfig) (string, error) {
	m.calls = append(m.calls, "CreateContainer")
	return "container-id", m.errCreateContainer
}

func (m *mockRuntimeService) StartContainer(_ context.Context, _ string) error {
	m.calls = append(m.calls, "StartContainer")
	return m.errStartContainer
}

func (m *mockRuntimeService) StopContainer(_ context.Context, _ string, timeout int64) error {
	m.calls = append(m.calls, "StopContainer")
	m.stopCalled = true
	m.stopTimeout = timeout
	return nil
}

func (m *mockRuntimeService) RemoveContainer(_ context.Context, _ string) error {
	m.calls = append(m.calls, "RemoveContainer")
	return m.errRemoveContainer
}

func (m *mockRuntimeService) ListContainers(_ context.Context, _ *runtimeapi.ContainerFilter) ([]*runtimeapi.Container, error) {
	panic("unexpected call to ListContainers")
}

func (m *mockRuntimeService) ContainerStatus(_ context.Context, _ string, _ bool) (*runtimeapi.ContainerStatusResponse, error) {
	panic("unexpected call to ContainerStatus")
}

func (m *mockRuntimeService) UpdateContainerResources(_ context.Context, _ string, _ *runtimeapi.ContainerResources) error {
	panic("unexpected call to UpdateContainerResources")
}

func (m *mockRuntimeService) ExecSync(_ context.Context, _ string, _ []string, _ time.Duration) ([]byte, []byte, error) {
	m.calls = append(m.calls, "ExecSync")
	if m.errExecSync != nil {
		return nil, []byte("stderr output"), m.errExecSync
	}
	return []byte(""), []byte(""), nil
}

func (m *mockRuntimeService) Exec(_ context.Context, _ *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
	panic("unexpected call to Exec")
}

func (m *mockRuntimeService) Attach(_ context.Context, _ *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error) {
	panic("unexpected call to Attach")
}

func (m *mockRuntimeService) ReopenContainerLog(_ context.Context, _ string) error {
	panic("unexpected call to ReopenContainerLog")
}

func (m *mockRuntimeService) CheckpointContainer(_ context.Context, _ *runtimeapi.CheckpointContainerRequest) error {
	panic("unexpected call to CheckpointContainer")
}

func (m *mockRuntimeService) GetContainerEvents(_ context.Context, _ chan *runtimeapi.ContainerEventResponse, _ func(runtimeapi.RuntimeService_GetContainerEventsClient)) error {
	panic("unexpected call to GetContainerEvents")
}

func (m *mockRuntimeService) RunPodSandbox(_ context.Context, _ *runtimeapi.PodSandboxConfig, _ string) (string, error) {
	m.calls = append(m.calls, "RunPodSandbox")
	return "sandbox-id", m.errRunPodSandbox
}

func (m *mockRuntimeService) StopPodSandbox(_ context.Context, _ string) error {
	m.calls = append(m.calls, "StopPodSandbox")
	return nil
}

func (m *mockRuntimeService) RemovePodSandbox(_ context.Context, _ string) error {
	m.calls = append(m.calls, "RemovePodSandbox")
	return m.errRemovePodSandbox
}

func (m *mockRuntimeService) PodSandboxStatus(_ context.Context, _ string, _ bool) (*runtimeapi.PodSandboxStatusResponse, error) {
	panic("unexpected call to PodSandboxStatus")
}

func (m *mockRuntimeService) ListPodSandbox(_ context.Context, _ *runtimeapi.PodSandboxFilter) ([]*runtimeapi.PodSandbox, error) {
	panic("unexpected call to ListPodSandbox")
}

func (m *mockRuntimeService) PortForward(_ context.Context, _ *runtimeapi.PortForwardRequest) (*runtimeapi.PortForwardResponse, error) {
	panic("unexpected call to PortForward")
}

func (m *mockRuntimeService) ContainerStats(_ context.Context, _ string) (*runtimeapi.ContainerStats, error) {
	panic("unexpected call to ContainerStats")
}

func (m *mockRuntimeService) ListContainerStats(_ context.Context, _ *runtimeapi.ContainerStatsFilter) ([]*runtimeapi.ContainerStats, error) {
	panic("unexpected call to ListContainerStats")
}

func (m *mockRuntimeService) PodSandboxStats(_ context.Context, _ string) (*runtimeapi.PodSandboxStats, error) {
	panic("unexpected call to PodSandboxStats")
}

func (m *mockRuntimeService) ListPodSandboxStats(_ context.Context, _ *runtimeapi.PodSandboxStatsFilter) ([]*runtimeapi.PodSandboxStats, error) {
	panic("unexpected call to ListPodSandboxStats")
}

func (m *mockRuntimeService) ListMetricDescriptors(_ context.Context) ([]*runtimeapi.MetricDescriptor, error) {
	panic("unexpected call to ListMetricDescriptors")
}

func (m *mockRuntimeService) ListPodSandboxMetrics(_ context.Context) ([]*runtimeapi.PodSandboxMetrics, error) {
	panic("unexpected call to ListPodSandboxMetrics")
}

func (m *mockRuntimeService) UpdateRuntimeConfig(_ context.Context, _ *runtimeapi.RuntimeConfig) error {
	panic("unexpected call to UpdateRuntimeConfig")
}

func (m *mockRuntimeService) Status(_ context.Context, _ bool) (*runtimeapi.StatusResponse, error) {
	panic("unexpected call to Status")
}

func (m *mockRuntimeService) RuntimeConfig(_ context.Context) (*runtimeapi.RuntimeConfigResponse, error) {
	panic("unexpected call to RuntimeConfig")
}

func indexOf(calls []string, name string) int {
	for i, c := range calls {
		if c == name {
			return i
		}
	}
	return -1
}

func newRuntimeWithMock(svc *mockRuntimeService) *ContainerRuntimeImpl {
	return &ContainerRuntimeImpl{
		cgroupDriver: "",
		ctrsvc:       svc,
		RuntimeImpl:  nil,
	}
}

func TestBug2_StopContainerCalledBeforeRemove(t *testing.T) {
	svc := &mockRuntimeService{}
	rt := newRuntimeWithMock(svc)
	files := map[string]string{
		"/usr/local/bin/edgecore": "/usr/local/bin/edgecore",
	}
	err := rt.CopyResources(context.Background(), "kubeedge/installation-package:latest", files)
	if err != nil {
		t.Fatalf("CopyResources returned unexpected error: %v", err)
	}
	if !svc.stopCalled {
		t.Fatal("BUG CONFIRMED: StopContainer was never called. RemoveContainer will fail on a running container, leaking it on the host.")
	}
	stopIdx := indexOf(svc.calls, "StopContainer")
	removeIdx := indexOf(svc.calls, "RemoveContainer")
	if stopIdx > removeIdx {
		t.Fatalf("StopContainer (pos %d) must be called BEFORE RemoveContainer (pos %d). Call order: %v", stopIdx, removeIdx, svc.calls)
	}
}

func TestBug2_StopContainerCalledOnExecSyncFailure(t *testing.T) {
	svc := &mockRuntimeService{
		errExecSync: errors.New("cp: cannot stat '/usr/local/bin/edgecore': No such file or directory"),
	}
	rt := newRuntimeWithMock(svc)
	files := map[string]string{
		"/usr/local/bin/edgecore": "/usr/local/bin/edgecore",
	}
	err := rt.CopyResources(context.Background(), "kubeedge/installation-package:latest", files)
	if err == nil {
		t.Fatal("expected CopyResources to return an error when ExecSync fails, but got nil")
	}
	if !svc.stopCalled {
		t.Fatal("BUG CONFIRMED: StopContainer not called on ExecSync failure. Running container leaked on host after copy error.")
	}
	stopIdx := indexOf(svc.calls, "StopContainer")
	removeIdx := indexOf(svc.calls, "RemoveContainer")
	if stopIdx > removeIdx {
		t.Fatalf("Cleanup ordering wrong on error path. StopContainer pos=%d, RemoveContainer pos=%d. Calls: %v", stopIdx, removeIdx, svc.calls)
	}
}

func TestBug2_StopCalledEvenWhenCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &mockRuntimeService{
		errExecSync: context.Canceled,
	}
	rt := newRuntimeWithMock(svc)
	cancel()
	files := map[string]string{
		"/usr/local/bin/edgecore": "/usr/local/bin/edgecore",
	}
	_ = rt.CopyResources(ctx, "kubeedge/installation-package:latest", files)
	if !svc.stopCalled {
		t.Fatal("BUG CONFIRMED: StopContainer not called when context is cancelled. Container leaked on cancellation path.")
	}
}

func TestBug2_FullCallSequenceHappyPath(t *testing.T) {
	svc := &mockRuntimeService{}
	rt := newRuntimeWithMock(svc)
	files := map[string]string{
		"/usr/local/bin/edgecore": "/usr/local/bin/edgecore",
	}
	err := rt.CopyResources(context.Background(), "kubeedge/installation-package:latest", files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{
		"RunPodSandbox",
		"CreateContainer",
		"StartContainer",
		"ExecSync",
		"StopContainer",
		"RemoveContainer",
		"RemovePodSandbox",
	}
	if len(svc.calls) != len(expected) {
		t.Fatalf("call count mismatch.\nwant: %v\ngot:  %v", expected, svc.calls)
	}
	for i, want := range expected {
		if svc.calls[i] != want {
			t.Errorf("call[%d]: want %q, got %q", i, want, svc.calls[i])
		}
	}
}
