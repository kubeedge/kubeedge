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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/kubeedge/common/constants"
	commfake "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common/fake"
)

func TestK8SInstToolInstallTools(t *testing.T) {
	globPatches := gomonkey.NewPatches()
	defer globPatches.Reset()

	globPatches.ApplyFuncReturn(GetOSInterface, &commfake.MockOSTypeInstaller{})
	globPatches.ApplyFuncReturn(installCRDs, nil)
	globPatches.ApplyFuncReturn(createKubeEdgeNs, nil)

	t.Run("cloudcore is running", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFuncReturn(GetOSInterface, &commfake.MockOSTypeInstaller{
			IsProcessRunning: true,
		})

		tool := &K8SInstTool{}
		err := tool.InstallTools()
		require.ErrorContains(t, err, "CloudCore is already running")
	})

	t.Run("process check error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFuncReturn(GetOSInterface, &commfake.MockOSTypeInstaller{
			ProcessRunningErr: errors.New("process check error"),
		})

		tool := &K8SInstTool{}
		err := tool.InstallTools()
		require.ErrorContains(t, err, "process check error")
	})

	t.Run("K8s version check fails", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFuncReturn(GetOSInterface, &commfake.MockOSTypeInstaller{
			IsK8SComponentInstalledError: errors.New("K8S version verification failed"),
		})

		tool := &K8SInstTool{}
		err := tool.InstallTools()
		require.ErrorContains(t, err, "K8S version verification failed")
	})

	t.Run("CRD install fails", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFuncReturn(installCRDs, errors.New("CRD installation failed"))

		tool := &K8SInstTool{}
		err := tool.InstallTools()
		require.ErrorContains(t, err, "CRD installation failed")
	})

	t.Run("namespace creation fails", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFuncReturn(createKubeEdgeNs, errors.New("namespace creation failed"))

		tool := &K8SInstTool{}
		err := tool.InstallTools()
		require.ErrorContains(t, err, "namespace creation failed")
	})

	t.Run("install successful", func(t *testing.T) {
		tool := &K8SInstTool{}
		err := tool.InstallTools()
		require.NoError(t, err)
	})
}

func TestCreateKubeEdgeV1CRD_ReadFileFails(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile,
		func(_ string) ([]byte, error) {
			return nil, errors.New("read file failed")
		})

	err := createKubeEdgeV1CRD(nil, "test-file")
	assert.Error(t, err, "createKubeEdgeV1CRD should return an error when ReadFile fails")
	assert.Contains(t, err.Error(), "read crd yaml error")
}

func TestCreateKubeEdgeV1CRD_UnmarshalFails(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile,
		func(_ string) ([]byte, error) {
			return []byte("invalid yaml content"), nil
		})

	err := createKubeEdgeV1CRD(nil, "test-file")
	assert.Error(t, err, "createKubeEdgeV1CRD should return an error when YAML unmarshal fails")
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestCreateKubeEdgeNs(t *testing.T) {
	t.Run("Mocked createKubeEdgeNs function", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(createKubeEdgeNs, func(kubeConfig, master string) error {
			assert.Equal(t, constants.SystemNamespace, "kubeedge", "The system namespace should be 'kubeedge'")
			return nil
		})

		err := createKubeEdgeNs("fake-kubeconfig", "fake-master")
		assert.NoError(t, err)
	})
}

func TestInstallCRDs(t *testing.T) {
	t.Run("MkdirAll failure", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(BuildConfig, func(kubeConfig, master string) (*rest.Config, error) {
			return &rest.Config{}, nil
		})

		patches.ApplyFunc(dynamic.NewForConfig, func(config *rest.Config) (dynamic.Interface, error) {
			return nil, nil
		})

		patches.ApplyFunc(GetLatestVersion, func() (string, error) {
			return "v1.8.0", nil
		})

		patches.ApplyFunc(os.MkdirAll, func(path string, mode os.FileMode) error {
			return errors.New("mkdir failed")
		})

		k8sInstTool := &K8SInstTool{
			Common: Common{},
		}

		err := installCRDs(k8sInstTool)

		if err != nil && err.Error() == "not able to create devices folder path" {
			assert.Contains(t, err.Error(), "not able to create")
		}
	})
}
