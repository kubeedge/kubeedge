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

package main

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestNewHollowEdgeNodeCommand(t *testing.T) {
	assert := assert.New(t)

	cmd := newHollowEdgeNodeCommand()
	assert.Equal("edgemark", cmd.Use, "The command's use should be 'edgemark'")
	assert.Equal("edgemark", cmd.Long, "The command's long description should be 'edgemark'")
	assert.NotNil(cmd.Run)

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		fs.AddFlag(f)
	})

	flags := []struct {
		name         string
		usage        string
		defaultValue interface{}
	}{
		{
			name:         "token",
			usage:        "Token indicates the priority of joining the cluster for the edge.",
			defaultValue: "",
		},
		{
			name:         "name",
			usage:        "Name of this Hollow Node.",
			defaultValue: "fake-node",
		},
		{
			name:         "websocket-server",
			usage:        "Server indicates websocket server address.",
			defaultValue: "",
		},
		{
			name:         "http-server",
			usage:        "HTTPServer indicates the server for edge to apply for the certificate.",
			defaultValue: "",
		},
		{
			name:         "node-labels",
			usage:        "Additional node labels",
			defaultValue: "",
		},
	}

	for _, f := range flags {
		flag := fs.Lookup(f.name)
		assert.NotNil(flag)
		assert.Equal(f.usage, flag.Usage)
		assert.Equal(f.defaultValue, flag.DefValue)
	}
}

func TestAddFlags(t *testing.T) {
	assert := assert.New(t)

	config := &hollowEdgeNodeConfig{
		NodeLabels: make(map[string]string),
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	config.addFlags(fs)

	flags := []struct {
		name         string
		usage        string
		defaultValue interface{}
	}{
		{
			name:         "token",
			usage:        "Token indicates the priority of joining the cluster for the edge.",
			defaultValue: "",
		},
		{
			name:         "name",
			usage:        "Name of this Hollow Node.",
			defaultValue: "fake-node",
		},
		{
			name:         "websocket-server",
			usage:        "Server indicates websocket server address.",
			defaultValue: "",
		},
		{
			name:         "http-server",
			usage:        "HTTPServer indicates the server for edge to apply for the certificate.",
			defaultValue: "",
		},
		{
			name:         "node-labels",
			usage:        "Additional node labels",
			defaultValue: "",
		},
	}

	for _, f := range flags {
		flag := fs.Lookup(f.name)
		assert.NotNil(flag)
		assert.Equal(f.usage, flag.Usage)
		assert.Equal(f.defaultValue, flag.DefValue)
	}
}

func TestEdgeCoreConfig(t *testing.T) {
	assert := assert.New(t)

	config := &hollowEdgeNodeConfig{
		Token:           "test-token",
		NodeName:        "test-node",
		HTTPServer:      "http://localhost:8080",
		WebsocketServer: "ws://localhost:8080",
		NodeLabels:      map[string]string{"key1": "value1", "key2": "value2"},
	}

	edgeCoreConfig := EdgeCoreConfig(config)
	assert.NotNil(edgeCoreConfig)

	assert.Equal("/edgecore.db", edgeCoreConfig.DataBase.DataSource)
	assert.Equal("test-token", edgeCoreConfig.Modules.EdgeHub.Token)
	assert.Equal("http://localhost:8080", edgeCoreConfig.Modules.EdgeHub.HTTPServer)
	assert.Equal("ws://localhost:8080", edgeCoreConfig.Modules.EdgeHub.WebSocket.Server)
	assert.Equal("test-node", edgeCoreConfig.Modules.Edged.HostnameOverride)
	assert.Equal(map[string]string{"key1": "value1", "key2": "value2"}, edgeCoreConfig.Modules.Edged.NodeLabels)
	assert.True(*edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.RegisterNode)
	assert.False(*edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.CgroupsPerQOS)
	assert.Equal(constants.DefaultRuntimeType, edgeCoreConfig.Modules.Edged.ContainerRuntime)
	assert.False(*edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.EnableControllerAttachDetach)
	assert.False(edgeCoreConfig.Modules.Edged.TailoredKubeletConfig.ProtectKernelDefaults)
}

func TestVolumePlugins(t *testing.T) {
	assert := assert.New(t)

	plugins := volumePlugins()

	pluginNames := make(map[string]bool)
	for _, plugin := range plugins {
		pluginNames[plugin.GetPluginName()] = true
	}

	// Expected plugins slice
	expectedPlugins := []string{
		"kubernetes.io/empty-dir",
		"kubernetes.io/host-path",
		"kubernetes.io/secret",
		"kubernetes.io/downward-api",
		"kubernetes.io/configmap",
		"kubernetes.io/projected",
		"kubernetes.io/local-volume",
	}

	for _, expectedPlugin := range expectedPlugins {
		assert.True(pluginNames[expectedPlugin], "Plugin %s should be present", expectedPlugin)
	}
}
