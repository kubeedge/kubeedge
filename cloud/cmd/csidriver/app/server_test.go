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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCSIDriverCommand(t *testing.T) {
	assert := assert.New(t)

	cmd := NewCSIDriverCommand()

	assert.NotNil(cmd)
	assert.Equal("csidriver", cmd.Use)
	assert.Equal(cmd.Long,
		`CSI Driver from KubeEdge: this is more like CSI Driver proxy,
		and it implements all of the CSI Identity and Controller interfaces.
		It sends messages to CloudHub which will forward to edge. Actually all of the actions
		about the Volume Lifecycle are executed in the CSI Driver from Vendor at edge`)

	assert.NotNil(cmd.Run)

	flags := cmd.Flags()
	assert.NotNil(flags)

	endpointFlag := flags.Lookup("endpoint")
	assert.NotNil(endpointFlag)
	assert.Equal("CSI endpoint", endpointFlag.Usage)
	assert.Equal("unix:///csi/csi.sock", endpointFlag.DefValue)

	drivernameFlag := flags.Lookup("drivername")
	assert.NotNil(drivernameFlag)
	assert.Equal("name of the driver", drivernameFlag.Usage)
	assert.Equal("csidriver", drivernameFlag.DefValue)

	nodeIDFlag := flags.Lookup("nodeid")
	assert.NotNil(nodeIDFlag)
	assert.Equal("node id determines which node will be used to create/delete volumes", nodeIDFlag.Usage)
	assert.Equal("", nodeIDFlag.DefValue)

	kubeEdgeEndpointFlag := flags.Lookup("kubeedge-endpoint")
	assert.NotNil(kubeEdgeEndpointFlag)
	assert.Equal("kubeedge endpoint", kubeEdgeEndpointFlag.Usage)
	assert.Equal("unix:///kubeedge/kubeedge.sock", kubeEdgeEndpointFlag.DefValue)

	versionFlag := flags.Lookup("version")
	assert.NotNil(versionFlag)
	assert.Equal("Print version information and quit", versionFlag.Usage)
	assert.Equal("false", versionFlag.DefValue)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	usageFunc := cmd.UsageFunc()
	err := usageFunc(cmd)
	assert.NoError(err)

	output := buf.String()
	expectedUsage := "Usage:\n  csidriver [flags]\n"
	assert.Contains(output, expectedUsage)

	buf.Reset()
	helpFunc := cmd.HelpFunc()
	helpFunc(cmd, []string{})

	output = buf.String()
	expectedHelp := "CSI Driver from KubeEdge: this is more like CSI Driver proxy"
	assert.Contains(output, expectedHelp)
}
