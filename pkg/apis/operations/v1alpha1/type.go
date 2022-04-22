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

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeUpgradeJob is used to upgrade edge node from cloud side.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type NodeUpgradeJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of NodeUpgradeJob.
	// +optional
	Spec NodeUpgradeJobSpec `json:"spec,omitempty"`
	// Most recently observed status of the NodeUpgradeJob.
	// +optional
	Status NodeUpgradeJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeUpgradeJobList is a list of NodeUpgradeJob.
type NodeUpgradeJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of NodeUpgradeJobs.
	Items []NodeUpgradeJob `json:"items"`
}

// NodeUpgradeJobSpec is the specification of the desired behavior of the NodeUpgradeJob.
type NodeUpgradeJobSpec struct {
	// +Required: Version is the EdgeCore version to upgrade.
	Version string `json:"version,omitempty"`
	// UpgradeTool is a request to decide use which upgrade tool. If it is empty,
	// the upgrade job simply use default upgrade tool keadm to do upgrade operation.
	// +optional
	UpgradeTool string `json:"upgradeTool,omitempty"`
	// TimeoutSeconds limits the duration of the node upgrade job.
	// Default to 300.
	// If set to 0, we'll use the default value 300.
	// +optional
	TimeoutSeconds *uint32 `json:"timeoutSeconds,omitempty"`
	// NodeNames is a request to select some specific nodes. If it is non-empty,
	// the upgrade job simply select these edge nodes to do upgrade operation.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	NodeNames []string `json:"nodeNames,omitempty"`
	// LabelSelector is a filter to select member clusters by labels.
	// It must match a node's labels for the NodeUpgradeJob to be operated on that node.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	// Image specifies a container image name, the image contains: keadm and edgecore.
	// keadm is used as upgradetool, to install the new version of edgecore.
	// The image name consists of registry hostname and repository name, but cannot includes the tag,
	// Version above will be used as the tag.
	// If the registry hostname is empty, docker.io will be used as default.
	// The default image name is: kubeedge/installation-package.
	// +optional
	Image string `json:"image,omitempty"`
}

// UpgradeResult describe the result status of upgrade operation on edge nodes.
// +kubebuilder:validation:Enum=upgrade_success;upgrade_failed_rollback_success;upgrade_failed_rollback_failed
type UpgradeResult string

// upgrade operation status
const (
	UpgradeSuccess               UpgradeResult = "upgrade_success"
	UpgradeFailedRollbackSuccess UpgradeResult = "upgrade_failed_rollback_success"
	UpgradeFailedRollbackFailed  UpgradeResult = "upgrade_failed_rollback_failed"
)

// UpgradeState describe the UpgradeState of upgrade operation on edge nodes.
// +kubebuilder:validation:Enum=upgrading;completed
type UpgradeState string

// Valid values of UpgradeState
const (
	InitialValue UpgradeState = ""
	Upgrading    UpgradeState = "upgrading"
	Completed    UpgradeState = "completed"
)

// NodeUpgradeJobStatus stores the status of NodeUpgradeJob.
// contains multiple edge nodes upgrade status.
// +kubebuilder:validation:Type=object
type NodeUpgradeJobStatus struct {
	// State represents for the state phase of the NodeUpgradeJob.
	// There are three possible state values: "", upgrading and completed.
	State UpgradeState `json:"state,omitempty"`
	// Status contains upgrade Status for each edge node.
	Status []UpgradeStatus `json:"status,omitempty"`
}

// UpgradeStatus stores the status of Upgrade for each edge node.
// +kubebuilder:validation:Type=object
type UpgradeStatus struct {
	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`
	// State represents for the upgrade state phase of the edge node.
	// There are three possible state values: "", upgrading and completed.
	State UpgradeState `json:"state,omitempty"`
	// History is the last upgrade result of the edge node.
	History History `json:"history,omitempty"`
}

// History stores the information about upgrade history record.
// +kubebuilder:validation:Type=object
type History struct {
	// HistoryID is to uniquely identify an Upgrade Operation.
	HistoryID string `json:"historyID,omitempty"`
	// FromVersion is the version which the edge node is upgraded from.
	FromVersion string `json:"fromVersion,omitempty"`
	// ToVersion is the version which the edge node is upgraded to.
	ToVersion string `json:"toVersion,omitempty"`
	// Result represents the result of upgrade.
	Result UpgradeResult `json:"result,omitempty"`
	// Reason is the error reason of Upgrade failure.
	// If the upgrade is successful, this reason is an empty string.
	Reason string `json:"reason,omitempty"`
	// UpgradeTime is the time of this Upgrade.
	UpgradeTime string `json:"upgradeTime,omitempty"`
}
