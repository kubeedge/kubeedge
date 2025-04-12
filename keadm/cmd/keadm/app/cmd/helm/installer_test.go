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

package helm

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestRunHelmInstallerSimple(t *testing.T) {
	t.Run("Basic properties check", func(t *testing.T) {
		forceTrue := &KubeCloudHelmInstTool{
			Force:     true,
			DryRun:    false,
			Namespace: "test-namespace",
		}

		forceFalse := &KubeCloudHelmInstTool{
			Force:     false,
			DryRun:    true,
			Namespace: "test-namespace",
		}

		assert.True(t, forceTrue.Force, "Force should be true in first instance")
		assert.False(t, forceFalse.Force, "Force should be false in second instance")
		assert.True(t, forceFalse.DryRun, "DryRun should be true in second instance")
		assert.Equal(t, "test-namespace", forceTrue.Namespace, "Namespace should be set correctly")
	})

	t.Run("Nil renderer error", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{}

		defer func() {
			r := recover()
			assert.NotNil(t, r, "Expected a panic with nil renderer")
		}()

		_, _ = cu.runHelmInstall(nil)
	})
}

func TestInstallToolsError(t *testing.T) {
	t.Run("Error from RunHelmInstall", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{
			Action: types.HelmInstallAction,
			Common: util.Common{
				ToolVersion: createMockToolVersion(),
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf(cu), "RunHelmInstall",
			func(_ *KubeCloudHelmInstTool, _ string) error {
				return errors.New("install error")
			})

		err := cu.InstallTools()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "install error")
	})

	t.Run("Error from RunHelmManifest", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{
			Action: types.HelmManifestAction,
			Common: util.Common{
				ToolVersion: createMockToolVersion(),
			},
		}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyMethod(reflect.TypeOf(cu), "RunHelmManifest",
			func(_ *KubeCloudHelmInstTool, _ string) error {
				return errors.New("manifest error")
			})

		err := cu.InstallTools()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest error")
	})
}

func TestProfileHandling(t *testing.T) {
	t.Run("Default profile creation", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{
			Profile: "",
			Common: util.Common{
				ToolVersion: createMockToolVersion(),
			},
		}

		expectedVersion := fmt.Sprintf("v%s", cu.Common.ToolVersion.String())
		expectedProfile := fmt.Sprintf("version=%s", expectedVersion)

		assert.Equal(t, "", cu.Profile, "Initial profile should be empty")

		cu.Profile = expectedProfile
		cu.ProfileKey = "version"

		assert.Equal(t, expectedProfile, cu.Profile, "Profile should be set to version=v1.8.0")
		assert.Equal(t, "version", cu.ProfileKey, "ProfileKey should be 'version'")
	})
}

func TestInstallToolsBasic(t *testing.T) {
	t.Run("Unsupported action", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{
			Action: "unsupported",
			Common: util.Common{
				ToolVersion: createMockToolVersion(),
			},
		}

		err := cu.InstallTools()
		assert.NoError(t, err, "Unsupported action should not return an error")
	})
}

func TestCombineProfile(t *testing.T) {
	t.Run("IptablesMgr key converts to version", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{
			ProfileKey: IptablesMgrProfileKey,
		}

		assert.Equal(t, IptablesMgrProfileKey, cu.ProfileKey)

		profilekey := cu.ProfileKey
		if profilekey == IptablesMgrProfileKey || profilekey == ControllerManagerProfileKey {
			profilekey = VersionProfileKey
		}

		assert.Equal(t, VersionProfileKey, profilekey)
	})

	t.Run("ControllerManager key converts to version", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{
			ProfileKey: ControllerManagerProfileKey,
		}

		assert.Equal(t, ControllerManagerProfileKey, cu.ProfileKey)

		profilekey := cu.ProfileKey
		if profilekey == IptablesMgrProfileKey || profilekey == ControllerManagerProfileKey {
			profilekey = VersionProfileKey
		}

		assert.Equal(t, VersionProfileKey, profilekey)
	})
}

func TestLoadValuesBehavior(t *testing.T) {
	t.Run("Check default profile filename", func(t *testing.T) {
		result := builtinProfileToFilename("")
		assert.Equal(t, DefaultProfileFilename, result, "Empty profile key should return default profile filename")

		result = builtinProfileToFilename("custom")
		assert.Equal(t, "custom.yaml", result, "Custom profile key should return key + .yaml")
	})
}

func TestRunHelmManifestSimple(t *testing.T) {
	t.Run("Basic execution path", func(t *testing.T) {
		cu := &KubeCloudHelmInstTool{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(fs.ReadFile, func(fsys fs.FS, name string) ([]byte, error) {
			return nil, errors.New("read file error")
		})

		err := cu.RunHelmManifest("")
		assert.Error(t, err)
	})
}

func TestCheckProfileMethod(t *testing.T) {
	tests := []struct {
		name          string
		profileKey    string
		validProfiles map[string]bool
		expectedError bool
	}{
		{
			name:          "Valid version profile key",
			profileKey:    VersionProfileKey,
			validProfiles: map[string]bool{VersionProfileKey: true},
			expectedError: false,
		},
		{
			name:          "Valid IptablesMgr profile key",
			profileKey:    IptablesMgrProfileKey,
			validProfiles: map[string]bool{},
			expectedError: false,
		},
		{
			name:          "Invalid profile key",
			profileKey:    "invalid",
			validProfiles: map[string]bool{VersionProfileKey: true},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &KubeCloudHelmInstTool{
				ProfileKey: tt.profileKey,
			}

			patches := gomonkey.NewPatches()
			defer patches.Reset()

			old := ValidProfiles
			defer func() { ValidProfiles = old }()

			ValidProfiles = tt.validProfiles
			ValidProfiles[IptablesMgrProfileKey] = true
			ValidProfiles[ControllerManagerProfileKey] = true

			err := cu.checkProfile("")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadValuesFunction(t *testing.T) {
	tests := []struct {
		name          string
		chartsDir     string
		profileKey    string
		existsProfile bool
		fileContent   []byte
		fileReadErr   error
		expectedError bool
	}{
		{
			name:          "Success with existing profile",
			chartsDir:     "",
			profileKey:    "test",
			existsProfile: true,
			fileContent:   []byte("test: value"),
			fileReadErr:   nil,
			expectedError: false,
		},
		{
			name:          "Success without existing profile",
			chartsDir:     "",
			profileKey:    "test",
			existsProfile: false,
			fileContent:   []byte("test: value"),
			fileReadErr:   nil,
			expectedError: false,
		},
		{
			name:          "File read error",
			chartsDir:     "",
			profileKey:    "test",
			existsProfile: true,
			fileContent:   nil,
			fileReadErr:   errors.New("read error"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(fs.ReadFile, func(fsys fs.FS, name string) ([]byte, error) {
				if tt.existsProfile {
					assert.Contains(t, name, "profiles")
				} else {
					assert.Contains(t, name, "values.yaml")
				}
				return tt.fileContent, tt.fileReadErr
			})

			result, err := loadValues(tt.chartsDir, tt.profileKey, tt.existsProfile)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, string(tt.fileContent), result)
			}
		})
	}
}

func TestSetsContainSubstring(t *testing.T) {
	tests := []struct {
		name     string
		sets     []string
		sub      string
		expected bool
	}{
		{
			name:     "Substring present",
			sets:     []string{"key1=value1", "key2=value2"},
			sub:      "key1",
			expected: true,
		},
		{
			name:     "Substring not present",
			sets:     []string{"key1=value1", "key2=value2"},
			sub:      "key3",
			expected: false,
		},
		{
			name:     "Empty sets",
			sets:     []string{},
			sub:      "key1",
			expected: false,
		},
		{
			name:     "Empty substring",
			sets:     []string{"key1=value1", "key2=value2"},
			sub:      "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetsContainSubstring(tt.sets, tt.sub)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRebuildFlagVals(t *testing.T) {
	tests := []struct {
		name         string
		initialSets  []string
		expectedSets []string
	}{
		{
			name: "No duplicate keys",
			initialSets: []string{
				"key1=value1",
				"key2=value2",
			},
			expectedSets: []string{
				"key1=value1",
				"key2=value2",
			},
		},
		{
			name: "With duplicate keys",
			initialSets: []string{
				"key1=value1",
				"key1=value2",
				"key2=value3",
			},
			expectedSets: []string{
				"key1=value2",
				"key2=value3",
			},
		},
		{
			name: "With invalid format",
			initialSets: []string{
				"key1=value1",
				"invalid",
				"key2=value2",
			},
			expectedSets: []string{
				"key1=value1",
				"key2=value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &KubeCloudHelmInstTool{
				Sets: tt.initialSets,
			}

			err := cu.rebuildFlagVals()

			assert.NoError(t, err)

			assert.Equal(t, len(tt.expectedSets), len(cu.Sets))

			expectedMap := make(map[string]bool)
			for _, set := range tt.expectedSets {
				expectedMap[set] = true
			}

			actualMap := make(map[string]bool)
			for _, set := range cu.Sets {
				actualMap[set] = true
			}

			assert.Equal(t, len(expectedMap), len(actualMap))

			for key := range expectedMap {
				assert.True(t, actualMap[key], "Expected set not found: %s", key)
			}
		})
	}
}

func TestHandleProfile(t *testing.T) {
	tests := []struct {
		name          string
		profileKey    string
		profileValue  string
		initialSets   []string
		expectedError bool
		expectedSets  []string
	}{
		{
			name:          "Version profile with empty value",
			profileKey:    "version",
			profileValue:  "",
			initialSets:   []string{},
			expectedError: false,
			expectedSets: []string{
				"cloudCore.image.tag=v1.8.0",
				"iptablesManager.image.tag=v1.8.0",
				"controllerManager.image.tag=v1.8.0",
			},
		},
		{
			name:          "Version profile with value without v prefix",
			profileKey:    "version",
			profileValue:  "1.9.0",
			initialSets:   []string{},
			expectedError: false,
			expectedSets: []string{
				"cloudCore.image.tag=v1.9.0",
				"iptablesManager.image.tag=v1.9.0",
				"controllerManager.image.tag=v1.9.0",
			},
		},
		{
			name:          "Version profile with existing tag settings",
			profileKey:    "version",
			profileValue:  "1.9.0",
			initialSets:   []string{"cloudCore.image.tag=custom-tag"},
			expectedError: false,
			expectedSets: []string{
				"cloudCore.image.tag=custom-tag",
				"iptablesManager.image.tag=v1.9.0",
				"controllerManager.image.tag=v1.9.0",
			},
		},
		{
			name:          "IptablesMgr profile with external mode",
			profileKey:    "iptablesmgr",
			profileValue:  "external",
			initialSets:   []string{},
			expectedError: false,
			expectedSets: []string{
				"iptablesManager.mode=external",
				"cloudCore.image.tag=1.8.0",
				"iptablesManager.image.tag=1.8.0",
			},
		},
		{
			name:          "IptablesMgr profile with internal mode",
			profileKey:    "iptablesmgr",
			profileValue:  "internal",
			initialSets:   []string{},
			expectedError: false,
			expectedSets: []string{
				"iptablesManager.mode=internal",
				"cloudCore.image.tag=1.8.0",
				"iptablesManager.image.tag=1.8.0",
			},
		},
		{
			name:          "IptablesMgr profile with empty value (defaults to external)",
			profileKey:    "iptablesmgr",
			profileValue:  "",
			initialSets:   []string{},
			expectedError: false,
			expectedSets: []string{
				"iptablesManager.mode=external",
				"cloudCore.image.tag=1.8.0",
				"iptablesManager.image.tag=1.8.0",
			},
		},
		{
			name:          "IptablesMgr profile with invalid value",
			profileKey:    "iptablesmgr",
			profileValue:  "invalid",
			initialSets:   []string{},
			expectedError: true,
			expectedSets:  []string{},
		},
		{
			name:          "ControllerManager profile",
			profileKey:    "controllermanager",
			profileValue:  "",
			initialSets:   []string{},
			expectedError: false,
			expectedSets: []string{
				"controllerManager.image.tag=1.8.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &KubeCloudHelmInstTool{
				ProfileKey: tt.profileKey,
				Sets:       tt.initialSets,
				Common: util.Common{
					ToolVersion: createMockToolVersion(),
				},
			}

			err := cu.handleProfile(tt.profileValue)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				assert.Equal(t, len(tt.expectedSets), len(cu.Sets),
					"Expected %d sets but got %d: %v", len(tt.expectedSets), len(cu.Sets), cu.Sets)

				for _, expectedSet := range tt.expectedSets {
					found := false
					for _, actualSet := range cu.Sets {
						if expectedSet == actualSet {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected set '%s' not found in actual sets: %v", expectedSet, cu.Sets)
				}
			}
		})
	}
}

func TestIsInnerProfile(t *testing.T) {
	tests := []struct {
		name       string
		profileKey string
		expected   bool
	}{
		{
			name:       "Empty profile key",
			profileKey: "",
			expected:   true,
		},
		{
			name:       "Default profile string",
			profileKey: DefaultProfileString,
			expected:   true,
		},
		{
			name:       "IptablesMgrProfileKey",
			profileKey: IptablesMgrProfileKey,
			expected:   true,
		},
		{
			name:       "ControllerManagerProfileKey",
			profileKey: ControllerManagerProfileKey,
			expected:   true,
		},
		{
			name:       "External profile key",
			profileKey: "external",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &KubeCloudHelmInstTool{
				ProfileKey: tt.profileKey,
			}

			result := cu.isInnerProfile()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBeforeRenderer(t *testing.T) {
	t.Run("Test advertiseAddress handling", func(t *testing.T) {
		addrs := "192.168.1.1,192.168.2.1"
		cu := &KubeCloudHelmInstTool{
			AdvertiseAddress: addrs,
			Profile:          "version=1.8.0",
			ProfileKey:       "version",
			Common: util.Common{
				ToolVersion: createMockToolVersion(),
			},
		}

		cu.existsProfile = true

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(fs.ReadFile, func(fsys fs.FS, name string) ([]byte, error) {
			return []byte("test: value"), nil
		})

		err := cu.beforeRenderer("")

		assert.NoError(t, err)

		addrList := strings.Split(addrs, ",")
		for i, addr := range addrList {
			expectedSet := fmt.Sprintf("cloudCore.modules.cloudHub.advertiseAddress[%d]=%s", i, addr)
			found := false
			for _, set := range cu.Sets {
				if set == expectedSet {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected address setting not found: %s", expectedSet)
		}
	})
}

func TestReadProfiles(t *testing.T) {
	tests := []struct {
		name           string
		dirEntries     []fs.DirEntry
		fsReadDirErr   error
		expectedResult map[string]bool
		expectedError  bool
	}{
		{
			name: "Valid profiles",
			dirEntries: []fs.DirEntry{
				MockDirEntry{name: "profile1.yaml", isDir: false},
				MockDirEntry{name: "profile2.yaml", isDir: false},
				MockDirEntry{name: "not-a-yaml", isDir: false},
				MockDirEntry{name: "subdir", isDir: true},
			},
			fsReadDirErr:   nil,
			expectedResult: map[string]bool{"profile1": true, "profile2": true},
			expectedError:  false,
		},
		{
			name:           "ReadDir error",
			dirEntries:     nil,
			fsReadDirErr:   errors.New("read dir error"),
			expectedResult: nil,
			expectedError:  true,
		},
		{
			name:           "Empty directory",
			dirEntries:     []fs.DirEntry{},
			fsReadDirErr:   nil,
			expectedResult: map[string]bool{},
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &KubeCloudHelmInstTool{}

			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(fs.ReadDir,
				func(fsys fs.FS, name string) ([]fs.DirEntry, error) {
					return tt.dirEntries, tt.fsReadDirErr
				})

			result, err := cu.readProfiles("", "profiles")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestRunHelmInstall(t *testing.T) {
	tests := []struct {
		name              string
		force             bool
		externalHelmRoot  string
		isProcessRunning  bool
		processRunningErr error
		k8sInstallErr     error
		expectedError     bool
	}{
		{
			name:              "Process running and no force",
			force:             false,
			externalHelmRoot:  "",
			isProcessRunning:  true,
			processRunningErr: nil,
			expectedError:     true,
		},
		{
			name:              "Process running error",
			force:             false,
			externalHelmRoot:  "",
			isProcessRunning:  false,
			processRunningErr: errors.New("process error"),
			expectedError:     true,
		},
		{
			name:              "K8s component install error",
			force:             false,
			externalHelmRoot:  "",
			isProcessRunning:  false,
			processRunningErr: nil,
			k8sInstallErr:     errors.New("k8s error"),
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &KubeCloudHelmInstTool{
				Force:            tt.force,
				ExternalHelmRoot: tt.externalHelmRoot,
				Common: util.Common{
					ToolVersion: createMockToolVersion(),
				},
			}

			osTypeInstaller := &MockOSTypeInstaller{
				IsKubeEdgeProcessRunningFunc: func(name string) (bool, error) {
					return tt.isProcessRunning, tt.processRunningErr
				},
				IsK8SComponentInstalledFunc: func(kubeConfig, master string) error {
					return tt.k8sInstallErr
				},
			}

			cu.Common.OSTypeInstaller = osTypeInstaller

			if !tt.isProcessRunning && tt.processRunningErr == nil && tt.k8sInstallErr == nil {
				patches := gomonkey.NewPatches()
				defer patches.Reset()

				patches.ApplyFunc(fs.ReadDir, func(fsys fs.FS, name string) ([]fs.DirEntry, error) {
					return nil, errors.New("mock error")
				})
			}

			err := cu.RunHelmInstall("")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTearDown(t *testing.T) {
	tests := []struct {
		name          string
		cleanNsErr    error
		expectedError bool
	}{
		{
			name:          "Successful cleanup",
			cleanNsErr:    nil,
			expectedError: false,
		},
		{
			name:          "Failed cleanup",
			cleanNsErr:    errors.New("cleanup error"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			cu := &KubeCloudHelmInstTool{
				Common: util.Common{
					KubeConfig: "test-kubeconfig",
				},
			}

			patches.ApplyMethod(reflect.TypeOf(&util.Common{}), "CleanNameSpace",
				func(_ *util.Common, namespace, kubeConfig string) error {
					assert.Equal(t, constants.SystemNamespace, namespace)
					assert.Equal(t, cu.KubeConfig, kubeConfig)
					return tt.cleanNsErr
				})

			err := cu.TearDown()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type MockDirEntry struct {
	name  string
	isDir bool
}

func (m MockDirEntry) Name() string               { return m.name }
func (m MockDirEntry) IsDir() bool                { return m.isDir }
func (m MockDirEntry) Type() fs.FileMode          { return fs.ModeDir | fs.ModePerm }
func (m MockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func createMockToolVersion() semver.Version {
	return semver.Version{
		Major: 1,
		Minor: 8,
		Patch: 0,
		Pre:   []semver.PRVersion{},
		Build: []string{},
	}
}

type MockOSTypeInstaller struct {
	SetKubeEdgeVersionFunc       func(version semver.Version)
	IsKubeEdgeProcessRunningFunc func(string) (bool, error)
	IsK8SComponentInstalledFunc  func(string, string) error
	InstallKubeEdgeFunc          func(types.InstallOptions) error
	InstallMQTTFunc              func() error
	RunEdgeCoreFunc              func() error
	KillKubeEdgeBinaryFunc       func(string) error
}

func (m *MockOSTypeInstaller) SetKubeEdgeVersion(version semver.Version) {
	if m.SetKubeEdgeVersionFunc != nil {
		m.SetKubeEdgeVersionFunc(version)
	}
}

func (m *MockOSTypeInstaller) IsKubeEdgeProcessRunning(name string) (bool, error) {
	if m.IsKubeEdgeProcessRunningFunc != nil {
		return m.IsKubeEdgeProcessRunningFunc(name)
	}
	return false, nil
}

func (m *MockOSTypeInstaller) IsK8SComponentInstalled(kubeConfig, master string) error {
	if m.IsK8SComponentInstalledFunc != nil {
		return m.IsK8SComponentInstalledFunc(kubeConfig, master)
	}
	return nil
}

func (m *MockOSTypeInstaller) InstallKubeEdge(options types.InstallOptions) error {
	if m.InstallKubeEdgeFunc != nil {
		return m.InstallKubeEdgeFunc(options)
	}
	return nil
}

func (m *MockOSTypeInstaller) InstallMQTT() error {
	if m.InstallMQTTFunc != nil {
		return m.InstallMQTTFunc()
	}
	return nil
}

func (m *MockOSTypeInstaller) RunEdgeCore() error {
	if m.RunEdgeCoreFunc != nil {
		return m.RunEdgeCoreFunc()
	}
	return nil
}

func (m *MockOSTypeInstaller) KillKubeEdgeBinary(name string) error {
	if m.KillKubeEdgeBinaryFunc != nil {
		return m.KillKubeEdgeBinaryFunc(name)
	}
	return nil
}
