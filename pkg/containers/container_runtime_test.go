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
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).Return("sb-123", nil)
				m.EXPECT().CreateContainer(gomock.Any(), "sb-123", gomock.Any(), gomock.Any()).Return("cnt-123", nil)
				m.EXPECT().StartContainer(gomock.Any(), "cnt-123").Return(nil)

				// Use .Do to verify the command manually instead of MatchedBy
				m.EXPECT().ExecSync(gomock.Any(), "cnt-123", gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, containerID string, cmd []string, timeout time.Duration) {
						expected := "cp '/tmp/src' '/tmp/usr/local/bin/dest'"
						if cmd[2] != expected {
							t.Errorf("expected command %s, got %s", expected, cmd[2])
						}
					}).Return([]byte("stdout"), []byte(""), nil)

				m.EXPECT().StopContainer(gomock.Any(), "cnt-123", int64(0)).Return(nil)
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-123").Return(nil)
				m.EXPECT().StopPodSandbox(gomock.Any(), "sb-123").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-123").Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "Success path: systemd cgroup driver validation",
			cgroupDriver: v1alpha2.CGroupDriverSystemd,
			image:        "kubeedge/pause:3.1",
			files:        map[string]string{"/src": "/dest"},
			setupMock: func(t *testing.T, m *mock.MockRuntimeService) {
				// Use .Do to verify CgroupParent manually
				m.EXPECT().RunPodSandbox(gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(ctx context.Context, config *runtimeapi.PodSandboxConfig, runtimeHandler string) {
						if config.Linux == nil || config.Linux.CgroupParent == "" {
							t.Errorf("CgroupParent should be set for systemd driver")
						}
					}).Return("sb-systemd", nil)

				m.EXPECT().CreateContainer(gomock.Any(), "sb-systemd", gomock.Any(), gomock.Any()).Return("cnt-systemd", nil)
				m.EXPECT().StartContainer(gomock.Any(), "cnt-systemd").Return(nil)
				m.EXPECT().ExecSync(gomock.Any(), "cnt-systemd", gomock.Any(), gomock.Any()).Return(nil, nil, nil)
				m.EXPECT().StopContainer(gomock.Any(), "cnt-systemd", int64(0)).Return(nil)
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-systemd").Return(nil)
				m.EXPECT().StopPodSandbox(gomock.Any(), "sb-systemd").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-systemd").Return(nil)
			},
			wantErr: false,
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
				m.EXPECT().StopContainer(gomock.Any(), "cnt-fail", int64(0)).Return(nil)
				m.EXPECT().RemoveContainer(gomock.Any(), "cnt-fail").Return(nil)
				m.EXPECT().StopPodSandbox(gomock.Any(), "sb-fail").Return(nil)
				m.EXPECT().RemovePodSandbox(gomock.Any(), "sb-fail").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockSvc := mock.NewMockRuntimeService(ctrl)

			runtime := &ContainerRuntimeImpl{
				cgroupDriver: tt.cgroupDriver,
				ctrsvc:       mockSvc,
			}

			tt.setupMock(t, mockSvc)

			err := runtime.CopyResources(context.Background(), tt.image, tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
				return res == "cp '/image/path' '/tmp/host/path'"
			},
		},
		{
			name: "Multiple files command",
			files: map[string]string{
				"/src1": "/dest1",
				"/src2": "/dest2",
			},
			check: func(res string) bool {
				// Map iteration is random, so check for both segments
				return strings.Contains(res, "cp '/src1' '/tmp/dest1'") &&
					strings.Contains(res, "cp '/src2' '/tmp/dest2'") &&
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
