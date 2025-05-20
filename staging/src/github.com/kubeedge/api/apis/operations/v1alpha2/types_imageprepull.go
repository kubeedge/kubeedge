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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fsmv1alpha1 "github.com/kubeedge/api/apis/fsm/v1alpha1"
	opsv1alpha1 "github.com/kubeedge/api/apis/operations/v1alpha1"
)

const (
	ResourceImagePrePullJob = "imageprepulljob"

	FinalizerImagePrePullJob = "kubeedge.io/imageprepulljob-controller"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImagePrePullJob is used to prepull images on edge node.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion
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

	// Concurrency specifies the maximum number of concurrent that edge nodes associated with
	// each CloudCore instance can pull images at the same time.
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

type ImagePrePullJobAction string

const (
	ImagePrePullJobActionCheck ImagePrePullJobAction = "Check"
	ImagePrePullJobActionPull  ImagePrePullJobAction = "Pull"
)

// ImagePrePullJobStatus stores the status of ImagePrePullJob.
// contains images prepull status on multiple edge nodes.
// +kubebuilder:validation:Type=object
type ImagePrePullJobStatus struct {
	// Phase represents for the phase of the NodeUpgradeJob
	Phase JobPhase `json:"phase"`

	// NodeStatus contains image prepull status for each edge node.
	NodeStatus []ImagePrePullNodeTaskStatus `json:"nodeStatus,omitempty"`

	// Reason represents for the reason of the ImagePrePullJob.
	// +optional
	Reason string `json:"reason,omitempty"`

	// State represents for the state phase of the ImagePrePullJob.
	// There are several possible state values: "", Upgrading, BackingUp, RollingBack and Checking.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	State fsmv1alpha1.State `json:"state,omitempty"`

	// Time represents for the running time of the ImagePrePullJob.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Time string `json:"time,omitempty"`

	// Event represents for the event of the ImagePrePullJob.
	// There are four possible event values: Init, Check, Pull, TimeOut.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Event string `json:"event,omitempty"`

	// Action represents for the action of the ImagePrePullJob.
	// There are two possible action values: Success, Failure.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Action fsmv1alpha1.Action `json:"action,omitempty"`

	// Status contains image prepull status for each edge node.
	// +optional
	// Deprecated: For compatibility with v1alpha1 version, It will be removed in v1.23
	Status []opsv1alpha1.ImagePrePullStatus `json:"status,omitempty"`
}

// ImagePrePullNodeTaskStatus stores image prepull status for each edge node.
// +kubebuilder:validation:Type=object
type ImagePrePullNodeTaskStatus struct {
	// ActionFlow represents for the results of executing the action flow.
	ActionFlow []ImagePrePullJobActionStatus `json:"actionFlow,omitempty"`

	// ImageStatus represents the prepull status for each image
	ImageStatus []ImageStatus `json:"imageStatus,omitempty"`

	// NodeName is the name of edge node.
	NodeName string `json:"nodeName,omitempty"`

	// Phase represents for the phase of the node task.
	Phase NodeTaskPhase `json:"phase,omitempty"`

	// Reason represents the reason for the failure of the node task.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// ImagePrePullJobActionStatus defines the results of executing the action.
// +kubebuilder:validation:Type=object
type ImagePrePullJobActionStatus struct {
	// Action represents for the action name
	Action ImagePrePullJobAction `json:"action,omitempty"`

	// State represents for the status of this image pull on the edge node.
	Status metav1.ConditionStatus `json:"status,omitempty"`

	// Reason represents the reason for the failure of the action.
	// +optional
	Reason string `json:"reason,omitempty"`

	// Time represents for the running time of the node task.
	Time string `json:"time,omitempty"`
}

// ImageStatus stores the prepull status for each image.
// +kubebuilder:validation:Type=object
type ImageStatus struct {
	// Image is the name of the image
	Image string `json:"image,omitempty"`

	// State represents for the status of this image pull on the edge node.
	Status metav1.ConditionStatus `json:"status,omitempty"`

	// Reason represents the fail reason if image pull failed
	// +optional
	Reason string `json:"reason,omitempty"`
}
