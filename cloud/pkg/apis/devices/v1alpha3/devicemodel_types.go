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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeviceSpec defines the desired state of Device
type DeviceModelSpec struct {
	DeviceProperties []DeviceProperties `json:"properties,omitempty"`
}

// DeviceStatus defines the observed state of Device
type DeviceModelStatus struct {
	DeviceRefs []NamespaceName `json:"deviceRefs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Device is the Schema for the devices API
type DeviceModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceModelSpec   `json:"spec,omitempty"`
	Status DeviceModelStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeviceServiceList contains a list of DeviceService
type DeviceModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceModel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeviceModel{}, &DeviceModelList{})
}

type DeviceProperties struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	ProfileProperty `json:",inline"`
}

type ProfileProperty struct {
	Type         PropertyValueType `json:"type,omitempty"`
	Mutable      bool              `json:"mutable,omitempty"`
	Minimum      string            `json:"minimum,omitempty"`
	Maximum      string            `json:"maximum,omitempty"`
	DefaultValue string            `json:"defaultValue,omitempty"`
	ReadWrite    string            `json:"readWrite" yaml:"readWrite" validate:"required,oneof='R' 'W' 'RW'"`
	Units        string            `json:"units,omitempty" yaml:"units,omitempty"`
	Mask         string            `json:"mask,omitempty" yaml:"mask,omitempty"`
	Shift        string            `json:"shift,omitempty" yaml:"shift,omitempty"`
	Scale        string            `json:"scale,omitempty" yaml:"scale,omitempty"`
	Offset       string            `json:"offset,omitempty" yaml:"offset,omitempty"`
	Base         string            `json:"base,omitempty" yaml:"base,omitempty"`
	Assertion    string            `json:"assertion,omitempty" yaml:"assertion,omitempty"`
	MediaType    string            `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
}

type NamespaceName struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type PropertyValueType string

const (
	ValueTypeBool PropertyValueType = "Bool"
)
