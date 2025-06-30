/*
Copyright 2025 The KubeEdge Authors.

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

package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvToCRIImage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "malformed name - return as-is",
			input:    "invalid@@image!!name",
			expected: "invalid@@image!!name",
		},
		{
			name:     "docker.io with tag",
			input:    "docker.io/kubeedge/installation-package:v1.20.0",
			expected: "docker.io/kubeedge/installation-package:v1.20.0",
		},
		{
			name:     "docker.io default registry",
			input:    "kubeedge/cloudcore:v1.21.0",
			expected: "docker.io/kubeedge/cloudcore:v1.21.0",
		},
		{
			name:     "docker.io with digest",
			input:    "docker.io/kubeedge/installation-package@sha256:abcd1234",
			expected: "docker.io/kubeedge/installation-package@sha256:abcd1234",
		},
		{
			name:     "docker.io without tag",
			input:    "docker.io/kubeedge/installation-package",
			expected: "docker.io/kubeedge/installation-package",
		},
		{
			name:     "no registry or tag - default to docker.io",
			input:    "kubeedge/installation-package",
			expected: "docker.io/kubeedge/installation-package",
		},
		{
			name:     "private registry with tag",
			input:    "registry.example.com/kubeedge/installation-package:v1.20.0",
			expected: "registry.example.com/kubeedge/installation-package:v1.20.0",
		},
		{
			name:     "private registry without tag",
			input:    "registry.example.com/kubeedge/installation-package",
			expected: "registry.example.com/kubeedge/installation-package",
		},
		{
			name:     "internal registry no namespace",
			input:    "internal-registry.net/cloudcore:v1.20.0",
			expected: "internal-registry.net/cloudcore:v1.20.0",
		},
		{
			name:     "internal registry without tag",
			input:    "internal-registry.net/installation-package",
			expected: "internal-registry.net/installation-package",
		},
		{
			name:     "port registry no namespace",
			input:    "registry.local:5000/cloudcore:v1.20.0",
			expected: "registry.local:5000/cloudcore:v1.20.0",
		},
		{
			name:     "port registry with namespace",
			input:    "registry.local:5000/kubeedge/installation-package:latest",
			expected: "registry.local:5000/kubeedge/installation-package:latest",
		},
		{
			name:     "port registry no tag",
			input:    "registry.local:5000/kubeedge/installation-package",
			expected: "registry.local:5000/kubeedge/installation-package",
		},
		{
			name:     "port registry no namespace (alt)",
			input:    "registry:5000/cloudcore:v1.20.0",
			expected: "registry:5000/cloudcore:v1.20.0",
		},
		{
			name:     "port registry with namespace (alt)",
			input:    "registry:5000/kubeedge/installation-package:latest",
			expected: "registry:5000/kubeedge/installation-package:latest",
		},
		{
			name:     "port registry no tag (alt)",
			input:    "registry:5000/kubeedge/installation-package",
			expected: "registry:5000/kubeedge/installation-package",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := convertCRIImage(tc.input)
			require.Equal(t, tc.expected, output)
		})
	}
}
