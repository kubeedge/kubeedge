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

const testKubeEdgeVersion = "1.0.0"

func TestDebOS_SetKubeEdgeVersion(t *testing.T) {
	version := semver.MustParse(testKubeEdgeVersion)
	deb := DebOS{}
	deb.SetKubeEdgeVersion(version)
	assert.Equal(t, version, deb.KubeEdgeVersion)
}

func TestDebOS_InstallMQTT(t *testing.T) {
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
			deb := DebOS{}

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

			err := deb.InstallMQTT()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebOS_IsK8SComponentInstalled(t *testing.T) {
	tests := []struct {
		name          string
		patchResult   error
		expectedError bool
	}{
		{
			name:          "Component installed",
			patchResult:   nil,
			expectedError: false,
		},
		{
			name:          "Component not installed",
			patchResult:   errors.New("component not installed"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deb := DebOS{}

			patch := gomonkey.ApplyFunc(isK8SComponentInstalled,
				func(kubeConfig, master string) error {
					return tt.patchResult
				})
			defer patch.Reset()

			err := deb.IsK8SComponentInstalled("kubeconfig", "master")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebOS_InstallKubeEdge(t *testing.T) {
	tests := []struct {
		name          string
		patchResult   error
		expectedError bool
	}{
		{
			name:          "Installation successful",
			patchResult:   nil,
			expectedError: false,
		},
		{
			name:          "Installation failed",
			patchResult:   errors.New("installation failed"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deb := DebOS{
				KubeEdgeVersion: semver.MustParse(testKubeEdgeVersion),
			}
			options := types.InstallOptions{
				ComponentType: types.EdgeCore,
				TarballPath:   "/path/to/tarball",
			}

			patch := gomonkey.ApplyFunc(installKubeEdge,
				func(opt types.InstallOptions, version semver.Version) error {
					return tt.patchResult
				})
			defer patch.Reset()

			err := deb.InstallKubeEdge(options)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebOS_RunEdgeCore(t *testing.T) {
	tests := []struct {
		name          string
		patchResult   error
		expectedError bool
	}{
		{
			name:          "Run successful",
			patchResult:   nil,
			expectedError: false,
		},
		{
			name:          "Run failed",
			patchResult:   errors.New("run failed"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deb := DebOS{}

			patch := gomonkey.ApplyFunc(runEdgeCore,
				func() error {
					return tt.patchResult
				})
			defer patch.Reset()

			err := deb.RunEdgeCore()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebOS_KillKubeEdgeBinary(t *testing.T) {
	tests := []struct {
		name          string
		patchResult   error
		expectedError bool
	}{
		{
			name:          "Kill successful",
			patchResult:   nil,
			expectedError: false,
		},
		{
			name:          "Kill failed",
			patchResult:   errors.New("kill failed"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deb := DebOS{}

			patch := gomonkey.ApplyFunc(KillKubeEdgeBinary,
				func(proc string) error {
					return tt.patchResult
				})
			defer patch.Reset()

			err := deb.KillKubeEdgeBinary("edgecore")

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDebOS_IsKubeEdgeProcessRunning(t *testing.T) {
	tests := []struct {
		name            string
		patchRunning    bool
		patchError      error
		expectedRunning bool
		expectedError   bool
	}{
		{
			name:            "Process running",
			patchRunning:    true,
			patchError:      nil,
			expectedRunning: true,
			expectedError:   false,
		},
		{
			name:            "Process not running",
			patchRunning:    false,
			patchError:      nil,
			expectedRunning: false,
			expectedError:   false,
		},
		{
			name:            "Check failed",
			patchRunning:    false,
			patchError:      errors.New("check failed"),
			expectedRunning: false,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deb := DebOS{}

			patch := gomonkey.ApplyFunc(IsKubeEdgeProcessRunning,
				func(proc string) (bool, error) {
					return tt.patchRunning, tt.patchError
				})
			defer patch.Reset()

			running, err := deb.IsKubeEdgeProcessRunning("edgecore")

			assert.Equal(t, tt.expectedRunning, running)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
