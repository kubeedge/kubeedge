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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DeviceSpec defines the desired state of Device
type DeviceSpec struct {
	ModelRef        string                  `json:"model,omitempty"`
	DeviceAccessRef *corev1.ObjectReference `json:"deviceAccessRef,omitempty"`
	DeviceService   map[string]v1.JSON      `json:"service,omitempty"`

	// Foo is an example field of Device. Edit device_types.go to remove/update
	Protocol       DeviceProtocol  `json:"protocol,omitempty"`
	DeviceCommands []DeviceCommand `json:"commands,omitempty"`
}

// DeviceStatus defines the observed state of Device
type DeviceStatus struct {
	//ID               string           `json:"deviceId,omitempty"`
	Ready          bool                  `json:"ready"`
	Conditions     Conditions            `json:"conditions,omitempty"`
	DeviceCommands []DeviceCommandStatus `json:"commands,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Device is the Schema for the devices API
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// DeviceList contains a list of Device
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Device{}, &DeviceList{})
}

type DeviceProtocol struct {
	Name        string                     `json:"name,omitempty"`
	Type        string                     `json:"type,omitempty"`
	Address     string                     `json:"address,omitempty"`
	Port        int                        `json:"port,omitempty"`
	Timeout     int                        `json:"timeout,omitempty"`
	IdleTimeout int                        `json:"idleTimeout,omitempty"`
	Args        *unstructured.Unstructured `json:"args,omitempty"`
}

type DeviceCommand struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type DeviceCommandStatus struct {
	DeviceCommand `json:",inline"`
	ReadWrite     string `json:"readWrite,omitempty"`
}