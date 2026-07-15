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
	// Reset the sync.Once and Config just in case for clean test
	once = sync.Once{}
	Config = Configure{}

	dc := &v1alpha1.DeviceController{
		Enable: true,
	}

	InitConfigure(dc)

	require.Equal(t, true, Config.Enable)
	require.Equal(t, *dc, Config.DeviceController)

	// Test sync.Once works, subsequent calls should not overwrite
	dc2 := &v1alpha1.DeviceController{
		Enable: false,
	}
	InitConfigure(dc2)

	require.Equal(t, true, Config.Enable, "Config should not be overwritten after first initialization")
}
