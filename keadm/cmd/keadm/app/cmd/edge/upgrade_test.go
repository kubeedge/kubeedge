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

package edge

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/kubeedge/api/apis/common/constants"
	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func TestUpgradeRun(t *testing.T) {
	var releaseCalled bool

	commonpatches := gomonkey.NewPatches()
	defer commonpatches.Reset()

	commonpatches.ApplyMethodFunc(reflect.TypeOf(&upgrdeedge.JSONFileReporter{}), "Report",
		func(_err error) error {
			return nil
		})
	commonpatches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "release",
		func() {
			releaseCalled = true
		})

	t.Run("aborted the upgrade operation", func(t *testing.T) {
		releaseCalled = false
		var upgradeCalled int

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Scan",
			func() bool {
				fmt.Print("N\n")
				return false
			})
		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Text",
			func() string {
				return "n"
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "prerun",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "upgrade",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})

		cmd := NewUpgradeCommand()
		err := cmd.RunE(nil, nil)
		assert.NoError(t, err)
		assert.False(t, releaseCalled)
		assert.Equal(t, 0, upgradeCalled)
	})

	t.Run("agree to upgrade operation", func(t *testing.T) {
		releaseCalled = false
		var upgradeCalled int

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Scan",
			func() bool {
				return false
			})
		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Text",
			func() string {
				fmt.Print("y\n")
				return "y"
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "prerun",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "upgrade",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})

		cmd := NewUpgradeCommand()
		err := cmd.RunE(nil, nil)
		assert.NoError(t, err)
		assert.True(t, releaseCalled)
		assert.Equal(t, 2, upgradeCalled)
	})

	t.Run("force upgrade", func(t *testing.T) {
		releaseCalled = false
		var (
			upgradeCalled int
			scanCalled    bool
		)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Scan",
			func() bool {
				scanCalled = true
				return false
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "prerun",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "upgrade",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})

		cmd := NewUpgradeCommand()

		err := cmd.Flags().Set("force", "true")
		assert.NoError(t, err)

		err = cmd.RunE(nil, nil)
		assert.NoError(t, err)
		assert.False(t, scanCalled)
		assert.True(t, releaseCalled)
		assert.Equal(t, 2, upgradeCalled)
	})

	// For compatibility with historical versions, It will be removed in v1.23
	t.Run("also force upgrade when upgradeID not empty", func(t *testing.T) {
		releaseCalled = false
		var (
			upgradeCalled int
			scanCalled    bool
		)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&upgrdeedge.TaskEventReporter{}), "Report",
			func(_err error) error {
				return nil
			})
		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Scan",
			func() bool {
				scanCalled = true
				return false
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "prerun",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "upgrade",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})

		cmd := NewUpgradeCommand()

		err := cmd.Flags().Set("upgradeID", "test-job")
		assert.NoError(t, err)

		err = cmd.RunE(nil, nil)
		assert.NoError(t, err)
		assert.False(t, scanCalled)
		assert.True(t, releaseCalled)
		assert.Equal(t, 2, upgradeCalled)
	})

	t.Run("occupied error no need to release", func(t *testing.T) {
		releaseCalled = false
		var (
			upgradeCalled int
			scanCalled    bool
		)

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethodFunc(reflect.TypeOf(&bufio.Scanner{}), "Scan",
			func() bool {
				scanCalled = true
				return false
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "prerun",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return OccupiedError
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&upgradeExecutor{}), "upgrade",
			func(_opts UpgradeOptions) error {
				upgradeCalled++
				return nil
			})

		cmd := NewUpgradeCommand()

		err := cmd.Flags().Set("force", "true")
		assert.NoError(t, err)

		err = cmd.RunE(nil, nil)
		assert.ErrorIs(t, err, OccupiedError)
		assert.False(t, scanCalled)
		assert.False(t, releaseCalled)
		assert.Equal(t, 1, upgradeCalled)
	})
}

func TestUpgradeExecutorPreRun(t *testing.T) {
	executor := newUpgradeExecutor()
	opts := UpgradeOptions{
		BaseOptions: BaseOptions{
			Config: constants.EdgecoreConfigPath,
		},
	}

	t.Run("get current version successfully", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "prePreRun",
			func(_configpath string) error {
				executor.currentVersion = fakeCurrentVersion
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "postPreRun",
			func(_prerunHook string) error {
				return nil
			})

		err := executor.prerun(opts)
		assert.NoError(t, err)
	})
}

func TestUpgradeExecutorUpgrade(t *testing.T) {
	const (
		edgecoreBin = "/etc/kubeedge/v1.0.0/edgecore"
	)

	var fileChecked bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(getEdgeCoreBinary, func(_opts UpgradeOptions, _config *cfgv1alpha2.EdgeCoreConfig,
	) (string, error) {
		return edgecoreBin, nil
	})
	patches.ApplyFunc(os.RemoveAll, func(_path string) error {
		return nil
	})
	patches.ApplyFunc(util.KillKubeEdgeBinary, func(_proc string) error {
		return nil
	})
	patches.ApplyFunc(files.FileCopy, func(src, dst string) error {
		assert.Equal(t, edgecoreBin, src)
		assert.Equal(t, "/usr/local/bin/edgecore", dst)
		fileChecked = true
		return nil
	})
	patches.ApplyFunc(runEdgeCore, func() error {
		return nil
	})

	executor := newUpgradeExecutor()
	opts := UpgradeOptions{
		BaseOptions: BaseOptions{
			Config: constants.EdgecoreConfigPath,
		},
	}
	err := executor.upgrade(opts)
	assert.NoError(t, err)
	assert.True(t, fileChecked)
}

func TestGetEdgeCoreBinary(t *testing.T) {
	const wantHostPath = "/etc/kubeedge/upgrade/v1.1.0/edgecore"
	var checked int

	opts := UpgradeOptions{
		Image:     "kubeedge/installation-package",
		ToVersion: "v1.1.0",
		BaseOptions: BaseOptions{
			Config: constants.EdgecoreConfigPath,
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

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.NewContainerRuntime, func(endpoint, cgroupDriver string,
	) (util.ContainerRuntime, error) {
		assert.Equal(t, cfg.Modules.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint, endpoint)
		assert.Equal(t, cfg.Modules.Edged.TailoredKubeletConfig.CgroupDriver, cgroupDriver)
		checked++
		return &util.ContainerRuntimeImpl{}, nil
	})
	patches.ApplyMethodFunc(reflect.TypeOf(&util.ContainerRuntimeImpl{}), "PullImage",
		func(_ctx context.Context, image string, _authConfig *runtimeapi.AuthConfig, _sandboxConfig *runtimeapi.PodSandboxConfig) error {
			assert.Equal(t, opts.Image+":"+opts.ToVersion, image)
			checked++
			return nil
		})
	patches.ApplyMethodFunc(reflect.TypeOf(&util.ContainerRuntimeImpl{}), "CopyResources",
		func(_ctx context.Context, edgeImage string, files map[string]string) error {
			assert.Equal(t, opts.Image+":"+opts.ToVersion, edgeImage)
			hostpath, ok := files["/usr/local/bin/edgecore"]
			assert.True(t, ok)
			assert.Equal(t, wantHostPath, hostpath)
			checked++
			return nil
		})

	path, err := getEdgeCoreBinary(opts, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 3, checked)
	assert.Equal(t, wantHostPath, path)
}
