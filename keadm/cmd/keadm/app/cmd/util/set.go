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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// ParseSetByCommma splits a set line according to the comma.
//
// A set line is of the form "hello, world" or {a, b, c}
func parseSetByComma(set string) []string {
	var vals []string
	var buffer strings.Builder
	var inQuotes, inBraces bool

	for _, char := range set {
		switch {
		case char == ',' && !inBraces && !inQuotes:
			val := buffer.String()
			if val != "" {
				vals = append(vals, val)
			}
			buffer.Reset()
		case char == '{' && !inQuotes:
			inBraces = true
			buffer.WriteRune(char)
		case char == '}' && !inQuotes:
			inBraces = false
			buffer.WriteRune(char)
		case char == '"':
			inQuotes = !inQuotes
			buffer.WriteRune(char)
		case !unicode.IsSpace(char) || inQuotes || inBraces:
			buffer.WriteRune(char)
		}
	}
	if buffer.Len() > 0 {
		vals = append(vals, buffer.String())
	}

	return vals
}

// ParseSetByEqual splits a set line according to the equal.
//
// A set line is of the form ["name1=value1","name2=value2"]
func parseSetByEqual(set []string) ([]string, []string) {
	var names []string
	var vals []string
	for _, s := range set {
		parts := strings.Split(s, "=")
		if len(parts) != 2 {
			continue
		}
		names = append(names, parts[0])
		vals = append(vals, parts[1])
	}
	return names, vals
}

// ParseSetValue parses the value in the split set line.
// The type of value must be interpreted by int, float64, string and array.
func parseSetValue(vals []string) []interface{} {
	parsedvals := make([]interface{}, len(vals))
	for i, s := range vals {
		parsedvals[i] = parseValue(s)
	}
	return parsedvals
}

// ParseValue parses the value and interprets it to int, float64, string and array.
// The representation of {value} will be interpreted by array.
func parseValue(s string) interface{} {
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		if strings.Contains(s, ":") {
			return parseMap(s)
		}
		return parseArray(s)
	}

	if intValue, err := strconv.Atoi(s); err == nil {
		return intValue
	}
	if floatValue, err := strconv.ParseFloat(s, 64); err == nil {
		return floatValue
	}
	if boolvalue, err := strconv.ParseBool(s); err == nil {
		return boolvalue
	}
	return trimStringVal(s)
}

// ParseMap parses the value of map.
func parseMap(s string) interface{} {
	s = s[1 : len(s)-1]
	keyValuePairs := strings.Split(s, ",")
	keyValue := strings.Split(keyValuePairs[0], ":")
	myMap := makeMap(reflect.TypeOf(parseValue(keyValue[0])), reflect.TypeOf(parseValue(keyValue[1])))
	for _, pair := range keyValuePairs {
		keyValue := strings.Split(pair, ":")
		parseKey := parseValue(keyValue[0])
		parseVal := parseValue(keyValue[1])
		setValue(myMap, parseKey, parseVal)
	}
	return myMap
}

// use reflect build map
func makeMap(keyType, valueType reflect.Type) interface{} {
	mapType := reflect.MapOf(keyType, valueType)
	return reflect.MakeMap(mapType).Interface()
}

// append <k,v> into map
func setValue(m interface{}, key interface{}, value interface{}) {
	v := reflect.ValueOf(m)
	k := reflect.ValueOf(key)
	vv := reflect.ValueOf(value)
	v.SetMapIndex(k, vv)
}

// ParseArray parses the value of array.
func parseArray(s string) interface{} {
	s = s[1 : len(s)-1]
	vals := strings.Split(s, ",")
	switch parseType(vals[0]) {
	case "int":
		intArray := make([]int, len(vals))
		for i, v := range vals {
			v = strings.TrimSpace(v)
			intValue, _ := strconv.Atoi(v)
			intArray[i] = intValue
		}
		return intArray
	case "float":
		floatArray := make([]float64, len(vals))
		for i, v := range vals {
			v = strings.TrimSpace(v)
			floatValue, _ := strconv.ParseFloat(v, 64)
			floatArray[i] = floatValue
		}
		return floatArray
	default:
		stringArray := make([]string, len(vals))
		for i, v := range vals {
			stringArray[i] = trimStringVal(v)
		}
		return stringArray
	}
}

func trimStringVal(s string) string {
	// trim space
	s = strings.TrimSpace(s)
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) || (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		// trim paired single or double quotation marks
		s = s[1 : len(s)-1]
	}
	return s
}

// ParseType parses the type of array and interprets it to int, float64, string.
func parseType(s string) string {
	// Check if it's an integer
	if _, err := strconv.Atoi(s); err == nil {
		return "int"
	}
	// Check if it's a float
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return "float"
	}
	// Check if it's a bool
	if _, err := strconv.ParseBool(s); err == nil {
		return "bool"
	}
	// Otherwise, it's a string
	return "string"
}

// GetNameFormStatus judges the types of names.
// The name has three forms:
//
//	1)name1 = value1, return 0
//	2)name1[0].variable1 = value1, return 1
//	3)name1[0] = value1, return 2
func getNameFormStatus(s string) int {
	if !strings.Contains(s, "[") && !strings.Contains(s, "]") {
		return 0
	}
	closeBracketIndex := strings.Index(s, "]")
	if closeBracketIndex != (len(s) - 1) {
		return 1
	}
	if closeBracketIndex == (len(s) - 1) {
		return 2
	}
	return -1
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// SetCommonValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string, splic.
// The name is represented by name1.name2.(...).nameM.
func setCommonValue(structPtr interface{}, fieldPath string, value interface{}) error {
	structVal := reflect.ValueOf(structPtr).Elem()

	path := strings.Split(fieldPath, ".")
	var fieldVal reflect.Value
	fieldVal = structVal

	// Traverse all but the last path element
	for i := 0; i < len(path)-1; i++ {
		p := path[i]
		fieldVal = getFieldValue(fieldVal, p)
		if !fieldVal.IsValid() {
			return fmt.Errorf("%s: No such field: %s in Config", fieldPath, p)
		}
	}

	// Handle the last path element
	lastPath := path[len(path)-1]

	// Check if the current fieldVal is a map
	if fieldVal.Kind() == reflect.Map {
		return setMapValue(fieldVal, lastPath, value)
	}

	// For non-map types, get the final field
	fieldVal = getFieldValue(fieldVal, lastPath)
	if !fieldVal.IsValid() {
		return fmt.Errorf("%s: No such field: %s in Config", fieldPath, lastPath)
	}

	// Set new value
	val := reflect.ValueOf(value)
	convertedValue, err := convertTargetValue(fieldVal.Type(), val)
	if err != nil {
		return fmt.Errorf("%s: Convert to target value failed, err: %s", fieldPath, err)
	}
	fieldVal.Set(convertedValue)

	return nil
}

// setMapValue sets a value in a map using the provided key
func setMapValue(mapVal reflect.Value, key string, value interface{}) error {
	if mapVal.Kind() != reflect.Map {
		return fmt.Errorf("target is not a map")
	}

	// If map is nil, initialize it
	if mapVal.IsNil() {
		mapVal.Set(reflect.MakeMap(mapVal.Type()))
	}

	// Convert the key string to the map's key type
	keyType := mapVal.Type().Key()
	mapKey := reflect.ValueOf(key)
	if mapKey.Type() != keyType {
		// For string keys, no conversion needed
		if keyType.Kind() != reflect.String {
			return fmt.Errorf("unsupported map key type: %v", keyType)
		}
	}

	// Convert the value to the map's value type
	elemType := mapVal.Type().Elem()
	valueRef := reflect.ValueOf(value)
	convertedValue, err := convertTargetValue(elemType, valueRef)
	if err != nil {
		return fmt.Errorf("failed to convert map value: %w", err)
	}

	mapVal.SetMapIndex(mapKey, convertedValue)
	return nil
}

// GetFieldValue gets the fieldname of the struct.
func getFieldValue(v reflect.Value, fieldName string) reflect.Value {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	f := v.FieldByName(fieldName)
	if !f.IsValid() {
		return reflect.Value{}
	}
	return f
}

// ParseAndSetArrayValue combines the function of SetArrayValue and ParseFieldPath1.
func parseAndSetArrayValue(structPtr interface{}, fieldPath string, newValue interface{}) error {
	path, index, _ := parseFieldPath1(fieldPath)
	err := setArrayValue(structPtr, path, index, newValue)
	return err
}

// SetArrayValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string.
// The name is represented by name1.name2.(...).nameM[N].
func setArrayValue(structPtr interface{}, fieldPath string, index int, newValue interface{}) error {
	v := reflect.ValueOf(structPtr).Elem()

	pathParts := strings.Split(fieldPath, ".")
	var fieldVal reflect.Value
	fieldVal = v
	for _, part := range pathParts {
		fieldVal = getFieldValue(fieldVal, part)
		if !fieldVal.IsValid() {
			return fmt.Errorf("%s: No such field: %s in Config", fieldPath, part)
		}
	}
	if fieldVal.Kind() != reflect.Array && fieldVal.Kind() != reflect.Slice && fieldVal.Kind() != reflect.Map {
		return fmt.Errorf("%s is not an array,slice and map", pathParts[len(pathParts)-1])
	}

	if fieldVal.Kind() == reflect.Array {
		// If it is an array, you need to check whether the index is within the range
		if index < 0 || index >= fieldVal.Len() {
			return fmt.Errorf("index out of range for %s", pathParts[len(pathParts)-1])
		}
	} else if fieldVal.Kind() == reflect.Slice {
		if index < 0 {
			return fmt.Errorf("index out of range for %s", pathParts[len(pathParts)-1])
		}
		// If it is a slice, you need to ensure its capacity
		ensureSliceCapacity