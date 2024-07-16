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

func TestNewCloudInit(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCloudInit()

	assert.Equal(cmd.Use, "init")
	assert.Equal(cmd.Short, "Bootstraps cloud component. Checks and install (if required) the pre-requisites.")
	assert.Equal(cmd.Long, cloudInitLongDescription)
	assert.Equal(cmd.Example, fmt.Sprintf(cloudInitExample, types.DefaultKubeEdgeVersion))

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
			types.FlagNameForce,
			"false",
		},
	}

	for _, flag := range expectedFlags {
		assert.Equal(flag.defaultValue, cmd.Flag(flag.name).DefValue)
		assert.Equal(flag.name, cmd.Flag(flag.name).Name)
	}
}

func TestNewInitOptions(t *testing.T) {
	assert := assert.New(t)

	opts := newInitOptions()
	assert.Equal(opts.KubeConfig, types.DefaultKubeConfig)
}

func TestAddInitOtherFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}
	opts := newInitOptions()

	addInitOtherFlags(cmd, opts)

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
	}

	for _, flag := range expectedFlags {
		assert.Equal(flag.defaultValue, cmd.Flag(flag.name).DefValue)
		assert.Equal(flag.name, cmd.Flag(flag.name).Name)
	}
}

func TestAddHelmValueOptionsFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}
	opts := newInitOptions()

	addHelmValueOptionsFlags(cmd, opts)

	assert.Equal("", cmd.Flag(types.FlagNameProfile).DefValue)
	assert.Equal(types.FlagNameProfile, cmd.Flag(types.FlagNameProfile).Name)

	assert.Equal("[]", cmd.Flag(types.FlagNameSet).DefValue)
	assert.Equal(types.FlagNameSet, cmd.Flag(types.FlagNameSet).Name)
}

func TestAddForceOptionsFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}
	opts := newInitOptions()

	addForceOptionsFlags(cmd, opts)

	assert.Equal("false", cmd.Flag(types.FlagNameForce).Value.String())
	assert.Equal(types.FlagNameForce, cmd.Flag(types.FlagNameForce).Name)
}

func TestAddInit2ToolsList(t *testing.T) {
	assert := assert.New(t)
	toolList := make(map[string]types.ToolsInstaller)
	opts := newInitOptions()

	err := AddInit2ToolsList(toolList, opts)
	assert.Nil(err)
	assert.NotNil(toolList["helm"])
}
