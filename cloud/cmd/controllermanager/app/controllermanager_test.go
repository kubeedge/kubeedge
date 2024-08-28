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
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewControllerManagerCommand(t *testing.T) {
	assert := assert.New(t)

	ctx := context.TODO()
	cmd := NewControllerManagerCommand(ctx)
	assert.NotNil(cmd)
	assert.IsType(&cobra.Command{}, cmd)

	assert.Equal("controller-manager", cmd.Use)
	assert.Contains(cmd.Long, "The node group controller manager run a bunch of controllers")
	assert.NotNil(cmd.Run)

	fs := cmd.Flags()
	assert.NotNil(fs, "Flags should not be nil")

	expectedFlags := []struct {
		Name     string
		DefValue string
		Usage    string
	}{
		{
			Name:     "use-server-side-apply",
			DefValue: "false",
			Usage:    "If use server-side apply when updating templates",
		},
	}

	for _, ef := range expectedFlags {
		flag := fs.Lookup(ef.Name)
		assert.NotNil(flag)
		assert.Equal(ef.Name, flag.Name)
		assert.Equal(ef.DefValue, flag.DefValue)
		assert.Contains(flag.Usage, ef.Usage)
	}
}
