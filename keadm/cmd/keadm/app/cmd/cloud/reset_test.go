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

package cloud

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/helm"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestNewCloudReset(t *testing.T) {
	assert := assert.New(t)

	cmd := NewCloudReset()

	assert.Equal("cloud", cmd.Use)
	assert.Equal("Teardowns CloudCore component", cmd.Short)
	assert.Equal(resetLongDescription, cmd.Long)
	assert.Equal(resetExample, cmd.Example)

	assert.NotNil(cmd.PreRunE)
	assert.NotNil(cmd.RunE)

	kubeconfigFlag := cmd.Flag(common.FlagNameKubeConfig)
	assert.Equal(common.DefaultKubeConfig, kubeconfigFlag.DefValue)
	assert.Equal(common.FlagNameKubeConfig, kubeconfigFlag.Name)

	forceFlag := cmd.Flag("force")
	assert.Equal("false", forceFlag.DefValue)
	assert.Equal("force", forceFlag.Name)
}

func TestAddResetFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}
	resetOpts := &common.ResetOptions{}

	addResetFlags(cmd, resetOpts)

	kubeconfigFlag := cmd.Flag(common.FlagNameKubeConfig)
	assert.NotNil(kubeconfigFlag)
	assert.Equal(common.DefaultKubeConfig, kubeconfigFlag.DefValue)
	assert.Equal(common.FlagNameKubeConfig, kubeconfigFlag.Name)

	forceFlag := cmd.Flag("force")
	assert.NotNil(forceFlag)
	assert.Equal("false", forceFlag.DefValue)
	assert.Equal("force", forceFlag.Name)
}

func TestTearDownCloudCore(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	tearDownCalled := false

	originalNewKubeCloudHelmInstTool := func(kubeConfig string) *helm.KubeCloudHelmInstTool {
		return &helm.KubeCloudHelmInstTool{
			Common: util.Common{
				KubeConfig: kubeConfig,
			},
		}
	}

	mockNewKubeCloudHelmInstTool := func(kubeConfig string) *helm.KubeCloudHelmInstTool {
		tool := originalNewKubeCloudHelmInstTool(kubeConfig)

		patches.ApplyMethodFunc(tool, "TearDown", func() error {
			tearDownCalled = true
			return nil
		})

		return tool
	}

	tempTearDownCloudCore := func(kubeConfig string) error {
		ke := mockNewKubeCloudHelmInstTool(kubeConfig)
		err := ke.TearDown()
		if err != nil {
			return fmt.Errorf("TearDown failed, err:%v", err)
		}
		return nil
	}

	err := tempTearDownCloudCore("fake-kubeconfig")
	assert.NoError(err)
	assert.True(tearDownCalled)

	patches.Reset()
	tearDownCalled = false

	mockNewKubeCloudHelmInstTool = func(kubeConfig string) *helm.KubeCloudHelmInstTool {
		tool := originalNewKubeCloudHelmInstTool(kubeConfig)

		patches.ApplyMethodFunc(tool, "TearDown", func() error {
			tearDownCalled = true
			return errors.New("teardown failed")
		})

		return tool
	}

	tempTearDownCloudCore = func(kubeConfig string) error {
		ke := mockNewKubeCloudHelmInstTool(kubeConfig)
		err := ke.TearDown()
		if err != nil {
			return fmt.Errorf("TearDown failed, err:%v", err)
		}
		return nil
	}

	err = tempTearDownCloudCore("fake-kubeconfig")
	assert.Error(err)
	assert.Contains(err.Error(), "TearDown failed")
	assert.True(tearDownCalled)
}

func TestPreRunE_NoComponentsRunning(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.CloudCoreRunningModuleV2,
		func(_ *common.ResetOptions) common.ModuleRunning {
			return common.NoneRunning
		})

	exitCalled := false
	exitCode := 0
	patches.ApplyFunc(os.Exit, func(code int) {
		exitCalled = true
		exitCode = code
	})

	cmd := NewCloudReset()

	_ = cmd.PreRunE(cmd, []string{})

	assert.True(exitCalled)
	assert.Equal(0, exitCode)
}

func TestPreRunE_CloudComponentRunning(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.CloudCoreRunningModuleV2,
		func(_ *common.ResetOptions) common.ModuleRunning {
			return common.KubeEdgeCloudRunning
		})

	exitCalled := false
	patches.ApplyFunc(os.Exit, func(code int) {
		exitCalled = true
	})

	cmd := NewCloudReset()

	err := cmd.PreRunE(cmd, []string{})

	assert.NoError(err)
	assert.False(exitCalled)
}

func TestPreRunE_EdgeComponentRunning(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.CloudCoreRunningModuleV2,
		func(_ *common.ResetOptions) common.ModuleRunning {
			return common.KubeEdgeEdgeRunning
		})

	exitCalled := false
	patches.ApplyFunc(os.Exit, func(code int) {
		exitCalled = true
	})

	cmd := NewCloudReset()

	err := cmd.PreRunE(cmd, []string{})

	assert.NoError(err)
	assert.False(exitCalled)
}

func TestRunE_ForceFlag(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	tearDownCalled := false
	patches.ApplyFunc(TearDownCloudCore,
		func(_ string) error {
			tearDownCalled = true
			return nil
		})

	cleanDirCalled := false
	patches.ApplyFunc(util.CleanDirectories,
		func(_ bool) error {
			cleanDirCalled = true
			return nil
		})

	cmd := NewCloudReset()
	err := cmd.Flags().Set("force", "true")
	assert.NoError(err)

	err = cmd.RunE(cmd, []string{})

	assert.NoError(err)
	assert.True(tearDownCalled)
	assert.True(cleanDirCalled)
}

func TestRunE_UserConfirmation(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()

	r, w, err := os.Pipe()
	assert.NoError(err)
	os.Stdin = r

	go func() {
		defer w.Close()
		_, err := w.Write([]byte("y\n"))
		assert.NoError(err)
	}()

	tearDownCalled := false
	patches.ApplyFunc(TearDownCloudCore,
		func(_ string) error {
			tearDownCalled = true
			return nil
		})

	cleanDirCalled := false
	patches.ApplyFunc(util.CleanDirectories,
		func(_ bool) error {
			cleanDirCalled = true
			return nil
		})

	cmd := NewCloudReset()

	err = cmd.RunE(cmd, []string{})

	assert.NoError(err)
	assert.True(tearDownCalled)
	assert.True(cleanDirCalled)
}

func TestRunE_UserRejection(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()

	r, w, err := os.Pipe()
	assert.NoError(err)
	os.Stdin = r

	go func() {
		defer w.Close()
		_, err := w.Write([]byte("n\n"))
		assert.NoError(err)
	}()

	tearDownCalled := false
	patches.ApplyFunc(TearDownCloudCore,
		func(_ string) error {
			tearDownCalled = true
			return nil
		})

	cleanDirCalled := false
	patches.ApplyFunc(util.CleanDirectories,
		func(_ bool) error {
			cleanDirCalled = true
			return nil
		})

	cmd := NewCloudReset()

	err = cmd.RunE(cmd, []string{})

	assert.Error(err)
	assert.Contains(err.Error(), "aborted reset operation")
	assert.False(tearDownCalled)
	assert.False(cleanDirCalled)
}

func TestRunE_TearDownError(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(TearDownCloudCore,
		func(_ string) error {
			return errors.New("teardown failed")
		})

	cleanDirCalled := false
	patches.ApplyFunc(util.CleanDirectories,
		func(_ bool) error {
			cleanDirCalled = true
			return nil
		})

	cmd := NewCloudReset()
	err := cmd.Flags().Set("force", "true")
	assert.NoError(err)

	err = cmd.RunE(cmd, []string{})

	assert.Error(err)
	assert.Contains(err.Error(), "teardown failed")
	assert.False(cleanDirCalled)
}

func TestRunE_CleanDirectoriesError(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	tearDownCalled := false
	patches.ApplyFunc(TearDownCloudCore,
		func(_ string) error {
			tearDownCalled = true
			return nil
		})

	patches.ApplyFunc(util.CleanDirectories,
		func(_ bool) error {
			return errors.New("clean failed")
		})

	cmd := NewCloudReset()
	err := cmd.Flags().Set("force", "true")
	assert.NoError(err)

	err = cmd.RunE(cmd, []string{})

	assert.Error(err)
	assert.True(tearDownCalled)
	assert.Contains(err.Error(), "clean failed")
}

func TestAddResetFlags_WithCustomValues(t *testing.T) {
	assert := assert.New(t)

	cmd := &cobra.Command{}
	resetOpts := &common.ResetOptions{
		Kubeconfig: "/custom/kubeconfig",
		Force:      true,
	}

	addResetFlags(cmd, resetOpts)

	err := cmd.Flags().Set(common.FlagNameKubeConfig, "/custom/kubeconfig")
	assert.NoError(err)

	err = cmd.Flags().Set("force", "true")
	assert.NoError(err)

	assert.Equal("/custom/kubeconfig", cmd.Flag(common.FlagNameKubeConfig).Value.String())
	assert.Equal("true", cmd.Flag("force").Value.String())
}
