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

package edge

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/common/constants"
	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func TestBackupCommandRun(t *testing.T) {
	var releaseCalled bool

	commonpatches := gomonkey.NewPatches()
	defer commonpatches.Reset()

	commonpatches.ApplyMethodFunc(reflect.TypeOf(&upgrdeedge.JSONFileReporter{}), "Report",
		func(_err error) error {
			return nil
		})
	commonpatches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "release",
		func() {
			releaseCalled = true
		})

	t.Run("backup successful", func(t *testing.T) {
		releaseCalled = false
		var backupCalled int

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&backupExecutor{}), "prerun",
			func(_opts BaseOptions) error {
				backupCalled++
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&backupExecutor{}), "backup",
			func(_opts BaseOptions) error {
				backupCalled++
				return nil
			})

		cmd := NewBackupCommand()
		err := cmd.RunE(nil, nil)
		assert.NoError(t, err)
		assert.True(t, releaseCalled)
		assert.Equal(t, 2, backupCalled)
	})

	t.Run("occupied error no need to release", func(t *testing.T) {
		releaseCalled = false
		var backupCalled int

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&backupExecutor{}), "prerun",
			func(_opts BaseOptions) error {
				backupCalled++
				return OccupiedError
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&backupExecutor{}), "backup",
			func(_opts BaseOptions) error {
				backupCalled++
				return nil
			})

		cmd := NewBackupCommand()
		err := cmd.RunE(nil, nil)
		assert.ErrorIs(t, err, OccupiedError)
		assert.False(t, releaseCalled)
		assert.Equal(t, 1, backupCalled)
	})
}

func TestBackupExecutorPrerun(t *testing.T) {
	executor := newBackupExecutor()
	opts := BaseOptions{
		Config: constants.EdgecoreConfigPath,
	}

	t.Run("cannot get current version", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "prePreRun",
			func(_configpath string) error {
				executor.currentVersion = unknownEdgeCoreVersion
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "postPreRun",
			func(_prerunHook string) error {
				return nil
			})

		err := executor.prerun(opts)
		assert.ErrorContains(t, err, "cannot get the required current version")
	})

	t.Run("get current version successfully", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "prePreRun",
			func(_configpath string) error {
				executor.currentVersion = fakeCurrentVersion
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "postPreRun",
			func(_prerunHook string) error {
				return nil
			})

		err := executor.prerun(opts)
		assert.NoError(t, err)
	})
}

func TestBackupExecutorBackup(t *testing.T) {
	var copied bool
	executor := &backupExecutor{
		baseUpgradeExecutor: baseUpgradeExecutor{
			cfg: &cfgv1alpha2.EdgeCoreConfig{
				DataBase: &cfgv1alpha2.DataBase{
					DataSource: cfgv1alpha2.DataBaseDataSource,
				},
			},
			currentVersion: fakeCurrentVersion,
		},
	}
	opts := BaseOptions{
		Config: constants.EdgecoreConfigPath,
	}

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(os.MkdirAll, func(_path string, _perm os.FileMode) error {
		return nil
	})
	patches.ApplyFunc(files.FileCopy, func(src, dst string) error {
		copied = true
		srcFileName := filepath.Base(src)
		want := filepath.Join(common.KubeEdgeBackupPath, executor.currentVersion, srcFileName)
		assert.Equal(t, want, dst)
		return nil
	})

	err := executor.backup(opts)
	assert.NoError(t, err)
	assert.True(t, copied)
}
