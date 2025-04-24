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

package actions

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	operationsv1alpha2 "github.com/kubeedge/api/apis/operations/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/cmd/edgecore/app/options"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/dbclient"
	"github.com/kubeedge/kubeedge/pkg/containers"
	taskmsg "github.com/kubeedge/kubeedge/pkg/nodetask/message"
	upgradeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
)

func TestNodeUpgradeJobPreRun(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.NodeUpgradeJobSpec{},
	}
	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}

	var saveUpgradeSpecCalled bool
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.Upgrade)(nil)), "Save",
		func(jobname, nodename string, spec *operationsv1alpha2.NodeUpgradeJobSpec) error {
			saveUpgradeSpecCalled = true
			return nil
		})

	err := h.preRun(ctx, "", "", "", specser)
	require.NoError(t, err)
	assert.True(t, saveUpgradeSpecCalled)
}

func TestNodeUpgradeJobPostRun(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.NodeUpgradeJobSpec{},
	}
	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}

	var deleteUpgradeSpecCalled bool
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethodFunc(reflect.TypeOf((*dbclient.Upgrade)(nil)), "Delete",
		func() error {
			deleteUpgradeSpecCalled = true
			return nil
		})

	err := h.postRun(ctx, "", "", "", specser)
	require.NoError(t, err)
	assert.True(t, deleteUpgradeSpecCalled)
}

func TestNodeUpgradeJobCheckItems(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.NodeUpgradeJobSpec{
			CheckItems: []string{"cpu", "mem", "disk"},
			Image:      "kubeedge/installation-package",
			Version:    "v1.21.0",
		},
	}
	cfg := &cfgv1alpha2.EdgeCoreConfig{
		Modules: &cfgv1alpha2.Modules{
			Edged: &cfgv1alpha2.Edged{
				TailoredKubeletConfig: &cfgv1alpha2.TailoredKubeletConfiguration{
					ContainerRuntimeEndpoint: "unix:///var/run/containerd/containerd.sock",
					CgroupDriver:             "systemd",
				},
			},
		},
	}
	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}

	t.Run("check items failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(PreCheck, func([]string) error {
			return errors.New("test error")
		})

		resp := h.checkItems(ctx, "", "", specser)
		assert.EqualError(t, resp.Error(), "test error")
	})

	t.Run("check items success", func(t *testing.T) {
		var pullImageCalled, copyResourcesCalled bool
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(PreCheck, func([]string) error {
			return nil
		})
		patches.ApplyFunc(options.GetEdgeCoreConfig, func() *cfgv1alpha2.EdgeCoreConfig {
			return cfg
		})
		patches.ApplyFunc(containers.NewContainerRuntime, func(endpoint, cgroupDriver string,
		) (containers.ContainerRuntime, error) {
			assert.Equal(t, cfg.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint, endpoint)
			assert.Equal(t, cfg.Modules.Edged.TailoredKubeletConfig.CgroupDriver, cgroupDriver)
			return &containers.ContainerRuntimeImpl{}, nil
		})
		patches.ApplyMethodFunc(reflect.TypeOf(&containers.ContainerRuntimeImpl{}), "PullImage",
			func(_ctx context.Context, image string, _authConfig *runtimeapi.AuthConfig, _sandboxConfig *runtimeapi.PodSandboxConfig) error {
				pullImageCalled = true
				assert.Equal(t, "kubeedge/installation-package:v1.21.0", image)
				return nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf(&containers.ContainerRuntimeImpl{}), "CopyResources",
			func(_ctx context.Context, edgeImage string, files map[string]string) error {
				copyResourcesCalled = true
				assert.Equal(t, "kubeedge/installation-package:v1.21.0", edgeImage)
				hostpath, ok := files["/usr/local/bin/keadm"]
				assert.True(t, ok)
				assert.Equal(t, "/usr/local/bin/keadm", hostpath)
				return nil
			})

		resp := h.checkItems(ctx, "", "", specser)
		require.NoError(t, resp.Error())
		assert.True(t, pullImageCalled)
		assert.True(t, copyResourcesCalled)
	})
}

func TestNodeUpgradeJobWaitingConfirmation(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.NodeUpgradeJobSpec{
			RequireConfirmation: true,
		},
	}

	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}
	resp := h.waitingConfirmation(ctx, "", "", specser)
	assert.NoError(t, resp.Error())
	assert.True(t, resp.NeedInterrupt())
}

func TestNodeUpgradeJobBackup(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{
		spec: &operationsv1alpha2.NodeUpgradeJobSpec{},
	}
	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}

	t.Run("exec backup command failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
			func(_cmd *execs.Command) error {
				return errors.New("test run command error")
			})

		resp := h.backup(ctx, "", "", specser)
		require.ErrorContains(t, resp.Error(), "test run command error")
	})

	t.Run("exec backup command failed", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
			func(_cmd *execs.Command) error {
				return errors.New("test run command error")
			})

		resp := h.backup(ctx, "", "", specser)
		require.ErrorContains(t, resp.Error(), "test run command error")
	})

	t.Run("backup command reports a successful result", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
			func(cmd *execs.Command) error {
				assert.Equal(t, "bash -c keadm backup edge", cmd.GetCommand())
				return nil
			})
		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{
				Success: true,
			}, nil
		})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			return nil
		})

		resp := h.backup(ctx, "", "", specser)
		require.NoError(t, resp.Error())
	})

	t.Run("backup command reports a failed result", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
			func(cmd *execs.Command) error {
				assert.Equal(t, "bash -c keadm backup edge", cmd.GetCommand())
				return nil
			})
		patches.ApplyFunc(upgradeedge.ParseJSONReporterInfo, func() (upgradeedge.JSONReporterInfo, error) {
			return upgradeedge.JSONReporterInfo{
				Success: false,
			}, nil
		})
		patches.ApplyFunc(upgradeedge.RemoveJSONReporterInfo, func() error {
			return nil
		})

		resp := h.backup(ctx, "", "", specser)
		require.ErrorContains(t, resp.Error(), "keadm backup failed")
	})
}

func TestNodeUpgradeJobUpgrade(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{}
	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}

	t.Run("get spec failed", func(t *testing.T) {
		resp := h.upgrade(ctx, "", "", specser)
		require.ErrorContains(t, resp.Error(), "failed to conv spec to NodeUpgradeJobSpec, actual type <nil>")
		require.True(t, resp.NeedInterrupt())
	})

	t.Run("standard upgrade command", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
			func(cmd *execs.Command) error {
				assert.Equal(t, "bash -c keadm upgrade edge --force --toVersion 1.21.0 >> /tmp/keadm.log 2>&1", cmd.GetCommand())
				return nil
			})

		specser.spec = &operationsv1alpha2.NodeUpgradeJobSpec{
			Version: "1.21.0",
		}
		resp := h.upgrade(ctx, "", "", specser)
		require.NoError(t, resp.Error())
	})

	t.Run("custom image repository upgrade command", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
			func(cmd *execs.Command) error {
				assert.Equal(t, "bash -c keadm upgrade edge --force --toVersion 1.21.0 --image custom.com/kubeedge/installation-package >> /tmp/keadm.log 2>&1", cmd.GetCommand())
				return nil
			})

		specser.spec = &operationsv1alpha2.NodeUpgradeJobSpec{
			Version: "1.21.0",
			Image:   "custom.com/kubeedge/installation-package",
		}
		resp := h.upgrade(ctx, "", "", specser)
		require.NoError(t, resp.Error())
	})
}

func TestNodeUpgradeJobRollback(t *testing.T) {
	ctx := context.TODO()
	specser := &cachedSpecSerializer{}
	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf((*execs.Command)(nil)), "Exec",
		func(cmd *execs.Command) error {
			assert.Equal(t, "bash -c keadm rollback edge >> /tmp/keadm.log 2>&1", cmd.GetCommand())
			return nil
		})

	resp := h.rollback(ctx, "", "", specser)
	require.NoError(t, resp.Error())
}

func TestNodeUpgradeJobReportActionStatus(t *testing.T) {
	var (
		jobName  = "test-job"
		nodeName = "node1"
	)
	cases := []struct {
		name        string
		resp        ActionResponse
		action      string
		extendEmpty bool
	}{
		{
			name:        "check successful",
			resp:        &nodeUpgradeJobActionResponse{},
			action:      string(operationsv1alpha2.NodeUpgradeJobActionCheck),
			extendEmpty: true,
		},
		{
			name: "upgrade failed",
			resp: &nodeUpgradeJobActionResponse{
				FromVersion: "v1.20.0",
				ToVersion:   "v1.21.0",
				baseActionResponse: baseActionResponse{
					err: errors.New("test error"),
				},
			},
			action: string(operationsv1alpha2.NodeUpgradeJobActionUpgrade),
		},
		{
			name: "rollback successful",
			resp: &nodeUpgradeJobActionResponse{
				FromVersion: "v1.21.0",
				ToVersion:   "v1.20.0",
			},
			action: string(operationsv1alpha2.NodeUpgradeJobActionBackUp),
		},
	}

	h := nodeUpgradeJobActionHandler{
		logger: klog.Background(),
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(message.ReportNodeTaskStatus, func(_res taskmsg.Resource, msgbody taskmsg.UpstreamMessage) {
				assert.Equal(t, c.action, msgbody.Action)
				if c.extendEmpty {
					assert.Empty(t, msgbody.Extend)
				} else {
					assert.NotEmpty(t, msgbody.Extend)
				}
				if c.resp.Error() == nil {
					assert.True(t, msgbody.Succ)
				} else {
					assert.False(t, msgbody.Succ)
					assert.Equal(t, c.resp.Error().Error(), msgbody.Reason)
				}
			})

			h.reportActionStatus(jobName, nodeName, c.action, c.resp)
		})
	}
}
