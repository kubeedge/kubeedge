/*
Copyright 2026 The KubeEdge Authors.

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

package nodeselect

import (
	"fmt"
	"strings"
)

// NodeSelector provides methods to select multiple nodes for batch operations.
// It supports selection by explicit node names or by label selectors.
type NodeSelector struct {
	nodes    []string
	selector map[string]string
}

// NewNodeSelector creates a new NodeSelector instance.
func NewNodeSelector() *NodeSelector {
	return &NodeSelector{
		nodes:    []string{},
		selector: make(map[string]string),
	}
}

// AddNodes adds specific node names to the selector.
// Multiple nodes can be provided as a comma-separated string.
// Example: "node1,node2,node3"
func (ns *NodeSelector) AddNodes(nodes string) {
	if nodes == "" {
		return
	}
	nodeList := strings.Split(nodes, ",")
	for _, node := range nodeList {
		trimmed := strings.TrimSpace(node)
		if trimmed != "" {
			ns.nodes = append(ns.nodes, trimmed)
		}
	}
}

// AddSelector adds a label selector key-value pair.
// Example: AddSelector("region", "us-west")
func (ns *NodeSelector) AddSelector(key, value string) {
	if key != "" && value != "" {
		ns.selector[key] = value
	}
}

// GetNodes returns the list of explicitly selected node names.
func (ns *NodeSelector) GetNodes() []string {
	return ns.nodes
}

// HasSelector returns true if label selectors are configured.
func (ns *NodeSelector) HasSelector() bool {
	return len(ns.selector) > 0
}

// GetSelector returns the label selector map.
func (ns *NodeSelector) GetSelector() map[string]string {
	return ns.selector
}

// Validate ensures that the node selection is valid.
// Returns an error if:
// - Neither nodes nor selector is specified
// - Both nodes and selector are specified (mutually exclusive)
func (ns *NodeSelector) Validate() error {
	hasNodes := len(ns.nodes) > 0
	hasSelector := len(ns.selector) > 0

	if !hasNodes && !hasSelector {
		return fmt.Errorf("must specify either --nodes or --selector")
	}
	if hasNodes && hasSelector {
		return fmt.Errorf("cannot specify both --nodes and --selector, they are mutually exclusive")
	}
	return nil
}

// Count returns the number of explicitly selected nodes.
// Returns 0 if using selector-based selection.
func (ns *NodeSelector) Count() int {
	return len(ns.nodes)
}

// String returns a human-readable representation of the selector.
func (ns *NodeSelector) String() string {
	if len(ns.nodes) > 0 {
		return fmt.Sprintf("nodes: %s", strings.Join(ns.nodes, ", "))
	}
	if len(ns.selector) > 0 {
		var parts []string
		for k, v := range ns.selector {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		return fmt.Sprintf("selector: %s", strings.Join(parts, ", "))
	}
	return "empty selector"
}