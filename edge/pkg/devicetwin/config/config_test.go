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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func TestInitConfigure(t *testing.T) {
	// Use t.Cleanup to restore global state after the test execution
	t.Cleanup(func() {
		config = Configure{}
	})

	dt := &v1alpha2.DeviceTwin{}
	nodeName := "test-node"

	InitConfigure(dt, nodeName)

	require.Equal(t, *dt, config.DeviceTwin)
	require.Equal(t, nodeName, config.NodeName)

	c := Get()
	require.NotNil(t, c)
	require.Equal(t, *dt, c.DeviceTwin)
	require.Equal(t, nodeName, c.NodeName)

	// Save the initialized config state
	expectedConfig := config

	// Test sync.Once works, subsequent calls should not overwrite
	dt2 := &v1alpha2.DeviceTwin{}
	InitConfigure(dt2, "different-node")

	require.Equal(t, expectedConfig, config, "Entire config should remain unchanged after first initialization")
}
