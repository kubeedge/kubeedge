/*
Copyright 2020 The KubeEdge Authors.

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

package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Convert string to other types
func Convert(valueType string, value string) (result interface{}, err error) {
	switch valueType {
	case "int":
		return strconv.ParseInt(value, 10, 64)
	case "float":
		return strconv.ParseFloat(value, 32)
	case "double":
		return strconv.ParseFloat(value, 64)
	case "boolean":
		return strconv.ParseBool(value)
	case "string":
		return value, nil
	default:
		return nil, errors.New("Convert failed")
	}
}

// ConvertToString other types to string
func ConvertToString(value interface{}) (string, error) {
	var result string
	if value == nil {
		return result, nil
	}
	switch v := value.(type) {
	case float64:
		ft := v
		result = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := v
		result = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := v
		result = strconv.Itoa(it)
	case uint:
		it := v
		result = strconv.Itoa(int(it))
	case int8:
		it := v
		result = strconv.Itoa(int(it))
	case uint8:
		it := v
		result = strconv.Itoa(int(it))
	case int16:
		it := v
		result = strconv.Itoa(int(it))
	case uint16:
		it := v
		result = strconv.Itoa(int(it))
	case int32:
		it := v
		result = strconv.Itoa(int(it))
	case uint32:
		it := v
		result = strconv.Itoa(int(it))
	case int64:
		it := v
		result = strconv.FormatInt(it, 10)
	case uint64:
		it := v
		result = strconv.FormatUint(it, 10)
	case string:
		result = v
	case []byte:
		result = string(v)
	default:
		newValue, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		result = string(newValue)
	}
	return result, nil
}

// DecodeAnyValue Any to interface
func DecodeAnyValue(value *anypb.Any) (interface{}, error) {
	typeURL := value.GetTypeUrl()

	messageTypeName := getMessageTypeName(typeURL)
	if messageTypeName == "" {
		return nil, fmt.Errorf("cant get message type：%s", typeURL)
	}
	if strings.Contains(messageTypeName, "google.protobuf.") {
		switch messageTypeName {
		case "google.protobuf.Int32Value":
			return decodeWrapperValue(value, &wrapperspb.Int32Value{})
		case "google.protobuf.StringValue":
			return decodeWrapperValue(value, &wrapperspb.StringValue{})
		case "google.protobuf.FloatValue":
			return decodeWrapperValue(value, &wrapperspb.FloatValue{})
		case "google.protobuf.BoolValue":
			return decodeWrapperValue(value, &wrapperspb.BoolValue{})
		case "google.protobuf.Int64Value":
			return decodeWrapperValue(value, &wrapperspb.Int64Value{})
		default:
			return nil, fmt.Errorf("unknown type : %s", messageTypeName)
		}
	}
	messageType := proto.MessageType(messageTypeName)
	if messageType == nil {
		return nil, fmt.Errorf("cant get message type：%s", messageTypeName)
	}

	if !reflect.TypeOf((*proto.Message)(nil)).Elem().AssignableTo(messageType) {
		return nil, fmt.Errorf("assiganbleto proto.Message error：%s", messageTypeName)
	}
	message := reflect.New(messageType.Elem()).Interface().(proto.Message)
	if err := proto.Unmarshal(value.Value, message); err != nil {
		return nil, fmt.Errorf("unmarshal value error：%v", err)
	}
	return message, nil
}

// decodeWrapperValue get proto.Message, then convert to interface
func decodeWrapperValue(value *anypb.Any, wrapper proto.Message) (interface{}, error) {
	if err := proto.Unmarshal(value.Value, wrapper); err != nil {
		return nil, fmt.Errorf("decode wrapperValue,proto unmarshal error：%v", err)
	}
	wrapperValue := reflect.ValueOf(wrapper).Elem()
	valueField := wrapperValue.FieldByName("Value")
	if !valueField.IsValid() {
		return nil, fmt.Errorf("cant get wrapperValue")
	}
	return valueField.Interface(), nil
}

// getMessageTypeName get type by parse type url
func getMessageTypeName(typeURL string) string {
	index := len(typeURL) - 1
	for index >= 0 && typeURL[index] != '/' {
		index--
	}
	if index >= 0 && index < len(typeURL)-1 {
		return typeURL[index+1:]
	}
	return ""
}
