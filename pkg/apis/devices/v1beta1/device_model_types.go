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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeviceModelSpec defines the model for a device.It is a blueprint which describes the device
// capabilities and access mechanism via property visitors.
type DeviceModelSpec struct {
	// Required: List of device properties.
	Properties []ModelProperty `json:"properties,omitempty"`
	// Required: Protocol name used by the device.
	Protocol string `json:"protocol,omitempty"`
}

// ModelProperty describes an individual device property / attribute like temperature / humidity etc.
type ModelProperty struct {
	// Required: The device property name.
	// Note: If you need to use the built-in stream data processing function, you need to define Name as saveFrame or saveVideo
	Name string `json:"name,omitempty"`
	// The device property description.
	// +optional
	Description string `json:"description,omitempty"`
	// Required: Type of device property, ENUM: INT,FLOAT,DOUBLE,STRING,BOOLEAN,BYTES,STREAM
	Type PropertyType `json:"type,omitempty"`
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	Minimum string `json:"minimum,omitempty"`
	// +optional
	Maximum string `json:"maximum,omitempty"`
	// The unit of the property
	// +optional
	Unit string `json:"unit,omitempty"`
}

// The type of device property.
// +kubebuilder:validation:Enum=INT;FLOAT;DOUBLE;STRING;BOOLEAN;BYTES;STREAM
type PropertyType string

const (
	INT     PropertyType = "INT"
	FLOAT   PropertyType = "FLOAT"
	DOUBLE  PropertyType = "DOUBLE"
	STRING  PropertyType = "STRING"
	BOOLEAN PropertyType = "BOOLEAN"
	BYTES   PropertyType = "BYTES"
	STREAM  PropertyType = "STREAM"
)

// The access mode for  a device property.
// +kubebuilder:validation:Enum=ReadWrite;ReadOnly
type PropertyAccessMode string

// Access mode constants for a device property.
const (
	ReadWrite PropertyAccessMode = "ReadWrite"
	ReadOnly  PropertyAccessMode = "ReadOnly"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceModel is the Schema for the device model API
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
type DeviceModel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeviceModelSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceModelList contains a list of DeviceModel
type DeviceModelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceModel `json:"items"`
}
