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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeGroupSpec defines the desired state of NodeGroup
type NodeGroupSpec struct {
	// Nodes contains names of all the nodes in the nodegroup.
	// +optional
	Nodes []string `json:"nodes,omitempty"`

	// MatchLabels are used to select nodes that have these labels.
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// NodeGroupStatus contains the observed status of all selected nodes in
// this NodeGroup, including nodes that have been one of the members of this NodeGroup
// and those have not.
type NodeGroupStatus struct {
	// NodeStatuses is a status list of all selected nodes.
	// +optional
	NodeStatuses []NodeStatus `json:"nodeStatuses,omitempty"`
}

// NodeStatus contains status of node that selected by this NodeGroup.
type NodeStatus struct {
	// NodeName contains name of this node.
	// +required
	NodeName string `json:"nodeName"`
	// ReadyStatus contains ready status of this node.
	// +required
	ReadyStatus ReadyStatus `json:"readyStatus"`
	// SelectionStatus contains status of the selection result for this node.
	// +required
	SelectionStatus SelectionStatus `json:"selectionStatus"`
	// SelectionStatusReason contains human-readable reason for this SelectionStatus.
	// +optional
	SelectionStatusReason string `json:"selectionStatusReason,omitempty"`
}

// HealthyStatus represents the healthy status of node.
type ReadyStatus string

const (
	// NodeReady indicates that this node is ready.
	NodeReady ReadyStatus = "Ready"

	// NodeNotReady indicates that this node is not ready.
	NodeNotReady ReadyStatus = "NotReady"

	// Unknown indicates that the status of this node is unknown.
	Unknown ReadyStatus = "Unknown"
)

// SelectionStatus represents the status of selecting a node as a member of this NodeGroup.
type SelectionStatus string

const (
	// SucceededSelection represents that this node has been selected as a member of this NodeGroup.
	SucceededSelection SelectionStatus = "Succeeded"
	// FailedSelection represents that this node failed to become a member of this NodeGroup.
	FailedSelection SelectionStatus = "Failed"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=ng

// NodeGroup is the Schema for the nodegroups API
type NodeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the specification of the desired behavior of member nodegroup.
	// +required
	Spec NodeGroupSpec `json:"spec,omitempty"`

	// Status represents the status of member nodegroup.
	// +optional
	Status NodeGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeGroupList contains a list of NodeGroup
type NodeGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeGroup `json:"items"`
}
