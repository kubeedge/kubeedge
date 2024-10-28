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

	api "github.com/kubeedge/api/apis/fsm/v1alpha1"
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

	// FailureTolerate specifies the task tolerance failure ratio.
	// The default FailureTolerate value is 0.1.
	// +optional
	FailureTolerate string `json:"failureTolerate,omitempty"`

	// Concurrency specifies the maximum number of edge nodes that can pull images at the same time.
	// The default Concurrency value is 1.
	// +optional
	Concurrency int32 `json:"concurrency,omitempty"`

	// TimeoutSeconds limits the duration of the node prepull job on each edgenode.
	// Default to 300.
	// If set to 0, we'll use the default value 300.
	// +optional
	TimeoutSeconds *uint32 `json:"timeoutSeconds,omitempty"`

	// ImageSecret specifies the secret for image pull if private registry used.
	// Use {namespace}/{secretName} in format.
	// +optional
	ImageSecret string `json:"imageSecrets,omitempty"`

	// RetryTimes specifies the retry times if image pull failed on each edgenode.
	// Default to 0
	// +optional
	RetryTimes int32 `json:"retryTimes,omitempty"`
}

// ImagePrePullJobStatus stores the status of ImagePrePullJob.
// contains images prepull status on multiple edge nodes.
// +kubebuilder:validation:Type=object
type ImagePrePullJobStatus struct {
	// State represents for the state phase of the ImagePrePullJob.
	// There are five possible state values: "", checking, pulling, successful, failed.
	State api.State `json:"state,omitempty"`

	// Event represents for the event of the ImagePrePullJob.
	// There are four possible event values: Init, Check, Pull, TimeOut.
	Event string `json:"event,omitempty"`

	// Action represents for the action of the ImagePrePullJob.
	// There are two possible action values: Success, Failure.
	Action api.Action `json:"action,omitempty"`

	// Reason represents for the reason of the ImagePrePullJob.
	Reason string `json:"reason,omitempty"`

	// Time represents for the running time of the ImagePrePullJob.
	Time string `json:"time,omitempty"`

	// Status contains image prepull status for each edge node.
	Status []ImagePrePullStatus `json:"status,omitempty"`
}

// ImagePrePullStatus stores image prepull status for each edge node.
// +kubebuilder:validation:Type=object
type ImagePrePullStatus struct {
	// TaskStatus represents the status for each node
	*TaskStatus `json:"nodeStatus,omitempty"`
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
	State api.State `json:"state,omitempty"`

	// Reason represents the fail reason if image pull failed
	// +optional
	Reason string `json:"reason,omitempty"`
}
