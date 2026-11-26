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

package slices

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{
			name:     "single element",
			input:    []string{"x"},
			expected: []string{"x"},
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

	// Test with integers
	intInput := []int{1, 2, 2, 3, 1, 4}
	expectedInts := []int{1, 2, 3, 4}
	assert.Equal(t, expectedInts, RemoveDuplicateElement(intInput))

	// Test with custom comparable struct
	type testStruct struct {
		ID   int
		Name string
	}
	structInput := []testStruct{
		{ID: 1, Name: "alpha"},
		{ID: 1, Name: "alpha"},
		{ID: 2, Name: "beta"},
	}
	expectedStruct := []testStruct{
		{ID: 1, Name: "alpha"},
		{ID: 2, Name: "beta"},
	}
	assert.Equal(t, expectedStruct, RemoveDuplicateElement(structInput))
}

func TestIn(t *testing.T) {
	intArr := []int{1, 2, 3}
	assert.True(t, In(intArr, 1))
	assert.False(t, In(intArr, 4))

	strArr := []string{"a", "b", "c"}
	assert.True(t, In(strArr, "b"))
	assert.False(t, In(strArr, "d"))

	type testStruct struct {
		ID int
	}
	structArr := []testStruct{{ID: 10}, {ID: 20}}
	assert.True(t, In(structArr, testStruct{ID: 10}))
	assert.False(t, In(structArr, testStruct{ID: 30}))
}
