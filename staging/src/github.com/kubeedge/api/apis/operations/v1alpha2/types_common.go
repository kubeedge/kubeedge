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

type JobState string

const (
	JobStateInit       JobState = "Init"
	JobStateInProgress JobState = "InProgress"
	JobStateComplated  JobState = "Complated"
	JobStateFailure    JobState = "Failure"
)

type NodeExecutionState string

const (
	NodeExecutionStateInProgress NodeExecutionState = "InProgress"
	NodeExecutionStateSuccessful NodeExecutionState = "Successful"
	NodeExecutionStateFailure    NodeExecutionState = "Failure"
)

// BasicNodeTaskStatus defines basic fields of node execution status.
// +kubebuilder:validation:Type=object
type BasicNodeTaskStatus struct {
	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`
	// Status represents for the status of the NodeTask.
	Status metav1.ConditionStatus `json:"status,omitempty"`
	// Reason represents for the reason of the NodeTask.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Time represents for the running time of the NodeTask.
	Time string `json:"time,omitempty"`
}
