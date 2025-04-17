//go:build windows
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
	"github.com/stretchr/testify/assert"
)

const testServiceName = "testService"

func setupCommandMock(t *testing.T, execError error, expectedCmd string) (*Command, []*gomonkey.Patches) {
	cmd := &Command{}
	patches := make([]*gomonkey.Patches, 0, 2)

	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return execError
		})
	patches = append(patches, p1)

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			if expectedCmd != "" {
				assert.Equal(t, expectedCmd, command)
			}
			return cmd
		})
	patches = append(patches, p2)

	return cmd, patches
}

func resetPatches(patches []*gomonkey.Patches) {
	for _, p := range patches {
		p.Reset()
	}
}

func TestIsNSSMInstalled(t *testing.T) {
	_, patches := setupCommandMock(t, nil, "nssm version")
	defer resetPatches(patches)

	result := IsNSSMInstalled()
	assert.True(t, result)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("command failed"), "nssm version")
	defer resetPatches(patches)

	result = IsNSSMInstalled()
	assert.False(t, result)
}

func TestInstallNSSM(t *testing.T) {
	_, patches := setupCommandMock(t, nil, installNSSMScript)
	defer resetPatches(patches)

	err := InstallNSSM()
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("installation failed"), installNSSMScript)
	defer resetPatches(patches)

	err = InstallNSSM()
	assert.Error(t, err)
}

func TestInstallNSSMService(t *testing.T) {
	path := "C:\\path\\to\\service.exe"
	args := []string{"arg1", "arg2"}
	expectedCmd := "nssm install " + testServiceName + " " + path + " " + args[0] + " " + args[1]

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	err := InstallNSSMService(testServiceName, path, args...)
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("service installation failed"), expectedCmd)
	defer resetPatches(patches)

	err = InstallNSSMService(testServiceName, path, args...)
	assert.Error(t, err)
}

func TestIsNSSMServiceExist(t *testing.T) {
	expectedCmd := "nssm status " + testServiceName

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	result := IsNSSMServiceExist(testServiceName)
	assert.True(t, result)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("service not found"), expectedCmd)
	defer resetPatches(patches)

	result = IsNSSMServiceExist(testServiceName)
	assert.False(t, result)
}

func TestStartNSSMService(t *testing.T) {
	expectedCmd := "nssm start " + testServiceName

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	err := StartNSSMService(testServiceName)
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("start failed"), expectedCmd)
	defer resetPatches(patches)

	err = StartNSSMService(testServiceName)
	assert.Error(t, err)
}

func TestStopNSSMService(t *testing.T) {
	expectedCmd := "nssm stop " + testServiceName

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	err := StopNSSMService(testServiceName)
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("stop failed"), expectedCmd)
	defer resetPatches(patches)

	err = StopNSSMService(testServiceName)
	assert.Error(t, err)
}

func TestSetNSSMServiceStdout(t *testing.T) {
	logFile := "C:\\logs\\stdout.log"
	expectedCmd := "nssm set " + testServiceName + " AppStdout " + logFile

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	err := SetNSSMServiceStdout(testServiceName, logFile)
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("set stdout failed"), expectedCmd)
	defer resetPatches(patches)

	err = SetNSSMServiceStdout(testServiceName, logFile)
	assert.Error(t, err)
}

func TestSetNSSMServiceStderr(t *testing.T) {
	logFile := "C:\\logs\\stderr.log"
	expectedCmd := "nssm set " + testServiceName + " AppStderr " + logFile

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	err := SetNSSMServiceStderr(testServiceName, logFile)
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("set stderr failed"), expectedCmd)
	defer resetPatches(patches)

	err = SetNSSMServiceStderr(testServiceName, logFile)
	assert.Error(t, err)
}

func TestUninstallNSSMService(t *testing.T) {
	expectedCmd := "nssm remove " + testServiceName + " confirm"

	_, patches := setupCommandMock(t, nil, expectedCmd)
	defer resetPatches(patches)

	err := UninstallNSSMService(testServiceName)
	assert.NoError(t, err)

	resetPatches(patches)
	_, patches = setupCommandMock(t, errors.New("uninstall failed"), expectedCmd)
	defer resetPatches(patches)

	err = UninstallNSSMService(testServiceName)
	assert.Error(t, err)
}
