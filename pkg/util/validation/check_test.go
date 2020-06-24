/*
Copyright 2019 The KubeEdge Authors.

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
