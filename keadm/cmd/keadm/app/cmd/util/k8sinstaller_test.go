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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/kubeedge/kubeedge/common/constants"
	commontypes "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

// setupTest creates and returns common test fixtures
func setupTest() (*gomonkey.Patches, *K8SInstTool, *MockOSTypeInstaller) {
	patches := gomonkey.NewPatches()
	k8sInstTool := &K8SInstTool{}

	mockOSInterface := &MockOSTypeInstaller{}
	patches.ApplyFunc(GetOSInterface, func() commontypes.OSTypeInstaller {
		return mockOSInterface
	})

	return patches, k8sInstTool, mockOSInterface
}

func TestK8SInstTool_TearDown(t *testing.T) {
	k8sInstTool := K8SInstTool{}
	err := k8sInstTool.TearDown()
	assert.NoError(t, err, "TearDown should not return an error")
}

type MockOSTypeInstaller struct{}

func (m *MockOSTypeInstaller) SetKubeEdgeVersion(version semver.Version) {
}

func (m *MockOSTypeInstaller) InstallMQTT() error {
	return nil
}

func (m *MockOSTypeInstaller) IsK8SComponentInstalled(kubeConfig, master string) error {
	return nil
}

func (m *MockOSTypeInstaller) InstallKubeEdge(options commontypes.InstallOptions) error {
	return nil
}

func (m *MockOSTypeInstaller) InstallDocker() error {
	return nil
}

func (m *MockOSTypeInstaller) InstallKubeEdgeService(componentType commontypes.ComponentType) error {
	return nil
}

func (m *MockOSTypeInstaller) IsKubeEdgeProcessRunning(name string) (bool, error) {
	return false, nil
}

func (m *MockOSTypeInstaller) KillKubeEdgeProcess(name string) error {
	return nil
}

func (m *MockOSTypeInstaller) KillKubeEdgeBinary(name string) error {
	return nil
}

func (m *MockOSTypeInstaller) RunEdgeCore() error {
	return nil
}

func (m *MockOSTypeInstaller) IsProcessRunning(name string) (bool, error) {
	return false, nil
}

func (m *MockOSTypeInstaller) GetOSVersion() (string, error) {
	return "mock-version", nil
}

func (m *MockOSTypeInstaller) RunningCommand(command string) (string, error) {
	return "", nil
}

func TestK8SInstTool_InstallTools_CloudCoreRunning(t *testing.T) {
	patches, k8sInstTool, mockOSInterface := setupTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsKubeEdgeProcessRunning",
		func(_ *MockOSTypeInstaller, name string) (bool, error) {
			if name == KubeCloudBinaryName {
				return true, nil
			}
			return false, nil
		})

	err := k8sInstTool.InstallTools()
	assert.Error(t, err, "InstallTools should return an error when CloudCore is running")
	assert.Contains(t, err.Error(), "CloudCore is already running")
}

func TestK8SInstTool_InstallTools_ProcessCheckError(t *testing.T) {
	patches, k8sInstTool, mockOSInterface := setupTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsKubeEdgeProcessRunning",
		func(_ *MockOSTypeInstaller, _ string) (bool, error) {
			return false, errors.New("process check error")
		})

	err := k8sInstTool.InstallTools()
	assert.Error(t, err, "InstallTools should return an error when process check fails")
	assert.Contains(t, err.Error(), "process check error")
}

func TestK8SInstTool_InstallTools_K8SVersionCheckFails(t *testing.T) {
	patches, k8sInstTool, mockOSInterface := setupTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsKubeEdgeProcessRunning",
		func(_ *MockOSTypeInstaller, _ string) (bool, error) {
			return false, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsK8SComponentInstalled",
		func(_ *MockOSTypeInstaller, _, _ string) error {
			return errors.New("K8S version verification failed")
		})

	err := k8sInstTool.InstallTools()
	assert.Error(t, err, "InstallTools should return an error when K8S version check fails")
	assert.Contains(t, err.Error(), "K8S version verification failed")
}

func TestK8SInstTool_InstallTools_CRDInstallFails(t *testing.T) {
	patches, k8sInstTool, mockOSInterface := setupTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsKubeEdgeProcessRunning",
		func(_ *MockOSTypeInstaller, _ string) (bool, error) {
			return false, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsK8SComponentInstalled",
		func(_ *MockOSTypeInstaller, _, _ string) error {
			return nil
		})

	patches.ApplyFunc(installCRDs,
		func(_ *K8SInstTool) error {
			return errors.New("CRD installation failed")
		})

	err := k8sInstTool.InstallTools()
	assert.Error(t, err, "InstallTools should return an error when CRD installation fails")
	assert.Contains(t, err.Error(), "CRD installation failed")
}

func TestK8SInstTool_InstallTools_NamespaceCreationFails(t *testing.T) {
	patches, k8sInstTool, mockOSInterface := setupTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsKubeEdgeProcessRunning",
		func(_ *MockOSTypeInstaller, _ string) (bool, error) {
			return false, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsK8SComponentInstalled",
		func(_ *MockOSTypeInstaller, _, _ string) error {
			return nil
		})

	patches.ApplyFunc(installCRDs,
		func(_ *K8SInstTool) error {
			return nil
		})

	patches.ApplyFunc(createKubeEdgeNs,
		func(_, _ string) error {
			return errors.New("namespace creation failed")
		})

	err := k8sInstTool.InstallTools()
	assert.Error(t, err, "InstallTools should return an error when namespace creation fails")
	assert.Contains(t, err.Error(), "namespace creation failed")
}

func TestK8SInstTool_InstallTools_Success(t *testing.T) {
	patches, k8sInstTool, mockOSInterface := setupTest()
	defer patches.Reset()

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsKubeEdgeProcessRunning",
		func(_ *MockOSTypeInstaller, _ string) (bool, error) {
			return false, nil
		})

	patches.ApplyMethod(reflect.TypeOf(mockOSInterface), "IsK8SComponentInstalled",
		func(_ *MockOSTypeInstaller, _, _ string) error {
			return nil
		})

	patches.ApplyFunc(installCRDs,
		func(_ *K8SInstTool) error {
			return nil
		})

	patches.ApplyFunc(createKubeEdgeNs,
		func(_, _ string) error {
			return nil
		})

	err := k8sInstTool.InstallTools()
	assert.NoError(t, err, "InstallTools should not return an error when everything succeeds")
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
