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

package debug

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util"
)

func TestCollect_NewCollect(t *testing.T) {
	assert := assert.New(t)
	cmd := NewCollect()

	assert.NotNil(cmd)
	assert.Equal("collect", cmd.Use)
	assert.Equal("Obtain all the data of the current node", cmd.Short)
	assert.Equal(edgecollectLongDescription, cmd.Long)
	assert.Equal(edgecollectExample, cmd.Example)
	assert.NotNil(cmd.Run)

	subcommands := cmd.Commands()
	assert.Empty(subcommands)

	expectedFlags := []struct {
		flagName    string
		shorthand   string
		defaultVal  string
		expectedVal string
	}{
		{
			flagName:    "config",
			shorthand:   "c",
			defaultVal:  common.EdgecoreConfigPath,
			expectedVal: common.EdgecoreConfigPath,
		},
		{
			flagName:    "detail",
			shorthand:   "d",
			defaultVal:  "false",
			expectedVal: "false",
		},
		{
			flagName:    "output-path",
			shorthand:   "o",
			defaultVal:  ".",
			expectedVal: ".",
		},
		{
			flagName:    "log-path",
			shorthand:   "l",
			defaultVal:  util.KubeEdgeLogPath,
			expectedVal: util.KubeEdgeLogPath,
		},
	}

	for _, tt := range expectedFlags {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flag(tt.flagName)
			assert.Equal(tt.flagName, flag.Name)
			assert.Equal(tt.defaultVal, flag.DefValue)
			assert.Equal(tt.expectedVal, flag.Value.String())
			assert.Equal(tt.shorthand, flag.Shorthand)
		})
	}
}

func TestCollect_AddCollectOtherFlags(t *testing.T) {
	assert := assert.New(t)
	cmd := &cobra.Command{}

	co := newCollectOptions()
	addCollectOtherFlags(cmd, co)

	expectedFlags := []struct {
		flagName    string
		shorthand   string
		defaultVal  string
		expectedVal string
	}{
		{
			flagName:    "config",
			shorthand:   "c",
			defaultVal:  common.EdgecoreConfigPath,
			expectedVal: common.EdgecoreConfigPath,
		},
		{
			flagName:    "detail",
			shorthand:   "d",
			defaultVal:  "false",
			expectedVal: "false",
		},
		{
			flagName:    "output-path",
			shorthand:   "o",
			defaultVal:  ".",
			expectedVal: ".",
		},
		{
			flagName:    "log-path",
			shorthand:   "l",
			defaultVal:  util.KubeEdgeLogPath,
			expectedVal: util.KubeEdgeLogPath,
		},
	}

	for _, tt := range expectedFlags {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flag(tt.flagName)
			assert.Equal(tt.flagName, flag.Name)
			assert.Equal(tt.defaultVal, flag.DefValue)
			assert.Equal(tt.expectedVal, flag.Value.String())
			assert.Equal(tt.shorthand, flag.Shorthand)
		})
	}
}

func TestCollect_NewCollectOptions(t *testing.T) {
	assert := assert.New(t)

	co := newCollectOptions()
	assert.NotNil(co)

	assert.Equal(common.EdgecoreConfigPath, co.Config)
	assert.Equal(".", co.OutputPath)
	assert.Equal(false, co.Detail)
}
