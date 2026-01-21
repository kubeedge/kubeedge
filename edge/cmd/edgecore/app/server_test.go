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

package app

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha2"
)

func TestNewEdgeCoreCommand(t *testing.T) {
	assert := assert.New(t)

	cmd := NewEdgeCoreCommand()

	assert.NotNil(cmd)
	assert.IsType(&cobra.Command{}, cmd)
	assert.Equal("edgecore", cmd.Use)

	expectedLongDescription := `Edgecore is the core edge part of KubeEdge, which contains six modules: devicetwin, edged,
edgehub, eventbus, metamanager, and servicebus. DeviceTwin is responsible for storing device status
and syncing device status to the cloud. It also provides query interfaces for applications. Edged is an
agent that runs on edge nodes and manages containerized applications and devices. Edgehub is a web socket
client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge
Architecture). This includes syncing cloud-side resource updates to the edge, and reporting
edge-side host and device status changes to the cloud. EventBus is a MQTT client to interact with MQTT
servers (mosquito), offering publish and subscribe capabilities to other components. MetaManager
is the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata
to/from a lightweight database (SQLite).ServiceBus is a HTTP client to interact with HTTP servers (REST),
offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge. `
	assert.Equal(expectedLongDescription, cmd.Long)

	assert.NotNil(cmd.Run)

	fs := cmd.Flags()

	expectedFlags := []struct {
		name  string
		usage string
	}{
		{
			name:  "config",
			usage: "The path to the configuration file. Flags override values in this file.",
		},
	}

	for _, f := range expectedFlags {
		flag := fs.Lookup(f.name)
		assert.NotNil(flag)
		assert.Equal(flag.Usage, f.usage)
	}
}

func TestCleanupToken(t *testing.T) {
	assert := assert.New(t)

	tmpFile, err := os.CreateTemp("", "edgecore-config-*.yaml")
	assert.NoError(err)
	defer os.Remove(tmpFile.Name())

	// giving a value to the Token
	initialConfig := v1alpha2.NewDefaultEdgeCoreConfig()
	initialConfig.Modules.EdgeHub.Token = "token-for-test"
	data, err := yaml.Marshal(initialConfig)
	assert.NoError(err)
	_, err = tmpFile.Write(data)
	assert.NoError(err)
	tmpFile.Close()

	err = cleanupToken(*initialConfig, tmpFile.Name())
	assert.NoError(err)

	// verify that the token is empty now
	updatedData, err := os.ReadFile(tmpFile.Name())
	assert.NoError(err)

	var updatedConfig v1alpha2.EdgeCoreConfig
	err = yaml.Unmarshal(updatedData, &updatedConfig)
	assert.NoError(err)
	assert.Empty(updatedConfig.Modules.EdgeHub.Token, "Expected token to be empty after running cleanupToken")
}
