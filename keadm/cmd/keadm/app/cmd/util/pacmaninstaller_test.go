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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const pacmanTestVersion = "1.6.0"
const testEdgeCoreProcessName = "edgecore"

func TestSetKubeEdgeVersion_Pacman(t *testing.T) {
	version := semver.MustParse(pacmanTestVersion)
	pacmanOS := PacmanOS{}

	pacmanOS.SetKubeEdgeVersion(version)

	assert.Equal(t, version, pacmanOS.KubeEdgeVersion)
}

func TestIsK8SComponentInstalled_Pacman(t *testing.T) {
	const (
		testKubeConfigPath = "/path/to/kubeconfig"
		testMasterNodeName = "master-node"
	)
	pacmanOS := PacmanOS{}

	p1 := gomonkey.ApplyFunc(isK8SComponentInstalled, func(kubeConfig, master string) error {
		assert.Equal(t, testKubeConfigPath, kubeConfig)
		assert.Equal(t, testMasterNodeName, master)
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.IsK8SComponentInstalled(testKubeConfigPath, testMasterNodeName)
	assert.NoError(t, err)
}

func TestInstallKubeEdge_Pacman(t *testing.T) {
	version := semver.MustParse(pacmanTestVersion)
	pacmanOS := PacmanOS{
		KubeEdgeVersion: version,
	}

	options := types.InstallOptions{}

	p1 := gomonkey.ApplyFunc(installKubeEdge, func(options types.InstallOptions, version semver.Version) error {
		assert.Equal(t, semver.MustParse(pacmanTestVersion), version)
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.InstallKubeEdge(options)
	assert.NoError(t, err)
}

func TestRunEdgeCore_Pacman(t *testing.T) {
	pacmanOS := PacmanOS{}

	p1 := gomonkey.ApplyFunc(runEdgeCore, func() error {
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.RunEdgeCore()
	assert.NoError(t, err)
}

func TestKillKubeEdgeBinary_Pacman(t *testing.T) {
	pacmanOS := PacmanOS{}

	p1 := gomonkey.ApplyFunc(KillKubeEdgeBinary, func(proc string) error {
		assert.Equal(t, testEdgeCoreProcessName, proc)
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.KillKubeEdgeBinary(testEdgeCoreProcessName)
	assert.NoError(t, err)
}

func TestIsKubeEdgeProcessRunning_Pacman(t *testing.T) {
	pacmanOS := PacmanOS{}

	p1 := gomonkey.ApplyFunc(IsKubeEdgeProcessRunning, func(proc string) (bool, error) {
		assert.Equal(t, testEdgeCoreProcessName, proc)
		return true, nil
	})
	defer p1.Reset()

	running, err := pacmanOS.IsKubeEdgeProcessRunning(testEdgeCoreProcessName)
	assert.NoError(t, err)
	assert.True(t, running)
}
