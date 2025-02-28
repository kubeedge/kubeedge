/*
Copyright 2024 The KubeEdge Authors.

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

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigUpdateJob is used to update edge configuration from cloud side.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type ConfigUpdateJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the specification of the desired behavior of ConfigUpdateJob.
	// +required
	Spec ConfigUpdateJobSpec `json:"spec"`

	// Status represents the status of ConfigUpdateJob.
	// +optional
	Status ConfigUpdateJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigUpdateJobList is a list of ConfigUpdateJob.
type ConfigUpdateJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ConfigUpdateJob.
	Items []ConfigUpdateJob `json:"items"`
}

// ConfigUpdateJobSpec represents the specification of the desired behavior of ConfigUpdateJob.
type ConfigUpdateJobSpec struct {
	// NodeNames is a request to select some specific nodes. If it is non-empty,
	// the update job simply select these edge nodes to do config update operation.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	NodeNames []string `json:"nodeNames,omitempty"`

	// LabelSelector is a filter to select member clusters by labels.
	// It must match a node's labels for the ConfigUpdateJo to be operated on that node.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// TimeoutSeconds limits the duration of the edge config update job.
	// Default to 300.
	// If set to 0, we'll use the default value 300.
	// +optional
	TimeoutSeconds *uint32 `json:"timeoutSeconds,omitempty"`

	// Concurrency specifies the max number of edge nodes that can be updated at the same time.
	// The default Concurrency value is 1.
	// +optional
	Concurrency int32 `json:"concurrency,omitempty"`

	// UpdateFields specify certain fields in EdgeCore configurations to update.
	// +required
	UpdateFields map[string]string `json:"updateFields,omitempty"`

	// FailureTolerate specifies the task tolerance failure ratio.
	// The default FailureTolerate value is 0.1.
	// +optional
	FailureTolerate string `json:"failureTolerate,omitempty"`
}

// ConfigUpdateJobStatus stores the status of ConfigUpdateJob.
// Contains multiple edge nodes config udpate status.
// +kubebuilder:validation:Type=object
type ConfigUpdateJobStatus struct {
	// State represents for the state phase of the ConfigUpdateJob.
	// There are several possible state values: "", Updating, BackingUp and RollingBack.
	State api.State `json:"state,omitempty"`

	// Event represents for the event of the ConfigUpdateJob.
	// There are six possible event values: Init, BackUp, Update, TimeOut, Rollback.
	Event string `json:"event,omitempty"`
	// Action represents for the action of the ConfigUpdateJob.
	// There are two possible action values: Success, Failure.
	Action api.Action `json:"action,omitempty"`
	// Reason represents for the reason of the ConfigUpdateJob.
	Reason string `json:"reason,omitempty"`
	// Time represents for the running time of the ConfigUpdateJob.
	Time string `json:"time,omitempty"`
	// Status contains update Status for each edge node.
	Status []TaskStatus `json:"nodeStatus,omitempty"`
}
