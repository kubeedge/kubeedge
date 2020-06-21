/*
Copyright 2020 The KubeEdge Authors.

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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// license compatible for Go and Proto files.
	license = `/*
Copyright %s The KubeEdge Authors.

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

`

	licenseShell = `
# Copyright %s The KubeEdge Authors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

`

	// since different year in Copyright, just check this exist or not.
	licenseFlag = []byte(`http://www.apache.org/licenses/LICENSE-2.0`)

	usage = []byte(`
Usage:
  verify:
    go run copyright.go verify
  update:
    go run copyright.go
`)
)

func checkAndApplyLicense(update bool) error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Filter out stuff that does not need copyright.
		if info.IsDir() {
			switch path {
			case "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".pb.go") {
			return nil
		}
		if filepath.Ext(path) != ".proto" && filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sh" {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		if !strings.Contains(string(b), string(licenseFlag)) {
			if update {
				log.Println("file", path, "is missing Copyright header. Adding.")
				cmd := exec.Command("bash", "-c", fmt.Sprintf("git log --format=%%aD %s | tail -1 | cut -d ' ' -f4", path))
				stdout, err := cmd.Output()

				if err != nil {
					return err
				}

				l := ""
				if filepath.Ext(path) == ".sh" {
					l = fmt.Sprintf(licenseShell, strings.TrimSuffix(string(stdout), "\n"))
				} else {
					l = fmt.Sprintf(license, strings.TrimSuffix(string(stdout), "\n"))
				}

				var bb bytes.Buffer
				_, _ = bb.Write([]byte(l))
				_, _ = bb.Write(b)
				if err = ioutil.WriteFile(path, bb.Bytes(), 0666); err != nil {
					return err
				}
			} else {
				return errors.New("some file are missing Copyright header. Please run `go run hack/copyright.go` to add")
			}
		}
		return nil
	})
}

func main() {
	update := true
	args := os.Args

	switch l := len(args); {
	case l > 2:
		log.Fatalf("invalid usage, usage is as below:%s", string(usage))
	case l == 2:
		if args[1] != "verify" {
			log.Fatalf("invalid usage, usage is as below:%s", string(usage))
		}
		update = false
	}

	if err := checkAndApplyLicense(update); err != nil {
		log.Fatal(err)
	}
}
