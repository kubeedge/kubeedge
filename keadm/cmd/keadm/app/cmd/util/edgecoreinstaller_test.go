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
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2/validation"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const (
	edgeCoreTestVersion     = "1.0.0"
	testCloudCoreIP         = "192.168.1.1:10000"
	testEdgeNodeName        = "edge-node"
	testRemoteRuntimeSocket = "unix:///var/run/cri.sock"
	testToken               = "token123"
	testCertPort            = "10002"
	testProtocolQuic        = "quic"
	testProtocolWebsocket   = "websocket"
	testCGroupDriverSystemd = "systemd"
)

type mockBehavior struct {
	processRunning   bool
	processError     string
	installError     string
	mkdirError       string
	validationError  bool
	writeFileError   string
	runEdgeCoreError string
	killError        string
}

func setupBasicMocks(patches *gomonkey.Patches, behavior mockBehavior) *DebOS {
	osInterface := &DebOS{}
	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return osInterface
	})

	patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
		func(_ types.OSTypeInstaller, _ semver.Version) {
		})

	return osInterface
}

func mockProcessCheck(patches *gomonkey.Patches, osInterface *DebOS, behavior mockBehavior) {
	patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
		func(_ types.OSTypeInstaller, _ string) (bool, error) {
			if behavior.processError != "" {
				return false, errors.New(behavior.processError)
			}
			return behavior.processRunning, nil
		})
}

func mockInstallKubeEdge(patches *gomonkey.Patches, osInterface *DebOS, behavior mockBehavior) {
	patches.ApplyMethod(reflect.TypeOf(osInterface), "InstallKubeEdge",
		func(_ types.OSTypeInstaller, _ types.InstallOptions) error {
			if behavior.installError != "" {
				return errors.New(behavior.installError)
			}
			return nil
		})
}

func mockFileOperations(patches *gomonkey.Patches, behavior mockBehavior) {
	patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
		if behavior.mkdirError != "" {
			return errors.New(behavior.mkdirError)
		}
		return nil
	})

	patches.ApplyMethod(reflect.TypeOf(&v1alpha2.EdgeCoreConfig{}), "WriteTo",
		func(_ *v1alpha2.EdgeCoreConfig, _ string) error {
			if behavior.writeFileError != "" {
				return errors.New(behavior.writeFileError)
			}
			return nil
		})
}

func mockValidation(patches *gomonkey.Patches, behavior mockBehavior) {
	patches.ApplyFunc(validation.ValidateEdgeCoreConfiguration,
		func(_ *v1alpha2.EdgeCoreConfig) field.ErrorList {
			if behavior.validationError {
				return field.ErrorList{
					field.Invalid(field.NewPath("Modules", "EdgeHub"),
						false,
						"validation error"),
				}
			}
			return field.ErrorList{}
		})
}

func mockRunEdgeCore(patches *gomonkey.Patches, osInterface *DebOS, behavior mockBehavior) {
	patches.ApplyMethod(reflect.TypeOf(osInterface), "RunEdgeCore",
		func(_ types.OSTypeInstaller) error {
			if behavior.runEdgeCoreError != "" {
				return errors.New(behavior.runEdgeCoreError)
			}
			return nil
		})
}

func mockKillProcess(patches *gomonkey.Patches, osInterface *DebOS, behavior mockBehavior) {
	patches.ApplyMethod(reflect.TypeOf(osInterface), "KillKubeEdgeBinary",
		func(_ types.OSTypeInstaller, _ string) error {
			if behavior.killError != "" {
				return errors.New(behavior.killError)
			}
			return nil
		})
}

func TestKubeEdgeInstTool_InstallTools(t *testing.T) {
	tests := []struct {
		name          string
		kubeEdgeInst  *KubeEdgeInstTool
		setup         func(*gomonkey.Patches)
		expectedError bool
		errorContains string
	}{
		{
			name: "EdgeCore already running",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{processRunning: true}
				osInterface := setupBasicMocks(patches, behavior)
				mockProcessCheck(patches, osInterface, behavior)
			},
			expectedError: true,
			errorContains: "EdgeCore is already running",
		},
		{
			name: "IsKubeEdgeProcessRunning returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{processError: "process check failed"}
				osInterface := setupBasicMocks(patches, behavior)
				mockProcessCheck(patches, osInterface, behavior)
			},
			expectedError: true,
			errorContains: "process check failed",
		},
		{
			name: "InstallKubeEdge returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{installError: "installation failed"}
				osInterface := setupBasicMocks(patches, behavior)
				mockProcessCheck(patches, osInterface, behavior)
				mockInstallKubeEdge(patches, osInterface, behavior)
			},
			expectedError: true,
			errorContains: "installation failed",
		},
		{
			name: "createEdgeConfigFiles returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{mkdirError: "mkdir failed"}
				osInterface := setupBasicMocks(patches, behavior)
				mockProcessCheck(patches, osInterface, behavior)
				mockInstallKubeEdge(patches, osInterface, behavior)
				mockFileOperations(patches, behavior)
			},
			expectedError: true,
			errorContains: "not able to create",
		},
		{
			name: "RunEdgeCore returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
				HubProtocol: testProtocolQuic,
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{runEdgeCoreError: "run failed"}
				osInterface := setupBasicMocks(patches, behavior)
				mockProcessCheck(patches, osInterface, behavior)
				mockInstallKubeEdge(patches, osInterface, behavior)
				mockFileOperations(patches, behavior)
				mockValidation(patches, behavior)
				mockRunEdgeCore(patches, osInterface, behavior)
			},
			expectedError: true,
			errorContains: "run failed",
		},
		{
			name: "Installation successful",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
				HubProtocol: testProtocolQuic,
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{}
				osInterface := setupBasicMocks(patches, behavior)
				mockProcessCheck(patches, osInterface, behavior)
				mockInstallKubeEdge(patches, osInterface, behavior)
				mockFileOperations(patches, behavior)
				mockValidation(patches, behavior)
				mockRunEdgeCore(patches, osInterface, behavior)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tt.setup(patches)

			err := tt.kubeEdgeInst.InstallTools()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKubeEdgeInstTool_createEdgeConfigFiles(t *testing.T) {
	tests := []struct {
		name          string
		kubeEdgeInst  *KubeEdgeInstTool
		setup         func(*gomonkey.Patches)
		expectedError bool
		errorContains string
	}{
		{
			name:         "MkdirAll returns error",
			kubeEdgeInst: &KubeEdgeInstTool{},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{mkdirError: "mkdir failed"}
				mockFileOperations(patches, behavior)
			},
			expectedError: true,
			errorContains: "not able to create",
		},
		{
			name: "Invalid CGroupDriver",
			kubeEdgeInst: &KubeEdgeInstTool{
				CGroupDriver: "invalid",
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{}
				mockFileOperations(patches, behavior)
			},
			expectedError: true,
			errorContains: "unsupported CGroupDriver",
		},
		{
			name: "Invalid HubProtocol",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol: "invalid",
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{}
				mockFileOperations(patches, behavior)
			},
			expectedError: true,
			errorContains: "unsupported hub of protocol",
		},
		{
			name: "Validation returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol: testProtocolQuic,
				CloudCoreIP: testCloudCoreIP,
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{validationError: true}
				mockFileOperations(patches, behavior)
				mockValidation(patches, behavior)
			},
			expectedError: true,
			errorContains: "validation error",
		},
		{
			name: "Write2File returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol: testProtocolQuic,
				CloudCoreIP: testCloudCoreIP,
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{writeFileError: "write failed"}
				mockFileOperations(patches, behavior)
				mockValidation(patches, behavior)
			},
			expectedError: true,
			errorContains: "write failed",
		},
		{
			name: "Successful configuration with websocket protocol",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol:           testProtocolWebsocket,
				CloudCoreIP:           testCloudCoreIP,
				EdgeNodeName:          testEdgeNodeName,
				CGroupDriver:          testCGroupDriverSystemd,
				RemoteRuntimeEndpoint: testRemoteRuntimeSocket,
				Token:                 testToken,
				CertPort:              testCertPort,
				Labels:                []string{"key1=value1", "key2=value2"},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{}
				mockFileOperations(patches, behavior)
				mockValidation(patches, behavior)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tt.setup(patches)

			err := tt.kubeEdgeInst.createEdgeConfigFiles()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKubeEdgeInstTool_TearDown(t *testing.T) {
	tests := []struct {
		name          string
		kubeEdgeInst  *KubeEdgeInstTool
		setup         func(*gomonkey.Patches)
		expectedError bool
		errorContains string
	}{
		{
			name: "TearDown successful",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{}
				osInterface := setupBasicMocks(patches, behavior)
				mockKillProcess(patches, osInterface, behavior)
			},
			expectedError: false,
		},
		{
			name: "KillKubeEdgeBinary returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse(edgeCoreTestVersion),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				behavior := mockBehavior{killError: "kill failed"}
				osInterface := setupBasicMocks(patches, behavior)
				mockKillProcess(patches, osInterface, behavior)
			},
			expectedError: true,
			errorContains: "kill failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			tt.setup(patches)

			err := tt.kubeEdgeInst.TearDown()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
