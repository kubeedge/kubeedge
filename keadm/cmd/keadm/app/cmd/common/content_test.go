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

package common

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/common/constants"
)

const (
	testProcess     = "test-process"
	testCommand     = "/usr/bin/test-cmd"
	testServicePath = "/etc/systemd/system/test-process.service"
)

func TestGenerateServiceFile(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	writeFileCallCheck := false
	expectedContent := fmt.Sprintf(serviceFileTemplate, testProcess, testCommand,
		fmt.Sprintf("%s=%t", constants.DeployMqttContainerEnv, true))

	patches.ApplyFunc(os.WriteFile, func(path string, data []byte, perm os.FileMode) error {
		writeFileCallCheck = true
		assert.Equal(testServicePath, path)
		assert.Equal(expectedContent, string(data))
		assert.Equal(os.ModePerm, perm)
		return nil
	})

	err := GenerateServiceFile(testProcess, testCommand, true)
	assert.NoError(err)
	assert.True(writeFileCallCheck)

	patches.Reset()
	writeFileCallCheck = false
	expectedContent = fmt.Sprintf(serviceFileTemplate, testProcess, testCommand,
		fmt.Sprintf("%s=%t", constants.DeployMqttContainerEnv, false))

	patches.ApplyFunc(os.WriteFile, func(path string, data []byte, perm os.FileMode) error {
		writeFileCallCheck = true
		assert.Equal(testServicePath, path)
		assert.Equal(expectedContent, string(data))
		assert.Equal(os.ModePerm, perm)
		return nil
	})

	err = GenerateServiceFile(testProcess, testCommand, false)
	assert.NoError(err)
	assert.True(writeFileCallCheck)

	patches.Reset()
	patches.ApplyFunc(os.WriteFile, func(path string, data []byte, perm os.FileMode) error {
		return errors.New("write file error")
	})

	err = GenerateServiceFile(testProcess, testCommand, true)
	assert.Error(err)
	assert.Contains(err.Error(), "write file error")
}

func TestRemoveServiceFile(t *testing.T) {
	assert := assert.New(t)

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	removeCallCheck := false

	patches.ApplyFunc(os.Remove, func(path string) error {
		removeCallCheck = true
		assert.Equal(testServicePath, path)
		return nil
	})

	err := RemoveServiceFile(testProcess)
	assert.NoError(err)
	assert.True(removeCallCheck)

	patches.Reset()
	patches.ApplyFunc(os.Remove, func(path string) error {
		return errors.New("remove file error")
	})

	err = RemoveServiceFile(testProcess)
	assert.Error(err)
	assert.Contains(err.Error(), "remove file error")
}
