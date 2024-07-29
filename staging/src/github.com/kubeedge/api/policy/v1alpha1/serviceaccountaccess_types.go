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
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=saaccess

// ServiceAccountAccess is the Schema for the ServiceAccountAccess API
type ServiceAccountAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the specification of rbac.
	// +required
	Spec AccessSpec `json:"spec,omitempty"`

	// Status represents the node list which store the rules.
	// +optional
	Status AccessStatus `json:"status,omitempty"`
}

// AccessStatus defines the observed state of ServiceAccountAccess
type AccessStatus struct {
	// NodeList represents the node name which store the rules.
	NodeList []string `json:"nodeList,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccountAccessList contains a list of ServiceAccountAccess
type ServiceAccountAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccountAccess `json:"items"`
}

// AccessSpec defines the desired state of AccessSpec
type AccessSpec struct {
	// ServiceAccount is one-to-one corresponding relations with the serviceaccountaccess.
	ServiceAccount corev1.ServiceAccount `json:"serviceAccount,omitempty"`
	// ServiceAccountUID is the uid of serviceaccount.
	ServiceAccountUID types.UID `json:"serviceAccountUid,omitempty"`
	// AccessRoleBinding represents rbac rolebinding plus detailed role info.
	AccessRoleBinding []AccessRoleBinding `json:"accessRoleBinding,omitempty"`
	// AccessClusterRoleBinding represents rbac ClusterRoleBinding plus detailed ClusterRole info.
	AccessClusterRoleBinding []AccessClusterRoleBinding `json:"accessClusterRoleBinding,omitempty"`
}

// AccessRoleBinding represents rbac rolebinding plus detailed role info.
type AccessRoleBinding struct {
	// RoleBinding represents rbac rolebinding.
	RoleBinding rbac.RoleBinding `json:"roleBinding,omitempty"`
	// Rules contains role rules.
	Rules []rbac.PolicyRule `json:"rules,omitempty"`
}

// AccessClusterRoleBinding represents rbac ClusterRoleBinding plus detailed ClusterRole info.
type AccessClusterRoleBinding struct {
	// ClusterRoleBinding represents rbac ClusterRoleBinding.
	ClusterRoleBinding rbac.ClusterRoleBinding `json:"clusterRoleBinding,omitempty"`
	// Rules contains role rules.
	Rules []rbac.PolicyRule `json:"rules,omitempty"`
}
