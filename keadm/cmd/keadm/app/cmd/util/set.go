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
	"strconv"
	"strings"
	"unicode"
)
// ParseSetByCommma splits a set line according to the comma.
//
// A set line is of the form "hello, world" or {a, b, c}
func ParseSetByComma(set string) []string {
	var vals []string
	var buffer strings.Builder
	var inQuotes, inBraces bool

	for _, char := range set {
		switch {
		case char == ',' && !inBraces && !inQuotes:
			val := buffer.String()
			if val != ""{
				vals = append(vals,val)
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
func ParseSetByEqual(set []string) ([]string, []string) {
	var names []string
	var vals []string
	if len(set) == 0{
		return names, vals
	}
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
func ParseSetValue(vals []string) []interface{} {
	parsedvals := make([]interface{}, len(vals))
	for i, s := range vals {
		parsedvals[i] = ParseValue(s)
	}
	return parsedvals
}

// ParseValue parses the value and interprets it to int, float64, string and array.
// The representation of {value} will be interpreted by array.
func ParseValue(s string) interface{} {
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		return ParseArray(s)
	}
	if intValue, err := strconv.Atoi(s); err == nil {
		return intValue
	}
	if floatValue, err := strconv.ParseFloat(s, 64); err == nil {
		return floatValue
	}
	if boolvalue, err :=strconv.ParseBool(s);err == nil{
		return boolvalue
	}
	return s
}

// ParseArray parses the value of array.
func ParseArray(s string) interface{} {
	if s == "{}" {
		return make([]interface{}, 0)
	}
	s = s[1 : len(s)-1]
	vals := strings.Split(s, ",")
	switch ParseType(vals[0]) {
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
func ParseType(s string) string {
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
func GetNameFormStatus(s string) int {
	if strings.Contains(s, "[") && strings.Contains(s, "]") {
		if strings.Contains(s, ".") {
			return 1 // name1[0].variable1 = value1
		}
		return 2 // name1[0] = value1
	}
	return 0 // name1 = value1
}

// SetCommonValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string, splic.
// The name is represented by name1.name2.(...).nameM.
func SetCommonValue(structPtr interface{}, fieldPath string, value interface{}) error {
	structVal := reflect.ValueOf(structPtr).Elem()

	path := strings.Split(fieldPath, ".")
	var fieldVal reflect.Value = structVal
	for _, p := range path {
		fieldVal = GetFieldValue(fieldVal, p)
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
func GetFieldValue(v reflect.Value, fieldName string) reflect.Value {
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
func ParseAndSetArrayValue(structPtr interface{}, fieldPath string, newValue interface{}) error {
	path, index, _ := ParseFieldPath1(fieldPath)
	// fmt.Println(path,index)
	err :=SetArrayValue(structPtr, path, index, newValue)
	// fmt.Println(structPtr)
	return err
}

// SetArrayValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string.
// The name is represented by name1.name2.(...).nameM[N].
func SetArrayValue(structPtr interface{}, fieldPath string, index int, newValue interface{}) error {
	v := reflect.ValueOf(structPtr).Elem()

	pathParts := strings.Split(fieldPath, ".")
	var fieldVal reflect.Value = v
	for _, part := range pathParts {
		fieldVal = fieldVal.FieldByName(part)
		if !fieldVal.IsValid() {
			return fmt.Errorf("%s: No such field: %s in Config", fieldPath, part)
		}
	}

	if fieldVal.Kind() != reflect.Array {
		return fmt.Errorf("%s is not an array", pathParts[len(pathParts)-1])
	}

	if index < 0 || index >= fieldVal.Len() {
		return fmt.Errorf("index out of range for %s", pathParts[len(pathParts)-1])
	}

	elem := reflect.ValueOf(newValue)
	if elem.Type() != fieldVal.Type().Elem() {
		return fmt.Errorf("type mismatch for field %s", pathParts[len(pathParts)-1])
	}

	fieldVal.Index(index).Set(elem)
	return nil
}

// ParseFieldPath1 parses the names in form of "name1.name2.(...).nameM[N]".
// path : name1.name2.(...).nameM, index : N
func ParseFieldPath1(fieldPath string) (string, int, error) {
	parts := strings.Split(fieldPath, "[")
	if len(parts) != 2 {
		return "", -1, fmt.Errorf("invalid field path")
	}

	path := parts[0]
	indexStr := strings.TrimSuffix(parts[1], "]")

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return "", -1, fmt.Errorf("invalid index")
	}

	return path, index, fmt.Errorf("")
}


// SetVariableValue modifies the new value of the name in the config represented by struct.
// The type of new value may be int, float, string.
// The name is represented by name1.name2.(...).nameM[N].variable1.
// func SetVariableValue(structPtr interface{}, fieldPath string, index int, variable string, newValue interface{}) error {
// 	pathParts := strings.Split(fieldPath, ".")
// 	val := reflect.ValueOf(structPtr)
// 	val = val.Elem()

// 	for _, part := range pathParts {
// 		if val.Kind() != reflect.Struct {
// 			return fmt.Errorf("path %s does not point to a struct", fieldPath)
// 		}
// 		field := val.FieldByName(part)
// 		if !field.IsValid() {
// 			return fmt.Errorf("field %s not found", part)
// 		}

// 		if field.Kind() == reflect.Ptr {
// 			if field.IsNil() {
// 				field.Set(reflect.New(field.Type().Elem()))
// 			}
// 			field = field.Elem()
// 		}

// 		val = field
// 		if val.Kind() != reflect.Struct {
//             return fmt.Errorf("path %s does not point to a struct", fieldPath)
//         }
// 	}

// 	if val.Kind() == reflect.Slice {
// 		if index < 0 || index >= val.Len() {
// 			return fmt.Errorf("index %d out of range for slice %s", index, fieldPath)
// 		}
// 		val = val.Index(index)
// 	}

// 	field := val.FieldByName(variable)
// 	if !field.IsValid() {
// 		return fmt.Errorf("field %s not found in struct", variable)
// 	}

// 	if !reflect.TypeOf(newValue).ConvertibleTo(field.Type()) {
// 		return fmt.Errorf("type mismatch, expected %s but got %s", field.Type(), reflect.TypeOf(newValue))
// 	}

// 	newVal := reflect.ValueOf(newValue).Convert(field.Type())
// 	field.Set(newVal)

// 	return nil
// }
func SetVariableValue(config interface{},name string,newValue interface{}) error{
	// Use reflection to get the value of config
	val := reflect.ValueOf(config)
	// Make sure config is a pointer type
	if val.Kind() !=reflect.Ptr{
		return fmt.Errorf("%s must be a pointer",config)
	}
	val = val.Elem()
	nameParts :=strings.Split(name,".")
	for i,part :=range(nameParts){
		if val.Kind() != reflect.Struct{
			return fmt.Errorf("path %s doen not point to a struct",name)
		}
		// Get the field corresponding to name
		field := val.FieldByName(part)
		fmt.Println(part)
		if !field.IsValid(){
			return fmt.Errorf("field %s not found",part)
		}

		if field.Kind() == reflect.Ptr && field.IsNil(){
			field.Set(reflect.New(field.Type().Elem()))
		}

		if i==len(nameParts)-1 && field.Kind() ==reflect.Slice{
			sliceLen := field.Len()
			if sliceLen == 0{
				return fmt.Errorf("slice %s is empty",part)
			}
			// Get index
			indexStr :=strings.TrimSuffix(strings.TrimPrefix(part,"["),"]")
			index,err := strconv.Atoi(indexStr)
			if err != nil {
                return fmt.Errorf("invalid index for slice %s", part)
            }
			// Make sure the index is within range
			if index < 0 || index >= sliceLen {
                return fmt.Errorf("index %d out of range for slice %s", index, part)
            }

			sliceVal := field.Index(index)
			if !reflect.TypeOf(newValue).ConvertibleTo(sliceVal.Type()){
				return fmt.Errorf("type mismatch, expected %s but got %s", sliceVal.Type(), reflect.TypeOf(newValue))
			}
			newVal := reflect.ValueOf(newValue).Convert(sliceVal.Type())
			sliceVal.Set(newVal)
			return nil
		}
		if field.Kind()==reflect.Slice{
			if field.Len() ==0 {
				return fmt.Errorf("slice %s is empty", part)
			}
			field = field.Index(0)
		}
		val = field
	}

	if val.Kind() == reflect.Slice {
        return fmt.Errorf("path %s cannot end with a slice", name)
    }
	if !reflect.TypeOf(newValue).ConvertibleTo(val.Type()) {
        return fmt.Errorf("type mismatch, expected %s but got %s", val.Type(), reflect.TypeOf(newValue))
    }

	//Set new value
	newVal := reflect.ValueOf(newValue).Convert(val.Type())
    val.Set(newVal)

    return nil
}


