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

package fake

import (
	"github.com/blang/semver"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

type MockOSTypeInstaller struct {
	IsProcessRunning             bool
	IsK8SComponentInstalledError error
	ProcessRunningErr            error
	InstallErr                   error
	KillErr                      error
}

func (m *MockOSTypeInstaller) InstallMQTT() error {
	return nil
}

func (m *MockOSTypeInstaller) IsK8SComponentInstalled(kubeConfig, master string) error {
	return m.IsK8SComponentInstalledError
}

func (m *MockOSTypeInstaller) SetKubeEdgeVersion(version semver.Version) {
}

func (m *MockOSTypeInstaller) InstallKubeEdge(options common.InstallOptions) error {
	return m.InstallErr
}

func (m *MockOSTypeInstaller) RunEdgeCore() error {
	return nil
}

func (m *MockOSTypeInstaller) KillKubeEdgeBinary(proc string) error {
	return m.KillErr
}

func (m *MockOSTypeInstaller) IsKubeEdgeProcessRunning(proc string) (bool, error) {
	return m.IsProcessRunning, m.ProcessRunningErr
}
