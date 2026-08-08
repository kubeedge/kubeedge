package certs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWrite(t *testing.T) {
	file := "./testdata/ca/ca.key"
	if _, err := WriteDERToPEMFile(file, "test data", []byte("test")); err != nil {
		t.Fatal(err)
	}
	if block, err := ReadPEMFile(file); err != nil {
		t.Fatal(err)
	} else {
		if block.Type != "test data" {
			t.Fatalf("want block type '%s', actual '%s'", "test data", block.Type)
		}
	}
	// Clean
	if err := os.RemoveAll("testdata"); err != nil {
		t.Fatalf("failed to clean testdata, err: %v", err)
	}
}

func TestWriteDERToPEMFilePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs")
	file := filepath.Join(dir, "edge.key")
	if _, err := WriteDERToPEMFile(file, "test data", []byte("test")); err != nil {
		t.Fatal(err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Fatalf("want dir permissions %o, actual %o", 0700, perm)
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0600 {
		t.Fatalf("want file permissions %o, actual %o", 0600, perm)
	}
}

func TestWriteDERToPEMFileTightensExistingPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs")
	file := filepath.Join(dir, "edge.key")

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteDERToPEMFile(file, "test data", []byte("test")); err != nil {
		t.Fatal(err)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0700 {
		t.Fatalf("want dir permissions %o, actual %o", 0700, perm)
	}

	fileInfo, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0600 {
		t.Fatalf("want file permissions %o, actual %o", 0600, perm)
	}
}

func TestReadPEMFileNoBlock(t *testing.T) {
	file := filepath.Join(t.TempDir(), "invalid.crt")
	if err := os.WriteFile(file, []byte("not a pem block"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPEMFile(file); err == nil {
		t.Fatal("expected error when file contains no PEM block, got nil")
	}
}
