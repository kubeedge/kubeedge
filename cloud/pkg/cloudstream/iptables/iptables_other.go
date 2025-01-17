//go:build !linux

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
package iptables

import (
	"context"

	cloudcoreConfig "github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

// Used to solve the compilation error when developing on non-Linux platforms

type Manager struct{}

type TunnelPortRecord struct {
	IPTunnelPort map[string]int `json:"ipTunnelPort"`
	Port         map[int]bool   `json:"port"`
}

func NewIptablesManager(_ *cloudcoreConfig.KubeAPIConfig, _ int) *Manager {
	return &Manager{}
}

func (im *Manager) Run(_ context.Context) {
	panic("This method on non-Linux platforms is not supported")
}
