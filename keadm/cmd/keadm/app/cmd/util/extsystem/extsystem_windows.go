//go:build windows

/*
Copyright 2024 The KubeEdge Authors.

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

package extsystem

import "errors"

// GetExtSystem returns an ExtSystem for the current system, or nil
// if we cannot detect a supported init system.
// This indicates we will skip init system checks, not an error.
func GetExtSystem() (ExtSystem, error) {
	// TODO: Implement this method when we need.
	// Refer to: k8s.io/kubernetes/cmd/kubeadm/app/util/initsystem/initsystem_windows.go
	return nil, errors.New("no supported init system detected")
}
