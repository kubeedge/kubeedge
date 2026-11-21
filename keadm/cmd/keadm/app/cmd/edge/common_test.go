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
	"errors"
	"os"
	"path/filepath"
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

func TestReplaceFiles(t *testing.T) {
	var stopped bool
	stop := func() error {
		stopped = true
		return nil
	}
	newFiles := func(t *testing.T) (string, string) {
		stopped = false
		dir := t.TempDir()
		src, dest := filepath.Join(dir, "a-src"), filepath.Join(dir, "a-dest")
		assert.NoError(t, os.WriteFile(src, []byte("new"), 0600))
		assert.NoError(t, os.WriteFile(dest, []byte("old"), 0600))
		return src, dest
	}
	content := func(t *testing.T, path string) string {
		data, err := os.ReadFile(path)
		assert.NoError(t, err)
		return string(data)
	}

	t.Run("replace files, keep the mode of an existing destination", func(t *testing.T) {
		src, dest := newFiles(t)
		assert.NoError(t, os.Chmod(dest, 0666))
		newSrc, newDest := filepath.Join(filepath.Dir(dest), "b-src"), filepath.Join(filepath.Dir(dest), "b-dest")
		assert.NoError(t, os.WriteFile(newSrc, []byte("new"), 0600))

		err := replaceFiles(map[string]string{src: dest, newSrc: newDest}, stop)
		assert.NoError(t, err)
		assert.True(t, stopped)
		for path, mode := range map[string]os.FileMode{dest: 0666, newDest: 0600} {
			info, err := os.Stat(path)
			assert.NoError(t, err)
			assert.Equal(t, mode, info.Mode())
			assert.Equal(t, "new", content(t, path))
			assert.NoFileExists(t, path+stagedFileSuffix)
		}
	})

	t.Run("source does not exist, do not stop", func(t *testing.T) {
		_, dest := newFiles(t)

		err := replaceFiles(map[string]string{dest + "-missing": dest}, stop)
		assert.Error(t, err)
		assert.False(t, stopped)
		assert.Equal(t, "old", content(t, dest))
		assert.NoFileExists(t, dest+stagedFileSuffix)
	})

	t.Run("source is not a regular file, do not stop", func(t *testing.T) {
		_, dest := newFiles(t)

		err := replaceFiles(map[string]string{filepath.Dir(dest): dest}, stop)
		assert.ErrorContains(t, err, "not a regular file")
		assert.False(t, stopped)
	})

	t.Run("destination directory does not exist, do not stop", func(t *testing.T) {
		src, dest := newFiles(t)

		err := replaceFiles(map[string]string{src: filepath.Join(dest+"-missing", "dest")}, stop)
		assert.Error(t, err)
		assert.False(t, stopped)
	})

	t.Run("second file cannot be written, remove the first one and do not stop", func(t *testing.T) {
		src, dest := newFiles(t)

		err := replaceFiles(map[string]string{src: dest, src + "-missing": dest + "-new"}, stop)
		assert.Error(t, err)
		assert.False(t, stopped)
		assert.Equal(t, "old", content(t, dest))
		assert.NoFileExists(t, dest+stagedFileSuffix)
	})

	t.Run("stop failed, do not replace", func(t *testing.T) {
		src, dest := newFiles(t)

		err := replaceFiles(map[string]string{src: dest}, func() error {
			return errors.New("test error")
		})
		assert.ErrorContains(t, err, "test error")
		assert.Equal(t, "old", content(t, dest))
		assert.NoFileExists(t, dest+stagedFileSuffix)
	})

	t.Run("destination is a symlink, replace its target", func(t *testing.T) {
		src, dest := newFiles(t)
		link := filepath.Join(filepath.Dir(dest), "link")
		assert.NoError(t, os.Symlink(dest, link))

		err := replaceFiles(map[string]string{src: link}, stop)
		assert.NoError(t, err)
		info, err := os.Lstat(link)
		assert.NoError(t, err)
		assert.NotZero(t, info.Mode()&os.ModeSymlink)
		assert.Equal(t, "new", content(t, dest))
	})

	t.Run("destination is a dangling symlink, do not stop", func(t *testing.T) {
		src, dest := newFiles(t)
		link := filepath.Join(filepath.Dir(dest), "link")
		assert.NoError(t, os.Symlink(dest+"-missing", link))

		err := replaceFiles(map[string]string{src: link}, stop)
		assert.Error(t, err)
		assert.False(t, stopped)
	})

	t.Run("rename failed, copy in place", func(t *testing.T) {
		src, dest := newFiles(t)

		err := replaceFiles(map[string]string{src: dest}, func() error {
			return os.Remove(dest + stagedFileSuffix)
		})
		assert.NoError(t, err)
		assert.Equal(t, "new", content(t, dest))
		assert.NoFileExists(t, dest+stagedFileSuffix)
	})

	t.Run("rename and copy in place failed, return error", func(t *testing.T) {
		src, dest := newFiles(t)

		err := replaceFiles(map[string]string{src: dest}, func() error {
			assert.NoError(t, os.Remove(dest+stagedFileSuffix))
			return os.Remove(src)
		})
		assert.Error(t, err)
	})
}
