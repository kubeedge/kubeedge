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

package restart

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewEdgePodRestart(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgePodRestart()

	assert.NotNil(cmd)
	assert.Equal("pod", cmd.Use)
	assert.Equal(edgePodRestartShortDescription, cmd.Short)
	assert.Equal(edgePodRestartShortDescription, cmd.Long)

	assert.NotNil(cmd.RunE)

	assert.Equal(cmd.Flags().Lookup(common.FlagNameNamespace).Name, "namespace")
}

func TestNewRestartPodOpts(t *testing.T) {
	assert := assert.New(t)

	podRestartOptions := NewRestartPodOpts()
	assert.NotNil(podRestartOptions)
	assert.Equal(podRestartOptions.Namespace, "default")
}

func TestAddRestartPodFlags(t *testing.T) {
	assert := assert.New(t)
	getOptions := NewRestartPodOpts()

	cmd := &cobra.Command{}

	AddRestartPodFlags(cmd, getOptions)

	namespaceFlag := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.Equal("default", namespaceFlag.DefValue)
	assert.Equal("namespace", namespaceFlag.Name)
}
