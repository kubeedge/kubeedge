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
}
