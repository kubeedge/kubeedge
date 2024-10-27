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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCSIDriverOptions(t *testing.T) {
	assert := assert.New(t)

	opt := NewCSIDriverOptions()
	assert.NotNil(opt)
	assert.Empty(opt.Endpoint)
	assert.Empty(opt.DriverName)
	assert.Empty(opt.NodeID)
	assert.Empty(opt.KubeEdgeEndpoint)
	assert.Empty(opt.Version)
}

func TestFlags(t *testing.T) {
	assert := assert.New(t)

	opt := NewCSIDriverOptions()
	fss := opt.Flags()
	fs := fss.FlagSet("csidriver")
	assert.NotNil(fs)

	assert.Equal("unix:///csi/csi.sock", opt.Endpoint)
	assert.Equal("csidriver", opt.DriverName)
	assert.Empty(opt.NodeID)
	assert.Equal("unix:///kubeedge/kubeedge.sock", opt.KubeEdgeEndpoint)

	flagTests := []struct {
		name         string
		expectedType string
		usage        string
	}{
		{
			name:         "endpoint",
			expectedType: "string",
			usage:        "CSI endpoint",
		},
		{
			name:         "drivername",
			expectedType: "string",
			usage:        "name of the driver",
		},
		{
			name:         "nodeid",
			expectedType: "string",
			usage:        "node id determines which node will be used to create/delete volumes",
		},
		{
			name:         "kubeedge-endpoint",
			expectedType: "string",
			usage:        "kubeedge endpoint",
		},
	}

	for _, ft := range flagTests {
		flag := fs.Lookup(ft.name)
		assert.NotNil(flag)
		assert.Equal(ft.expectedType, flag.Value.Type())
		assert.Equal(ft.usage, flag.Usage)
	}
}
