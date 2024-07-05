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

	"github.com/stretchr/testify/assert"
)

func TestNewEdgeDebug(t *testing.T) {
	assert := assert.New(t)

	cmd := NewEdgeDebug()

	assert.NotNil(cmd)
	assert.Equal("debug", cmd.Use)
	assert.Equal(edgeDebugShortDescription, cmd.Short)
	assert.Equal(edgeDebugLongDescription, cmd.Long)

	expectedSubCommands := []string{"get", "diagnose", "check", "collect"}
	for _, subCmd := range expectedSubCommands {
		found := false
		for _, cmd := range cmd.Commands() {
			if cmd.Use == subCmd {
				found = true
				break
			}
		}
		assert.True(found)
	}
}
