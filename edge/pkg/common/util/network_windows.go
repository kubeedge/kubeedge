//go:build windows
// +build windows

/*
Copyright 2025 The KubeEdge Authors.

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

package util

import "errors"

const npipeProtocol = "npipe"

// LocalEndpoint returns the full path to a named pipe at the given endpoint - unlike on unix, we can't use sockets.
func LocalEndpoint(_path, file string) (string, error) {
	if file == "" {
		return "", errors.New("file must not be empty")
	}
	// windows pipes are expected to use forward slashes: https://learn.microsoft.com/windows/win32/ipc/pipe-names
	// so using `url` like we do on unix gives us unclear benefits - see https://github.com/kubernetes/kubernetes/issues/78628
	// So we just construct the path from scratch.
	// Format: \\ServerName\pipe\PipeName
	// Where ServerName is either the name of a remote computer or a period, to specify the local computer.
	// We only consider PipeName as regular windows path, while the pipe path components are fixed, hence we use constants.
	return npipeProtocol + `://\\.\pipe\edgecore-` + file, nil
}
