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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EdgeApplicationSpec defines the desired state of EdgeApplication
type EdgeApplicationSpec struct {
	// WorkloadTemplate contains original templates of resources to be deployed
	// as an EdgeApplication.
	WorkloadTemplate ResourceTemplate `json:"workloadTemplate,omitempty"`
	// WorkloadScope represents which node groups the workload will be deployed in.
	WorkloadScope WorkloadScope `json:"workloadScope"`
}

// WorkloadScope represents which node groups the workload should be deployed in.
type WorkloadScope struct {
	// TargetNodeGroups represents the target node groups of workload to be deployed.
	// +optional
	TargetNodeGroups []TargetNodeGroup `json:"targetNodeGroups,omitempty"`
}

// TargetNodeGroup represents the target node group of workload to be deployed, including
// override rules to apply for this node group.
type TargetNodeGroup struct {
	// Name represents the name of target node group
	Name string `json:"name"`
	// Overriders represents the override rules that would apply on workload.
	Overriders Overriders `json:"overriders,omitempty"`
}

// ResourceTemplate represents original templates of resources to be deployed
// as an EdgeApplication.
type ResourceTemplate struct {
	// Manifests represent a list of Kubernetes resources to be deployed on the managed node groups.
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`
}

// Overriders represents the override rules that would apply on resources.
type Overriders struct {
	// Replicas will override the replicas field of deployment
	// +optional
	Replicas int `json:"replicas,omitempty"`
	// ImageOverrider represents the rules dedicated to handling image overrides.
	// +optional
	ImageOverrider []ImageOverrider `json:"imageOverrider,omitempty"`
}

// ImageOverrider represents the rules dedicated to handling image overrides.
type ImageOverrider struct {
	// Predicate filters images before applying the rule.
	//
	// Defaults to nil, in that case, the system will automatically detect image fields if the resource type is
	// Pod, ReplicaSet, Deployment or StatefulSet by following rule:
	//   - Pod: spec/containers/<N>/image
	//   - ReplicaSet: spec/template/spec/containers/<N>/image
	//   - Deployment: spec/template/spec/containers/<N>/image
	//   - StatefulSet: spec/template/spec/containers/<N>/image
	// In addition, all images will be processed if the resource object has more than one containers.
	//
	// If not nil, only images matches the filters will be processed.
	// +optional
	Predicate *ImagePredicate `json:"predicate,omitempty"`

	// Component is part of image name.
	// Basically we presume an image can be made of '[registry/]repository[:tag]'.
	// The registry could be:
	// - k8s.gcr.io
	// - fictional.registry.example:10443
	// The repository could be:
	// - kube-apiserver
	// - fictional/nginx
	// The tag cloud be:
	// - latest
	// - v1.19.1
	// - @sha256:dbcc1c35ac38df41fd2f5e4130b32ffdb93ebae8b3dbe638c23575912276fc9c
	//
	// +kubebuilder:validation:Enum=Registry;Repository;Tag
	// +required
	Component ImageComponent `json:"component"`

	// Operator represents the operator which will apply on the image.
	// +kubebuilder:validation:Enum=add;remove;replace
	// +required
	Operator OverriderOperator `json:"operator"`

	// Value to be applied to image.
	// Must not be empty when operator is 'add' or 'replace'.
	// Defaults to empty and ignored when operator is 'remove'.
	// +optional
	Value string `json:"value,omitempty"`
}

// ImagePredicate describes images filter.
type ImagePredicate struct {
	// Path indicates the path of target field
	// +required
	Path string `json:"path"`
}

// ImageComponent indicates the components for image.
type ImageComponent string

const (
	// Registry is the registry component of an image with format '[registry/]repository[:tag]'.
	Registry ImageComponent = "Registry"

	// Repository is the repository component of an image with format '[registry/]repository[:tag]'.
	Repository ImageComponent = "Repository"

	// Tag is the tag component of an image with format '[registry/]repository[:tag]'.
	Tag ImageComponent = "Tag"
)

// OverriderOperator is the set of operators that can be used in an overrider.
type OverriderOperator string

// These are valid overrider operators.
const (
	OverriderOpAdd     OverriderOperator = "add"
	OverriderOpRemove  OverriderOperator = "remove"
	OverriderOpReplace OverriderOperator = "replace"
)

// Manifest represents a resource to be deployed on managed node groups.
type Manifest struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	runtime.RawExtension `json:",inline"`
}

// EdgeApplicationStatus defines the observed state of EdgeApplication
type EdgeApplicationStatus struct {
	// WorkloadStatus contains running statuses of generated resources.
	// +optional
	WorkloadStatus []ManifestStatus `json:"workloadStatus,omitempty"`
}

// ManifestStatus contains running status of a specific manifest in spec.
type ManifestStatus struct {
	// Identifier represents the identity of a resource linking to manifests in spec.
	// +required
	Identifier ResourceIdentifier `json:"identifier"`

	// Conditions contain the different condition statuses for this manifest.
	// Valid condition types are:
	// 1. Processing: this workload is under processing and the current state of manifest does not match the desired.
	// 2. Available: the current status of this workload matches the desired.
	// +kubebuilder:validation:Enum=Processing;Available
	// +optional
	Condition ManifestCondition `json:"conditions,omitempty"`
}

// ResourceIdentifier provides the identifiers needed to interact with any arbitrary object.
type ResourceIdentifier struct {
	// Ordinal represents an index in manifests list, so the condition can still be linked
	// to a manifest even though manifest cannot be parsed successfully.
	// +kubebuilder:validation:Minimum=0
	Ordinal int `json:"ordinal"`

	// Group is the group of the resource.
	Group string `json:"group,omitempty"`

	// Version is the version of the resource.
	Version string `json:"version"`

	// Kind is the kind of the resource.
	Kind string `json:"kind"`

	// Resource is the resource type of the resource
	Resource string `json:"resource"`

	// Namespace is the namespace of the resource
	Namespace string `json:"namespace"`

	// Name is the name of the resource
	Name string `json:"name"`
}

type ManifestCondition string

const (
	// EdgeAppProcessing represents that the manifest is under processing and currently
	// the status of this manifest does not match the desired.
	EdgeAppProcessing ManifestCondition = "Processing"
	// EdgeAppAvailable represents that the manifest has been applied successfully and the current
	// status matches the desired.
	EdgeAppAvailable ManifestCondition = "Available"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=eapp

// EdgeApplication is the Schema for the edgeapplications API
type EdgeApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of EdgeApplication.
	// +required
	Spec EdgeApplicationSpec `json:"spec,omitempty"`
	// Status represents the status of PropagationStatus.
	// +optional
	Status EdgeApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EdgeApplicationList contains a list of EdgeApplication
type EdgeApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeApplication `json:"items"`
}
