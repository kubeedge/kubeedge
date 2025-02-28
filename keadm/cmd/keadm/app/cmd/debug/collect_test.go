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

package debug

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestCollect_NewCollect(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCollect()

	assert.NotNil(cmd)
	assert.Equal("collect", cmd.Use)
	assert.Equal("Obtain all the data of the current node", cmd.Short)
	assert.Equal(edgecollectLongDescription, cmd.Long)
	assert.Equal(edgecollectExample, cmd.Example)
	assert.NotNil(cmd.Run)

	subcommands := cmd.Commands()
	assert.Empty(subcommands)

	expectedFlags := []struct {
		flagName    string
		shorthand   string
		defaultVal  string
		expectedVal string
	}{
		{
			flagName:    "config",
			shorthand:   "c",
			defaultVal:  common.EdgecoreConfigPath,
			expectedVal: common.EdgecoreConfigPath,
		},
		{
			flagName:    "detail",
			shorthand:   "d",
			defaultVal:  "false",
			expectedVal: "false",
		},
		{
			flagName:    "output-path",
			shorthand:   "o",
			defaultVal:  ".",
			expectedVal: ".",
		},
		{
			flagName:    "log-path",
			shorthand:   "l",
			defaultVal:  util.KubeEdgeLogPath,
			expectedVal: util.KubeEdgeLogPath,
		},
	}

	for _, tt := range expectedFlags {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flag(tt.flagName)
			assert.Equal(tt.flagName, flag.Name)
			assert.Equal(tt.defaultVal, flag.DefValue)
			assert.Equal(tt.expectedVal, flag.Value.String())
			assert.Equal(tt.shorthand, flag.Shorthand)
		})
	}
}

func TestCollect_AddCollectOtherFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}

	co := newCollectOptions()
	addCollectOtherFlags(cmd, co)

	expectedFlags := []struct {
		flagName    string
		shorthand   string
		defaultVal  string
		expectedVal string
	}{
		{
			flagName:    "config",
			shorthand:   "c",
			defaultVal:  common.EdgecoreConfigPath,
			expectedVal: common.EdgecoreConfigPath,
		},
		{
			flagName:    "detail",
			shorthand:   "d",
			defaultVal:  "false",
			expectedVal: "false",
		},
		{
			flagName:    "output-path",
			shorthand:   "o",
			defaultVal:  ".",
			expectedVal: ".",
		},
		{
			flagName:    "log-path",
			shorthand:   "l",
			defaultVal:  util.KubeEdgeLogPath,
			expectedVal: util.KubeEdgeLogPath,
		},
	}

	for _, tt := range expectedFlags {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flag(tt.flagName)
			assert.Equal(tt.flagName, flag.Name)
			assert.Equal(tt.defaultVal, flag.DefValue)
			assert.Equal(tt.expectedVal, flag.Value.String())
			assert.Equal(tt.shorthand, flag.Shorthand)
		})
	}
}

func TestCollect_NewCollectOptions(t *testing.T) {
	assert := assert.New(t)

	co := newCollectOptions()
	assert.NotNil(co)

	assert.Equal(common.EdgecoreConfigPath, co.Config)
	assert.Equal(".", co.OutputPath)
	assert.Equal(false, co.Detail)
}
func TestCollect_VerificationParameters(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-collect-*")
	assert.NoError(err)
	defer os.RemoveAll(tmpDir)

	// Create a mock config file
	mockConfigPath := filepath.Join(tmpDir, "config")
	err = os.WriteFile(mockConfigPath, []byte("mock config"), 0644)
	assert.NoError(err)

	tests := []struct {
		name        string
		options     *common.CollectOptions
		expectError bool
	}{
		{
			name: "valid parameters",
			options: &common.CollectOptions{
				Config:     mockConfigPath,
				OutputPath: tmpDir,
				Detail:     true,
			},
			expectError: false,
		},
		{
			name: "non-existent config",
			options: &common.CollectOptions{
				Config:     "/nonexistent/config",
				OutputPath: tmpDir,
			},
			expectError: true,
		},
		{
			name: "non-existent output path",
			options: &common.CollectOptions{
				Config:     mockConfigPath,
				OutputPath: "/nonexistent/path",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerificationParameters(tt.options)
			if tt.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestCollect_MakeDirTmp(t *testing.T) {
	assert := assert.New(t)

	tmpName, timeNow, err := makeDirTmp()
	assert.NoError(err)
	defer os.RemoveAll(tmpName)

	// Verify the directory was created
	_, err = os.Stat(tmpName)
	assert.NoError(err)

	// Verify the directory name format
	assert.Contains(tmpName, "/tmp/edge_")
	assert.Contains(tmpName, timeNow)
}

func TestCollect_PrintDetail(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test when printDeatilFlag is false
	printDeatilFlag = false
	printDetail("test message")

	// Test when printDeatilFlag is true
	printDeatilFlag = true
	printDetail("test message")

	// Restore stdout
	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	// When flag is true, message should be printed
	assert.Contains(t, string(out), "test message")
}

func TestCollect_ExecuteShell(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-shell-*")
	assert.NoError(err)
	defer os.RemoveAll(tmpDir)

	// Test with a simple command
	err = ExecuteShell("echo 'test' > %s/test.txt", tmpDir)
	assert.NoError(err)

	// Verify the file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	assert.NoError(err)
	assert.Contains(string(content), "test")
}

func TestCollect_CopyFile(t *testing.T) {
	assert := assert.New(t)

	// Create temporary directories for testing
	srcDir, err := os.MkdirTemp("", "test-src-*")
	assert.NoError(err)
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "test-dst-*")
	assert.NoError(err)
	defer os.RemoveAll(dstDir)

	// Create a test file
	testFile := filepath.Join(srcDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	assert.NoError(err)

	// Test copying the file
	err = CopyFile(testFile, dstDir)
	assert.NoError(err)

	// Verify the file was copied
	copiedFile := filepath.Join(dstDir, "test.txt")
	_, err = os.Stat(copiedFile)
	assert.NoError(err)
}
func TestCollect_CollectRuntimeData(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-runtime-*")
	assert.NoError(err)
	defer os.RemoveAll(tmpDir)

	// Create runtime directory
	runtimeDir := filepath.Join(tmpDir, "runtime")

	tests := []struct {
		name        string
		setupFunc   func() error
		expectError bool
	}{
		{
			name: "successful collection",
			setupFunc: func() error {
				// Create mock docker service file
				serviceDir := filepath.Join(tmpDir, "lib/systemd/system")
				if err := os.MkdirAll(serviceDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(
					filepath.Join(serviceDir, "docker.service"),
					[]byte("docker service file"),
					0644,
				)
			},
			expectError: false,
		},
		{
			name: "mkdir error",
			setupFunc: func() error {
				// Create a file where directory should be to force error
				return os.WriteFile(runtimeDir, []byte(""), 0644)
			},
			expectError: true,
		}, {
			name: "directory exists as file",
			setupFunc: func() error {
				// Create a file where directory should be
				return os.WriteFile(runtimeDir, []byte(""), 0644)
			},
			expectError: true,
		},
		{
			name: "copy file error",
			setupFunc: func() error {
				// Create directory but don't create required files
				return os.MkdirAll(runtimeDir, 0755)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(runtimeDir)

			if tt.setupFunc != nil {
				err := tt.setupFunc()
				assert.NoError(err)
			}

			err := collectRuntimeData(runtimeDir)
			if tt.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
func TestCollect_CollectEdgecoreData(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-edgecore-*")
	assert.NoError(err)
	defer os.RemoveAll(tmpDir)

	// Create edgecore directory for test
	edgecoreDir := filepath.Join(tmpDir, "edgecore")

	// Create a valid config
	config := &v1alpha2.EdgeCoreConfig{
		DataBase: &v1alpha2.DataBase{
			DataSource: "/var/lib/kubeedge/edgecore.db",
		},
		Modules: &v1alpha2.Modules{
			EdgeHub: &v1alpha2.EdgeHub{
				TLSCertFile:       "/etc/kubeedge/certs/server.crt",
				TLSPrivateKeyFile: "/etc/kubeedge/certs/server.key",
				TLSCAFile:         "/etc/kubeedge/ca/rootCA.crt",
			},
		},
	}
	ops := &common.CollectOptions{
		LogPath: "/var/log/kubeedge/edgecore.log",
	}

	tests := []struct {
		name        string
		setup       func() error
		expectError bool
	}{
		{
			name: "directory creation failed - already exists as file",
			setup: func() error {
				return os.WriteFile(edgecoreDir, []byte(""), 0644)
			},
			expectError: true,
		},
		{
			name: "execution with missing files",
			setup: func() error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cleanup from previous test
			os.RemoveAll(edgecoreDir)

			if tt.setup != nil {
				err := tt.setup()
				assert.NoError(err)
			}

			err := collectEdgecoreData(edgecoreDir, config, ops)
			if tt.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
func TestCollect_ExecuteCollect(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-execute-collect-*")
	assert.NoError(err)
	defer os.RemoveAll(tmpDir)

	// Create a mock config file with proper YAML formatting
	mockConfigPath := filepath.Join(tmpDir, "edgecore.yaml")
	mockConfig := `
modules:
edged:
	hostName: test-node
edgeHub:
	tlsCertFile: tls.crt
	tlsPrivateKeyFile: tls.key
`
	err = os.WriteFile(mockConfigPath, []byte(mockConfig), 0644)
	assert.NoError(err)

	tests := []struct {
		name        string
		options     *common.CollectOptions
		expectError bool
	}{
		{
			name: "invalid config path",
			options: &common.CollectOptions{
				Config:     "/nonexistent/config",
				OutputPath: tmpDir,
			},
			expectError: true,
		},
		{
			name: "invalid output path",
			options: &common.CollectOptions{
				Config:     mockConfigPath,
				OutputPath: "/nonexistent/path",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteCollect(tt.options)
			if tt.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
