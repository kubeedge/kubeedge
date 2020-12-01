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
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/blang/semver"
	"k8s.io/apimachinery/pkg/version"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

var PrivateDownloadServiceFile = downloadServiceFile

func TestManagedKubernetesVersion(t *testing.T) {
	vers := version.Info{Minor: "17"}
	t.Run("test with minor version of 17", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "17+"
	t.Run("test with minor version of 17+", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "100"
	t.Run("test with minor version of 100", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "100+"
	t.Run("test with minor version of 100+", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err != nil {
			t.Fatalf("checked errored with: %s\n", err)
		}
	})

	vers.Minor = "3"
	t.Run("test with minor version of 3", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err == nil {
			t.Fatalf("check should return an error and didn't")
		}
	})

	vers.Minor = "3+"
	t.Run("test with minor version of 3+", func(t *testing.T) {
		err := checkKubernetesVersion(&vers)
		if err == nil {
			t.Fatalf("check should return an error and didn't")
		}
	})
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
	targetVersion = semver.Version{Major: 1, Minor: 0}
	serviceFilePath = testTmpDir + "/" + CloudServiceFile
	t.Run("test with downloading cloudcore service file if version < 1.1.0", func(t *testing.T) {
		err := downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("download should not return an error:{%s}\n", err.Error())
		}
		if err = check(serviceFilePath); !os.IsNotExist(err) {
			if err == nil {
				err = errors.New("nil")
			}
			t.Fatalf("check should return error{%s} but not,error:{%s}\n", os.ErrNotExist, err.Error())
		}
		clean(testTmpDir)
	})

	componentType = types.CloudCore
	targetVersion = semver.Version{Major: 1, Minor: 1}
	serviceFilePath = testTmpDir + "/" + CloudServiceFile
	t.Run("test with downloading cloudcore service file if version = 1.1.0", func(t *testing.T) {
		err := downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("download should not return an error:{%s}\n", err.Error())
		}
		if err = check(serviceFilePath); !os.IsNotExist(err) {
			if err == nil {
				err = errors.New("nil")
			}
			t.Fatalf("check should return error{%s} but not,error:{%s}\n", os.ErrNotExist, err.Error())
		}
		clean(testTmpDir)
	})

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
	targetVersion = semver.Version{Major: 1, Minor: 0}
	serviceFilePath = testTmpDir + "/" + EdgeServiceFile
	t.Run("test with downloading edgecore service file if version < 1.1.0", func(t *testing.T) {
		err := downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("download should not return an error:{%s}\n", err.Error())
		}
		if err = check(serviceFilePath); !os.IsNotExist(err) {
			if err == nil {
				err = errors.New("nil")
			}
			t.Fatalf("check should return error{%s} but not,error:{%s}\n", os.ErrNotExist, err.Error())
		}
		clean(testTmpDir)
	})

	componentType = types.EdgeCore
	targetVersion = semver.Version{Major: 1, Minor: 1}
	serviceFilePath = testTmpDir + "/" + OldEdgeServiceFile
	t.Run("test with downloading edgecore service file if version = 1.1.0", func(t *testing.T) {
		err := downloadServiceFile(componentType, targetVersion, testTmpDir)
		if err != nil {
			t.Fatalf("download should not return an error:{%s}\n", err.Error())
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
