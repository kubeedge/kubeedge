package certs

import (
	"os"
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

func TestReadPEMFileNoBlock(t *testing.T) {
	file := "./testdata/ca/invalid.crt"
	if err := os.MkdirAll("./testdata/ca", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("not a pem block"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPEMFile(file); err == nil {
		t.Fatal("expected error when file contains no PEM block, got nil")
	}
	// Clean
	if err := os.RemoveAll("testdata"); err != nil {
		t.Fatalf("failed to clean testdata, err: %v", err)
	}
}
