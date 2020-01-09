/*
Copyright 2020 The KubeEdge Authors.

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

// ClusterObjectSync stores the state of the cluster level, nonNamespaced object that was successfully persisted to the edge node.
// ClusterObjectSync name is a concatenation of the node name which receiving the object and the object UUID.
// +k8s:openapi-gen=true
type ClusterObjectSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObjectSyncSpec   `json:"spec,omitempty"`
	Status ObjectSyncStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterObjectSyncList is a list of ObjectSync.
type ClusterObjectSyncList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ClusterObjectSync.
	Items []ObjectSync `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ObjectSync stores the state of the namespaced object that was successfully persisted to the edge node.
// ObjectSync name is a concatenation of the node name which receiving the object and the object UUID.
// +k8s:openapi-gen=true
type ObjectSync struct {
	// Standard Kubernetes type metadata.
	metav1.TypeMeta `json:",inline"`
	// Standard Kubernetes object's metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObjectSyncSpec   `json:"spec,omitempty"`
	Status ObjectSyncStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ObjectSyncList is a list of ObjectSync.
type ObjectSyncList struct {
	// Standard type metadata.
	metav1.TypeMeta `json:",inline"`

	// Standard list metadata.
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of ObjectSync.
	Items []ObjectSync `json:"items"`
}

// ObjectSyncSpec stores the details of objects that persist to the edge.
type ObjectSyncSpec struct {
	// ObjectAPIVersion is the APIVersion of the object
	// that was successfully persist to the edge node.
	ObjectAPIVersion string `json:"objectAPIVersion,omitempty"`
	// ObjectType is the kind of the object
	// that was successfully persist to the edge node.
	ObjectKind string `json:"objectKind,omitempty"`
	// ObjectName is the name of the object
	// that was successfully persist to the edge node.
	ObjectName string `json:"objectName,omitempty"`
}

// ObjectSyncSpec stores the resourceversion of objects that persist to the edge.
type ObjectSyncStatus struct {
	// ObjectResourceVersion is the resourceversion of the object
	// that was successfully persist to the edge node.
	ObjectResourceVersion string `json:"objectResourceVersion,omitempty"`
}
