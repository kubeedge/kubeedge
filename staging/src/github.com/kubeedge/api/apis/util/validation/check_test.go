package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileIsExist(t *testing.T) {
	dir := t.TempDir()

	// case: file exists and is readable
	ef, err := os.CreateTemp(dir, "CheckFileIsExist")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	ef.Close()
	if !FileIsExist(ef.Name()) {
		t.Errorf("existing file should be reported as existing")
	}

	// case: file does not exist
	notExistFile := filepath.Join(dir, "not_exist_dir", "not_exist_file")
	if FileIsExist(notExistFile) {
		t.Errorf("non-existent file should be reported as not existing")
	}

	// case: stat returns an error other than ErrNotExist
	// e.g., filename too long
	longName := filepath.Join(dir, strings.Repeat("a", 1000))
	// FileIsExist conservatively returns true for errors other than ErrNotExist
	if !FileIsExist(longName) {
		t.Errorf("file with very long name should conservatively report as existing")
	}
}
