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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commfake "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common/fake"
)

func TestMQTTInstToolInstallTools(t *testing.T) {
	t.Run("install success", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		inst := &commfake.MockOSTypeInstaller{}
		called := false

		patches.ApplyFuncReturn(GetOSInterface, inst)
		patches.ApplyMethod(inst, "InstallMQTT", func(*commfake.MockOSTypeInstaller) error {
			called = true
			return nil
		})

		tool := &MQTTInstTool{}
		err := tool.InstallTools()
		require.NoError(t, err)
		assert.True(t, called, "InstallMQTT should be called")
	})

	t.Run("install error", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		inst := &commfake.MockOSTypeInstaller{}
		patches.ApplyFuncReturn(GetOSInterface, inst)
		patches.ApplyMethod(inst, "InstallMQTT", func(*commfake.MockOSTypeInstaller) error {
			return errors.New("install mqtt failed")
		})

		tool := &MQTTInstTool{}
		err := tool.InstallTools()
		require.ErrorContains(t, err, "install mqtt failed")
	})
}

func TestMQTTInstToolTearDown(t *testing.T) {
	tool := &MQTTInstTool{}
	err := tool.TearDown()
	assert.NoError(t, err)
}
