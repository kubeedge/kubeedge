package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=`.status.conditions[?(@.type=="Applied")].status`,name="Applied",type=string
// +kubebuilder:printcolumn:JSONPath=`.metadata.creationTimestamp`,name="Age",type=date

// EdgeApplication defines a list of resources to be deployed on the node groups.
type EdgeApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of EdgeApplication.
	Spec EdgeAppSpec `json:"spec"`

	// Status represents the status of PropagationStatus.
	// +optional
	Status EdgeAppStatus `json:"status,omitempty"`
}

// EdgeAppSpec defines the desired state of EdgeApplication.
type EdgeAppSpec struct {
	// ResourceTemplate represents the manifest workload to be deployed on managed node groups.
	ResourceTemplate ResourceTemplate `json:"resourceTemplate,omitempty"`
	// WorkloadScopes represents the scope of workload to be deployed.
	WorkloadScopes WorkloadScope `json:"workloadScopes"`
}

// WorkloadScope represents the scope of workload to be deployed.
type WorkloadScope struct {
	// TargetNodeGroups represents the target node groups of workload to be deployed.
	// +optional
	TargetNodeGroups []TargetNodeGroups `json:"targetNodeGroups,omitempty"`
	// TargetNodes represents the target nodes of workload to be deployed.
	// +optional
	TargetNodes []TargetNodes `json:"targetNodes,omitempty"`
}

// TargetNodeGroups represents the target node groups of workload to be deployed.
type TargetNodeGroups struct {
	// Name represents the name of target node group
	Name string `json:"name"`
	// Overriders offers various alternatives to represent the override rules.
	Overriders Overriders `json:"overriders,omitempty"`
}

// TargetNodes represents the target nodes of workload to be deployed.
type TargetNodes struct {
	// Label of target nodes
	Label labels.Selector `json:"label"`
	// Overriders offers various alternatives to represent the override rules.
	Overriders Overriders `json:"overriders"`
}

// ResourceTemplate represents the manifest workload to be deployed on managed node groups.
type ResourceTemplate struct {
	// Manifests represents a list of Kubernetes resources to be deployed on the managed node groups.
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`
}

// Overriders offers various alternatives to represent the override rules.
//
// If more than one alternatives exist, they will be applied with following order:
// - ImageOverrider
// - Plaintext
type Overriders struct {
	// Replicas of deployment
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

// EdgeAppStatus defines the observed state of EdgeApplication.
type EdgeAppStatus struct {
	// Conditions contain the different condition statuses for this work.
	// Valid condition types are:
	// 1. Applied represents workload in EdgeApplication is applied successfully on a managed node groups.
	// 2. Progressing represents workload in EdgeApplication is being applied on a managed node groups.
	// 3. Available represents workload in EdgeApplication exists on the managed node groups.
	// 4. Degraded represents the current state of workload does not match the desired
	// state for a certain period.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ManifestStatuses contains running status of manifests in spec.
	// +optional
	ManifestStatuses []ManifestStatus `json:"manifestStatuses,omitempty"`
}

// ManifestStatus contains running status of a specific manifest in spec.
type ManifestStatus struct {
	// Identifier represents the identity of a resource linking to manifests in spec.
	// +required
	Identifier ResourceIdentifier `json:"identifier"`

	// Status reflects running status of current manifest.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Status *runtime.RawExtension `json:"status,omitempty"`
}

// ResourceIdentifier provides the identifiers needed to interact with any arbitrary object.
type ResourceIdentifier struct {
	// Ordinal represents an index in manifests list, so the condition can still be linked
	// to a manifest even though manifest cannot be parsed successfully.
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

const (
	// EdgeAppApplied represents that the resource defined in node groups is
	// successfully applied on the managed node groups.
	EdgeAppApplied string = "Applied"
	// EdgeAppProgressing represents that the resource defined in node groups is
	// in the progress to be applied on the managed node groups.
	EdgeAppProgressing string = "Progressing"
	// EdgeAppAvailable represents that all resources of the node groups exists on
	// the managed node groups.
	EdgeAppAvailable string = "Available"
	// EdgeAppDegraded represents that the current state of node groups does not match
	// the desired state for a certain period.
	EdgeAppDegraded string = "Degraded"
)
