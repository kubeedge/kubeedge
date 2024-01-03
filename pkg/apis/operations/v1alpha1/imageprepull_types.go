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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImagePrePullJob is used to prepull images on edge node.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
type ImagePrePullJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the specification of the desired behavior of ImagePrePullJob.
	// +required
	Spec ImagePrePullJobSpec `json:"spec"`

	// Status represents the status of ImagePrePullJob.
	// +optional
	Status ImagePrePullJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImagePrePullJobList is a list of ImagePrePullJob.
type ImagePrePullJobList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ImagePrePullJob.
	Items []ImagePrePullJob `json:"items"`
}

// ImagePrePullSpec represents the specification of the desired behavior of ImagePrePullJob.
type ImagePrePullJobSpec struct {
	// ImagePrepullTemplate represents original templates of imagePrePull
	ImagePrePullTemplate ImagePrePullTemplate `json:"imagePrePullTemplate,omitempty"`
}

// ImagePrePullTemplate represents original templates of imagePrePull
type ImagePrePullTemplate struct {
	// Images is the image list to be prepull
	Images []string `json:"images,omitempty"`

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

	// CheckItems specifies the items need to be checked before the task is executed.
	// The default CheckItems value is disk.
	// +optional
	CheckItems []string `json:"checkItems,omitempty"`

	// ImageSecret specifies the secret for image pull if private registry used.
	// Use {namespace}/{secretName} in format.
	// +optional
	ImageSecret string `json:"imageSecrets,omitempty"`

	// TimeoutSecondsOnEachNode limits the duration of the image prepull job on each edgenode.
	// Default to 360.
	// If set to 0, we'll use the default value 300.
	// +optional
	TimeoutSecondsOnEachNode *uint32 `json:"timeoutSecondsOnEachNode,omitempty"`

	// RetryTimesOnEachNode specifies the retry times if image pull failed on each edgenode.
	// Default to 0
	// +optional
	RetryTimesOnEachNode int32 `json:"retryTimesOnEachNode,omitempty"`
}

// PrePullState describe the PrePullState of image prepull operation on edge nodes.
// +kubebuilder:validation:Enum=prepulling;successful;failed
type PrePullState string

// Valid values of PrepullState
const (
	PrePullInitialValue PrePullState = ""
	PrePulling          PrePullState = "prepulling"
	PrePullSuccessful   PrePullState = "successful"
	PrePullFailed       PrePullState = "failed"
)

// ImagePrePullJobStatus stores the status of ImagePrePullJob.
// contains images prepull status on multiple edge nodes.
// +kubebuilder:validation:Type=object
type ImagePrePullJobStatus struct {
	// State represents for the state phase of the ImagePrePullJob.
	// There are four possible state values: "", prechecking, prepulling, successful, failed.
	State PrePullState `json:"state,omitempty"`

	// Status contains image prepull status for each edge node.
	Status []ImagePrePullStatus `json:"status,omitempty"`
}

// ImagePrePullStatus stores image prepull status for each edge node.
// +kubebuilder:validation:Type=object
type ImagePrePullStatus struct {
	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`

	// State represents for the state phase of the ImagePrepullJob on the edge node.
	// There are five possible state values: "", prepulling, successful, failed.
	State PrePullState `json:"state,omitempty"`

	// Message represents the fail reason if images prepull failed on the edge node
	Reason string `json:"reason,omitempty"`

	// ImageStatus represents the prepull status for each image
	ImageStatus []ImageStatus `json:"imageStatus,omitempty"`
}

// ImageStatus stores the prepull status for each image.
// +kubebuilder:validation:Type=object
type ImageStatus struct {
	// Image is the name of the image
	Image string `json:"image,omitempty"`

	// State represents for the state phase of this image pull on the edge node
	// There are two possible state values: successful, failed.
	State PrePullState `json:"state,omitempty"`

	// Reason represents the fail reason if image pull failed
	// +optional
	Reason string `json:"reason,omitempty"`
}
