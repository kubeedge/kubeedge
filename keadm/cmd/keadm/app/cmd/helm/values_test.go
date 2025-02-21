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
package helm

import (
	"os"
	"reflect"
	"testing"
)

func TestOptionsWithEmptyValues(t *testing.T) {
	opts := &Options{}
	values, err := opts.MergeValues()
	if err != nil {
		t.Fatalf("Unexpected error with empty options: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("Expected empty map, got %v", values)
	}
}

func TestOptionsWithMultipleValueTypes(t *testing.T) {
	jsonFile, err := os.CreateTemp("", "test-json-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp JSON file: %v", err)
	}
	defer os.Remove(jsonFile.Name())
	_, err = jsonFile.Write([]byte(`{"key": "value"}`))
	if err != nil {
		t.Fatalf("Failed to write to temp JSON file: %v", err)
	}
	jsonFile.Close()

	yamlFile, err := os.CreateTemp("", "test-yaml-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp YAML file: %v", err)
	}
	defer os.Remove(yamlFile.Name())
	_, err = yamlFile.Write([]byte(`base:
  key: value`))
	if err != nil {
		t.Fatalf("Failed to write to temp YAML file: %v", err)
	}
	yamlFile.Close()

	opts := &Options{
		ValueFiles:    []string{yamlFile.Name()},
		Values:        []string{"set.key=value"},
		StringValues:  []string{"string.key=value"},
		FileValues:    []string{`file.content=` + jsonFile.Name()},
		LiteralValues: []string{"literal.key=value"},
	}

	values, err := opts.MergeValues()
	if err != nil {
		t.Fatalf("Unexpected error merging multiple value types: %v", err)
	}

	expectedKeys := []string{"base", "set", "string", "file", "literal"}
	for _, key := range expectedKeys {
		if _, exists := values[key]; !exists {
			t.Errorf("Expected key %s to exist in merged values", key)
		}
	}
}

func TestReadFileFromStdin(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		if _, err := w.Write([]byte("test stdin content")); err != nil {
			t.Error("Failed to write to pipe:", err)
		}
		w.Close()
	}()

	content, err := readFile("-")
	if err != nil {
		t.Fatalf("Unexpected error reading from stdin: %v", err)
	}

	if string(content) != "test stdin content" {
		t.Errorf("Unexpected stdin content: got %s, want 'test stdin content'", string(content))
	}
}

func TestMergeMapEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		mapA     map[string]interface{}
		mapB     map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:     "Empty maps",
			mapA:     map[string]interface{}{},
			mapB:     map[string]interface{}{},
			expected: map[string]interface{}{},
		},
		{
			name: "Overwriting nested map with new values",
			mapA: map[string]interface{}{
				"nested": map[string]interface{}{"key1": "value1"},
			},
			mapB: map[string]interface{}{
				"nested": "simple_value",
			},
			expected: map[string]interface{}{
				"nested": "simple_value",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mergeMaps(tc.mapA, tc.mapB)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("mergeMaps() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestErrorCases(t *testing.T) {
	testCases := []struct {
		name        string
		opts        *Options
		expectError bool
	}{
		{
			name: "Invalid JSON input",
			opts: &Options{
				JSONValues: []string{"{invalid json}"},
			},
			expectError: true,
		},
		{
			name: "Invalid set input",
			opts: &Options{
				Values: []string{"invalid set format"},
			},
			expectError: true,
		},
		{
			name: "Non-existent file",
			opts: &Options{
				ValueFiles: []string{"/path/to/non/existent/file"},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.opts.MergeValues()
			if (err != nil) != tc.expectError {
				t.Errorf("MergeValues() error = %v, wantErr %v", err, tc.expectError)
			}
		})
	}
}
func TestMergeValues(t *testing.T) {
	nestedMap := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]string{
			"cool": "stuff",
		},
	}
	anotherNestedMap := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]string{
			"cool":    "things",
			"awesome": "stuff",
		},
	}
	flatMap := map[string]interface{}{
		"foo": "bar",
		"baz": "stuff",
	}
	anotherFlatMap := map[string]interface{}{
		"testing": "fun",
	}

	testMap := mergeMaps(flatMap, nestedMap)
	equal := reflect.DeepEqual(testMap, nestedMap)
	if !equal {
		t.Fatalf("Expected a nested map to overwrite a flat value. Expected: %v, got %v", nestedMap, testMap)
	}

	testMap = mergeMaps(nestedMap, flatMap)
	equal = reflect.DeepEqual(testMap, flatMap)
	if !equal {
		t.Fatalf("Expected a flat value to overwrite a map. Expected: %v, got %v", flatMap, testMap)
	}

	testMap = mergeMaps(nestedMap, anotherNestedMap)
	equal = reflect.DeepEqual(testMap, anotherNestedMap)
	if !equal {
		t.Fatalf("Expected a nested map to overwrite another nested map. Expected: %v, got %v", anotherNestedMap, testMap)
	}

	testMap = mergeMaps(anotherFlatMap, anotherNestedMap)
	expectedMap := map[string]interface{}{
		"testing": "fun",
		"foo":     "bar",
		"baz": map[string]string{
			"cool":    "things",
			"awesome": "stuff",
		},
	}
	equal = reflect.DeepEqual(testMap, expectedMap)
	if !equal {
		t.Fatalf("Expected a map with different keys to merge properly with another map. Expected: %v, got %v", expectedMap, testMap)
	}
}

func TestReadFile(t *testing.T) {
	filePath := "%a.txt"
	_, err := readFile(filePath)
	if err == nil {
		t.Fatalf("Expected error when has special strings")
	}
}
