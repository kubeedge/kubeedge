/*
Copyright 2022 The KubeEdge Authors.

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

package cmd

import (
	"embed"
	"fmt"
)

// FS embeds the shell scripts
//go:embed *.sh
var FS embed.FS

// BuiltinFile returns a FS for the provided file.
func BuiltinFile(file string) ([]byte, error) {
	if file == "" {
		return nil, fmt.Errorf("not valid file")
	}
	return FS.ReadFile(file)
}
