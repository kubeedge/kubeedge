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

import "reflect"

// RemoveDuplicateElement deduplicate
func RemoveDuplicateElement[T any](slice []T) []T {
	result := make([]T, 0, len(slice))
	temp := make(map[any]struct{}, len(slice))

	for _, item := range slice {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func In[T any](slice []T, item T) bool {
	for i := range slice {
		if reflect.DeepEqual(slice[i], item) {
			return true
		}
	}
	return false
}
