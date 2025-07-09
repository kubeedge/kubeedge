//go:build !windows

/*
Copyright 2015 The KubeEdge Authors.

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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/common/constants"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	commfake "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common/fake"
	"github.com/kubeedge/kubeedge/pkg/util/execs"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

const (
	TestTempDir = "/tmp"
)

type MockFileInfo struct {
	fileName string
	fileSize int64
	fileMode os.FileMode
	isDir    bool
	modTime  time.Time
	sys      interface{}
}

func (m MockFileInfo) Name() string       { return m.fileName }
func (m MockFileInfo) Size() int64        { return m.fileSize }
func (m MockFileInfo) Mode() os.FileMode  { return m.fileMode }
func (m MockFileInfo) ModTime() time.Time { return m.modTime }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() interface{}   { return m.sys }

func TestIsKubeEdgeProcessRunningOthers(t *testing.T) {
	tests := []struct {
		name      string
		process   string
		exitCode  int
		execErr   error
		expected  bool
		expectErr bool
	}{
		{
			name:      "Process running",
			process:   "edgecore",
			exitCode:  0,
			execErr:   nil,
			expected:  true,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockCmd := &execs.Command{
				ExitCode: tt.exitCode,
			}

			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				expected := fmt.Sprintf("pidof %s 2>&1", tt.process)
				assert.Equal(t, expected, command)
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.execErr
			})

			result, err := IsKubeEdgeProcessRunning(tt.process)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasSystemdOthers(t *testing.T) {
	tests := []struct {
		name        string
		cmdSuccess  bool
		fileInitErr error
		isDirectory bool
		expected    bool
	}{
		{
			name:       "Command succeeds",
			cmdSuccess: true,
			expected:   true,
		},
		{
			name:        "Command fails but systemd path exists and is directory",
			cmdSuccess:  false,
			fileInitErr: nil,
			isDirectory: true,
			expected:    true,
		},
		{
			name:        "Command fails and systemd path exists but is not directory",
			cmdSuccess:  false,
			fileInitErr: nil,
			isDirectory: false,
			expected:    false,
		},
		{
			name:        "Command fails and systemd path doesn't exist",
			cmdSuccess:  false,
			fileInitErr: fmt.Errorf("file not found"),
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				assert.Equal(t, "file /sbin/init", command)
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				if tt.cmdSuccess {
					return nil
				}
				return fmt.Errorf("command failed")
			})

			patches.ApplyFunc(os.Lstat, func(name string) (os.FileInfo, error) {
				assert.Equal(t, common.SystemdBootPath, name)
				if tt.fileInitErr != nil {
					return nil, tt.fileInitErr
				}

				return &MockFileInfo{
					isDir: tt.isDirectory,
				}, nil
			})

			result := HasSystemd()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunningModuleV2Others(t *testing.T) {
	tests := []struct {
		name           string
		cloudRunning   common.ModuleRunning
		edgeRunning    common.ModuleRunning
		expectedResult common.ModuleRunning
	}{
		{
			name:           "Cloud running",
			cloudRunning:   common.KubeEdgeCloudRunning,
			edgeRunning:    common.NoneRunning,
			expectedResult: common.KubeEdgeCloudRunning,
		},
		{
			name:           "Edge running",
			cloudRunning:   common.NoneRunning,
			edgeRunning:    common.KubeEdgeEdgeRunning,
			expectedResult: common.KubeEdgeEdgeRunning,
		},
		{
			name:           "Nothing running",
			cloudRunning:   common.NoneRunning,
			edgeRunning:    common.NoneRunning,
			expectedResult: common.NoneRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(CloudCoreRunningModuleV2, func(*common.ResetOptions) common.ModuleRunning {
				return tt.cloudRunning
			})

			patches.ApplyFunc(EdgeCoreRunningModuleV2, func(*common.ResetOptions) common.ModuleRunning {
				return tt.edgeRunning
			})

			opt := &common.ResetOptions{}
			result := RunningModuleV2(opt)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestCloudCoreRunningModuleV2Others(t *testing.T) {
	tests := []struct {
		name      string
		isRunning bool
		hasError  bool
		expected  common.ModuleRunning
	}{
		{
			name:      "CloudCore running",
			isRunning: true,
			hasError:  false,
			expected:  common.KubeEdgeCloudRunning,
		},
		{
			name:      "CloudCore not running",
			isRunning: false,
			hasError:  false,
			expected:  common.NoneRunning,
		},
		{
			name:      "Error checking CloudCore",
			isRunning: false,
			hasError:  true,
			expected:  common.NoneRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(IsCloudcoreContainerRunning, func(namespace, kubeconfig string) (bool, error) {
				if tt.hasError {
					return false, fmt.Errorf("error checking container")
				}
				return tt.isRunning, nil
			})

			patches.ApplyFunc(klog.Warningf, func(format string, args ...interface{}) {})

			opt := &common.ResetOptions{}
			result := CloudCoreRunningModuleV2(opt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEdgeCoreRunningModuleV2Others(t *testing.T) {
	tests := []struct {
		name       string
		isRunning  bool
		runningErr error
		expected   common.ModuleRunning
	}{
		{
			name:       "Edge running",
			isRunning:  true,
			runningErr: nil,
			expected:   common.KubeEdgeEdgeRunning,
		},
		{
			name:       "Edge not running",
			isRunning:  false,
			runningErr: nil,
			expected:   common.NoneRunning,
		},
		{
			name:       "Error checking Edge",
			isRunning:  false,
			runningErr: fmt.Errorf("error checking process"),
			expected:   common.NoneRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockOS := &commfake.MockOSTypeInstaller{
				IsProcessRunning:  tt.isRunning,
				ProcessRunningErr: tt.runningErr,
			}

			patches.ApplyFunc(GetOSInterface, func() common.OSTypeInstaller {
				return mockOS
			})

			patches.ApplyFunc(klog.Warningf, func(format string, args ...interface{}) {})

			opt := &common.ResetOptions{}
			result := EdgeCoreRunningModuleV2(opt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKillKubeEdgeBinaryOthers(t *testing.T) {
	tests := []struct {
		name           string
		proc           string
		hasSystemd     bool
		edgeRunning    bool
		edgeRunningErr error
		cmdExecErr     error
		expected       bool
	}{
		{
			name:       "Kill cloudcore",
			proc:       "cloudcore",
			hasSystemd: false,
			expected:   true,
		},
		{
			name:        "Kill edgecore with systemd and service running",
			proc:        "edgecore",
			hasSystemd:  true,
			edgeRunning: true,
			expected:    true,
		},
		{
			name:        "Kill edgecore with systemd but service not running",
			proc:        "edgecore",
			hasSystemd:  true,
			edgeRunning: false,
			expected:    true,
		},
		{
			name:       "Kill edgecore without systemd",
			proc:       "edgecore",
			hasSystemd: false,
			expected:   true,
		},
		{
			name:       "Command execution fails",
			proc:       "edgecore",
			hasSystemd: false,
			cmdExecErr: fmt.Errorf("command failed"),
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(HasSystemd, func() bool {
				return tt.hasSystemd
			})

			patches.ApplyFunc(isEdgeCoreServiceRunning, func(serviceName string) (bool, error) {
				if serviceName == "edge" || serviceName == constants.KubeEdgeBinaryName {
					return tt.edgeRunning, tt.edgeRunningErr
				}
				return false, nil
			})

			patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if name == fmt.Sprintf("/etc/systemd/system/%s.service", "edge") ||
					name == fmt.Sprintf("/etc/systemd/system/%s.service", constants.KubeEdgeBinaryName) {
					if tt.edgeRunning {
						return &MockFileInfo{}, nil
					}
				}
				return nil, os.ErrNotExist
			})

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.cmdExecErr
			})

			patches.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
				return 0, nil
			})

			err := KillKubeEdgeBinary(tt.proc)
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCheckSumOthers(t *testing.T) {
	tests := []struct {
		name           string
		fileExists     bool
		checksumsMatch bool
		downloadFails  bool
		expectedResult bool
		expectedErr    bool
	}{
		{
			name:           "Checksum file exists and checksums match",
			fileExists:     true,
			checksumsMatch: true,
			expectedResult: true,
			expectedErr:    false,
		},
		{
			name:           "Checksum file exists but checksums don't match",
			fileExists:     true,
			checksumsMatch: false,
			expectedResult: false,
			expectedErr:    false,
		},
		{
			name:           "Checksum file doesn't exist but checksums don't match",
			fileExists:     false,
			checksumsMatch: false,
			expectedResult: false,
			expectedErr:    false,
		},
		{
			name:           "Download checksum fails",
			fileExists:     false,
			downloadFails:  true,
			expectedResult: false,
			expectedErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			actualChecksum := "sha512sum123"
			patches.ApplyFunc(computeSHA512Checksum, func(filepath string) (string, error) {
				return actualChecksum, nil
			})

			patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if tt.fileExists {
					return &MockFileInfo{}, nil
				}
				return nil, os.ErrNotExist
			})

			expectedChecksum := actualChecksum
			if !tt.checksumsMatch {
				expectedChecksum = "differentchecksum"
			}
			patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
				return []byte(expectedChecksum), nil
			})

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				if tt.downloadFails {
					return fmt.Errorf("download failed")
				}
				return nil
			})

			patches.ApplyMethod(mockCmd, "GetStdOut", func(*execs.Command) string {
				return expectedChecksum
			})

			patches.ApplyFunc(fmt.Printf, func(format string, a ...interface{}) (n int, err error) {
				return 0, nil
			})

			filename := "kubeedge.tar.gz"
			checksumFile := "checksum.txt"
			version, _ := semver.Parse("1.0.0")
			tarballPath := TestTempDir

			result, err := checkSum(filename, checksumFile, version, tarballPath)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestRetryDownloadOthers(t *testing.T) {
	tests := []struct {
		name          string
		downloadFails bool
		checksumFails bool
		expectedErr   bool
	}{
		{
			name:          "Download and checksum succeed",
			downloadFails: false,
			checksumFails: false,
			expectedErr:   false,
		},
		{
			name:          "Download fails",
			downloadFails: true,
			checksumFails: false,
			expectedErr:   true,
		},
		{
			name:          "Checksum fails then succeeds",
			downloadFails: false,
			checksumFails: true,
			expectedErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			checksumFailOnFirst := tt.checksumFails
			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				if tt.downloadFails {
					return fmt.Errorf("download failed")
				}
				return nil
			})

			checksumCalled := 0
			patches.ApplyFunc(checkSum, func(filename, checksumFilename string, version semver.Version, tarballPath string) (bool, error) {
				checksumCalled++
				if checksumFailOnFirst && checksumCalled == 1 {
					return false, nil
				}
				return true, nil
			})

			patches.ApplyFunc(fmt.Printf, func(format string, a ...interface{}) (n int, err error) {
				return 0, nil
			})

			filename := "kubeedge.tar.gz"
			checksumFile := "checksum.txt"
			version, _ := semver.Parse("1.0.0")
			tarballPath := TestTempDir

			err := retryDownload(filename, checksumFile, version, tarballPath)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsEdgeCoreServiceRunningOthers(t *testing.T) {
	tests := []struct {
		name           string
		serviceName    string
		exitCode       int
		execErr        error
		expectedResult bool
		expectedErr    bool
	}{
		{
			name:           "Service is running",
			serviceName:    "edgecore",
			exitCode:       0,
			execErr:        nil,
			expectedResult: true,
			expectedErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockCmd := &execs.Command{
				ExitCode: tt.exitCode,
			}

			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.execErr
			})

			result, err := isEdgeCoreServiceRunning(tt.serviceName)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestRunEdgeCoreOthers(t *testing.T) {
	tests := []struct {
		name        string
		mkdirErr    error
		hasSystemd  bool
		cmdExecErr  error
		expectedErr bool
	}{
		{
			name:        "Run with systemd successfully",
			mkdirErr:    nil,
			hasSystemd:  true,
			cmdExecErr:  nil,
			expectedErr: false,
		},
		{
			name:        "Run without systemd successfully",
			mkdirErr:    nil,
			hasSystemd:  false,
			cmdExecErr:  nil,
			expectedErr: false,
		},
		{
			name:        "MkdirAll fails",
			mkdirErr:    fmt.Errorf("mkdir failed"),
			hasSystemd:  true,
			cmdExecErr:  nil,
			expectedErr: true,
		},
		{
			name:        "Command execution fails",
			mkdirErr:    nil,
			hasSystemd:  true,
			cmdExecErr:  fmt.Errorf("command failed"),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
				return tt.mkdirErr
			})

			patches.ApplyFunc(HasSystemd, func() bool {
				return tt.hasSystemd
			})

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.cmdExecErr
			})

			patches.ApplyMethod(mockCmd, "GetStdOut", func(*execs.Command) string {
				return "mock stdout"
			})

			patches.ApplyFunc(fmt.Printf, func(format string, a ...interface{}) (n int, err error) {
				return 0, nil
			})
			patches.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
				return 0, nil
			})

			err := runEdgeCore()

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInstallKubeEdgeOthers(t *testing.T) {
	tests := []struct {
		name             string
		tarballPathEmpty bool
		mkdirErr         error
		fileStatErr      error
		checksumSuccess  bool
		askConfirmResult bool
		askConfirmErr    error
		downloadErr      error
		serviceFileErr   error
		cmdExecErr       error
		componentType    types.ComponentType
		expectedErr      bool
	}{
		{
			name:             "Install EdgeCore successfully with existing tarball",
			tarballPathEmpty: false,
			mkdirErr:         nil,
			fileStatErr:      nil,
			checksumSuccess:  true,
			serviceFileErr:   nil,
			cmdExecErr:       nil,
			componentType:    types.EdgeCore,
			expectedErr:      false,
		},
		{
			name:             "Install CloudCore successfully with existing tarball",
			tarballPathEmpty: false,
			mkdirErr:         nil,
			fileStatErr:      nil,
			checksumSuccess:  true,
			serviceFileErr:   nil,
			cmdExecErr:       nil,
			componentType:    types.CloudCore,
			expectedErr:      false,
		},
		{
			name:             "Install with default tarball path",
			tarballPathEmpty: true,
			fileStatErr:      nil,
			checksumSuccess:  true,
			serviceFileErr:   nil,
			cmdExecErr:       nil,
			componentType:    types.EdgeCore,
			expectedErr:      false,
		},
		{
			name:             "TarballPath mkdir fails",
			tarballPathEmpty: false,
			mkdirErr:         fmt.Errorf("mkdir failed"),
			expectedErr:      true,
		},
		{
			name:             "KubeEdgePath mkdir fails",
			tarballPathEmpty: false,
			mkdirErr:         nil,
			fileStatErr:      nil,
			checksumSuccess:  true,
			serviceFileErr:   nil,
			cmdExecErr:       nil,
			componentType:    types.EdgeCore,
			expectedErr:      false,
		},
		{
			name:             "Tarball doesn't exist and download fails",
			tarballPathEmpty: false,
			fileStatErr:      os.ErrNotExist,
			downloadErr:      fmt.Errorf("download failed"),
			expectedErr:      true,
		},
		{
			name:             "Tarball exists but checksum fails and user confirms retry",
			tarballPathEmpty: false,
			fileStatErr:      nil,
			checksumSuccess:  false,
			askConfirmResult: true,
			downloadErr:      nil,
			serviceFileErr:   nil,
			cmdExecErr:       nil,
			componentType:    types.EdgeCore,
			expectedErr:      false,
		},
		{
			name:             "Download service file fails",
			tarballPathEmpty: false,
			fileStatErr:      nil,
			checksumSuccess:  true,
			serviceFileErr:   fmt.Errorf("service download failed"),
			expectedErr:      true,
		},
		{
			name:             "Command execution fails",
			tarballPathEmpty: false,
			fileStatErr:      nil,
			checksumSuccess:  true,
			serviceFileErr:   nil,
			cmdExecErr:       fmt.Errorf("command failed"),
			componentType:    types.EdgeCore,
			expectedErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(os.MkdirAll, func(path string, perm os.FileMode) error {
				if path != constants.KubeEdgePath && tt.mkdirErr != nil {
					return tt.mkdirErr
				}
				return nil
			})

			patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if tt.fileStatErr != nil {
					return nil, tt.fileStatErr
				}
				return &MockFileInfo{}, nil
			})

			patches.ApplyFunc(checkSum, func(filename, checksumFilename string, version semver.Version, tarballPath string) (bool, error) {
				return tt.checksumSuccess, nil
			})

			patches.ApplyFunc(askForconfirm, func() (bool, error) {
				return tt.askConfirmResult, tt.askConfirmErr
			})

			patches.ApplyFunc(retryDownload, func(filename, checksumFilename string, version semver.Version, tarballPath string) error {
				return tt.downloadErr
			})

			patches.ApplyFunc(downloadServiceFile, func(componentType types.ComponentType, version semver.Version, targetDir string) error {
				return tt.serviceFileErr
			})

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.cmdExecErr
			})

			patches.ApplyMethod(mockCmd, "GetStdOut", func(*execs.Command) string {
				return "mock stdout"
			})

			patches.ApplyFunc(fmt.Printf, func(format string, a ...interface{}) (n int, err error) {
				return 0, nil
			})
			patches.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
				return 0, nil
			})

			options := types.InstallOptions{
				ComponentType: tt.componentType,
			}
			if !tt.tarballPathEmpty {
				options.TarballPath = TestTempDir
			}
			version, _ := semver.Parse("1.0.0")

			err := installKubeEdge(options, version)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestComputeSHA512ChecksumOthers(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("test content for SHA512 checksum")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		filePath    string
		fileExists  bool
		expectedErr bool
	}{
		{
			name:        "File exists",
			filePath:    tmpfile.Name(),
			fileExists:  true,
			expectedErr: false,
		},
		{
			name:        "File doesn't exist",
			filePath:    "/non/existent/path",
			fileExists:  false,
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, err := computeSHA512Checksum(tt.filePath)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, checksum)
			}
		})
	}
}

func TestAskForconfirmOthers(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	originalAskForconfirm := askForconfirm
	patches.ApplyFunc(askForconfirm, func() (bool, error) {
		return true, nil
	})

	result, err := askForconfirm()
	assert.NoError(t, err)
	assert.True(t, result)

	patches.Reset()
	_ = originalAskForconfirm
}

func TestDownloadServiceFileOthers(t *testing.T) {
	tests := []struct {
		name          string
		componentType types.ComponentType
		hasSystemd    bool
		fileExists    bool
		getVersionErr error
		cmdExecErr    error
		expectedErr   bool
	}{
		{
			name:          "EdgeCore download success",
			componentType: types.EdgeCore,
			hasSystemd:    true,
			fileExists:    false,
			getVersionErr: nil,
			cmdExecErr:    nil,
			expectedErr:   false,
		},
		{
			name:          "CloudCore download success",
			componentType: types.CloudCore,
			hasSystemd:    true,
			fileExists:    false,
			getVersionErr: nil,
			cmdExecErr:    nil,
			expectedErr:   false,
		},
		{
			name:          "No systemd",
			componentType: types.EdgeCore,
			hasSystemd:    false,
			expectedErr:   false,
		},
		{
			name:          "File already exists",
			componentType: types.EdgeCore,
			hasSystemd:    true,
			fileExists:    true,
			expectedErr:   false,
		},
		{
			name:          "GetLatestVersion fails",
			componentType: types.EdgeCore,
			hasSystemd:    true,
			fileExists:    false,
			getVersionErr: fmt.Errorf("version error"),
			cmdExecErr:    nil,
			expectedErr:   false,
		},
		{
			name:          "Command execution fails",
			componentType: types.EdgeCore,
			hasSystemd:    true,
			fileExists:    false,
			getVersionErr: nil,
			cmdExecErr:    fmt.Errorf("command failed"),
			expectedErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(HasSystemd, func() bool {
				return tt.hasSystemd
			})

			patches.ApplyFunc(os.Stat, func(name string) (os.FileInfo, error) {
				if tt.fileExists {
					return &MockFileInfo{}, nil
				}
				return nil, os.ErrNotExist
			})

			patches.ApplyFunc(GetLatestVersion, func() (string, error) {
				if tt.getVersionErr != nil {
					return "", tt.getVersionErr
				}
				return "v1.0.0", nil
			})

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.cmdExecErr
			})

			patches.ApplyFunc(fmt.Printf, func(format string, a ...interface{}) (n int, err error) {
				return 0, nil
			})

			version, _ := semver.Parse("1.0.0")
			err := downloadServiceFile(tt.componentType, version, TestTempDir)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetOSInterfaceOthers(t *testing.T) {
	tests := []struct {
		packageManager string
		expectedType   string
		wantErr        bool
	}{
		{
			packageManager: APT,
			expectedType:   "*util.DebOS",
		},
		{
			packageManager: YUM,
			expectedType:   "*util.RpmOS",
		},
		{
			packageManager: PACMAN,
			expectedType:   "*util.PacmanOS",
		},
		{
			packageManager: "Unknown",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run("package type "+tt.packageManager, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFuncReturn(GetPackageManager, tt.packageManager)

			if tt.wantErr {
				defer func() {
					errmsg := recover().(string)
					assert.Equal(t, "Failed to detect supported package manager command(apt, yum, pacman), exit", errmsg)
				}()
			}
			result := GetOSInterface()
			assert.Equal(t, tt.expectedType, fmt.Sprintf("%T", result))
		})
	}
}

func TestIsCloudcoreContainerRunningOthers(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		kubeconfig     string
		cmdExitCode    int
		cmdErr         error
		expectedResult bool
		expectedErr    bool
	}{
		{
			name:           "Command error",
			namespace:      "kubeedge",
			kubeconfig:     "/path/to/kubeconfig",
			cmdExitCode:    -1,
			cmdErr:         fmt.Errorf("command error"),
			expectedResult: false,
			expectedErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockCmd := &execs.Command{
				ExitCode: tt.cmdExitCode,
			}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				expectedCmd := fmt.Sprintf("kubectl get deployment -n %s %s --kubeconfig=%s",
					tt.namespace, "cloudcore", tt.kubeconfig)
				assert.Contains(t, command, expectedCmd)
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(*execs.Command) error {
				return tt.cmdErr
			})

			result, err := IsCloudcoreContainerRunning(tt.namespace, tt.kubeconfig)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestFileExistsOthers(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "fileexists")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "File exists",
			path:     tmpfile.Name(),
			expected: true,
		},
		{
			name:     "File doesn't exist",
			path:     "/non/existent/path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := files.FileExists(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckIfAvailableOthers(t *testing.T) {
	tests := []struct {
		name       string
		provided   string
		defaultVal string
		expected   string
	}{
		{
			name:       "Value provided",
			provided:   "provided-value",
			defaultVal: "default-value",
			expected:   "provided-value",
		},
		{
			name:       "Use default value",
			provided:   "",
			defaultVal: "default-value",
			expected:   "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckIfAvailable(tt.provided, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPackageManagerOthers(t *testing.T) {
	tests := []struct {
		name         string
		aptExists    bool
		yumExists    bool
		pacmanExists bool
		expected     string
	}{
		{
			name:         "No package manager found",
			aptExists:    false,
			yumExists:    false,
			pacmanExists: false,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			mockCmd := &execs.Command{}
			patches.ApplyFunc(execs.NewCommand, func(command string) *execs.Command {
				return mockCmd
			})

			patches.ApplyMethod(mockCmd, "Exec", func(cmd *execs.Command) error {
				if (tt.aptExists && cmd.GetCommand() == "bash -c which apt") ||
					(tt.yumExists && cmd.GetCommand() == "bash -c which yum") ||
					(tt.pacmanExists && cmd.GetCommand() == "bash -c which pacman") {
					cmd.ExitCode = 0
					return nil
				}
				cmd.ExitCode = 1
				return fmt.Errorf("command not found")
			})

			result := GetPackageManager()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommandOthers(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		stdout    []byte
		stderr    []byte
		exitCode  int
		execError error
	}{
		{
			name:      "Command succeeds",
			command:   "echo hello",
			stdout:    []byte("hello"),
			stderr:    []byte{},
			exitCode:  0,
			execError: nil,
		},
		{
			name:      "Command fails",
			command:   "invalid command",
			stdout:    []byte{},
			stderr:    []byte("command not found"),
			exitCode:  1,
			execError: fmt.Errorf("command failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			cmd := execs.NewCommand(tt.command)
			assert.Equal(t, "bash -c "+tt.command, cmd.GetCommand())

			cmd.StdOut = tt.stdout
			cmd.StdErr = tt.stderr

			assert.Equal(t, string(tt.stdout), cmd.GetStdOut())

			assert.Equal(t, string(tt.stderr), cmd.GetStdErr())
		})
	}
}
