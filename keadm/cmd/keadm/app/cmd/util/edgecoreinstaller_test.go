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
					ToolVersion: semver.MustParse("1.0.0"),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock IsKubeEdgeProcessRunning to return true (already running)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
					func(_ types.OSTypeInstaller, _ string) (bool, error) {
						return true, nil
					})
			},
			expectedError: true,
			errorContains: "EdgeCore is already running",
		},
		{
			name: "IsKubeEdgeProcessRunning returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse("1.0.0"),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock IsKubeEdgeProcessRunning to return an error
				patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
					func(_ types.OSTypeInstaller, _ string) (bool, error) {
						return false, errors.New("process check failed")
					})
			},
			expectedError: true,
			errorContains: "process check failed",
		},
		{
			name: "InstallKubeEdge returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse("1.0.0"),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock IsKubeEdgeProcessRunning to return false
				patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
					func(_ types.OSTypeInstaller, _ string) (bool, error) {
						return false, nil
					})

				// Mock SetKubeEdgeVersion (no-op)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
					func(_ types.OSTypeInstaller, _ semver.Version) {
						// Do nothing
					})

				// Mock InstallKubeEdge to return error
				patches.ApplyMethod(reflect.TypeOf(osInterface), "InstallKubeEdge",
					func(_ types.OSTypeInstaller, _ types.InstallOptions) error {
						return errors.New("installation failed")
					})
			},
			expectedError: true,
			errorContains: "installation failed",
		},
		{
			name: "createEdgeConfigFiles returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse("1.0.0"),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock IsKubeEdgeProcessRunning to return false
				patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
					func(_ types.OSTypeInstaller, _ string) (bool, error) {
						return false, nil
					})

				// Mock SetKubeEdgeVersion (no-op)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
					func(_ types.OSTypeInstaller, _ semver.Version) {
						// Do nothing
					})

				// Mock InstallKubeEdge to succeed
				patches.ApplyMethod(reflect.TypeOf(osInterface), "InstallKubeEdge",
					func(_ types.OSTypeInstaller, _ types.InstallOptions) error {
						return nil
					})

				// Mock os.MkdirAll to fail
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return errors.New("mkdir failed")
				})
			},
			expectedError: true,
			errorContains: "not able to create",
		},
		{
			name: "RunEdgeCore returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse("1.0.0"),
				},
				HubProtocol: "quic",
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock IsKubeEdgeProcessRunning to return false
				patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
					func(_ types.OSTypeInstaller, _ string) (bool, error) {
						return false, nil
					})

				// Mock SetKubeEdgeVersion (no-op)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
					func(_ types.OSTypeInstaller, _ semver.Version) {
						// Do nothing
					})

				// Mock InstallKubeEdge to succeed
				patches.ApplyMethod(reflect.TypeOf(osInterface), "InstallKubeEdge",
					func(_ types.OSTypeInstaller, _ types.InstallOptions) error {
						return nil
					})

				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})

				// Mock validation.ValidateEdgeCoreConfiguration to succeed
				patches.ApplyFunc(validation.ValidateEdgeCoreConfiguration,
					func(_ *v1alpha2.EdgeCoreConfig) field.ErrorList {
						return field.ErrorList{}
					})

				// Mock types.Write2File to succeed
				patches.ApplyFunc(types.Write2File,
					func(_ string, _ interface{}) error {
						return nil
					})

				// Mock RunEdgeCore to fail
				patches.ApplyMethod(reflect.TypeOf(osInterface), "RunEdgeCore",
					func(_ types.OSTypeInstaller) error {
						return errors.New("run failed")
					})
			},
			expectedError: true,
			errorContains: "run failed",
		},
		{
			name: "Installation successful",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse("1.0.0"),
				},
				HubProtocol: "quic",
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock IsKubeEdgeProcessRunning to return false
				patches.ApplyMethod(reflect.TypeOf(osInterface), "IsKubeEdgeProcessRunning",
					func(_ types.OSTypeInstaller, _ string) (bool, error) {
						return false, nil
					})

				// Mock SetKubeEdgeVersion (no-op)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
					func(_ types.OSTypeInstaller, _ semver.Version) {
						// Do nothing
					})

				// Mock InstallKubeEdge to succeed
				patches.ApplyMethod(reflect.TypeOf(osInterface), "InstallKubeEdge",
					func(_ types.OSTypeInstaller, _ types.InstallOptions) error {
						return nil
					})

				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})

				// Mock validation.ValidateEdgeCoreConfiguration to succeed
				patches.ApplyFunc(validation.ValidateEdgeCoreConfiguration,
					func(_ *v1alpha2.EdgeCoreConfig) field.ErrorList {
						return field.ErrorList{}
					})

				// Mock types.Write2File to succeed
				patches.ApplyFunc(types.Write2File,
					func(_ string, _ interface{}) error {
						return nil
					})

				// Mock RunEdgeCore to succeed
				patches.ApplyMethod(reflect.TypeOf(osInterface), "RunEdgeCore",
					func(_ types.OSTypeInstaller) error {
						return nil
					})
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create patches
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			// Apply test-specific mocks
			tt.setup(patches)

			// Call the method
			err := tt.kubeEdgeInst.InstallTools()

			// Verify
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
				// Mock os.MkdirAll to fail
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return errors.New("mkdir failed")
				})
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
				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})
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
				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})
			},
			expectedError: true,
			errorContains: "unsupported hub of protocol",
		},
		{
			name: "Validation returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol: "quic",
				CloudCoreIP: "192.168.1.1:10000",
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})

				// Mock validation.ValidateEdgeCoreConfiguration to fail
				patches.ApplyFunc(validation.ValidateEdgeCoreConfiguration,
					func(_ *v1alpha2.EdgeCoreConfig) field.ErrorList {
						return field.ErrorList{
							field.Invalid(field.NewPath("Modules", "EdgeHub"),
								false,
								"validation error"),
						}
					})
			},
			expectedError: true,
			errorContains: "validation error",
		},
		{
			name: "Write2File returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol: "quic",
				CloudCoreIP: "192.168.1.1:10000",
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})

				// Mock validation.ValidateEdgeCoreConfiguration to succeed
				patches.ApplyFunc(validation.ValidateEdgeCoreConfiguration,
					func(_ *v1alpha2.EdgeCoreConfig) field.ErrorList {
						return field.ErrorList{}
					})

				// Mock types.Write2File to fail
				patches.ApplyFunc(types.Write2File,
					func(_ string, _ interface{}) error {
						return errors.New("write failed")
					})
			},
			expectedError: true,
			errorContains: "write failed",
		},
		{
			name: "Successful configuration with websocket protocol",
			kubeEdgeInst: &KubeEdgeInstTool{
				HubProtocol:           "websocket",
				CloudCoreIP:           "192.168.1.1:10000",
				EdgeNodeName:          "edge-node",
				CGroupDriver:          "systemd",
				RemoteRuntimeEndpoint: "unix:///var/run/cri.sock",
				Token:                 "token123",
				CertPort:              "10002",
				Labels:                []string{"key1=value1", "key2=value2"},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock os.MkdirAll to succeed
				patches.ApplyFunc(os.MkdirAll, func(_ string, _ os.FileMode) error {
					return nil
				})

				// Mock validation.ValidateEdgeCoreConfiguration to succeed
				patches.ApplyFunc(validation.ValidateEdgeCoreConfiguration,
					func(_ *v1alpha2.EdgeCoreConfig) field.ErrorList {
						return field.ErrorList{}
					})

				// Mock types.Write2File to succeed
				patches.ApplyFunc(types.Write2File,
					func(_ string, _ interface{}) error {
						return nil
					})
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create patches
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			// Apply test-specific mocks
			tt.setup(patches)

			// Call the method
			err := tt.kubeEdgeInst.createEdgeConfigFiles()

			// Verify
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
					ToolVersion: semver.MustParse("1.0.0"),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock SetKubeEdgeVersion (no-op)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
					func(_ types.OSTypeInstaller, _ semver.Version) {
						// Do nothing
					})

				// Mock KillKubeEdgeBinary to succeed
				patches.ApplyMethod(reflect.TypeOf(osInterface), "KillKubeEdgeBinary",
					func(_ types.OSTypeInstaller, _ string) error {
						return nil
					})
			},
			expectedError: false,
		},
		{
			name: "KillKubeEdgeBinary returns error",
			kubeEdgeInst: &KubeEdgeInstTool{
				Common: Common{
					ToolVersion: semver.MustParse("1.0.0"),
				},
			},
			setup: func(patches *gomonkey.Patches) {
				// Mock GetOSInterface
				osInterface := &DebOS{}
				patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
					return osInterface
				})

				// Mock SetKubeEdgeVersion (no-op)
				patches.ApplyMethod(reflect.TypeOf(osInterface), "SetKubeEdgeVersion",
					func(_ types.OSTypeInstaller, _ semver.Version) {
						// Do nothing
					})

				// Mock KillKubeEdgeBinary to fail
				patches.ApplyMethod(reflect.TypeOf(osInterface), "KillKubeEdgeBinary",
					func(_ types.OSTypeInstaller, _ string) error {
						return errors.New("kill failed")
					})
			},
			expectedError: true,
			errorContains: "kill failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create patches
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			// Apply test-specific mocks
			tt.setup(patches)

			// Call the method
			err := tt.kubeEdgeInst.TearDown()

			// Verify
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
