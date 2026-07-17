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

package files

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileCopy(t *testing.T) {
	t.Run("source file not exists", func(t *testing.T) {
		err := FileCopy("a.txt", "b.txt")
		assert.ErrorContains(t, err, "failed to get source file a.txt sta")
	})

	t.Run("source file is a directory", func(t *testing.T) {
		const src = "a"
		var err error
		err = os.Mkdir(src, os.ModePerm)
		assert.NoError(t, err)

		defer func() {
			err := os.Remove(src)
			assert.NoError(t, err)
		}()

		err = FileCopy(src, "b.txt")
		assert.ErrorContains(t, err, "source file a is not a regular file")
	})

	t.Run("source file cannot be opened", func(t *testing.T) {
		src, err := os.CreateTemp("", "src_noperm_*.txt")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.Remove(src.Name()))
		}()
		assert.NoError(t, src.Close())

		// Remove read permission to force os.Open error
		err = os.Chmod(src.Name(), 0222)
		assert.NoError(t, err)

		err = FileCopy(src.Name(), "dst.txt")
		assert.ErrorContains(t, err, "failed to open source file")
	})

	t.Run("copy file successfully", func(t *testing.T) {
		const src, dest = "a.txt", "b.txt"
		var err error
		_, err = os.Create(src)
		assert.NoError(t, err)

		defer func() {
			err = os.Remove(src)
			assert.NoError(t, err)
			err = os.Remove(dest)
			assert.NoError(t, err)
		}()

		err = FileCopy(src, "b.txt")
		assert.NoError(t, err)
	})

	t.Run("destination cannot be created", func(t *testing.T) {
		src, err := os.CreateTemp("", "src_*.txt")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, os.Remove(src.Name()))
		}()
		assert.NoError(t, src.Close())

		// Use a guaranteed-nonexistent directory under TempDir to force OpenFile error
		tempDir := t.TempDir()
		dst := filepath.Join(tempDir, "does-not-exist", "b.txt")
		err = FileCopy(src.Name(), dst)
		assert.ErrorContains(t, err, "failed to open or create destination file")
	})
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

func TestGetSubDirs(t *testing.T) {
	testDir := "testdata"
	err := os.Mkdir(testDir, os.ModePerm)
	assert.NoError(t, err)

	defer func() {
		err = os.RemoveAll(testDir)
		assert.NoError(t, err)
	}()

	subdirs := []string{"dir1", "dir2", "dir3"}
	for i, subdir := range subdirs {
		path := filepath.Join(testDir, subdir)
		err = os.Mkdir(path, os.ModePerm)
		assert.NoError(t, err)
		modTime := time.Now().Add(-time.Duration(10-i) * time.Minute)
		err = os.Chtimes(path, modTime, modTime)
		assert.NoError(t, err)
	}

	t.Run("get subdirs no sort", func(t *testing.T) {
		dirs, err := GetSubDirs(testDir, false)
		assert.NoError(t, err)
		assert.Equal(t, subdirs, dirs)
	})

	t.Run("get subdirs sort by mod time", func(t *testing.T) {
		dirs, err := GetSubDirs(testDir, true)
		assert.NoError(t, err)
		assert.Equal(t, []string{"dir3", "dir2", "dir1"}, dirs)
	})

	t.Run("invalid directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonexistentDir := filepath.Join(tmpDir, "nonexistent")
		dirs, err := GetSubDirs(nonexistentDir, false)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to read directory")
		assert.Nil(t, dirs)
	})

	t.Run("sort stat error on i", func(t *testing.T) {
		tmpDir := t.TempDir()
		for _, name := range []string{"aaa", "bbb"} {
			err := os.Mkdir(filepath.Join(tmpDir, name), os.ModePerm)
			assert.NoError(t, err)
		}

		callCount := 0
		origStat := osStat
		osStat = func(path string) (os.FileInfo, error) {
			callCount++
			if callCount == 1 {
				return nil, os.ErrNotExist // fail on i
			}
			return origStat(path)
		}
		defer func() { osStat = origStat }()

		dirs, err := GetSubDirs(tmpDir, true)
		assert.NoError(t, err)
		assert.NotNil(t, dirs)
	})

	t.Run("sort stat error on j", func(t *testing.T) {
		tmpDir := t.TempDir()
		for _, name := range []string{"aaa", "bbb"} {
			err := os.Mkdir(filepath.Join(tmpDir, name), os.ModePerm)
			assert.NoError(t, err)
		}

		callCount := 0
		origStat := osStat
		osStat = func(path string) (os.FileInfo, error) {
			callCount++
			if callCount == 2 {
				return nil, os.ErrNotExist // fail on j
			}
			return origStat(path)
		}
		defer func() { osStat = origStat }()

		dirs, err := GetSubDirs(tmpDir, true)
		assert.NoError(t, err)
		assert.NotNil(t, dirs)
	})
}
