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
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewManifestGenerate(t *testing.T) {
	assert := assert.New(t)

	cmd := NewManifestGenerate()

	assert.Equal(cmd.Use, "manifest")
	assert.Equal(cmd.Short, "Checks and generate the manifests.")
	assert.Equal(cmd.Long, cloudManifestLongDescription)
	assert.Equal(cmd.Example, fmt.Sprintf(cloudManifestGenerateExample, types.DefaultKubeEdgeVersion))

	assert.NotNil(cmd.RunE)

	expectedFlags := []struct {
		name         string
		defaultValue string
	}{
		{
			types.FlagNameKubeEdgeVersion,
			"",
		},
		{
			types.FlagNameAdvertiseAddress,
			"",
		},
		{
			types.FlagNameKubeConfig,
			types.DefaultKubeConfig,
		},
		{
			types.FlagNameManifests,
			"",
		},
		{
			types.FlagNameFiles,
			"",
		},
		{
			types.FlagNameDryRun,
			"false",
		},
		{
			types.FlagNameExternalHelmRoot,
			"",
		},
		{
			types.FlagNameImageRepository,
			"",
		},
		{
			types.FlagNameSet,
			"[]",
		},
		{
			types.FlagNameProfile,
			"",
		},
		{
			types.FlagNameSkipCRDs,
			"false",
		},
	}

	for _, flag := range expectedFlags {
		assert.Equal(flag.defaultValue, cmd.Flag(flag.name).DefValue)
		assert.Equal(flag.name, cmd.Flag(flag.name).Name)
	}
}

func TestAddManifestsGenerateJoinOtherFlags(t *testing.T) {
	assert := assert.New(t)

	cmd := &cobra.Command{}
	initOpts := &types.InitOptions{}

	addManifestsGenerateJoinOtherFlags(cmd, initOpts)

	expectedFlags := []struct {
		name         string
		defaultValue string
	}{
		{
			types.FlagNameKubeEdgeVersion,
			"",
		},
		{
			types.FlagNameAdvertiseAddress,
			"",
		},
		{
			types.FlagNameKubeConfig,
			"",
		},
		{
			types.FlagNameManifests,
			"",
		},
		{
			types.FlagNameFiles,
			"",
		},
		{
			types.FlagNameDryRun,
			"false",
		},
		{
			types.FlagNameExternalHelmRoot,
			"",
		},
		{
			types.FlagNameImageRepository,
			"",
		},
		{
			types.FlagNameSet,
			"[]",
		},
		{
			types.FlagNameProfile,
			"",
		},
		{
			types.FlagNameSkipCRDs,
			"false",
		},
	}

	for _, flag := range expectedFlags {
		assert.Equal(flag.defaultValue, cmd.Flag(flag.name).DefValue)
		assert.Equal(flag.name, cmd.Flag(flag.name).Name)
	}
}

func TestAddManifestsGenerate2ToolsList(t *testing.T) {
	assert := assert.New(t)
	toolList := make(map[string]types.ToolsInstaller)
	flagData := make(map[string]types.FlagData)
	opts := newInitOptions()

	err := AddManifestsGenerate2ToolsList(toolList, flagData, opts)
	assert.Nil(err)
	assert.NotNil(toolList["helm"])
}
