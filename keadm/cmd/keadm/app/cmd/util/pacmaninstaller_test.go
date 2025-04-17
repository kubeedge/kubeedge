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

func TestSetKubeEdgeVersion(t *testing.T) {
	version := semver.MustParse("1.6.0")
	pacmanOS := PacmanOS{}

	pacmanOS.SetKubeEdgeVersion(version)

	assert.Equal(t, version, pacmanOS.KubeEdgeVersion)
}

func TestIsK8SComponentInstalled(t *testing.T) {
	pacmanOS := PacmanOS{}
	kubeConfig := "/path/to/kubeconfig"
	master := "master-node"

	p1 := gomonkey.ApplyFunc(isK8SComponentInstalled, func(kubeConfig, master string) error {
		assert.Equal(t, "/path/to/kubeconfig", kubeConfig)
		assert.Equal(t, "master-node", master)
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.IsK8SComponentInstalled(kubeConfig, master)
	assert.NoError(t, err)
}

func TestInstallKubeEdge(t *testing.T) {
	version := semver.MustParse("1.6.0")
	pacmanOS := PacmanOS{
		KubeEdgeVersion: version,
	}

	options := types.InstallOptions{}

	p1 := gomonkey.ApplyFunc(installKubeEdge, func(options types.InstallOptions, version semver.Version) error {
		assert.Equal(t, semver.MustParse("1.6.0"), version)
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.InstallKubeEdge(options)
	assert.NoError(t, err)
}

func TestRunEdgeCore(t *testing.T) {
	pacmanOS := PacmanOS{}

	p1 := gomonkey.ApplyFunc(runEdgeCore, func() error {
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.RunEdgeCore()
	assert.NoError(t, err)
}

func TestKillKubeEdgeBinary(t *testing.T) {
	pacmanOS := PacmanOS{}
	proc := "edgecore"

	p1 := gomonkey.ApplyFunc(KillKubeEdgeBinary, func(proc string) error {
		assert.Equal(t, "edgecore", proc)
		return nil
	})
	defer p1.Reset()

	err := pacmanOS.KillKubeEdgeBinary(proc)
	assert.NoError(t, err)
}

func TestIsKubeEdgeProcessRunning(t *testing.T) {
	pacmanOS := PacmanOS{}
	proc := "edgecore"

	p1 := gomonkey.ApplyFunc(IsKubeEdgeProcessRunning, func(proc string) (bool, error) {
		assert.Equal(t, "edgecore", proc)
		return true, nil
	})
	defer p1.Reset()

	running, err := pacmanOS.IsKubeEdgeProcessRunning(proc)
	assert.NoError(t, err)
	assert.True(t, running)
}
