package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileIsExist(t *testing.T) {
	dir := t.TempDir()

	ef, err := os.CreateTemp(dir, "CheckFileIsExist")
	if err == nil {
		if !FileIsExist(ef.Name()) {
			t.Fatalf("file %v should exist", ef.Name())
		}
	}

	nonexistentDir := filepath.Join(dir, "not_exist_dir")
	notExistFile := filepath.Join(nonexistentDir, "not_exist_file")

	if FileIsExist(notExistFile) {
		t.Fatalf("file %v should not exist", notExistFile)
	}
}
