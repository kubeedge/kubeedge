/*
Copyright 2021.

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

package v1alpha3

import (
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeviceAccessSpec defines the desired state of DeviceAccess
type DeviceAccessSpec struct {

	// Foo is an example field of Device. Edit device_types.go to remove/update
	AccessParameters []AccessParameter `json:"parameters,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// DeviceAccess is the Schema for the devicevisitors API
type DeviceAccess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeviceAccessSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceAccessList contains a list of DeviceAccess
type DeviceAccessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceAccess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceAccess{}, &DeviceAccessList{})
}

type AccessParameter struct {
	Name      string             `json:"name,omitempty"`
	Parameter map[string]v1.JSON `json:"parameter,omitempty"`
}
