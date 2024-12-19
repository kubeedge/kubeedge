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

package options

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/api/apis/common/constants"
)

func TestNewCloudCoreOptions(t *testing.T) {
	assert := assert.New(t)

	options := NewCloudCoreOptions()
	assert.NotNil(options)
	assert.Equal(path.Join(constants.DefaultConfigDir, "cloudcore.yaml"), options.ConfigFile)
}

func TestFlags(t *testing.T) {
	assert := assert.New(t)

	options := NewCloudCoreOptions()
	flagSets := options.Flags()

	globalFlagSet := flagSets.FlagSet("global")
	assert.NotNil(globalFlagSet)

	configFlag := globalFlagSet.Lookup("config")
	assert.NotNil(configFlag)
	assert.Equal("config", configFlag.Name)
	assert.Equal(path.Join(constants.DefaultConfigDir, "cloudcore.yaml"), configFlag.DefValue)
	assert.Equal("The path to the configuration file. Flags override values in this file.", configFlag.Usage)
}

func TestValidate(t *testing.T) {
	assert := assert.New(t)

	options := NewCloudCoreOptions()

	// TestCase 1: non-existent file
	options.ConfigFile = "/non/existent/file.yaml"
	errs := options.Validate()
	assert.Len(errs, 1)
	assert.Contains(errs[0].Error(), "config file /non/existent/file.yaml not exist. For the configuration file format, please refer to --minconfig and --defaultconfig command")

	// TestCase 2: file exists
	tempFile, err := os.CreateTemp("", "cloudcore_test_config_*.yaml")
	assert.NoError(err)
	defer os.Remove(tempFile.Name())

	options.ConfigFile = tempFile.Name()
	errs = options.Validate()
	assert.Empty(errs)
}

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	options := NewCloudCoreOptions()

	// TestCase 1: non-existent file
	options.ConfigFile = "/non/existent/file.yaml"
	cfg, err := options.Config()
	assert.Error(err)
	assert.Nil(cfg)

	// TestCase 2: file exists
	tempFile, err := os.CreateTemp("", "cloudcore_test_config_*.yaml")
	assert.NoError(err)
	defer os.Remove(tempFile.Name())

	options.ConfigFile = tempFile.Name()
	cfg, err = options.Config()
	assert.NoError(err)
	assert.NotNil(cfg)
}
