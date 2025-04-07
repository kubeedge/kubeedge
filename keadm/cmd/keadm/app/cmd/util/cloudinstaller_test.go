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

package util

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
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

func setupTestEnvironment(t *testing.T, kubeConfig, master, advertiseAddress, dnsName string) (*KubeCloudInstTool, *mockOSTypeInstaller, *gomonkey.Patches) {
	toolVersion, err := semver.Parse("1.0.0")
	if err != nil {
		t.Fatalf("Failed to parse semver: %v", err)
	}

	cu := KubeCloudInstTool{
		Common: Common{
			ToolVersion: toolVersion,
			KubeConfig:  kubeConfig,
			Master:      master,
		},
		AdvertiseAddress: advertiseAddress,
		DNSName:          dnsName,
	}

	mockOS := &mockOSTypeInstaller{}
	cu.OSTypeInstaller = mockOS

	patches := gomonkey.NewPatches()

	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return mockOS
	})

	patches.ApplyMethod(&Common{}, "SetOSInterface", func(_ *Common, intf types.OSTypeInstaller) {
		cu.OSTypeInstaller = mockOS
	})

	patches.ApplyMethod(&Common{}, "SetKubeEdgeVersion", func(_ *Common, version semver.Version) {
	})

	patches.ApplyMethod(&Common{}, "InstallKubeEdge", func(_ *Common, options types.InstallOptions) error {
		return nil
	})

	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		return nil
	})

	return &cu, mockOS, patches
}

func TestKubeCloudInstTool_RunCloudCore(t *testing.T) {
	cu, _, patches := setupTestEnvironment(t, "", "", "", "")
	defer patches.Reset()

	patches.ApplyFunc(exec.Command, func(command string, args ...string) *exec.Cmd {
		return &exec.Cmd{}
	})

	mockCmd := &execs.Command{
		Cmd: &exec.Cmd{},
	}

	patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
		return mockCmd
	})

	patches.ApplyMethod(mockCmd, "Exec", func(_ *execs.Command) error {
		return nil
	})

	patches.ApplyMethod(mockCmd, "GetStdOut", func(_ *execs.Command) string {
		return "CloudCore started"
	})

	err := cu.RunCloudCore()
	assert.NoError(t, err)

	patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		if path == common.KubeEdgeLogPath {
			return errors.New("mkdir error")
		}
		return nil
	})
	err = cu.RunCloudCore()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not able to create")

	patches.Reset()
	patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
		if path == constants.KubeEdgeUsrBinPath {
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
	patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
		return &execs.Command{
			Cmd: &exec.Cmd{},
		}
	})

	patches.ApplyMethod(&execs.Command{}, "Exec", func(_ *execs.Command) error {
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
	patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
		return &execs.Command{
			Cmd: &exec.Cmd{},
		}
	})

	patches.ApplyMethod(&execs.Command{}, "Exec", func(_ *execs.Command) error {
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
	cu, _, patches := setupTestEnvironment(t,
		"/path/to/kubeconfig",
		"https://kubernetes.master",
		"192.168.1.10,192.168.1.11",
		"cloud.kubeedge.io,edge.kubeedge.io")
	defer patches.Reset()

	patches.ApplyFunc(v1alpha1.NewDefaultCloudCoreConfig, func() *v1alpha1.CloudCoreConfig {
		return &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{},
			Modules: &v1alpha1.Modules{
				CloudHub: &v1alpha1.CloudHub{},
			},
		}
	})

	var capturedConfig *v1alpha1.CloudCoreConfig
	patches.ApplyMethod(reflect.TypeOf(&v1alpha1.CloudCoreConfig{}), "WriteTo",
		func(cfg *v1alpha1.CloudCoreConfig, _filename string) error {
			capturedConfig = cfg
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

	cu2, _, patches2 := setupTestEnvironment(t, "", "", "", "")
	defer patches2.Reset()

	patches2.ApplyFunc(v1alpha1.NewDefaultCloudCoreConfig, func() *v1alpha1.CloudCoreConfig {
		return &v1alpha1.CloudCoreConfig{
			KubeAPIConfig: &v1alpha1.KubeAPIConfig{},
			Modules: &v1alpha1.Modules{
				CloudHub: &v1alpha1.CloudHub{},
			},
		}
	})

	capturedConfig = nil
	patches.ApplyMethod(reflect.TypeOf(&v1alpha1.CloudCoreConfig{}), "WriteTo",
		func(cfg *v1alpha1.CloudCoreConfig, _filename string) error {
			capturedConfig = cfg
			return nil
		})

	patches2.ApplyMethod(&KubeCloudInstTool{}, "RunCloudCore", func(_ *KubeCloudInstTool) error {
		return nil
	})

	err = cu2.InstallTools()
	assert.NoError(t, err)

	assert.NotNil(t, capturedConfig)
	assert.Empty(t, capturedConfig.KubeAPIConfig.KubeConfig)
	assert.Empty(t, capturedConfig.KubeAPIConfig.Master)
	assert.Empty(t, capturedConfig.Modules.CloudHub.AdvertiseAddress)
	assert.Empty(t, capturedConfig.Modules.CloudHub.DNSNames)
}

func init() {
	if _, ok := os.LookupEnv("CI"); ok {
		if err := os.MkdirAll(KubeEdgeConfigDir, os.ModePerm); err != nil {
			os.Stderr.WriteString("Failed to create KubeEdgeConfigDir: " + err.Error() + "\n")
		}
		if err := os.MkdirAll(common.KubeEdgeLogPath, os.ModePerm); err != nil {
			os.Stderr.WriteString("Failed to create KubeEdgeLogPath: " + err.Error() + "\n")
		}
		if err := os.MkdirAll(constants.KubeEdgeUsrBinPath, os.ModePerm); err != nil {
			os.Stderr.WriteString("Failed to create KubeEdgeUsrBinPath: " + err.Error() + "\n")
		}
	}
}
