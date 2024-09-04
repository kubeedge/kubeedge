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

package csidriver

import (
	"context"
	"testing"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewIdentityServer(t *testing.T) {
	assert := assert.New(t)

	name := "test-server"
	version := "v1.0.0"

	ids := newIdentityServer(name, version)

	assert.NotNil(ids)
	assert.Equal(name, ids.name)
	assert.Equal(version, ids.version)
}

func TestGetPluginInfo(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		name        string
		version     string
		expectError bool
		errorCode   codes.Code
	}{
		{
			name:        "test-driver",
			version:     "v1.0.0",
			expectError: false,
			errorCode:   codes.OK,
		},
		{
			name:        "",
			version:     "v1.0.0",
			expectError: true,
			errorCode:   codes.Unavailable,
		},
		{
			name:        "test-driver",
			version:     "",
			expectError: true,
			errorCode:   codes.Unavailable,
		},
		{
			name:        "",
			version:     "",
			expectError: true,
			errorCode:   codes.Unavailable,
		},
	}

	for _, tc := range testCases {
		ids := newIdentityServer(tc.name, tc.version)
		result, err := ids.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})

		if tc.expectError {
			assert.Error(err)
			assert.Equal(tc.errorCode, status.Code(err))
		} else {
			assert.NoError(err)
			assert.NotNil(result)
			assert.Equal(tc.name, result.Name)
			assert.Equal(tc.version, result.VendorVersion)
		}
	}
}

func TestProbe(t *testing.T) {
	assert := assert.New(t)

	ids := newIdentityServer("test-driver", "v1.0.0")
	resp, err := ids.Probe(context.Background(), &csi.ProbeRequest{})

	assert.NoError(err)
	assert.NotNil(resp)
}

func TestGetPluginCapabilities(t *testing.T) {
	assert := assert.New(t)

	ids := newIdentityServer("test-driver", "v1.0.0")
	result, err := ids.GetPluginCapabilities(context.Background(), &csi.GetPluginCapabilitiesRequest{})

	assert.NoError(err)
	assert.NotNil(result)
	assert.Len(result.Capabilities, 1)

	capabilities := result.Capabilities[0]
	assert.NotNil(capabilities.GetService())
	assert.Equal(csi.PluginCapability_Service_CONTROLLER_SERVICE, capabilities.GetService().Type)
}
