/*
Copyright 2023 The KubeEdge Authors.

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
	metav1 "k8s.io/api/core/v1"

	"github.com/kubeedge/kubeedge/common/constants"
)

// IsEdgeNode checks whether a node is an Edge Node
// only if label {"node-role.kubernetes.io/edge": ""} exists, it is an edge node
func IsEdgeNode(node *metav1.Node) bool {
	if node.Labels == nil {
		return false
	}
	if _, ok := node.Labels[constants.EdgeNodeRoleKey]; !ok {
		return false
	}

	if node.Labels[constants.EdgeNodeRoleKey] != constants.EdgeNodeRoleValue {
		return false
	}

	return true
}

// RemoveDuplicateElement deduplicate
func RemoveDuplicateElement[T any](s []T) []T {
	result := make([]T, 0, len(s))
	temp := make(map[any]struct{}, len(s))

	for _, item := range s {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}
