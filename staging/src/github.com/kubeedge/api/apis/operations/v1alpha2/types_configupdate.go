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

package v1alpha2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	ResourceConfigUpdateJob = "configupdatejob"

	FinalizerConfigUpdateJob = "kubeedge.io/configupdatejob-controller"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigUpdateJob is used to update edge configuration from cloud side.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
type ConfigUpdateJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of ConfigUpdateJob.
	// +optional
	Spec ConfigUpdateJobSpec `json:"spec,omitempty"`
	// Most recently observed status of the ConfigUpdateJob.
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
	// It must match a node's labels for the ConfigUpdateJob to be operated on that node.
	// Please note that sets of NodeNames and LabelSelector are ORed.
	// Users must set one and can only set one.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// TimeoutSeconds limits the duration of the config update job.
	// Default to 300.
	// If set to 0, we'll use the default value 300.
	// +optional
	TimeoutSeconds *uint32 `json:"timeoutSeconds,omitempty"`

	// Concurrency specifies the maximum number of concurrent that edge nodes associated with
	// each CloudCore instance can be updated at the same time.
	// The default Concurrency value is 1.
	// +optional
	Concurrency int32 `json:"concurrency,omitempty"`

	// UpdateFields specify certain fields in EdgeCore configuration to update
	// +required
	UpdateFields map[string]string `json:"updateFields,omitempty"`

	// FailureTolerate specifies the task tolerance failure ratio.
	// The default FailureTolerate value is 0.1.
	// +optional
	FailureTolerate string `json:"failureTolerate,omitempty"`
}

type ConfigUpdateJobAction string

const (
	ConfigUpdateJobActionCheck    ConfigUpdateJobAction = "Check"
	ConfigUpdateJobActionBackUp   ConfigUpdateJobAction = "BackUp"
	ConfigUpdateJobActionUpdate   ConfigUpdateJobAction = "Update"
	ConfigUpdateJobActionRollBack ConfigUpdateJobAction = "RollBack"
)

// ConfigUpdateJobStatus stores the status of ConfigUpdateJob.
// contains multiple edge nodes config update status.
// +kubebuilder:validation:Type=object
type ConfigUpdateJobStatus struct {
	// Phase represents for the phase of the ConfigUpdateJob
	Phase JobPhase `json:"phase"`

	// NodeStatus contains config update status for each edge node.
	NodeStatus []ConfigUpdateJobNodeTaskStatus `json:"nodeStatus,omitempty"`

	// Reason represents for the reason of the ConfigUpdateJob.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// ConfigUpdateJobNodeTaskStatus stores the status of config update for each edge node.
// +kubebuilder:validation:Type=object
type ConfigUpdateJobNodeTaskStatus struct {
	// ActionFlow represents for the results of executing the action flow.
	ActionFlow []ConfigUpdateJobActionStatus `json:"actionFlow,omitempty"`

	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`

	// Phase represents for the phase of the node task.
	Phase NodeTaskPhase `json:"phase,omitempty"`

	// Reason represents the reason for the failure of the node task.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// ConfigUpdateJobActionStatus defines the results of executing the action.
// +kubebuilder:validation:Type=object
type ConfigUpdateJobActionStatus struct {
	// Action represents for the action phase of the ConfigUpdateJob
	Action ConfigUpdateJobAction `json:"action,omitempty"`

	// State represents for the status of this image pull on the edge node.
	Status metav1.ConditionStatus `json:"status,omitempty"`

	// Reason represents the reason for the failure of the action.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Time represents for the running time of the node task.
	Time string `json:"time,omitempty"`
}
