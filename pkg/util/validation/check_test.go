package validation

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestFileIsExist(t *testing.T) {
    assert := assert.New(t)

    dir := t.TempDir()

    ef, err := os.CreateTemp(dir, "CheckFileIsExist")
    assert.NoError(err, "Error creating temporary file")
    defer os.Remove(ef.Name())

    assert.True(FileIsExist(ef.Name()), "file %v should exist", ef.Name())

    nonexistentDir := filepath.Join(dir, "not_exist_dir")
    notExistFile := filepath.Join(nonexistentDir, "not_exist_file")

    assert.False(FileIsExist(notExistFile), "file %v should not exist", notExistFile)
}
