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

const (
	testKubeConfig         = "/path/to/kubeconfig"
	testMasterNode         = "master-node"
	rpmTestKubeEdgeVersion = "1.6.0"
)

func TestRpmOSSetKubeEdgeVersion(t *testing.T) {
	version := semver.MustParse(rpmTestKubeEdgeVersion)
	rpmOS := RpmOS{}
	rpmOS.SetKubeEdgeVersion(version)
	assert.Equal(t, version, rpmOS.KubeEdgeVersion)
}

func TestRpmOSIsK8SComponentInstalled(t *testing.T) {
	rpmOS := RpmOS{}

	p1 := gomonkey.ApplyFunc(isK8SComponentInstalled, func(kubeConfig, master string) error {
		assert.Equal(t, testKubeConfig, kubeConfig)
		assert.Equal(t, testMasterNode, master)
		return nil
	})
	defer p1.Reset()

	err := rpmOS.IsK8SComponentInstalled(testKubeConfig, testMasterNode)
	assert.NoError(t, err)
}

func TestRpmOSInstallKubeEdge(t *testing.T) {
	version := semver.MustParse(rpmTestKubeEdgeVersion)
	rpmOS := RpmOS{
		KubeEdgeVersion: version,
	}
	options := types.InstallOptions{}

	p1 := gomonkey.ApplyFunc(installKubeEdge, func(options types.InstallOptions, version semver.Version) error {
		assert.Equal(t, semver.MustParse(rpmTestKubeEdgeVersion), version)
		return nil
	})
	defer p1.Reset()

	err := rpmOS.InstallKubeEdge(options)
	assert.NoError(t, err)
}

func TestRunEdgeCore(t *testing.T) {
	rpmOS := RpmOS{}

	p1 := gomonkey.ApplyFunc(runEdgeCore, func() error {
		return nil
	})
	defer p1.Reset()

	err := rpmOS.RunEdgeCore()
	assert.NoError(t, err)
}

func TestRpmKillKubeEdgeBinary(t *testing.T) {
	rpmOS := RpmOS{}
	proc := "edgecore"

	p1 := gomonkey.ApplyFunc(KillKubeEdgeBinary, func(proc string) error {
		assert.Equal(t, "edgecore", proc)
		return nil
	})
	defer p1.Reset()

	err := rpmOS.KillKubeEdgeBinary(proc)
	assert.NoError(t, err)
}

func TestIsKubeEdgeProcessRunning(t *testing.T) {
	rpmOS := RpmOS{}
	proc := "edgecore"

	p1 := gomonkey.ApplyFunc(IsKubeEdgeProcessRunning, func(proc string) (bool, error) {
		assert.Equal(t, "edgecore", proc)
		return true, nil
	})
	defer p1.Reset()

	running, err := rpmOS.IsKubeEdgeProcessRunning(proc)
	assert.NoError(t, err)
	assert.True(t, running)
}
