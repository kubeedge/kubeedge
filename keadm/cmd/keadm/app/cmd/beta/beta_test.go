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

package beta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBeta(t *testing.T) {
	assert := assert.New(t)
	cmd := NewBeta()

	assert.NotNil(cmd)
	assert.Equal(cmd.Use, "beta")
	assert.Equal(cmd.Short, "keadm beta command")
	assert.Equal(cmd.Long, `keadm beta command provides some subcommands that are still in testing, but have complete functions and can be used in advance, but now it contains nothing`)

	flags := cmd.Flags()
	assert.NotNil(flags)
	assert.Equal("", flags.FlagUsages())
}
