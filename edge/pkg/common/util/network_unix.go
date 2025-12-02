//go:build freebsd || linux || darwin
// +build freebsd linux darwin

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

import (
	"errors"
	"net/url"
	"path/filepath"
)

// unixProtocol is the network protocol of unix socket.
const unixProtocol = "unix"

// LocalEndpoint returns the full path to a unix socket at the given endpoint
func LocalEndpoint(path, file string) (string, error) {
	if file == "" {
		return "", errors.New("file must not be empty")
	}
	u := url.URL{
		Scheme: unixProtocol,
		Path:   filepath.Join(path, file+".sock"),
	}
	return u.String(), nil
}
