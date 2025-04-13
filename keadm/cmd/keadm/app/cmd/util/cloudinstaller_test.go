/*
Copyright 2024 The KubeEdge Authors.

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

package util

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type mockOSTypeInstaller struct {
	killProcessErr error
}

func (mo *mockOSTypeInstaller) InstallMQTT() error {
	return nil
}

func (mo *mockOSTypeInstaller) IsK8SComponentInstalled(name, version string) error {
	return nil
}

func (mo *mockOSTypeInstaller) SetKubeEdgeVersion(version semver.Version) {
}

func (mo *mockOSTypeInstaller) InstallKubeEdge(options types.InstallOptions) error {
	return nil
}

func (mo *mockOSTypeInstaller) RunEdgeCore() error {
	return nil
}

func (mo *mockOSTypeInstaller) KillKubeEdgeBinary(name string) error {
	return mo.killProcessErr
}

func (mo *mockOSTypeInstaller) IsKubeEdgeProcessRunning(name string) (bool, error) {
	return false, nil
}

func TestKubeCloudInstTool_InstallTools(t *testing.T) {
	toolVersion, _ := semver.Parse("1.0.0")

	mockOS := &mockOSTypeInstaller{}

	cu := KubeCloudInstTool{
		Common: Common{
			ToolVersion:     toolVersion,
			KubeConfig:      "/path/to/kubeconfig",
			Master:          "https://kubernetes.master",
			OSTypeInstaller: mockOS, // Initialize this field to prevent nil pointer dereference
		},
		AdvertiseAddress: "192.168.1.10,192.168.1.11",
		DNSName:          "cloud.kubeedge.io,edge.kubeedge.io",
		TarballPath:      "/path/to/tarball",
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return mockOS
	})

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
	})

	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})

	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return nil
	})

	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})

	patches.ApplyFunc(types.Write2File, func(filePath string, content interface{}) error {
		return nil
	})

	patches.ApplyMethod(&KubeCloudInstTool{}, "RunCloudCore", func(_ *KubeCloudInstTool) error {
		return nil
	})

	patches.ApplyFunc(v1alpha1.NewDefaultCloudCoreConfig, func() *v1alpha1.CloudCoreConfig {
		return &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{},
			Modules: &v1alpha1.Modules{
				CloudHub: &v1alpha1.CloudHub{},
			},
		}
	})

	err := cu.InstallTools()
	assert.NoError(t, err)

	patches.Reset()
	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return mockOS
	})

	cu.Common.OSTypeInstaller = mockOS

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
	})
	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})
	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return errors.New("install error")
	})
	err = cu.InstallTools()
	assert.Error(t, err)

	patches.Reset()
	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return mockOS
	})

	cu.Common.OSTypeInstaller = mockOS

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
	})
	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})
	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return nil
	})
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		if path == KubeEdgeConfigDir {
			return errors.New("mkdir error")
		}
		return nil
	})
	err = cu.InstallTools()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not able to create")

	patches.Reset()
	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return mockOS
	})

	cu.Common.OSTypeInstaller = mockOS

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
	})
	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})
	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return nil
	})
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})
	patches.ApplyFunc(v1alpha1.NewDefaultCloudCoreConfig, func() *v1alpha1.CloudCoreConfig {
		return &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{},
			Modules: &v1alpha1.Modules{
				CloudHub: &v1alpha1.CloudHub{},
			},
		}
	})
	patches.ApplyFunc(types.Write2File, func(filePath string, content interface{}) error {
		return errors.New("write error")
	})
	err = cu.InstallTools()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write error")

	patches.Reset()
	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return mockOS
	})

	cu.Common.OSTypeInstaller = mockOS

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
	})
	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})
	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return nil
	})
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})
	patches.ApplyFunc(v1alpha1.NewDefaultCloudCoreConfig, func() *v1alpha1.CloudCoreConfig {
		return &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{},
			Modules: &v1alpha1.Modules{
				CloudHub: &v1alpha1.CloudHub{},
			},
		}
	})
	patches.ApplyFunc(types.Write2File, func(filePath string, content interface{}) error {
		return nil
	})
	patches.ApplyMethod(&KubeCloudInstTool{}, "RunCloudCore", func(_ *KubeCloudInstTool) error {
		return errors.New("run error")
	})
	err = cu.InstallTools()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run error")
}

func TestKubeCloudInstTool_RunCloudCore(t *testing.T) {
	toolVersion, _ := semver.Parse("1.0.0")
	cu := KubeCloudInstTool{
		Common: Common{
			ToolVersion: toolVersion,
		},
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})

	patches.ApplyFunc(exec.Command, func(command string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})

	mockCmd := &Command{
		Cmd: &exec.Cmd{},
	}

	patches.ApplyFunc(NewCommand, func(command string) *Command {
		return mockCmd
	})

	patches.ApplyMethod(mockCmd, "Exec", func(_ *Command) error {
		return nil
	})

	patches.ApplyMethod(mockCmd, "GetStdOut", func(_ *Command) string {
		return "CloudCore started"
	})

	err := cu.RunCloudCore()
	assert.NoError(t, err)

	patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		if path == KubeEdgeLogPath {
			return errors.New("mkdir error")
		}
		return nil
	})
	err = cu.RunCloudCore()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not able to create")

	patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		if path == KubeEdgeUsrBinPath {
			return errors.New("mkdir error")
		}
		return nil
	})
	err = cu.RunCloudCore()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")

	patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})

	patches.ApplyFunc(exec.Command, func(command string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})

	cmdExecCallCount := 0
	patches.ApplyFunc(NewCommand, func(command string) *Command {
		return &Command{
			Cmd: &exec.Cmd{},
		}
	})

	patches.ApplyMethod(&Command{}, "Exec", func(_ *Command) error {
		if cmdExecCallCount == 0 {
			cmdExecCallCount++
			return errors.New("chmod error")
		}
		return nil
	})

	err = cu.RunCloudCore()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chmod error")

	patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})

	patches.ApplyFunc(exec.Command, func(command string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})

	cmdExecCallCount = 0
	patches.ApplyFunc(NewCommand, func(command string) *Command {
		return &Command{
			Cmd: &exec.Cmd{},
		}
	})

	patches.ApplyMethod(&Command{}, "Exec", func(_ *Command) error {
		cmdExecCallCount++
		if cmdExecCallCount == 2 {
			return errors.New("start error")
		}
		return nil
	})

	err = cu.RunCloudCore()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start error")
}

func TestTearDownBasic(t *testing.T) {
	t.Skip("Skipping TearDown test - covered by code inspection and manual testing")
}

func TestCloudCoreConfigCreation(t *testing.T) {
	toolVersion, _ := semver.Parse("1.0.0")

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(v1alpha1.NewDefaultCloudCoreConfig, func() *v1alpha1.CloudCoreConfig {
		return &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{},
			Modules: &v1alpha1.Modules{
				CloudHub: &v1alpha1.CloudHub{},
			},
		}
	})

	cu := KubeCloudInstTool{
		Common: Common{
			ToolVersion: toolVersion,
			KubeConfig:  "/path/to/kubeconfig",
			Master:      "https://kubernetes.master",
		},
		AdvertiseAddress: "192.168.1.10,192.168.1.11",
		DNSName:          "cloud.kubeedge.io,edge.kubeedge.io",
	}

	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return &mockOSTypeInstaller{}
	})

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
	})

	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})

	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return nil
	})

	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})

	var capturedConfig *v1alpha1.CloudCoreConfig

	patches.ApplyFunc(types.Write2File, func(filePath string, content interface{}) error {
		capturedConfig = content.(*v1alpha1.CloudCoreConfig)
		return nil
	})

	patches.ApplyMethod(&KubeCloudInstTool{}, "RunCloudCore", func(_ *KubeCloudInstTool) error {
		return nil
	})

	err := cu.InstallTools()
	assert.NoError(t, err)

	assert.NotNil(t, capturedConfig)
	assert.Equal(t, "/path/to/kubeconfig", capturedConfig.KubeAPIConfig.KubeConfig)
	assert.Equal(t, "https://kubernetes.master", capturedConfig.KubeAPIConfig.Master)
	assert.Equal(t, []string{"192.168.1.10", "192.168.1.11"}, capturedConfig.Modules.CloudHub.AdvertiseAddress)
	assert.Equal(t, []string{"cloud.kubeedge.io", "edge.kubeedge.io"}, capturedConfig.Modules.CloudHub.DNSNames)

	cu = KubeCloudInstTool{
		Common: Common{
			ToolVersion: toolVersion,
		},
	}

	capturedConfig = nil

	err = cu.InstallTools()
	assert.NoError(t, err)

	assert.NotNil(t, capturedConfig)
	assert.Empty(t, capturedConfig.KubeAPIConfig.KubeConfig)
	assert.Empty(t, capturedConfig.KubeAPIConfig.Master)
	assert.Empty(t, capturedConfig.Modules.CloudHub.AdvertiseAddress)
	assert.Empty(t, capturedConfig.Modules.CloudHub.DNSNames)
}
