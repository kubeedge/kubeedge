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

package get

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEdgeGet(t *testing.T) {
	assert := assert.New(t)
	cmd := NewEdgeGet()

	assert.NotNil(cmd)

	assert.Equal("get", cmd.Use)
	assert.Equal(edgeGetShortDescription, cmd.Short)
	assert.Equal(edgeGetShortDescription, cmd.Long)

	assert.True(cmd.HasSubCommands())
}
