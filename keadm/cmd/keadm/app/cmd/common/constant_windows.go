//go:build windows

/*
Copyright 2023 The KubeEdge Authors.

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

package common

import (
	"os"
	"path/filepath"
)

const (
	// DefaultCertPath is the default certificate path in edge node
	DefaultCertPath = "c:/etc/kubeedge/certs"
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		DefaultKubeConfig = "C:\\Users\\Administrator\\.kube\\config"
		return
	}
	DefaultKubeConfig = filepath.Join(home, ".kube", "config")
}
