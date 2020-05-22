package validation

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestFileIsExist(t *testing.T) {
	dir, err := ioutil.TempDir("", "TestTempFile_BadDir")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer os.RemoveAll(dir)

	ef, err := ioutil.TempFile(dir, "CheckFileIsExist")
	if err == nil {
		if !FileIsExist(ef.Name()) {
			t.Fatalf("file %v should exist", ef.Name())
		}
	}

	nonexistentDir := filepath.Join(dir, "_not_exists_")
	nf, err := ioutil.TempFile(nonexistentDir, "foo")
	if err == nil {
		if FileIsExist(nf.Name()) {
			t.Fatalf("file %v should not exist", nf.Name())
		}
	}
}
