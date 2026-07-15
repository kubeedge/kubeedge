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

package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/cloudcore/v1alpha1"
)

func TestInitConfigure(t *testing.T) {
	once = sync.Once{}
	Config = Configure{}

	// Use t.Cleanup to restore global state after the test execution
	t.Cleanup(func() {
		once = sync.Once{}
		Config = Configure{}
	})

	r := &v1alpha1.Router{
		Enable: true,
	}

	InitConfigure(r)

	require.Equal(t, true, Config.Enable)
	require.Equal(t, *r, Config.Router)

	// Save the initialized Config state
	expectedConfig := Config

	// Test sync.Once works, subsequent calls should not overwrite
	r2 := &v1alpha1.Router{
		Enable: false,
	}
	InitConfigure(r2)

	require.Equal(t, expectedConfig, Config, "Entire config should remain unchanged after first initialization")
}
