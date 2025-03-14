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

package nodes

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/constants"
)

// IsEdgeNode checks whether a node is an Edge Node
// only if label {"node-role.kubernetes.io/edge": ""} exists, it is an edge node
func IsEdgeNode(node *corev1.Node) bool {
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

// IsReadyNode checks whether the node is ready.
func IsReadyNode(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady &&
			condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// GetManagedEdgeNodes returns the managed edge nodes through the edge node session
// (NodeSessions field of session.Manager) that connected to the current CloudCore instance.
func GetManagedEdgeNodes(nodeSessions *sync.Map) []string {
	nodes := make([]string, 0)
	nodeSessions.Range(func(key, _ any) bool {
		nodeID, ok := key.(string)
		if ok {
			nodes = append(nodes, nodeID)
		} else {
			klog.Warningf("invalid node session key %v, expected string type, actual type is %T", key, key)
		}
		return true
	})
	return nodes
}
