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
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	mock "github.com/kubeedge/kubeedge/pkg/containers/testing"
)

func TestCopyResources(t *testing.T) {
	tests := []struct {
		name         string
		cgroupDriver string
		image        string
		files        map[string]string
		setupMock    func(t *testing.T, m *mock.MockRuntimeService)
		wantErr      bool
	}{
		{
			name:         "Success path: copy single file with command verification",
			cgroupDriver: v1alpha2.CGroupDriverCGroupFS,
			image:        "kubeedge/pause:3.1",
			files:        map[string]string{"/tmp/src": "/usr/local/bin/dest"},
			setupMock: func(t *testing.T, m *mock.MockRuntimeService) {
				var logDirectory string
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, config *runtimeapi.PodSandboxConfig, _ string) {
						logDirectory = config.LogDirectory
						if !filepath.IsAbs(logDirectory) {
							t.Errorf("LogDirectory must be absolute, got %q", logDirectory)
						}
					}).Return("sb-123", nil)
				m.EXPECT().CreateContainer(gomock.Any(), "sb-123", gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, _ string, config *runtimeapi.ContainerConfig, sandboxConfig *runtimeapi.PodSandboxConfig) {
						if config.LogPath != filepath.Join(copyResourcesContainerName, "0.log") {
							t.Errorf("unexpected LogPath %q", config.LogPath)
						}
						if sandboxConfig.LogDirectory != logDirectory {
							t.Errorf("sandbox LogDirectory changed: got %q, want %q", sandboxConfig.LogDirectory, logDirectory)
						}
					}).Return("cnt-123", nil)
				m.EXPECT().StartContainer(gomock.Any(), "cnt-123").Return(nil)
				m.EXPECT().ExecSync(gomock.Any(), "cnt-123", gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, _ string, cmd []string, _ time.Duration) {
						expected := "cp /tmp/src /tmp/usr/local/bin/dest"
						if cmd[2] != expected {
							t.Errorf("expected command %s, got %s", expected, cmd[2])
						}
					}).Return([]byte("stdout"), []byte(""), nil)
				m.EXPECT().StopContainer(gomock.Any(), "cnt-123", copyResourcesStopTimeout).Return(nil)
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-123").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-123").Return(nil)
			},
		},
		{
			name:         "Success path: systemd cgroup driver validation",
			cgroupDriver: v1alpha2.CGroupDriverSystemd,
			image:        "kubeedge/pause:3.1",
			files:        map[string]string{"/src": "/dest"},
			setupMock: func(t *testing.T, m *mock.MockRuntimeService) {
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(_ context.Context, config *runtimeapi.PodSandboxConfig, _ string) {
						if goruntime.GOOS == "linux" && (config.Linux == nil || config.Linux.CgroupParent == "") {
							t.Errorf("CgroupParent should be set for systemd driver")
						}
					}).Return("sb-systemd", nil)
				m.EXPECT().CreateContainer(gomock.Any(), "sb-systemd", gomock.Any(), gomock.Any()).Return("cnt-systemd", nil)
				m.EXPECT().StartContainer(gomock.Any(), "cnt-systemd").Return(nil)
				m.EXPECT().ExecSync(gomock.Any(), "cnt-systemd", gomock.Any(), gomock.Any()).Return(nil, nil, nil)
				m.EXPECT().StopContainer(gomock.Any(), "cnt-systemd", copyResourcesStopTimeout).Return(nil)
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-systemd").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-systemd").Return(nil)
			},
		},
		{
			name:    "Failure: Sandbox creation fails",
			image:   "kubeedge/pause:3.1",
			wantErr: true,
			setupMock: func(t *testing.T, m *mock.MockRuntimeService) {
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).Return("", fmt.Errorf("CRI error"))
			},
		},
		{
			name:    "Failure: StartContainer fails",
			image:   "kubeedge/pause:3.1",
			wantErr: true,
			setupMock: func(t *testing.T, m *mock.MockRuntimeService) {
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).Return("sb-fail", nil)
				m.EXPECT().CreateContainer(gomock.Any(), "sb-fail", gomock.Any(), gomock.Any()).Return("cnt-fail", nil)
				m.EXPECT().StartContainer(gomock.Any(), "cnt-fail").Return(fmt.Errorf("internal error"))
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-fail").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-fail").Return(nil)
			},
		},
		{
			name:    "Success: cleanup error after copy is nonfatal",
			image:   "kubeedge/pause:3.1",
			files:   map[string]string{"/src": "/dest"},
			wantErr: false,
			setupMock: func(t *testing.T, m *mock.MockRuntimeService) {
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).Return("sb-cleanup", nil)
				m.EXPECT().CreateContainer(gomock.Any(), "sb-cleanup", gomock.Any(), gomock.Any()).Return("cnt-cleanup", nil)
				m.EXPECT().StartContainer(gomock.Any(), "cnt-cleanup").Return(nil)
				m.EXPECT().ExecSync(gomock.Any(), "cnt-cleanup", gomock.Any(), gomock.Any()).Return(nil, nil, nil)
				m.EXPECT().StopContainer(gomock.Any(), "cnt-cleanup", copyResourcesStopTimeout).Return(fmt.Errorf("stream terminated by RST_STREAM"))
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-cleanup").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-cleanup").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockSvc := mock.NewMockRuntimeService(ctrl)
			podLogsDirectory := t.TempDir()
			runtime := &ContainerRuntimeImpl{
				cgroupDriver:     tt.cgroupDriver,
				podLogsDirectory: podLogsDirectory,
				ctrsvc:           mockSvc,
			}

			tt.setupMock(t, mockSvc)
			err := runtime.CopyResources(context.Background(), tt.image, tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyResources() error = %v, wantErr %v", err, tt.wantErr)
			}
			entries, readErr := os.ReadDir(podLogsDirectory)
			if readErr != nil {
				t.Fatalf("read pod logs directory: %v", readErr)
			}
			if len(entries) != 0 {
				t.Errorf("resource copy log directory was not cleaned: %v", entries)
			}
		})
	}
}

func TestCopyResourcesCleanupUsesIndependentContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mock.NewMockRuntimeService(ctrl)
	runtime := &ContainerRuntimeImpl{
		podLogsDirectory: t.TempDir(),
		ctrsvc:           mockSvc,
	}

	ctx, cancel := context.WithCancel(context.Background())
	mockSvc.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).Return("sb-context", nil)
	mockSvc.EXPECT().CreateContainer(gomock.Any(), "sb-context", gomock.Any(), gomock.Any()).Return("cnt-context", nil)
	mockSvc.EXPECT().StartContainer(gomock.Any(), "cnt-context").Return(nil)
	mockSvc.EXPECT().ExecSync(gomock.Any(), "cnt-context", gomock.Any(), gomock.Any()).
		Do(func(context.Context, string, []string, time.Duration) {
			cancel()
		}).Return(nil, nil, nil)
	stopCall := mockSvc.EXPECT().StopContainer(gomock.Any(), "cnt-context", copyResourcesStopTimeout).
		Do(func(cleanupCtx context.Context, _ string, _ int64) {
			if err := cleanupCtx.Err(); err != nil {
				t.Errorf("cleanup context inherited cancellation: %v", err)
			}
		}).Return(nil)
	removeCall := mockSvc.EXPECT().RemoveContainer(gomock.Any(), "cnt-context").Return(nil).After(stopCall)
	mockSvc.EXPECT().RemovePodSandbox(gomock.Any(), "sb-context").Return(nil).After(removeCall)

	if err := runtime.CopyResources(ctx, "kubeedge/pause:3.1", map[string]string{"/src": "/dest"}); err != nil {
		t.Fatalf("CopyResources() error = %v", err)
	}
}

func Test_copyResourcesCmd(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		check func(string) bool
	}{
		{
			name:  "Single file command",
			files: map[string]string{"/image/path": "/host/path"},
			check: func(res string) bool {
				return res == "cp /image/path /tmp/host/path"
			},
		},
		{
			name: "Multiple files command",
			files: map[string]string{
				"/src1": "/dest1",
				"/src2": "/dest2",
			},
			check: func(res string) bool {
				return strings.Contains(res, "cp /src1 /tmp/dest1") &&
					strings.Contains(res, "cp /src2 /tmp/dest2") &&
					strings.Contains(res, " && ")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copyResourcesCmd(tt.files)
			if !tt.check(got) {
				t.Errorf("copyResourcesCmd() = %v, check failed", got)
			}
		})
	}
}
