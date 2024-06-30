/*
Copyright 2024 The KubeEdge Authors.

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

package cloud

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewCloudReset(t *testing.T) {
	assert := assert.New(t)

	cmd := NewCloudReset()

	assert.Equal("cloud", cmd.Use)
	assert.Equal("Teardowns CloudCore component", cmd.Short)
	assert.Equal(resetLongDescription, cmd.Long)
	assert.Equal(resetExample, cmd.Example)

	assert.NotNil(cmd.PreRunE)
	assert.NotNil(cmd.RunE)

	kubeconfigFlag := cmd.Flag(common.FlagNameKubeConfig)
	assert.Equal(common.DefaultKubeConfig, kubeconfigFlag.DefValue)
	assert.Equal(common.FlagNameKubeConfig, kubeconfigFlag.Name)

	forceFlag := cmd.Flag("force")
	assert.Equal("false", forceFlag.DefValue)
	assert.Equal("force", forceFlag.Name)
}

func TestAddResetFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}
	resetOpts := &common.ResetOptions{}

	addResetFlags(cmd, resetOpts)

	kubeconfigFlag := cmd.Flag(common.FlagNameKubeConfig)
	assert.NotNil(kubeconfigFlag)
	assert.Equal(common.DefaultKubeConfig, kubeconfigFlag.DefValue)
	assert.Equal(common.FlagNameKubeConfig, kubeconfigFlag.Name)

	forceFlag := cmd.Flag("force")
	assert.NotNil(forceFlag)
	assert.Equal("false", forceFlag.DefValue)
	assert.Equal("force", forceFlag.Name)
}
