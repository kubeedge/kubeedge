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

type JobState string

// Constants for job state
const (
	JobStateInit       JobState = "Init"
	JobStateInProgress JobState = "InProgress"
	JobStateComplated  JobState = "Complated"
	JobStateFailure    JobState = "Failure"
)

// IsFinal returns whether the node task is in the final state.
func (s JobState) IsFinal() bool {
	return s == JobStateComplated || s == JobStateFailure
}

type NodeTaskStatus string

// Constants for node task status.
const (
	NodeTaskStatusPending    NodeTaskStatus = "Pending"
	NodeTaskStatusInProgress NodeTaskStatus = "InProgress"
	NodeTaskStatusSuccessful NodeTaskStatus = "Successful"
	NodeTaskStatusFailure    NodeTaskStatus = "Failure"
	NodeTaskStatusUnknown    NodeTaskStatus = "Unknown"
)

// BasicNodeTaskStatus defines basic fields of node execution status.
// +kubebuilder:validation:Type=object
type BasicNodeTaskStatus struct {
	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`
	// Status represents for the status of the node task.
	Status NodeTaskStatus `json:"status,omitempty"`
	// Reason represents for the reason of the node task.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Time represents for the running time of the node task.
	Time string `json:"time,omitempty"`
}

// Constants for node job check items.
const (
	CheckItemCPU  string = "cpu"
	CheckItemMem  string = "mem"
	CheckItemDisk string = "disk"
)
