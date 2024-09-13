/*
Copyright 2023 The KubeEdge Authors.

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

	"github.com/kubeedge/api/apis/operations/v1alpha1"
)

func TestFilterVersion(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name         string
		version      string
		expected     string
		expectResult bool
	}{
		{
			name:         "not match expected version",
			version:      "v1.22.6-kubeedge-v1.9.0",
			expected:     "v1.10.0",
			expectResult: false,
		},
		{
			name:         "not match expected version",
			version:      "v1.22.6-kubeedge-v1.10.0-beta.0.194+77ea462f402efb",
			expected:     "v1.10.0",
			expectResult: true,
		},
		{
			name:         "no right format version",
			version:      "v1.22.6",
			expected:     "v1.10.0",
			expectResult: true,
		},
		{
			name:         "match expected version",
			version:      "v1.22.6-kubeedge-v1.10.0",
			expected:     "v1.10.0",
			expectResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := FilterVersion(test.version, test.expected)
			assert.Equal(test.expectResult, result)
		})
	}
}

func TestRemoveDuplicateElement(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "case 1",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "case 2",
			input:    []string{"a", "a", "b", "c", "b", "a", "a"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "case 3",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RemoveDuplicateElement(test.input)
			assert.Equal(test.expected, result)
		})
	}
}

func TestMergeAnnotationUpgradeHistory(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name        string
		origin      string
		fromVersion string
		toVersion   string
		expected    string
	}{
		{
			name:        "case 1: no history record exist",
			origin:      "",
			fromVersion: "v1.10.0",
			toVersion:   "v1.10.1",
			expected:    "v1.10.0->v1.10.1",
		},
		{
			name:        "case 2: 1 history record exist",
			origin:      "v1.10.0->v1.10.1",
			fromVersion: "v1.10.1",
			toVersion:   "v1.10.2",
			expected:    "v1.10.0->v1.10.1;v1.10.1->v1.10.2",
		},
		{
			name:        "case 2: 3 history record exist",
			origin:      "1.10.0->v1.10.1;v1.10.1->v1.10.2;v1.10.2->v1.10.3",
			fromVersion: "v1.10.3",
			toVersion:   "v1.10.4",
			expected:    "v1.10.1->v1.10.2;v1.10.2->v1.10.3;v1.10.3->v1.10.4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MergeAnnotationUpgradeHistory(test.origin, test.fromVersion, test.toVersion)
			assert.Equal(test.expected, result)
		})
	}
}

func TestGetImageRepo(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		Image      string
		ExpectRepo string
	}{
		{Image: "name", ExpectRepo: "docker.io/library/name"},
		{Image: "name:tag", ExpectRepo: "docker.io/library/name"},
		{Image: "name@sha256:59329e44d499406bd2e620473b0ba0b531abb7e326cef0156f33e5957cdfe259", ExpectRepo: "docker.io/library/name"},
		{Image: "org/name", ExpectRepo: "docker.io/org/name"},
		{Image: "org/name:tag", ExpectRepo: "docker.io/org/name"},
		{Image: "org/name@sha256:59329e44d499406bd2e620473b0ba0b531abb7e326cef0156f33e5957cdfe259", ExpectRepo: "docker.io/org/name"},
		{Image: "registry:8080/name", ExpectRepo: "registry:8080/name"},
		{Image: "registry:8080/name:tag", ExpectRepo: "registry:8080/name"},
		{Image: "registry:8080/name@sha256:59329e44d499406bd2e620473b0ba0b531abb7e326cef0156f33e5957cdfe259", ExpectRepo: "registry:8080/name"},
		{Image: "registry:8080/org/name", ExpectRepo: "registry:8080/org/name"},
		{Image: "registry:8080/org/name:tag", ExpectRepo: "registry:8080/org/name"},
		{Image: "registry:8080/org/name@sha256:59329e44d499406bd2e620473b0ba0b531abb7e326cef0156f33e5957cdfe259", ExpectRepo: "registry:8080/org/name"},
	}

	for _, test := range tests {
		t.Run(test.Image, func(t *testing.T) {
			repo, err := GetImageRepo(test.Image)
			assert.NoError(err)
			assert.Equal(test.ExpectRepo, repo)
		})
	}
}

func TestGetNodeName(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		resource string
		expected string
	}{
		{
			name:     "Valid resource string",
			resource: "task/taskId/node/node123",
			expected: "node123",
		},
		{
			name:     "Resource string with extra segments",
			resource: "task/taskId/node/node456/extra",
			expected: "node456",
		},
		{
			name:     "Resource string without NodeID",
			resource: "task/taskId/node/",
			expected: "",
		},
		{
			name:     "Invalid resource",
			resource: "///",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetNodeName(test.resource)
			assert.Equal(test.expected, result)
		})
	}
}

func TestGetTaskID(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name     string
		resource string
		expected string
	}{
		{
			name:     "Valid resource string",
			resource: "task/task123/node/nodeID",
			expected: "task123",
		},
		{
			name:     "Resource string with extra segments",
			resource: "task/task456/node/nodeID/extra",
			expected: "task456",
		},
		{
			name:     "Resource string without TaskID",
			resource: "task//nodeID/node123",
			expected: "",
		},
		{
			name:     "Invalid resource",
			resource: "///",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetTaskID(test.resource)
			assert.Equal(test.expected, result)
		})
	}
}

func TestVersionLess(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name        string
		version1    string
		version2    string
		stdResult   bool
		expectError bool
	}{
		{
			name:        "version1 less than version2",
			version1:    "v1.9.0",
			version2:    "v1.10.0",
			stdResult:   true,
			expectError: false,
		},
		{
			name:        "version1 equal to version2",
			version1:    "v1.10.0",
			version2:    "v1.10.0",
			stdResult:   false,
			expectError: false,
		},
		{
			name:        "version1 greater than version2",
			version1:    "v1.11.0",
			version2:    "v1.10.0",
			stdResult:   false,
			expectError: false,
		},
		{
			name:        "version1 is less and has major version difference",
			version1:    "v1.9.0",
			version2:    "v2.0.0",
			stdResult:   true,
			expectError: false,
		},
		{
			name:        "version1 is less and has patch version difference",
			version1:    "v1.10.0",
			version2:    "v1.10.1",
			stdResult:   true,
			expectError: false,
		},
		{
			name:        "Invalid version1",
			version1:    "invalid",
			version2:    "v1.10.0",
			stdResult:   false,
			expectError: true,
		},
		{
			name:        "Invalid version2",
			version1:    "v1.10.0",
			version2:    "invalid",
			stdResult:   false,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := VersionLess(test.version1, test.version2)

			if test.expectError {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}

			assert.Equal(test.stdResult, result)
		})
	}
}

func TestNodeUpdated(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name      string
		oldStatus v1alpha1.TaskStatus
		newStatus v1alpha1.TaskStatus
		stdResult bool
	}{
		{
			name:      "Different node names",
			oldStatus: v1alpha1.TaskStatus{NodeName: "node1", State: "Running"},
			newStatus: v1alpha1.TaskStatus{NodeName: "node2", State: "Completed"},
			stdResult: false,
		},
		{
			name:      "Same node names",
			oldStatus: v1alpha1.TaskStatus{NodeName: "node1", State: "Running"},
			newStatus: v1alpha1.TaskStatus{NodeName: "node1", State: "Running"},
			stdResult: false,
		},
		{
			name:      "New state empty",
			oldStatus: v1alpha1.TaskStatus{NodeName: "node1", State: "Running"},
			newStatus: v1alpha1.TaskStatus{NodeName: "node1", State: ""},
			stdResult: false,
		},
		{
			name:      "State updated",
			oldStatus: v1alpha1.TaskStatus{NodeName: "node1", State: "Running"},
			newStatus: v1alpha1.TaskStatus{NodeName: "node1", State: "Completed"},
			stdResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NodeUpdated(test.oldStatus, test.newStatus)
			assert.Equal(test.stdResult, result)
		})
	}
}
