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
