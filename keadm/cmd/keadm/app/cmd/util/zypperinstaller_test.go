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
	"github.com/kubeedge/kubeedge/pkg/util/execs"
)

const zypperTestVersion = "1.6.0"

func TestSetKubeEdgeVersion_Zypper(t *testing.T) {
	version := semver.MustParse(zypperTestVersion)
	zypperOS := ZypperOS{}

	zypperOS.SetKubeEdgeVersion(version)

	assert.Equal(t, version, zypperOS.KubeEdgeVersion)
}

func TestZypperOS_InstallMQTT(t *testing.T) {
	tests := []struct {
		name          string
		execResults   []error
		stdOutResults []string
		expectedError bool
	}{
		{
			name:          "MQTT install success",
			execResults:   []error{nil, nil},
			stdOutResults: []string{"", "Installing mosquitto..."},
			expectedError: false,
		},
		{
			name:          "MQTT check failed",
			execResults:   []error{errors.New("command failed")},
			stdOutResults: []string{""},
			expectedError: true,
		},
		{
			name:          "MQTT install failed",
			execResults:   []error{nil, errors.New("installation failed")},
			stdOutResults: []string{"", ""},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zypperOS := ZypperOS{}

			cmdMock := &execs.Command{}

			execCount := 0
			execPatch := gomonkey.ApplyMethod(reflect.TypeOf(cmdMock), "Exec",
				func(_ *execs.Command) error {
					result := tt.execResults[execCount]
					execCount++
					return result
				})
			defer execPatch.Reset()

			stdOutCount := 0
			stdOutPatch := gomonkey.ApplyMethod(reflect.TypeOf(cmdMock), "GetStdOut",
				func(_ *execs.Command) string {
					result := tt.stdOutResults[stdOutCount]
					stdOutCount++
					return result
				})
			defer stdOutPatch.Reset()

			newCmdPatch := gomonkey.ApplyFunc(execs.NewCommand,
				func(command string) *execs.Command {
					return cmdMock
				})
			defer newCmdPatch.Reset()

			err := zypperOS.InstallMQTT()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsK8SComponentInstalled_Zypper(t *testing.T) {
	const (
		testKubeConfigPath = "/path/to/kubeconfig"
		testMasterNodeName = "master-node"
	)
	zypperOS := ZypperOS{}

	p1 := gomonkey.ApplyFunc(isK8SComponentInstalled, func(kubeConfig, master string) error {
		assert.Equal(t, testKubeConfigPath, kubeConfig)
		assert.Equal(t, testMasterNodeName, master)
		return nil
	})
	defer p1.Reset()

	err := zypperOS.IsK8SComponentInstalled(testKubeConfigPath, testMasterNodeName)
	assert.NoError(t, err)
}

func TestInstallKubeEdge_Zypper(t *testing.T) {
	version := semver.MustParse(zypperTestVersion)
	zypperOS := ZypperOS{
		KubeEdgeVersion: version,
	}

	options := types.InstallOptions{}

	p1 := gomonkey.ApplyFunc(installKubeEdge, func(options types.InstallOptions, version semver.Version) error {
		assert.Equal(t, semver.MustParse(zypperTestVersion), version)
		return nil
	})
	defer p1.Reset()

	err := zypperOS.InstallKubeEdge(options)
	assert.NoError(t, err)
}

func TestRunEdgeCore_Zypper(t *testing.T) {
	zypperOS := ZypperOS{}

	p1 := gomonkey.ApplyFunc(runEdgeCore, func() error {
		return nil
	})
	defer p1.Reset()

	err := zypperOS.RunEdgeCore()
	assert.NoError(t, err)
}

func TestKillKubeEdgeBinary_Zypper(t *testing.T) {
	zypperOS := ZypperOS{}

	p1 := gomonkey.ApplyFunc(KillKubeEdgeBinary, func(proc string) error {
		assert.Equal(t, testEdgeCoreProcessName, proc)
		return nil
	})
	defer p1.Reset()

	err := zypperOS.KillKubeEdgeBinary(testEdgeCoreProcessName)
	assert.NoError(t, err)
}

func TestIsKubeEdgeProcessRunning_Zypper(t *testing.T) {
	zypperOS := ZypperOS{}

	p1 := gomonkey.ApplyFunc(IsKubeEdgeProcessRunning, func(proc string) (bool, error) {
		assert.Equal(t, testEdgeCoreProcessName, proc)
		return true, nil
	})
	defer p1.Reset()

	running, err := zypperOS.IsKubeEdgeProcessRunning(testEdgeCoreProcessName)
	assert.NoError(t, err)
	assert.True(t, running)
}
