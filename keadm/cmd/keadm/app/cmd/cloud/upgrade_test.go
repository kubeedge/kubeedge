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

	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewCloudUpgrade(t *testing.T) {
	assert := assert.New(t)

	cmd := NewCloudUpgrade()

	assert.Equal(cmd.Use, "cloud")
	assert.Equal(cmd.Short, "Upgrade the cloud components")
	assert.Equal(cmd.Long, "Upgrade the cloud components to the desired version, "+
		"it uses helm to upgrade the installed release of cloudcore chart, which includes all the cloud components")
	assert.NotNil(cmd.RunE)

	flag := cmd.Flags().Lookup(types.FlagNameKubeEdgeVersion)
	assert.NotNil(flag)
	assert.Equal("", flag.DefValue)
	assert.Equal(types.FlagNameKubeEdgeVersion, flag.Name)

	flag = cmd.Flags().Lookup(types.FlagNameAdvertiseAddress)
	assert.NotNil(flag)
	assert.Equal("", flag.DefValue)
	assert.Equal(types.FlagNameAdvertiseAddress, flag.Name)
}

func TestNewCloudUpgradeOptions(t *testing.T) {
	assert := assert.New(t)
	opts := newCloudUpgradeOptions()

	assert.NotNil(opts)
	assert.Equal(opts.KubeConfig, types.DefaultKubeConfig)
}

func TestAddUpgradeOptionFlags(t *testing.T) {
	assert := assert.New(t)

	cmd := &cobra.Command{}

	opts := newCloudUpgradeOptions()

	addUpgradeOptionFlags(cmd, opts)

	expectedFlags := []struct {
		name      string
		shorthand string
		defValue  interface{}
	}{
		{
			name:      types.FlagNameKubeEdgeVersion,
			shorthand: "",
			defValue:  "",
		},
		{
			name:      types.FlagNameAdvertiseAddress,
			shorthand: "",
			defValue:  "",
		},
		{
			name:      types.FlagNameKubeConfig,
			shorthand: "",
			defValue:  types.DefaultKubeConfig,
		},
		{
			name:      types.FlagNameDryRun,
			shorthand: "d",
			defValue:  "false",
		},
		{
			name:      types.FlagNameRequireConfirmation,
			shorthand: "r",
			defValue:  "false",
		},
		{
			name:      types.FlagNameSet,
			shorthand: "",
			defValue:  "[]",
		},
		{
			name:      types.FlagNameValueFiles,
			shorthand: "",
			defValue:  "[]",
		},
		{
			name:      types.FlagNameForce,
			shorthand: "",
			defValue:  "false",
		},
		{
			name:      types.FlagNameProfile,
			shorthand: "",
			defValue:  "",
		},
		{
			name:      types.FlagNameExternalHelmRoot,
			shorthand: "",
			defValue:  "",
		},
		{
			name:      types.FlagNameReuseValues,
			shorthand: "",
			defValue:  "false",
		},
		{
			name:      types.FlagNamePrintFinalValues,
			shorthand: "",
			defValue:  "false",
		},
		{
			name:      types.FlagNameImageRepository,
			shorthand: "",
			defValue:  "",
		},
	}

	for _, ef := range expectedFlags {
		flag := cmd.Flags().Lookup(ef.name)
		assert.NotNil(flag)
		if ef.shorthand != "" {
			assert.Equal(ef.shorthand, flag.Shorthand)
		}
		assert.Equal(ef.defValue, flag.DefValue)
	}
}
