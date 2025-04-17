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

func TestIsNSSMInstalled(t *testing.T) {
	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			return cmd
		})
	defer p2.Reset()

	result := IsNSSMInstalled()
	assert.True(t, result)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("command failed")
		})
	defer p3.Reset()

	result = IsNSSMInstalled()
	assert.False(t, result)
}

func TestInstallNSSM(t *testing.T) {
	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			assert.Equal(t, installNSSMScript, command)
			return cmd
		})
	defer p2.Reset()

	err := InstallNSSM()
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("installation failed")
		})
	defer p3.Reset()

	err = InstallNSSM()
	assert.Error(t, err)
}

func TestInstallNSSMService(t *testing.T) {
	path := "C:\\path\\to\\service.exe"
	args := []string{"arg1", "arg2"}

	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm install " + testServiceName + " " + path + " " + args[0] + " " + args[1]
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	err := InstallNSSMService(testServiceName, path, args...)
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("service installation failed")
		})
	defer p3.Reset()

	err = InstallNSSMService(testServiceName, path, args...)
	assert.Error(t, err)
}

func TestIsNSSMServiceExist(t *testing.T) {
	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm status " + testServiceName
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	result := IsNSSMServiceExist(testServiceName)
	assert.True(t, result)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("service not found")
		})
	defer p3.Reset()

	result = IsNSSMServiceExist(testServiceName)
	assert.False(t, result)
}

func TestStartNSSMService(t *testing.T) {
	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm start " + testServiceName
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	err := StartNSSMService(testServiceName)
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("start failed")
		})
	defer p3.Reset()

	err = StartNSSMService(testServiceName)
	assert.Error(t, err)
}

func TestStopNSSMService(t *testing.T) {
	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm stop " + testServiceName
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	err := StopNSSMService(testServiceName)
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("stop failed")
		})
	defer p3.Reset()

	err = StopNSSMService(testServiceName)
	assert.Error(t, err)
}

func TestSetNSSMServiceStdout(t *testing.T) {
	logFile := "C:\\logs\\stdout.log"

	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm set " + testServiceName + " AppStdout " + logFile
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	err := SetNSSMServiceStdout(testServiceName, logFile)
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("set stdout failed")
		})
	defer p3.Reset()

	err = SetNSSMServiceStdout(testServiceName, logFile)
	assert.Error(t, err)
}

func TestSetNSSMServiceStderr(t *testing.T) {
	logFile := "C:\\logs\\stderr.log"

	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm set " + testServiceName + " AppStderr " + logFile
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	err := SetNSSMServiceStderr(testServiceName, logFile)
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("set stderr failed")
		})
	defer p3.Reset()

	err = SetNSSMServiceStderr(testServiceName, logFile)
	assert.Error(t, err)
}

func TestUninstallNSSMService(t *testing.T) {
	cmd := &Command{}
	p1 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return nil
		})
	defer p1.Reset()

	p2 := gomonkey.ApplyFunc(NewCommand,
		func(command string) *Command {
			expectedCmd := "nssm remove " + testServiceName + " confirm"
			assert.Equal(t, expectedCmd, command)
			return cmd
		})
	defer p2.Reset()

	err := UninstallNSSMService(testServiceName)
	assert.NoError(t, err)

	p3 := gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Exec",
		func(_ *Command) error {
			return errors.New("uninstall failed")
		})
	defer p3.Reset()

	err = UninstallNSSMService(testServiceName)
	assert.Error(t, err)
}
