/*
Copyright 2022 The KubeEdge Authors.

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

package controller

import (
	"strings"
)

// filterVersion returns true only if the edge node version already on the upgrade req
// version is like: v1.22.6-kubeedge-v1.10.0-beta.0.185+95378fb019912a, expected is like v1.10.0
func filterVersion(version string, expected string) bool {
	// if not correct version format, also return true
	index := strings.Index(version, "-kubeedge-")
	if index == -1 {
		return false
	}

	if version[index+10:] != expected {
		// filter nodes that already in the required version
		return false
	}

	return true
}

func RemoveDuplicateElement(s []string) []string {
	result := make([]string, 0, len(s))
	temp := map[string]struct{}{}

	for _, item := range s {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}
