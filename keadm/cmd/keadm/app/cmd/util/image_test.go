/*
Copyright 2022 The KubeEdge Authors.

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

	"github.com/stretchr/testify/assert"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
)

func TestSet_Current(t *testing.T) {
	tests := []struct {
		name            string
		set             Set
		imageRepository string
		version         string
		expected        Set
	}{
		{
			name: "Update repository and version",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			imageRepository: "my-registry.io",
			version:         "v2.0",
			expected: Set{
				"component1": "my-registry.io/component1:v2.0",
				"component2": "my-registry.io/component2:v2.0",
			},
		},
		{
			name: "Update only repository",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			imageRepository: "my-registry.io",
			version:         "",
			expected: Set{
				"component1": "my-registry.io/component1:v1.0",
				"component2": "my-registry.io/component2:v1.0",
			},
		},
		{
			name: "Update only version",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			imageRepository: "",
			version:         "v2.0",
			expected: Set{
				"component1": "kubeedge/component1:v2.0",
				"component2": "kubeedge/component2:v2.0",
			},
		},
		{
			name: "No updates",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			imageRepository: "",
			version:         "",
			expected: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
		},
		{
			name: "Handle components without version",
			set: Set{
				"component1": "kubeedge/component1",
				"component2": "kubeedge/component2",
			},
			imageRepository: "my-registry.io",
			version:         "v2.0",
			expected: Set{
				"component1": "my-registry.io/component1:v2.0",
				"component2": "my-registry.io/component2:v2.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.set.Current(tt.imageRepository, tt.version)
			assert.Equal(t, tt.expected, result)

			for k, v := range tt.set {
				assert.Equal(t, v, tt.set[k])
			}
		})
	}
}

func TestSet_Get(t *testing.T) {
	set := Set{
		"component1": "kubeedge/component1:v1.0",
		"component2": "kubeedge/component2:v1.0",
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "Get existing component",
			key:      "component1",
			expected: "kubeedge/component1:v1.0",
		},
		{
			name:     "Get non-existing component",
			key:      "component3",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := set.Get(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSet_Merge(t *testing.T) {
	tests := []struct {
		name     string
		set      Set
		src      Set
		expected Set
	}{
		{
			name: "Merge non-overlapping sets",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			src: Set{
				"component3": "kubeedge/component3:v1.0",
				"component4": "kubeedge/component4:v1.0",
			},
			expected: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
				"component3": "kubeedge/component3:v1.0",
				"component4": "kubeedge/component4:v1.0",
			},
		},
		{
			name: "Merge with overlapping keys",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			src: Set{
				"component2": "kubeedge/component2:v2.0",
				"component3": "kubeedge/component3:v1.0",
			},
			expected: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v2.0",
				"component3": "kubeedge/component3:v1.0",
			},
		},
		{
			name: "Merge with empty source",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			src: Set{},
			expected: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
		},
		{
			name: "Merge into empty set",
			set:  Set{},
			src: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			expected: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.set.Merge(tt.src)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.set, result)
		})
	}
}

func TestSet_List(t *testing.T) {
	tests := []struct {
		name     string
		set      Set
		expected []string
	}{
		{
			name: "List all components",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			expected: []string{"kubeedge/component1:v1.0", "kubeedge/component2:v1.0"},
		},
		{
			name:     "List empty set",
			set:      Set{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.set.List()

			assert.Equal(t, len(tt.expected), len(result))
			for _, val := range result {
				assert.Contains(t, tt.expected, val)
			}
		})
	}
}

func TestSet_Remove(t *testing.T) {
	tests := []struct {
		name     string
		set      Set
		key      string
		expected Set
	}{
		{
			name: "Remove existing component",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			key: "component1",
			expected: Set{
				"component2": "kubeedge/component2:v1.0",
			},
		},
		{
			name: "Remove non-existing component",
			set: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
			key: "component3",
			expected: Set{
				"component1": "kubeedge/component1:v1.0",
				"component2": "kubeedge/component2:v1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.set.Remove(tt.key)
			assert.Equal(t, tt.expected, result)

			assert.Equal(t, tt.set, result)
		})
	}
}

func TestEdgeSet(t *testing.T) {
	tests := []struct {
		name    string
		opt     *common.JoinOptions
		wantSet Set
	}{
		{
			name: "Default options",
			opt: &common.JoinOptions{
				KubeEdgeVersion: "v1.10.0",
				ImageRepository: "",
			},
			wantSet: Set{
				EdgeCore: "kubeedge/installation-package:v1.10.0",
			},
		},
		{
			name: "Custom repository",
			opt: &common.JoinOptions{
				KubeEdgeVersion: "v1.10.0",
				ImageRepository: "my-registry.io",
			},
			wantSet: Set{
				EdgeCore: "my-registry.io/installation-package:v1.10.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EdgeSet(tt.opt)
			assert.Equal(t, tt.wantSet, result)
		})
	}
}

func TestCloudSet(t *testing.T) {
	tests := []struct {
		name            string
		imageRepository string
		version         string
		wantSet         Set
	}{
		{
			name:            "Default options",
			imageRepository: "",
			version:         "v1.10.0",
			wantSet: Set{
				CloudAdmission:         "kubeedge/admission:v1.10.0",
				CloudCloudcore:         "kubeedge/cloudcore:v1.10.0",
				CloudIptablesManager:   "kubeedge/iptables-manager:v1.10.0",
				CloudControllerManager: "kubeedge/controller-manager:v1.10.0",
			},
		},
		{
			name:            "Custom repository",
			imageRepository: "my-registry.io",
			version:         "v1.10.0",
			wantSet: Set{
				CloudAdmission:         "my-registry.io/admission:v1.10.0",
				CloudCloudcore:         "my-registry.io/cloudcore:v1.10.0",
				CloudIptablesManager:   "my-registry.io/iptables-manager:v1.10.0",
				CloudControllerManager: "my-registry.io/controller-manager:v1.10.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CloudSet(tt.imageRepository, tt.version)
			assert.Equal(t, tt.wantSet, result)
		})
	}
}
