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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Upgrade stores the state of the Upgrade, nonNamespaced object that was used to upgrade edge node from cloud side.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type Upgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of Upgrade.
	// +optional
	Spec UpgradeSpec `json:"spec,omitempty"`
	// Most recently observed status of the Upgrade.
	// +optional
	Status []UpgradeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UpgradeList is a list of Upgrade.
type UpgradeList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of Upgrades.
	Items []Upgrade `json:"items"`
}

// UpgradeSpec is a description of Upgrade.
type UpgradeSpec struct {
	// +Required: Version is the EdgeCore version to upgrade.
	Version string `json:"version,omitempty"`
	// UpgradeInstaller is the cmd to install the latest upgrader
	// If it is empty, will install the required version keadm from docker images.
	// +optional
	UpgradeInstaller string `json:"upgradeInstaller,omitempty"`
	// UpgradeCmd is the cmd to upgrade the edge node to the required version.
	// If it is empty, will use keadm to do upgrade operation.
	// +optional
	UpgradeCmd string `json:"upgradeCmd,omitempty"`
	// NodeNames is a request to select some specific nodes. If it is non-empty,
	// the Upgrade simply select these edge nodes to do upgrade operation
	// +optional
	NodeNames []string `json:"nodeNames,omitempty"`
	// LabelSelector is a selector which must be true for the Upgrade to fit on a node.
	// Selector which must match a node's labels for the Selector to be operated on that node.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// UpgradeOperationStatus describe the result status of upgrade operation on edge nodes.
// +kubebuilder:validation:Enum=upgrading;upgrade_success;upgrade_failed_rollback_success;upgrade_failed_rollback_failed
type UpgradeOperationStatus string

// upgrade operation status
const (
	UpgradeUpgrading             UpgradeOperationStatus = "upgrading"
	UpgradeSuccess               UpgradeOperationStatus = "upgrade_success"
	UpgradeFailedRollbackSuccess UpgradeOperationStatus = "upgrade_failed_rollback_success"
	UpgradeFailedRollbackFailed  UpgradeOperationStatus = "upgrade_failed_rollback_failed"
)

const MaxStatusHistory = 20

// UpgradeStatus stores the status of Upgrade.
// contains multi nodes status
// key is edge node name, value is []Status
// +kubebuilder:validation:Type=object
type UpgradeStatus struct {
	NodeName string    `json:"nodeName,omitempty"`
	History  []History `json:"history,omitempty"`
}

type History struct {
	// FromVersion is the version Upgrade from
	FromVersion string `json:"fromVersion,omitempty"`
	// ToVersion is the version Upgrade to
	ToVersion string `json:"toVersion,omitempty"`
	// Status is the status of Upgrade..
	Status UpgradeOperationStatus `json:"operationStatus,omitempty"`
	// Reason is the error reason of Upgrade failure.
	Reason string `json:"reason,omitempty"`
}
