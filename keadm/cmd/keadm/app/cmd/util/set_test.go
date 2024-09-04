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

package util

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
)

func TestParseSetByComma(t *testing.T) {
	testCases := []struct {
		set        string
		expectVals []string
	}{
		// Test case 1: Normal input
		{
			set:        "value1,value2,value3",
			expectVals: []string{"value1", "value2", "value3"},
		},
		// Test case 2: Input with spaces
		{
			set:        "value1=1, value2[1].var=2 , value3.var=helo",
			expectVals: []string{"value1=1", "value2[1].var=2", "value3.var=helo"},
		},
		// Test case 3: Input with quotes
		{
			set:        `"value1","value2","value3"`,
			expectVals: []string{`"value1"`, `"value2"`, `"value3"`},
		},
		// Test case 4: Input with braces
		{
			set:        "{value1},{value2},{value3}",
			expectVals: []string{"{value1}", "{value2}", "{value3}"},
		},
		// Test case 5: Mixed input
		{
			set:        `value1,{"value2","value3"},value4`,
			expectVals: []string{"value1", `{"value2","value3"}`, "value4"},
		},
		// Test case 6: Set with empty values
		{
			set:        ",value1,,value2,",
			expectVals: []string{"value1", "value2"},
		},
	}

	for _, tc := range testCases {
		got := parseSetByComma(tc.set)
		if !reflect.DeepEqual(got, tc.expectVals) {
			t.Errorf("ParseSetByComma(%q) = %v, expect %v", tc.set, got, tc.expectVals)
		}
	}
}
func TestParseSetByEqual(t *testing.T) {
	testCases := []struct {
		set         []string
		expectNames []string
		expectVals  []string
	}{
		{
			set:         []string{"name1=value1", "name2=value2", "name3=value3"},
			expectNames: []string{"name1", "name2", "name3"},
			expectVals:  []string{"value1", "value2", "value3"},
		},
		{
			set:         []string{"name1value1", "name2=value2", "name3=value3"},
			expectNames: []string{"name2", "name3"},
			expectVals:  []string{"value2", "value3"},
		},
	}
	for _, tc := range testCases {
		names, vals := parseSetByEqual(tc.set)
		if !reflect.DeepEqual(names, tc.expectNames) || !reflect.DeepEqual(vals, tc.expectVals) {
			t.Errorf("Failed for input %v. Expected (%v,%v),got(%v,%v)", tc.set, tc.expectNames, tc.expectVals, names, vals)
		}
	}
}

func TestParseSetValue(t *testing.T) {
	testCases := []struct {
		vals        []string
		expectParse []interface{}
	}{
		// Test case 1: Normal input
		{
			vals:        []string{"123", "true", "3.14", "hello"},
			expectParse: []interface{}{123, true, 3.14, "hello"},
		},
		// Test case 2: Empty input
		{
			vals:        []string{},
			expectParse: []interface{}{},
		},
		// Test case 3: Input with mixed types
		{
			vals:        []string{"123", "true", "3.14", "hello", "false", "42"},
			expectParse: []interface{}{123, true, 3.14, "hello", false, 42},
		},
	}

	for _, tc := range testCases {
		got := parseSetValue(tc.vals)
		if !reflect.DeepEqual(got, tc.expectParse) {
			t.Errorf("ParseSetValue(%q) = %v, want %v", tc.vals, got, tc.expectParse)
		}
	}
}
func TestParseValue(t *testing.T) {
	testCases := []struct {
		s            string
		expectResult interface{}
	}{
		// Test case 1: Integer input
		{
			s:            "123",
			expectResult: 123,
		},
		// Test case 2: Float input
		{
			s:            "3.14",
			expectResult: 3.14,
		},
		// Test case 3: String input
		{
			s:            "hello",
			expectResult: "hello",
		},
		// Test case 4: Array input
		{
			s:            "{1, 2, 3}",
			expectResult: []int{1, 2, 3},
		},
		// Test case 5: Empty input
		{
			s:            "",
			expectResult: "",
		},
		// Test case 6: Bool true input
		{
			s:            "true",
			expectResult: true,
		},
		// Test case 7: Bool false input
		{
			s:            "false",
			expectResult: false,
		},
	}

	for _, tc := range testCases {
		result := parseValue(tc.s)
		if !reflect.DeepEqual(result, tc.expectResult) {
			t.Errorf("Failed for input %s. Expected %v, got %v", tc.s, tc.expectResult, result)
		}
	}
}

func TestParseArray(t *testing.T) {
	testCases := []struct {
		s            string
		expectResult interface{}
	}{
		// Test case 1: Integer array input
		{
			s:            "{1, 2, 3}",
			expectResult: []int{1, 2, 3},
		},
		// Test case 2: Float array input
		{
			s:            "{3.14, 2.718, 1.618}",
			expectResult: []float64{3.14, 2.718, 1.618},
		},
		// Test case 3: String array input
		{
			s:            `{"apple", "banana", "cherry"}`,
			expectResult: []string{`"apple"`, `"banana"`, `"cherry"`},
		},
	}

	for _, tc := range testCases {
		got := parseArray(tc.s)
		if !reflect.DeepEqual(got, tc.expectResult) {
			t.Errorf("ParseArray(%q) = %v, want %v", tc.s, got, tc.expectResult)
		}
	}
}
func TestParseType(t *testing.T) {
	testCases := []struct {
		s            string
		expectResult string
	}{
		{
			s:            "123",
			expectResult: "int",
		},
		{
			s:            "3.14",
			expectResult: "float",
		},
		{
			s:            "hello",
			expectResult: "string",
		},
		{
			s:            "true",
			expectResult: "bool",
		},
		{
			s:            "false",
			expectResult: "bool",
		},
	}
	for _, tc := range testCases {
		result := parseType(tc.s)
		if result != tc.expectResult {
			t.Errorf("Failed for input %s. Expected %s,got %s", tc.s, tc.expectResult, result)
		}
	}
}

func TestGetNameFormStatus(t *testing.T) {
	testCases := []struct {
		s            string
		expectResult int
	}{
		// Test case 1: String without brackets
		{
			s:            "name1",
			expectResult: 0,
		},
		// Test case 2
		{
			s:            "name1[0].variable1",
			expectResult: 1,
		},
		// Test case 3
		{
			s:            "name1[0]",
			expectResult: 2,
		},
		// Test case 4
		{
			s:            "name1[0].variable1.value2",
			expectResult: 1,
		},
	}

	for _, tc := range testCases {
		got := getNameFormStatus(tc.s)
		if got != tc.expectResult {
			t.Errorf("GetNameFormStatus(%q) = %d, want %d", tc.s, got, tc.expectResult)
		}
	}
}

func TestSetCommonValue(t *testing.T) {
	type Config struct {
		Name  string
		Value int
	}
	testStruct := Config{Name: "initial", Value: 42}

	err := setCommonValue(&testStruct, "Name", "updated")
	if err != nil {
		t.Errorf("Failed to set value:%v", err)
	}
	if testStruct.Name != "updated" {
		t.Errorf("Failed to set string value.Expected 'updated',got '%s'", testStruct.Name)
	}

	err = setCommonValue(&testStruct, "Value", 100)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}
	if testStruct.Value != 100 {
		t.Errorf("Failed to set int value. Expected 100, got %d", testStruct.Value)
	}

	err = setCommonValue(&testStruct, "Name", 123)
	if err == nil {
		t.Error("Expected an error for setting incorrect value type, but got nil")
	}

	err = setCommonValue(&testStruct, "NonExistentField", "value")
	if err == nil {
		t.Error("Expected an error for setting value to non-existent field, but got nil")
	}
}
func TestParseAndSetArrayValue(t *testing.T) {
	type config struct {
		ArrayField [3]string
	}
	testStruct := config{ArrayField: [3]string{"1", "2", "3"}}
	newVal := "10"
	err := parseAndSetArrayValue(&testStruct, "ArrayField[1]", newVal)
	if err != nil {
		t.Errorf("Failed to set array value: %v", err)
	}
	expectedArray := [3]string{"1", "10", "3"}
	if !reflect.DeepEqual(testStruct.ArrayField, expectedArray) {
		t.Errorf("Failed to set array value. Expected %v, got %v", expectedArray, testStruct.ArrayField)
	}
	err = parseAndSetArrayValue(&testStruct, "ArrayField[10]", "10")
	if err == nil {
		t.Error("Expected an error for setting value to non-existent index, but got nil")
	}
}

func TestSetArrayValue(t *testing.T) {
	type Config struct {
		ArrayField [3]string
	}
	// Initialize a test struct
	testStruct := Config{ArrayField: [3]string{"1", "2", "3"}}

	// Test case 1: Set array value
	err := setArrayValue(&testStruct, "ArrayField", 1, "10")
	if err != nil {
		t.Errorf("Failed to set array value: %v", err)
	}
	expectedArray := [3]string{"1", "10", "3"}
	if !reflect.DeepEqual(testStruct.ArrayField, expectedArray) {
		t.Errorf("Failed to set array value. Expected %v, got %v", expectedArray, testStruct.ArrayField)
	}

	// Test case 2: Set value to non-existent index
	err = setArrayValue(&testStruct, "ArrayField", 10, 5)
	if err == nil {
		t.Error("Expected an error for setting value to non-existent index, but got nil")
	}

	// Test case 3: Set value with incorrect type
	err = setArrayValue(&testStruct, "ArrayField", 1, 1)
	if err == nil {
		t.Error("Expected an error for setting incorrect value type, but got nil")
	}
}

func TestParseFieldPath1(t *testing.T) {
	testCases := []struct {
		fieldPath     string
		expectedPath  string
		expectedIndex int
		expectedError string
	}{
		// Test case 1: Valid field path with index
		{
			fieldPath:     "ArrayField[1]",
			expectedPath:  "ArrayField",
			expectedIndex: 1,
			expectedError: "",
		},
		// Test case 2: Invalid field path (no index)
		{
			fieldPath:     "ArrayField",
			expectedPath:  "",
			expectedIndex: -1,
			expectedError: "invalid field path",
		},
	}

	for _, tc := range testCases {
		path, index, err := parseFieldPath1(tc.fieldPath)
		// Check if the output matches the expected output
		if path != tc.expectedPath || index != tc.expectedIndex || fmt.Sprintf("%v", err) != tc.expectedError {
			t.Errorf("Failed for field path %s. Expected (%s, %d, %s), got (%s, %d, %v)", tc.fieldPath, tc.expectedPath, tc.expectedIndex, tc.expectedError, path, index, err)
		}
	}
}

type Config struct {
	Name1 Name1Config
}

type Name1Config struct {
	Name2 []Name2Config
}

type Name2Config struct {
	Variable1 int
	Variable2 float64
	Variable3 string
}

func TestSetVariableValue(t *testing.T) {
	config := &Config{
		Name1: Name1Config{
			Name2: []Name2Config{
				{Variable1: 10, Variable2: 3.14, Variable3: "hello"},
				{Variable1: 20, Variable2: 6.28, Variable3: "world"},
			},
		},
	}

	err := setVariableValue(config, "Name1.Name2[0].Variable1", 100)
	if err != nil {
		t.Errorf("Error updating field: %v", err)
	}
	if config.Name1.Name2[0].Variable1 != 100 {
		t.Errorf("Variable1 not updated properly")
	}

	err = setVariableValue(config, "Name1.Name2[1].Variable4", "new value")
	if err == nil {
		t.Error("Expected error for updating non-existent field, but got nil")
	}

	err = setVariableValue(config, "Name1.Name2[2].Variable1", 200)
	if err == nil {
		t.Error("Expected error for updating out of range index, but got nil")
	}
}

func TestSetVariableValue_TypeMismatch(t *testing.T) {
	config := &Config{
		Name1: Name1Config{
			Name2: []Name2Config{
				{Variable1: 10, Variable2: 3.14, Variable3: "hello"},
				{Variable1: 20, Variable2: 6.28, Variable3: "world"},
			},
		},
	}

	err := setVariableValue(config, "Name1.Name2[0].Variable1", "string value")
	if err == nil {
		t.Error("Expected error for type mismatch, but got nil")
	}
}

func TestSetVariableValue_EmptySlice(t *testing.T) {
	config := &Config{
		Name1: Name1Config{},
	}

	err := setVariableValue(config, "Name1.Name2[0].Variable1", 100)
	if err == nil {
		t.Error("Expected error for updating empty slice, but got nil")
	}
}

func TestEdgeCoreConfig(t *testing.T) {
	cfg := v1alpha2.NewDefaultEdgeCoreConfig()
	if err := ParseSet(cfg, `database.AliasName=test,database.driverName=mysql,modules.dbTest.enable=true,Modules.Edged.TailoredKubeletFlag.HostnameOverride=hy,Modules.MetaManager.MetaServer.ServiceAccountIssuers={ht,jl},featureGates={"alpha":true,"ht":false}`); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg.DataBase.AliasName, cfg.DataBase.DriverName, cfg.Modules.DBTest.Enable, cfg.Modules.Edged.TailoredKubeletFlag.HostnameOverride, cfg.Modules.MetaManager.MetaServer.ServiceAccountIssuers, cfg.FeatureGates)
}
