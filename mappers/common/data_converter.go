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

package mappercommon

import (
	"errors"
	"strconv"
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
