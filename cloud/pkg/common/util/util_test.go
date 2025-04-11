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
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/common/constants"
)

func TestIsEdgeNode(t *testing.T) {
	tests := []struct {
		name     string
		node     *corev1.Node
		expected bool
	}{
		{
			name: "edge node with correct label",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.EdgeNodeRoleKey: constants.EdgeNodeRoleValue,
					},
				},
			},
			expected: true,
		},
		{
			name: "node with edge label key but incorrect value",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.EdgeNodeRoleKey: "incorrect-value",
					},
				},
			},
			expected: false,
		},
		{
			name: "node without edge label",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"some-other-label": "some-value",
					},
				},
			},
			expected: false,
		},
		{
			name: "node with nil labels",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: nil,
				},
			},
			expected: false,
		},
		{
			name: "node with empty labels map",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsEdgeNode(test.node)
			if result != test.expected {
				t.Errorf("IsEdgeNode() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestRemoveDuplicateElementWithNumericTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{
			name:     "integers with duplicates",
			input:    []int{1, 2, 3, 1, 2, 3, 4, 5, 4},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "integers without duplicates",
			input:    []int{1, 2, 3, 4, 5},
			expected: []int{1, 2, 3, 4, 5},
		},
		{
			name:     "empty integer slice",
			input:    []int{},
			expected: []int{},
		},
		{
			name:     "single integer",
			input:    []int{42},
			expected: []int{42},
		},
		{
			name:     "all duplicates",
			input:    []int{7, 7, 7, 7, 7},
			expected: []int{7},
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

func TestRemoveDuplicateElementWithFloatTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []float64
		expected []float64
	}{
		{
			name:     "floats with duplicates",
			input:    []float64{1.1, 2.2, 3.3, 1.1, 2.2, 4.4},
			expected: []float64{1.1, 2.2, 3.3, 4.4},
		},
		{
			name:     "floats without duplicates",
			input:    []float64{1.1, 2.2, 3.3},
			expected: []float64{1.1, 2.2, 3.3},
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

func TestRemoveDuplicateElementWithStructTypes(t *testing.T) {
	type testStruct struct {
		ID   int
		Name string
	}

	tests := []struct {
		name     string
		input    []testStruct
		expected []testStruct
	}{
		{
			name: "structs with duplicates",
			input: []testStruct{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
				{ID: 1, Name: "Alice"},
				{ID: 3, Name: "Charlie"},
				{ID: 2, Name: "Bob"},
			},
			expected: []testStruct{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
				{ID: 3, Name: "Charlie"},
			},
		},
		{
			name: "structs without duplicates",
			input: []testStruct{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
				{ID: 3, Name: "Charlie"},
			},
			expected: []testStruct{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
				{ID: 3, Name: "Charlie"},
			},
		},
		{
			name:     "empty struct slice",
			input:    []testStruct{},
			expected: []testStruct{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RemoveDuplicateElement panicked with: %v", r)
				}
			}()

			result := RemoveDuplicateElement(test.input)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Got = %v, Want = %v", result, test.expected)
			}
		})
	}
}

func TestRemoveDuplicateElementWithBoolTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    []bool
		expected []bool
	}{
		{
			name:     "booleans with duplicates",
			input:    []bool{true, false, true, false, true},
			expected: []bool{true, false},
		},
		{
			name:     "only true values",
			input:    []bool{true, true, true},
			expected: []bool{true},
		},
		{
			name:     "only false values",
			input:    []bool{false, false, false},
			expected: []bool{false},
		},
		{
			name:     "empty boolean slice",
			input:    []bool{},
			expected: []bool{},
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

func TestRemoveDuplicateElementWithPointerTypes(t *testing.T) {
	a, b, c := 1, 2, 3

	// First test with duplicate pointers (same memory address)
	pointersDuplicates := []*int{&a, &b, &a, &c, &b}
	expectedUnique := []*int{&a, &b, &c}

	result := RemoveDuplicateElement(pointersDuplicates)
	if len(result) != len(expectedUnique) {
		t.Errorf("Wrong length: got %v, want %v", len(result), len(expectedUnique))
	}

	// Create a map to check if we have all unique addresses
	pointerMap := make(map[*int]bool)
	for _, ptr := range result {
		if pointerMap[ptr] {
			t.Errorf("Duplicate pointer found in result: %v", ptr)
		}
		pointerMap[ptr] = true
	}

	// Check that result contains all expected values
	for _, ptr := range expectedUnique {
		found := false
		for _, resultPtr := range result {
			if ptr == resultPtr {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pointer %v not found in result", ptr)
		}
	}
}
