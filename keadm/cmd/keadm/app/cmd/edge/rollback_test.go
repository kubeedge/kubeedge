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
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/common/constants"
	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	upgrdeedge "github.com/kubeedge/kubeedge/pkg/upgrade/edge"
	"github.com/kubeedge/kubeedge/pkg/util/files"
)

func TestRollbackRun(t *testing.T) {
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

	t.Run("rollback successful", func(t *testing.T) {
		releaseCalled = false
		var rollbackCalled int

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&rollbackExecutor{}), "prerun",
			func(_opts RollbackOptions) error {
				rollbackCalled++
				return nil
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&rollbackExecutor{}), "rollback",
			func(_opts RollbackOptions) error {
				rollbackCalled++
				return nil
			})

		cmd := NewRollbackCommand()
		err := cmd.RunE(nil, nil)
		assert.NoError(t, err)
		assert.True(t, releaseCalled)
		assert.Equal(t, 2, rollbackCalled)
	})

	t.Run("occupied error no need to release", func(t *testing.T) {
		releaseCalled = false
		var rollbackCalled int

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyPrivateMethod(reflect.TypeOf(&rollbackExecutor{}), "prerun",
			func(_opts RollbackOptions) error {
				rollbackCalled++
				return OccupiedError
			})
		patches.ApplyPrivateMethod(reflect.TypeOf(&rollbackExecutor{}), "rollback",
			func(_opts RollbackOptions) error {
				rollbackCalled++
				return nil
			})

		cmd := NewRollbackCommand()
		err := cmd.RunE(nil, nil)
		assert.ErrorIs(t, err, OccupiedError)
		assert.False(t, releaseCalled)
		assert.Equal(t, 1, rollbackCalled)
	})
}

func TestRollbackExecutorPreRun(t *testing.T) {
	commonpatches := gomonkey.NewPatches()
	defer commonpatches.Reset()

	commonpatches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "prePreRun",
		func(_configpath string) error {
			return nil
		})
	commonpatches.ApplyPrivateMethod(reflect.TypeOf(&baseUpgradeExecutor{}), "postPreRun",
		func(_prerunHook string) error {
			return nil
		})

	t.Run("historialVersion not empty, no backup dir", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		patches.Reset()

		patches.ApplyFunc(files.GetSubDirs, func(_dir string, _sorted bool) ([]string, error) {
			return []string{}, nil
		})

		executor := newRollbackExecutor()
		opts := RollbackOptions{
			HistoricalVersion: "v1.0.0",
		}
		err := executor.prerun(&opts)
		assert.ErrorContains(t, err, "the historical version v1.0.0 is not exist in backup dir")
	})

	t.Run("historialVersion not empty, exists backup dir", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		patches.Reset()

		patches.ApplyFunc(files.GetSubDirs, func(_dir string, _sorted bool) ([]string, error) {
			return []string{"v1.0.0"}, nil
		})

		executor := newRollbackExecutor()
		opts := RollbackOptions{
			HistoricalVersion: "v1.0.0",
		}
		err := executor.prerun(&opts)
		assert.NoError(t, err)
		assert.Equal(t, "v1.0.0", opts.HistoricalVersion)
	})

	t.Run("historialVersion is empty, backed up versions is empty", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		patches.Reset()

		patches.ApplyFunc(files.GetSubDirs, func(_dir string, _sorted bool) ([]string, error) {
			return []string{}, nil
		})

		executor := newRollbackExecutor()
		var opts RollbackOptions
		err := executor.prerun(&opts)
		assert.ErrorContains(t, err, "no historical version is found in backup dir")
	})

	t.Run("historialVersion is empty, set it to the latast version", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		patches.Reset()

		patches.ApplyFunc(files.GetSubDirs, func(_dir string, _sorted bool) ([]string, error) {
			return []string{"v1.1.0", "v1.0.0"}, nil
		})

		executor := newRollbackExecutor()
		var opts RollbackOptions
		err := executor.prerun(&opts)
		assert.NoError(t, err)
		assert.Equal(t, "v1.1.0", opts.HistoricalVersion)
	})
}

func TestRollbackExecutorRollback(t *testing.T) {
	var fileChecked, edgecoreStopped bool
	executor := newRollbackExecutor()
	executor.cfg = &cfgv1alpha2.EdgeCoreConfig{
		DataBase: &cfgv1alpha2.DataBase{
			DataSource: "/data/kubeedge/edge.db",
		},
	}
	opts := RollbackOptions{
		HistoricalVersion: "v1.0.0",
		BaseOptions: BaseOptions{
			Config: "/data/kubeedge/edge.yaml",
		},
	}
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.KillKubeEdgeBinary, func(_proc string) error {
		edgecoreStopped = true
		return nil
	})
	patches.ApplyFunc(replaceFiles, func(srcToDest map[string]string, stop func() error) error {
		backupPath := filepath.Join(common.KubeEdgeBackupPath, opts.HistoricalVersion)
		assert.Equal(t, map[string]string{
			filepath.Join(backupPath, "edgecore.db"):   executor.cfg.DataBase.DataSource,
			filepath.Join(backupPath, "edgecore.yaml"): opts.Config,
			filepath.Join(backupPath, "edgecore"):      filepath.Join(constants.KubeEdgeUsrBinPath, constants.KubeEdgeBinaryName),
		}, srcToDest)
		fileChecked = true
		return stop()
	})
	patches.ApplyFunc(runEdgeCore, func() error {
		return nil
	})

	err := executor.rollback(opts)
	assert.NoError(t, err)
	assert.True(t, fileChecked)
	assert.True(t, edgecoreStopped)
}
