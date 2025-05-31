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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/blang/semver"
	"github.com/stretchr/testify/assert"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

const (
	testVersion    = "1.6.0"
	testKubeConfig = "/path/to/kubeconfig"
	testMasterNode = "master-node"
)

func TestSetKubeEdgeVersion(t *testing.T) {
	version := semver.MustParse(testVersion)
	rpmOS := RpmOS{}
	rpmOS.SetKubeEdgeVersion(version)
	assert.Equal(t, version, rpmOS.KubeEdgeVersion)
}

func TestIsK8SComponentInstalled(t *testing.T) {
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

func TestInstallKubeEdge(t *testing.T) {
	version := semver.MustParse(testVersion)
	rpmOS := RpmOS{
		KubeEdgeVersion: version,
	}
	options := types.InstallOptions{}

	p1 := gomonkey.ApplyFunc(installKubeEdge, func(options types.InstallOptions, version semver.Version) error {
		assert.Equal(t, semver.MustParse(testVersion), version)
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

func TestKillKubeEdgeBinary(t *testing.T) {
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

func TestGetOSVendorName_Error(t *testing.T) {
	cmd := &Command{}

	p1 := gomonkey.ApplyFunc(NewCommand, func(command string) *Command {
		return cmd
	})
	p2 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec", func(*Command) error {
		return errors.New("os vendor name error")
	})

	vendor, err := getOSVendorName()
	assert.Error(t, err)
	assert.Equal(t, "os vendor name error", err.Error())
	assert.Empty(t, vendor)

	p1.Reset()
	p2.Reset()
}
