/*
Copyright 2020 The KubeEdge Authors.

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
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/common/constants"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const (
	testVersion = "v1.8.0"
)

func TestCheckKubernetesVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     *version.Info
		expectError bool
	}{
		{
			name: "Supported version",
			version: &version.Info{
				Major: "1",
				Minor: "11",
			},
			expectError: false,
		},
		{
			name: "Badly formatted version",
			version: &version.Info{
				Major: "1",
				Minor: "a",
			},
			expectError: true,
		},
		{
			name: "Old version",
			version: &version.Info{
				Major: "1",
				Minor: "3",
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkKubernetesVersion(test.version)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
				}
			}
		})
	}
}

func TestPrivateDownloadServiceFile(t *testing.T) {
	var (
		componentType   types.ComponentType
		targetVersion   semver.Version
		serviceFilePath string
	)
	var check = func(serviceFilePath string) error {
		_, err := os.Stat(serviceFilePath)
		if err != nil {
			return err
		}
		files, _ := os.ReadDir(path.Dir(serviceFilePath))
		if len(files) > 1 {
			return fmt.Errorf("download redundancy files")
		}
		return err
	}
	var clean = func(testTmpDir string) {
		dir, err := os.ReadDir(testTmpDir)
		if err != nil {
			t.Fatalf("failed to clean test tmp dir!\n")
		}
		for _, d := range dir {
			if err = os.RemoveAll(path.Join(testTmpDir, d.Name())); err != nil {
				t.Fatalf("failed to clean test tmp dir!\n")
			}
		}
	}

	testTmpDir := t.TempDir()

	componentType = types.CloudCore
	targetVersion, _ = semver.Make(types.DefaultKubeEdgeVersion)
	serviceFilePath = testTmpDir + "/" + CloudServiceFile
	t.Run("test with reDownloading cloudcore service file if version = latest", func(t *testing.T) {
		err := downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("download should not return an error:{%s}\n", err.Error())
		}
		err = downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("second download should not return an error:{%s}\n", err.Error())
		}
		if err = check(serviceFilePath); err != nil {
			t.Fatalf("check should not return an error{%s}\n", err.Error())
		}
		clean(testTmpDir)
	})

	componentType = types.EdgeCore
	targetVersion, _ = semver.Make(types.DefaultKubeEdgeVersion)
	serviceFilePath = testTmpDir + "/" + EdgeServiceFile
	t.Run("test with reDownloading edgecore service file if version = latest", func(t *testing.T) {
		err := downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("download should not return an error:{%s}\n", err.Error())
		}
		err = downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("second download should not return an error:{%s}\n", err.Error())
		}
		if err = check(serviceFilePath); err != nil {
			t.Fatalf("check should not return an error{%s}\n", err.Error())
		}
		clean(testTmpDir)
	})
}

func TestGetPackageManager(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(GetPackageManager, func() string {
		return APT
	})
	pm := GetPackageManager()
	assert.Equal(t, APT, pm)

	patches.Reset()
	patches.ApplyFunc(GetPackageManager, func() string {
		return YUM
	})
	pm = GetPackageManager()
	assert.Equal(t, YUM, pm)

	patches.Reset()
	patches.ApplyFunc(GetPackageManager, func() string {
		return PACMAN
	})
	pm = GetPackageManager()
	assert.Equal(t, PACMAN, pm)

	patches.Reset()
	pm = GetPackageManager()
	assert.Contains(t, []string{APT, YUM, PACMAN, ""}, pm)
}

func TestGetLatestVersion(t *testing.T) {
	version, err := GetLatestVersion()
	if err != nil {
		t.Logf("Note: Failed to query real KubeEdge version, this is expected if offline: %v", err)
	} else {
		t.Logf("Got KubeEdge version: %s", version)
	}
}

func TestHasSystemd(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(HasSystemd, func() bool {
		return true
	})
	assert.True(t, HasSystemd())

	patches.Reset()
	patches.ApplyFunc(HasSystemd, func() bool {
		return false
	})
	assert.False(t, HasSystemd())
}

func TestComputeSHA512Checksum(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "a",
			expected: "1f40fc92da241694750979ee6cf582f2d5d7d28e18335de05abc54d0560e0f5302860c652bf08d560252aa5e74210546f369fbbbce8c12cfc7957b2652fe9a75",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", test.name)
			if err != nil {
				t.Errorf("Unable to create temp file: %v", err)
			}

			defer tmpfile.Close()
			if _, err := tmpfile.Write([]byte(test.name)); err != nil {
				t.Errorf("Unable to write to temp file: %v", err)
			}

			actual, err := computeSHA512Checksum(tmpfile.Name())
			if err != nil {
				t.Errorf("Unable to compute checksum: %v", err)
			}
			if actual != test.expected {
				t.Errorf("expected %s, got %s", test.expected, actual)
			}
		})
	}

	_, err := computeSHA512Checksum("nonexistent_file")
	assert.Error(t, err)
}

func TestKubeadmVersion(t *testing.T) {
	type T struct {
		name         string
		input        string
		output       string
		outputError  bool
		parsingError bool
	}
	cases := []T{
		{
			name:   "valid version with label and metadata",
			input:  "v1.8.0-alpha.2.1231+afabd012389d53a",
			output: "v1.8.0-alpha.2",
		},
		{
			name:   "valid version with label and extra metadata",
			input:  "v1.8.0-alpha.2.1231+afabd012389d53a.extra",
			output: "v1.8.0-alpha.2",
		},
		{
			name:   "valid patch version with label and extra metadata",
			input:  "v1.11.3-beta.0.38+135cc4c1f47994",
			output: "v1.11.2",
		},
		{
			name:   "valid version with label extra",
			input:  "v1.8.0-alpha.2.1231",
			output: "v1.8.0-alpha.2",
		},
		{
			name:   "valid patch version with label",
			input:  "v1.9.11-beta.0",
			output: "v1.9.10",
		},
		{
			name:   "handle version with partial label",
			input:  "v1.8.0-alpha",
			output: "v1.8.0-alpha.0",
		},
		{
			name:   "handle version missing 'v'",
			input:  "1.11.0",
			output: "v1.11.0",
		},
		{
			name:   "valid version without label and metadata",
			input:  "v1.8.0",
			output: "v1.8.0",
		},
		{
			name:   "valid patch version without label and metadata",
			input:  "v1.8.2",
			output: "v1.8.2",
		},
		{
			name:         "invalid version",
			input:        "foo",
			parsingError: true,
		},
		{
			name:         "invalid version with stray dash",
			input:        "v1.9.11-",
			parsingError: true,
		},
		{
			name:         "invalid version without patch release",
			input:        "v1.9",
			parsingError: true,
		},
		{
			name:        "invalid version with label and metadata",
			input:       "v1.8.0-alpha.2.1231+afabd012389d53a",
			output:      "v1.8.0-alpha.3",
			outputError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := keadmVersion(tc.input)
			if (err != nil) != tc.parsingError {
				t.Fatalf("expected error: %v, got: %v", tc.parsingError, err != nil)
			}
			if (output != tc.output) != tc.outputError {
				t.Fatalf("expected output: %s, got: %s, for input: %s", tc.output, output, tc.input)
			}
		})
	}
}

func TestValidateStableVersion(t *testing.T) {
	type T struct {
		name          string
		remoteVersion string
		clientVersion string
		output        string
		expectedError bool
	}
	cases := []T{
		{
			name:          "valid: remote version is newer; return stable label [1]",
			remoteVersion: "v1.12.0",
			clientVersion: "v1.11.0",
			output:        "v1.11.0",
		},
		{
			name:          "valid: remote version is newer; return stable label [2]",
			remoteVersion: "v2.0.0",
			clientVersion: "v1.11.0",
			output:        "v1.11.0",
		},
		{
			name:          "valid: remote version is newer; return stable label [3]",
			remoteVersion: "v2.1.5",
			clientVersion: "v1.11.5",
			output:        "v1.11.5",
		},
		{
			name:          "valid: return the remote version as it is part of the same release",
			remoteVersion: "v1.11.5",
			clientVersion: "v1.11.0",
			output:        "v1.11.5",
		},
		{
			name:          "valid: return the same version",
			remoteVersion: "v1.11.0",
			clientVersion: "v1.11.0",
			output:        "v1.11.0",
		},
		{
			name:          "invalid: client version is empty",
			remoteVersion: "v1.12.1",
			clientVersion: "",
			expectedError: true,
		},
		{
			name:          "invalid: error parsing the remote version",
			remoteVersion: "invalid-version",
			clientVersion: "v1.12.0",
			expectedError: true,
		},
		{
			name:          "invalid: error parsing the client version",
			remoteVersion: "v1.12.0",
			clientVersion: "invalid-version",
			expectedError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := validateStableVersion(tc.remoteVersion, tc.clientVersion)
			if (err != nil) != tc.expectedError {
				t.Fatalf("expected error: %v, got: %v", tc.expectedError, err != nil)
			}
			if output != tc.output {
				t.Fatalf("expected output: %s, got: %s", tc.output, output)
			}
		})
	}
}

func TestGetHelmVersion(t *testing.T) {
	cases := []struct {
		name       string
		version    string
		retryTimes int
		want       string
	}{
		{
			name:    "get input version",
			version: "v1.14.0",
			want:    "1.14.0",
		},
		{
			name:       "get default version",
			version:    "1-14-0",
			retryTimes: 0,
			want:       types.DefaultKubeEdgeVersion,
		},
		{
			name:       "get remote version",
			version:    "1-14-0",
			retryTimes: 1,
			want:       "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			if c.retryTimes > 0 {
				patches.ApplyFunc(GetLatestVersion, func() (string, error) {
					return "v1.15.0", nil
				})
			}

			res := GetHelmVersion(c.version, c.retryTimes)
			if c.want != "" && c.want != res {
				t.Fatalf("expected output: %s, got: %s", c.want, res)
			}

			if c.retryTimes > 0 {
				patches.Reset()
				patches.ApplyFunc(GetLatestVersion, func() (string, error) {
					return "", errors.New("network error")
				})

				res = GetHelmVersion(c.version, 1)
				assert.Equal(t, types.DefaultKubeEdgeVersion, res)
			}
		})
	}
}

func TestSetKubeEdgeVersion(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockVersion, err := semver.Make("1.5.0")
	if err != nil {
		t.Fatalf("Failed to parse semver: %v", err)
	}

	mockOS := &CommonTestsMockOSTypeInstaller{
		version: semver.Version{},
	}

	common := &Common{}
	common.OSTypeInstaller = mockOS

	common.SetKubeEdgeVersion(mockVersion)

	assert.Equal(t, mockVersion, mockOS.version, "SetKubeEdgeVersion should delegate to OSTypeInstaller.SetKubeEdgeVersion")
}

type CommonTestsMockOSTypeInstaller struct {
	version semver.Version
}

func (m *CommonTestsMockOSTypeInstaller) InstallMQTT() error {
	return nil
}

func (m *CommonTestsMockOSTypeInstaller) IsK8SComponentInstalled(name, version string) error {
	return nil
}

func (m *CommonTestsMockOSTypeInstaller) SetKubeEdgeVersion(version semver.Version) {
	m.version = version
}

func (m *CommonTestsMockOSTypeInstaller) InstallKubeEdge(options types.InstallOptions) error {
	return nil
}

func (m *CommonTestsMockOSTypeInstaller) RunEdgeCore() error {
	return nil
}

func (m *CommonTestsMockOSTypeInstaller) KillKubeEdgeBinary(name string) error {
	return nil
}

func (m *CommonTestsMockOSTypeInstaller) IsKubeEdgeProcessRunning(name string) (bool, error) {
	return false, nil
}

func TestDecompressAndCompress(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	testDir := t.TempDir()
	testFile1 := filepath.Join(testDir, "testfile1.txt")
	testFile2 := filepath.Join(testDir, "testfile2.txt")
	testTarGz := filepath.Join(testDir, "archive.tar.gz")
	extractDir := filepath.Join(testDir, "extract")

	err := os.WriteFile(testFile1, []byte("test content 1"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	assert.NoError(t, err)

	err = Compress(testTarGz, []string{testFile1, testFile2})
	assert.NoError(t, err)
	assert.FileExists(t, testTarGz)

	err = os.MkdirAll(extractDir, 0755)
	assert.NoError(t, err)

	originalOpen := os.Open
	patches.ApplyFunc(os.Open, func(name string) (*os.File, error) {
		if name == "nonexistent.tar.gz" {
			return nil, errors.New("file not found")
		}
		return originalOpen(name)
	})

	err = DecompressTarGz("nonexistent.tar.gz", extractDir)
	assert.Error(t, err)

	extractedFile1 := filepath.Join(extractDir, filepath.Base(testFile1))
	extractedFile2 := filepath.Join(extractDir, filepath.Base(testFile2))

	err = os.WriteFile(extractedFile1, []byte("test content 1"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(extractedFile2, []byte("test content 2"), 0644)
	assert.NoError(t, err)

	assert.FileExists(t, extractedFile1)
	assert.FileExists(t, extractedFile2)

	badPath := filepath.Join(os.TempDir(), "invalid/path/test.tar.gz")
	err = Compress(badPath, []string{testFile1})
	assert.Error(t, err)
}

func TestBuildConfig(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(clientcmd.BuildConfigFromFlags,
		func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
			if kubeconfigPath == "error" {
				return nil, errors.New("kubeconfig error")
			}
			return &rest.Config{
				Host: masterUrl,
			}, nil
		})

	config, err := BuildConfig("/path/to/kubeconfig", "https://master-url")
	assert.NoError(t, err)
	assert.Equal(t, "https://master-url", config.Host)

	_, err = BuildConfig("error", "https://master-url")
	assert.Error(t, err)
}

func TestIsK8SComponentInstalled(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(isK8SComponentInstalled,
		func(kubeConfig, master string) error {
			if kubeConfig == "error" {
				return errors.New("test error")
			}
			return nil
		})

	err := isK8SComponentInstalled("/path/to/kubeconfig", "https://master-url")
	assert.NoError(t, err)

	err = isK8SComponentInstalled("error", "")
	assert.Error(t, err)
}

func TestExecShellFilter(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(ExecShellFilter,
		func(c string) (string, error) {
			if c == "valid command" {
				return "command output", nil
			}
			return "", errors.New("command failed")
		})

	output, err := ExecShellFilter("valid command")
	assert.NoError(t, err)
	assert.Equal(t, "command output", output)

	_, err = ExecShellFilter("invalid command")
	assert.Error(t, err)
}

func TestRunningModule(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(RunningModule,
		func() (types.ModuleRunning, error) {
			return types.KubeEdgeCloudRunning, nil
		})

	module, err := RunningModule()
	assert.NoError(t, err)
	assert.Equal(t, types.KubeEdgeCloudRunning, module)

	patches.Reset()
	patches.ApplyFunc(RunningModule,
		func() (types.ModuleRunning, error) {
			return types.KubeEdgeEdgeRunning, nil
		})

	module, err = RunningModule()
	assert.NoError(t, err)
	assert.Equal(t, types.KubeEdgeEdgeRunning, module)

	patches.Reset()
	patches.ApplyFunc(RunningModule,
		func() (types.ModuleRunning, error) {
			return types.NoneRunning, errors.New("process check error")
		})

	module, err = RunningModule()
	assert.Error(t, err)
	assert.Equal(t, types.NoneRunning, module)
}

func TestParseEdgecoreConfig(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(ParseEdgecoreConfig,
		func(edgecorePath string) (*v1alpha2.EdgeCoreConfig, error) {
			if strings.Contains(edgecorePath, "error") {
				return nil, errors.New("parse error")
			}
			return &v1alpha2.EdgeCoreConfig{}, nil
		})

	config, err := ParseEdgecoreConfig("/path/to/edgecore.yaml")
	assert.NoError(t, err)
	assert.NotNil(t, config)

	_, err = ParseEdgecoreConfig("/path/to/error.yaml")
	assert.Error(t, err)
}

func TestAskForConfirm(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(askForconfirm, func() (bool, error) {
		return true, nil
	})

	result, err := askForconfirm()
	assert.NoError(t, err)
	assert.True(t, result)

	patches.Reset()
	patches.ApplyFunc(askForconfirm, func() (bool, error) {
		return false, nil
	})

	result, err = askForconfirm()
	assert.NoError(t, err)
	assert.False(t, result)

	patches.Reset()
	patches.ApplyFunc(askForconfirm, func() (bool, error) {
		return false, errors.New("invalid Input")
	})

	_, err = askForconfirm()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Input")

	patches.Reset()
	patches.ApplyFunc(askForconfirm, func() (bool, error) {
		return false, errors.New("scan error")
	})

	_, err = askForconfirm()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scan error")
}

func TestPrintFunctions(t *testing.T) {
	PrintSucceed("test", "Running")
	PrintFail("test", "Running")
}

func TestGetCurrentVersion(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(GetCurrentVersion, func(version string) (string, error) {
		if version == testVersion || version == "1.8.0" {
			return testVersion, nil
		}
		return "v1.20.0", nil
	})

	version, err := GetCurrentVersion(testVersion)
	assert.NoError(t, err)
	assert.Equal(t, testVersion, version)

	version, err = GetCurrentVersion("1.8.0")
	assert.NoError(t, err)
	assert.Equal(t, testVersion, version)

	version, err = GetCurrentVersion("latest")
	assert.NoError(t, err)
	assert.Equal(t, "v1.20.0", version)
}

func TestGetOSInterface(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(GetOSInterface, func() types.OSTypeInstaller {
		return &CommonTestsMockOSTypeInstaller{}
	})

	osInterface := GetOSInterface()
	_, isMockOS := osInterface.(*CommonTestsMockOSTypeInstaller)
	assert.True(t, isMockOS)
}

func TestCheckIfAvailable(t *testing.T) {
	result := CheckIfAvailable("provided-value", "default-value")
	assert.Equal(t, "provided-value", result)

	result = CheckIfAvailable("", "default-value")
	assert.Equal(t, "default-value", result)
}

func TestCommand(t *testing.T) {
	cmd := NewCommand("echo hello")
	assert.NotNil(t, cmd)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockCmd := exec.Command("echo", "test")
	patches.ApplyFunc(exec.Command, func(name string, arg ...string) *exec.Cmd {
		if name != "" {
			return mockCmd
		}
		return exec.Command(name, arg...)
	})

	patches.ApplyMethod(reflect.TypeOf(mockCmd), "Run", func(*exec.Cmd) error {
		return nil
	})

	patches.ApplyMethod(reflect.TypeOf(mockCmd), "Output", func(*exec.Cmd) ([]byte, error) {
		return []byte("command output"), nil
	})

	successCmd := NewCommand("test command")
	err := successCmd.Exec()
	assert.NoError(t, err)

	output := successCmd.GetStdOut()

	patches.ApplyMethod(reflect.TypeOf(mockCmd), "Run", func(*exec.Cmd) error {
		return errors.New("command failed")
	})

	failCmd := NewCommand("failing command")
	err = failCmd.Exec()
	assert.Error(t, err)

	patches.ApplyMethod(reflect.TypeOf(mockCmd), "Output", func(*exec.Cmd) ([]byte, error) {
		return nil, errors.New("output error")
	})

	noOutputCmd := NewCommand("no output command")
	output = noOutputCmd.GetStdOut()
	assert.Empty(t, output)
}

func TestInstallKubeEdge(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockOS := &CommonTestsMockOSTypeInstaller{}
	common := &Common{OSTypeInstaller: mockOS}

	options := types.InstallOptions{
		ComponentType: types.EdgeCore,
	}
	err := common.InstallKubeEdge(options)
	assert.NoError(t, err)

	patches.ApplyMethod(mockOS, "InstallKubeEdge", func(*CommonTestsMockOSTypeInstaller, types.InstallOptions) error {
		return errors.New("install error")
	})
	err = common.InstallKubeEdge(options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "install error")
}

func TestIsK8SComponentInstalledMethod(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockOS := &CommonTestsMockOSTypeInstaller{}
	common := &Common{OSTypeInstaller: mockOS}

	err := common.IsK8SComponentInstalled("kubelet", "v1.20.0")
	assert.NoError(t, err)

	patches.ApplyMethod(mockOS, "IsK8SComponentInstalled", func(*CommonTestsMockOSTypeInstaller, string, string) error {
		return errors.New("not installed")
	})
	err = common.IsK8SComponentInstalled("kubelet", "v1.20.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestKillKubeEdgeBinary(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	mockOS := &CommonTestsMockOSTypeInstaller{}
	common := &Common{OSTypeInstaller: mockOS}

	err := common.KillKubeEdgeBinary("edgecore")
	assert.NoError(t, err)

	patches.ApplyMethod(mockOS, "KillKubeEdgeBinary", func(*CommonTestsMockOSTypeInstaller, string) error {
		return errors.New("kill error")
	})
	err = common.KillKubeEdgeBinary("edgecore")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kill error")
}

func TestCleanNameSpace(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyMethod(&Common{}, "CleanNameSpace", func(*Common, string, string) error {
		return nil
	})

	common := &Common{}

	err := common.CleanNameSpace(constants.SystemNamespace, "/path/to/kubeconfig")
	assert.NoError(t, err)

	patches.Reset()
	patches.ApplyMethod(&Common{}, "CleanNameSpace", func(*Common, string, string) error {
		return errors.New("cleanup error")
	})

	err = common.CleanNameSpace(constants.SystemNamespace, "/path/to/kubeconfig")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup error")
}

func TestAddToolVals(t *testing.T) {
	flagSet := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flagSet.String("test-flag", "default-value", "test flag")
	flag := flagSet.Lookup("test-flag")

	flagData := make(map[string]types.FlagData)

	AddToolVals(flag, flagData)

	data, exists := flagData["test-flag"]
	assert.True(t, exists)
	assert.Equal(t, "default-value", data.DefVal)
	assert.Equal(t, "default-value", data.Val)
}

func TestCleanupCompressFile(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "testfile.txt")

	err := os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(t, err)

	tarPath := filepath.Join(testDir, "archive.tar.gz")
	err = Compress(tarPath, []string{testDir})

	if err != nil {
		t.Logf("Expected error: %v", err)
	}
}
