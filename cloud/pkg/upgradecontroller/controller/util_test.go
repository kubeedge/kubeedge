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

package controller

import (
	"reflect"
	"testing"
)

func TestFilterVersion(t *testing.T) {
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
			expectResult: false,
		},
		{
			name:         "no right format version",
			version:      "v1.22.6",
			expected:     "v1.10.0",
			expectResult: false,
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
			result := filterVersion(test.version, test.expected)
			if result != test.expectResult {
				t.Errorf("Got = %v, Want = %v", result, test.expectResult)
			}
		})
	}
}

func TestRemoveDuplicateElement(t *testing.T) {
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
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Got = %v, Want = %v", result, test.expected)
			}
		})
	}
}
