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

// ParseSetValue parses the value in the splited set line.
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
	return s
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
			stringArray[i] = strings.TrimSpace(v)
		}
		return stringArray
	}
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

// SetCommonValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string, splic.
// The name is represented by name1.name2.(...).nameM.
func setCommonValue(structPtr interface{}, fieldPath string, value interface{}) error {
	structVal := reflect.ValueOf(structPtr).Elem()

	path := strings.Split(fieldPath, ".")
	var fieldVal reflect.Value
	fieldVal = structVal
	for _, p := range path {
		fieldVal = getFieldValue(fieldVal, p)
		if !fieldVal.IsValid() {
			return fmt.Errorf("%s: No such field: %s in Config", fieldPath, p)
		}
	}

	val := reflect.ValueOf(value)

	if fieldVal.Type() != val.Type() {
		return fmt.Errorf("%s: Provided value type %s does not match field type %s", fieldPath, val.Type(), fieldVal.Type())
	}
	fieldVal.Set(val)

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
	// If it is an array or slice, you need to check whether the index is within the range
	if index < 0 || index >= fieldVal.Len() {
		return fmt.Errorf("index out of range for %s", pathParts[len(pathParts)-1])
	}

	//Extend array or slice length
	if index >= fieldVal.Len() {
		newSlice := reflect.MakeSlice(fieldVal.Type(), index+1, index+1)
		reflect.Copy(newSlice, fieldVal)
		fieldVal.Set(newSlice)
	}
	//Set new value
	elem := reflect.ValueOf(newValue)
	if elem.Type() != fieldVal.Type().Elem() {
		return fmt.Errorf("type mismatch for field %s", pathParts[len(pathParts)-1])
	}
	fieldVal.Index(index).Set(elem)
	return nil
}

// ParseFieldPath1 parses the names in form of "name1.name2.(...).nameM[N]".
// path : name1.name2.(...).nameM, index : N
func parseFieldPath1(fieldPath string) (string, int, error) {
	parts := strings.Split(fieldPath, "[")
	if len(parts) != 2 {
		return "", -1, errors.New("invalid field path")
	}

	path := parts[0]
	indexStr := strings.TrimSuffix(parts[1], "]")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", -1, fmt.Errorf("invalid index: %v", err)
	}

	return path, index, errors.New("")
}

// SetVariableValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string.
// The name is represented by name1.name2.(...).nameM[N].variable1.
func setVariableValue(obj interface{}, fieldPath string, value interface{}) error {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() != reflect.Ptr {
		return errors.New("obj must be a pointer")
	}
	objValue = objValue.Elem()
	fieldNames := parseFieldPath(fieldPath)

	for i, fieldName := range fieldNames[:len(fieldNames)-1] {
		switch objValue.Kind() {
		case reflect.Struct:
			objValue = objValue.FieldByName(fieldName)
		case reflect.Slice:
			index, err := getIndex(fieldName)
			if err != nil {
				return err
			}
			if index < 0 || index >= objValue.Len() {
				return fmt.Errorf("index out of range for field %s", fieldName)
			}
			objValue = objValue.Index(index)
		case reflect.Ptr:
			if objValue.IsNil() {
				objValue.Set(reflect.New(objValue.Type().Elem()))
			}
			objValue = objValue.Elem()
		default:
			return fmt.Errorf("field %s is neither a struct nor a slice nor a pointer", fieldName)
		}
		if objValue.Kind() == reflect.Ptr && objValue.Elem().Kind() == reflect.Struct {
			objValue = objValue.Elem()
		}
		if i == len(fieldNames)-2 && objValue.Kind() != reflect.Struct {
			return fmt.Errorf("field %s is not a struct", fieldNames[i])
		}
	}

	targetFieldName := fieldNames[len(fieldNames)-1]
	if objValue.Kind() == reflect.Ptr {
		if objValue.IsNil() {
			objValue.Set(reflect.New(objValue.Type().Elem()))
		}
		objValue = objValue.Elem()
	}
	targetFieldValue := objValue.FieldByName(targetFieldName)
	if !targetFieldValue.IsValid() {
		return fmt.Errorf("field %s not found", targetFieldName)
	}
	if !targetFieldValue.CanSet() {
		return fmt.Errorf("field %s cannot be set", targetFieldName)
	}

	valueToSet := reflect.ValueOf(value)
	if !valueToSet.Type().AssignableTo(targetFieldValue.Type()) {
		return fmt.Errorf("value type %s is not assignable to field %s type %s", valueToSet.Type(), targetFieldName, targetFieldValue.Type())
	}

	targetFieldValue.Set(valueToSet)

	return nil
}

// Split fieldPath by "." and "[" to separate fields and indexes
func parseFieldPath(fieldPath string) []string {
	fields := strings.FieldsFunc(fieldPath, func(r rune) bool {
		return r == '.' || r == '[' || r == ']'
	})

	// Remove empty fields and trim brackets from indexes
	var cleanedFields []string
	for _, f := range fields {
		if f != "" {
			cleanedFields = append(cleanedFields, strings.Trim(f, "[]"))
		}
	}

	return cleanedFields
}

func getIndex(field string) (int, error) {
	index, err := strconv.Atoi(field)
	if err != nil {
		return -1, fmt.Errorf("invalid index: %s", field)
	}
	return index, nil
}

func findFieldByTag(obj interface{}, k int, tagName []string, fieldNames []string) {
	if k == len(tagName) {
		return
	}
	v := reflect.ValueOf(obj)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if isFirstLetterUpper(tagName[k]) {
		fieldNames[k] = tagName[k]
		findFieldByTag(v.FieldByName(tagName[k]).Interface(), k+1, tagName, fieldNames)
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagValue := field.Tag.Get("json")
		if !isFirstLetterUpper(tagName[k]) && tagValue == "" && i >= t.NumField() {
			upperTagName := upperFirstLetter(tagName[k])
			fieldNames[k] = upperTagName
			findFieldByTag(v.FieldByName(upperTagName).Interface(), k+1, tagName, fieldNames)
		}
		tagVals := strings.Split(tagValue, ",")
		if tagVals[0] == tagName[k] {
			fieldNames[k] = field.Name
			findFieldByTag(v.Field(i).Interface(), k+1, tagName, fieldNames)
		}
	}
}

func upperFirstLetter(s string) string {
	if len(s) > 0 {
		firstChar := []rune(s)[0]
		upperFirstChar := unicode.ToUpper(firstChar)
		return string(upperFirstChar) + s[1:]
	}
	return ""
}

// Determine whether the first letter is capitalized
func isFirstLetterUpper(s string) bool {
	if s == "" {
		return false
	}
	r := rune(s[0])
	return unicode.IsUpper(r)
}

func parseTag(cfg interface{}, name string) string {
	names := strings.Split(name, ".")
	var parseName string
	if isFirstLetterUpper(names[0]) {
		parseName = name
	} else {
		fieldNames := make([]string, len(names))
		findFieldByTag(cfg, 0, names, fieldNames)
		for i, fieldName := range fieldNames {
			if i == len(fieldNames)-1 {
				parseName = parseName + fieldName
			} else {
				parseName = parseName + fieldName + "."
			}
		}
	}
	return parseName
}

func ParseSet(cfg interface{}, set string) error {
	sets := parseSetByComma(set)
	names, vals := parseSetByEqual(sets)
	parseVals := parseSetValue(vals)
	for i, name := range names {
		status := getNameFormStatus(name)
		parseTagName := parseTag(cfg, name)
		switch status {
		//name1.nam2=val
		case 0:
			if err := setCommonValue(cfg, parseTagName, parseVals[i]); err != nil {
				return err
			}
		//name1.nam[1].var=val
		case 1:
			if err := setVariableValue(cfg, parseTagName, parseVals[i]); err != nil {
				return err
			}
		//name[0]=val
		case 2:
			if err := parseAndSetArrayValue(cfg, parseTagName, parseVals[i]); err != nil {
				return err
			}
		default:
			return errors.New("The field is not support")
		}
	}
	return nil
}
