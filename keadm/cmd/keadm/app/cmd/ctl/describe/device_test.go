/*
Copyright 2026 The KubeEdge Authors.

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

package describe

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestNewEdgeDescribeDevice(t *testing.T) {
	cmd := NewEdgeDescribeDevice()
	assert.NotNil(t, cmd)
	assert.Equal(t, "device", cmd.Use)
	assert.Equal(t, edgeDescribeDeviceShortDescription, cmd.Short)
	assert.Contains(t, cmd.Aliases, "devices")

	// Verify flag registration
	flagNamespace := cmd.Flags().Lookup(common.FlagNameNamespace)
	assert.NotNil(t, flagNamespace)
	assert.Equal(t, "n", flagNamespace.Shorthand)
	assert.Equal(t, "default", flagNamespace.DefValue)

	flagSelector := cmd.Flags().Lookup(common.FlagNameLabelSelector)
	assert.NotNil(t, flagSelector)
	assert.Equal(t, "l", flagSelector.Shorthand)

	flagAllNamespaces := cmd.Flags().Lookup(common.FlagNameAllNamespaces)
	assert.NotNil(t, flagAllNamespaces)
	assert.Equal(t, "A", flagAllNamespaces.Shorthand)

	flagShowEvents := cmd.Flags().Lookup(common.FlagNameShowEvents)
	assert.NotNil(t, flagShowEvents)
	assert.Equal(t, "If present, display events related to the described object.", flagShowEvents.Usage)

	flagChunkSize := cmd.Flags().Lookup(common.FlagNameChunkSize)
	assert.NotNil(t, flagChunkSize)
	assert.Equal(t, "500", flagChunkSize.DefValue)
}

func TestNewDescribeDeviceOptions(t *testing.T) {
	opts := NewDescribeDeviceOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.In)
	assert.NotNil(t, opts.Out)
	assert.NotNil(t, opts.ErrOut)
}
