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
	"context"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	cfgv1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
	edgecoreutil "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/edgecore"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/idempotency"
)

const (
	fakeCurrentVersion = "v1.0.0"
)

func TestPrePreRun(t *testing.T) {
	t.Run("returns error when occupied", func(t *testing.T) {
		executor := baseUpgradeExecutor{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(idempotency.Occupy, func() (bool, error) {
			return true, nil
		})

		err := executor.prePreRun("")
		assert.ErrorIs(t, err, OccupiedError)
		assert.Equal(t, unknownEdgeCoreVersion, executor.currentVersion)
	})

	t.Run("get version failed", func(t *testing.T) {
		executor := baseUpgradeExecutor{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(idempotency.Occupy, func() (bool, error) {
			return false, nil
		})
		patches.ApplyFunc(util.ParseEdgecoreConfig, func(_configPath string) (*cfgv1alpha2.EdgeCoreConfig, error) {
			return &cfgv1alpha2.EdgeCoreConfig{}, nil
		})
		patches.ApplyFunc(edgecoreutil.GetVersion, func(_ctx context.Context, _config *cfgv1alpha2.EdgeCoreConfig) string {
			return ""
		})

		err := executor.prePreRun("")
		assert.NoError(t, err)
		assert.Equal(t, unknownEdgeCoreVersion, executor.currentVersion)
	})

	t.Run("get version successfully", func(t *testing.T) {
		executor := baseUpgradeExecutor{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(idempotency.Occupy, func() (bool, error) {
			return false, nil
		})
		patches.ApplyFunc(util.ParseEdgecoreConfig, func(_configPath string) (*cfgv1alpha2.EdgeCoreConfig, error) {
			return &cfgv1alpha2.EdgeCoreConfig{}, nil
		})
		patches.ApplyFunc(edgecoreutil.GetVersion, func(_ctx context.Context, _config *cfgv1alpha2.EdgeCoreConfig) string {
			return fakeCurrentVersion
		})

		err := executor.prePreRun("")
		assert.NoError(t, err)
		assert.Equal(t, fakeCurrentVersion, executor.currentVersion)
	})
}

func TestPostPrerun(t *testing.T) {
	cases := []struct {
		name                 string
		hook                 string
		wantScriptHasBeenRun bool
	}{
		{
			name:                 "do not run prerun hook when flag is empty",
			hook:                 "",
			wantScriptHasBeenRun: false,
		},
		{
			name:                 "run prerun hook when flag is not empty",
			hook:                 "test.sh",
			wantScriptHasBeenRun: true,
		},
	}

	var scriptHasBeenRun bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.RunScript, func(scriptPath string) error {
		scriptHasBeenRun = true
		return nil
	})

	executor := baseUpgradeExecutor{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := executor.postPreRun(c.hook)
			assert.NoError(t, err)
			assert.Equal(t, c.wantScriptHasBeenRun, scriptHasBeenRun)
		})
	}
}

func TestRelease(t *testing.T) {
	t.Run("do not release when not occupied", func(t *testing.T) {
		var released bool
		executor := baseUpgradeExecutor{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(idempotency.IsOccupied, func() bool {
			return false
		})
		gomonkey.ApplyFunc(idempotency.Release, func() error {
			released = true
			return nil
		})

		executor.release()
		assert.False(t, released)
	})

	t.Run("do release when occupied", func(t *testing.T) {
		var released bool
		executor := baseUpgradeExecutor{}

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(idempotency.IsOccupied, func() bool {
			return true
		})
		gomonkey.ApplyFunc(idempotency.Release, func() error {
			released = true
			return nil
		})

		executor.release()
		assert.True(t, released)
	})
}

func TestRunPostrunHook(t *testing.T) {
	cases := []struct {
		name                 string
		hook                 string
		wantScriptHasBeenRun bool
	}{
		{
			name:                 "do not run prerun hook when flag is empty",
			hook:                 "",
			wantScriptHasBeenRun: false,
		},
		{
			name:                 "run prerun hook when flag is not empty",
			hook:                 "test.sh",
			wantScriptHasBeenRun: true,
		},
	}

	var scriptHasBeenRun bool

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(util.RunScript, func(scriptPath string) error {
		scriptHasBeenRun = true
		return nil
	})

	executor := baseUpgradeExecutor{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			executor.runPostRunHook(c.hook)
			assert.Equal(t, c.wantScriptHasBeenRun, scriptHasBeenRun)
		})
	}
}
