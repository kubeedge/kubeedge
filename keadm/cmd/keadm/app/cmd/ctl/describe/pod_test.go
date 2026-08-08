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

func TestNewEdgeDescribePod(t *testing.T) {
	cmd := NewEdgeDescribePod()
	assert.NotNil(t, cmd)
	assert.Equal(t, "pod", cmd.Use)
	assert.Equal(t, edgeDescribePodShortDescription, cmd.Short)
	assert.Contains(t, cmd.Aliases, "pods")
	assert.Contains(t, cmd.Aliases, "po")

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

func TestNewDescribePodOptions(t *testing.T) {
	opts := NewDescribePodOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.In)
	assert.NotNil(t, opts.Out)
	assert.NotNil(t, opts.ErrOut)
}
