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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	"k8s.io/apimachinery/pkg/version"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
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
	if pm := GetPackageManager(); pm == "" {
		t.Errorf("failed to get package manager")
	}
}

func TestGetLatestVersion(t *testing.T) {
	_, err := GetLatestVersion()
	if err != nil {
		t.Errorf("failed to query kubeedge version: %v", err)
	}
}

func TestHasSystemd(t *testing.T) {
	HasSystemd()
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	ef, err := os.CreateTemp(dir, "FileExist")
	if err == nil {
		if !FileExists(ef.Name()) {
			t.Fatalf("file %v should exist", ef.Name())
		}
	}

	nonexistentDir := filepath.Join(dir, "not_exists_dir")
	notExistFile := filepath.Join(nonexistentDir, "not_exist_file")

	if FileExists(notExistFile) {
		t.Fatalf("file %v should not exist", notExistFile)
	}
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
