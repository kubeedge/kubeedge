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
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestGetEdgeCoreOptions(t *testing.T) {
	assert := assert.New(t)

	NewEdgeCoreOptions()
	options := GetEdgeCoreOptions()

	assert.NotNil(options)
	expectedConfigFile := path.Join(constants.DefaultConfigDir, "edgecore.yaml")
	assert.Equal(expectedConfigFile, options.ConfigFile)
}

func TestGetEdgeCoreConfig(t *testing.T) {
	assert := assert.New(t)

	config := GetEdgeCoreConfig()
	assert.Nil(config)

	NewEdgeCoreOptions().Config()

	config = GetEdgeCoreConfig()
	assert.NotNil(config, "Expected non-nil EdgeCoreConfig after initialization")
}

func TestNewEdgeCoreOptions(t *testing.T) {
	assert := assert.New(t)

	options := NewEdgeCoreOptions()
	assert.NotNil(options)
	assert.Equal(path.Join(constants.DefaultConfigDir, "edgecore.yaml"), options.ConfigFile, "Expected default config file path")
}

func TestFlags(t *testing.T) {
	assert := assert.New(t)

	options := NewEdgeCoreOptions()
	fss := options.Flags()

	assert.NotNil(fss)

	fs := fss.FlagSet("global")
	assert.NotNil(fs)

	flag := fs.Lookup("config")
	assert.NotNil(flag)
	assert.Equal("The path to the configuration file. Flags override values in this file.", flag.Usage)
	assert.Equal(options.ConfigFile, flag.Value.String())
}

func TestValidate(t *testing.T) {
	assert := assert.New(t)

	// Creating a temporary valid config file
	validConfigFile, err := os.CreateTemp("", "edgecore.yaml")
	assert.NoError(err)
	defer os.Remove(validConfigFile.Name())

	options := NewEdgeCoreOptions()
	options.ConfigFile = validConfigFile.Name()
	errs := options.Validate()
	assert.Empty(errs, "Expected no validation errors for a valid config file")

	options.ConfigFile = "nonExistentFile.yaml"
	errs = options.Validate()
	assert.NotEmpty(errs, "Expected validation errors for a non-existent config file")
	expectedError := field.Required(field.NewPath("config"),
		fmt.Sprintf("config file %v not exist. For the configuration file format, please refer to --minconfig and --defaultconfig command", options.ConfigFile))

	assert.Contains(errs, expectedError)
}

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary valid config file with some content
	validConfigContent := `
apiVersion: edgecore.config.kubeedge.io/v1alpha2
kind: EdgeCoreConfig
modules:
  edged:
    hostname-override: "edge-node"
`

	validConfigFile, err := os.CreateTemp("", "edgecore.yaml")
	assert.NoError(err, "Expected no error creating a temporary valid config file")
	defer os.Remove(validConfigFile.Name())

	_, err = validConfigFile.WriteString(validConfigContent)
	assert.NoError(err, "Expected no error writing to the temporary valid config file")
	validConfigFile.Close()

	options := NewEdgeCoreOptions()
	options.ConfigFile = validConfigFile.Name()

	config, err := options.Config()
	assert.NoError(err)
	assert.NotNil(config)

	// creating an invalid config file
	invalidConfigFile, err := os.CreateTemp("", "invalid_file.yaml")
	assert.NoError(err)
	defer os.Remove(invalidConfigFile.Name())

	_, err = invalidConfigFile.WriteString("invalid content")
	assert.NoError(err)
	invalidConfigFile.Close()

	options.ConfigFile = invalidConfigFile.Name()

	config, err = options.Config()
	assert.Error(err)
	assert.Nil(config)
}
