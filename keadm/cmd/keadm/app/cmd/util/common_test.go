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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/blang/semver"
	"k8s.io/apimachinery/pkg/version"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

var PrivateDownloadServiceFile = downloadServiceFile

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
		files, _ := ioutil.ReadDir(path.Dir(serviceFilePath))
		if len(files) > 1 {
			return fmt.Errorf("download redundancy files")
		}
		return err
	}
	var clean = func(testTmpDir string) {
		dir, err := ioutil.ReadDir(testTmpDir)
		if err != nil {
			t.Fatalf("failed to clean test tmp dir!\n")
		}
		for _, d := range dir {
			if err = os.RemoveAll(path.Join(testTmpDir, d.Name())); err != nil {
				t.Fatalf("failed to clean test tmp dir!\n")
			}
		}
	}

	testTmpDir, err := ioutil.TempDir("", "kubeedge")
	if err != nil {
		t.Fatalf("failed to create temp dir for testing:{%s}\n", err.Error())
	}
	defer os.RemoveAll(testTmpDir)

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
	hasSystemd()
}

func TestFileExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestTempFile_BadDir")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer os.RemoveAll(dir)

	ef, err := ioutil.TempFile(dir, "FileExist")
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
			tmpfile, err := ioutil.TempFile("", test.name)
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
