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

package v1beta1

import (
	// "encoding/json"

	// v1 "k8s.io/api/core/v1"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceStatus is the Schema for the devices API
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
type DeviceStatus struct {
	// Standard object's metadata.
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DeviceStatusSpec   `json:"spec,omitempty"`
	Status            DeviceStatusStatus `json:"status,omitempty"`
}

// DeviceStatusSpec represents a single device status instance
// Since the association between device and device status is 1:1 and encoded in OwnerReference,
// there is no need to have a field to indicate which device this status belongs to.
type DeviceStatusSpec struct {
}

// DeviceStatusStatus reports the device state and the desired/reported values of twin attributes.
type DeviceStatusStatus struct {
	// A list of device twins containing desired/reported desired/reported values of twin properties.
	// Optional: A passive device won't have twin properties and this list could be empty.
	// +optional
	Twins []Twin `json:"twins,omitempty"`
	// Optional: The state of the device.
	// +optional
	State string `json:"state,omitempty"`
	// Optional: The last time the device was online.
	// +optional
	LastOnlineTime string `json:"lastOnlineTime,omitempty"`
	// Optional: Extensions can be used to add more status information.
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	Extensions DeviceStatusExtensions `json:"extensions,omitempty"`
}

// Twin provides a logical representation of control properties (writable properties in the
// device model). The properties can have a Desired state and a Reported state. The cloud configures
// the `Desired`state of a device property and this configuration update is pushed to the edge node.
// The mapper sends a command to the device to change this property value as per the desired state .
// It receives the `Reported` state of the property once the previous operation is complete and sends
// the reported state to the cloud. Offline device interaction in the edge is possible via twin
// properties for control/command operations.
type Twin struct {
	// Required: The property name for which the desired/reported values are specified.
	// This property should be present in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Required: the reported property value.
	Reported TwinProperty `json:"reported,omitempty"`
	// The meaning of here is to indicate desired value of `deviceProperty.Desired`
	// that the mapper has received in current cycle.
	// Useful in cases that people want to check whether the mapper is working
	// appropriately and its internal status is up-to-date.
	// This value should be only updated by devicecontroller upstream.
	ObservedDesired TwinProperty `json:"observedDesired,omitempty"`
}

// DeviceStatusExtensions defines a struct to hold extension info of device status
// +kubebuilder:validation:Type=object
type DeviceStatusExtensions struct {
	Data map[string]interface{} `json:"data"`
}

// MarshalJSON implements the Marshaler interface.
func (in *DeviceStatusExtensions) MarshalJSON() ([]byte, error) {
	return json.Marshal(in.Data)
}

// UnmarshalJSON implements the Unmarshaler interface.
func (in *DeviceStatusExtensions) UnmarshalJSON(data []byte) error {
	var out map[string]interface{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	in.Data = out
	return nil
}

// DeepCopyInto implements the DeepCopyInto interface.
func (in *DeviceStatusExtensions) DeepCopyInto(out *DeviceStatusExtensions) {
	bytes, err := json.Marshal(in.Data)
	if err != nil {
		panic(err)
	}
	var clone map[string]interface{}
	err = json.Unmarshal(bytes, &clone)
	if err != nil {
		panic(err)
	}
	out.Data = clone
}

// DeepCopy implements the DeepCopy interface.
func (in *DeviceStatusExtensions) DeepCopy() *DeviceStatusExtensions {
	if in == nil {
		return nil
	}
	out := new(DeviceStatusExtensions)
	in.DeepCopyInto(out)
	return out
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceStatusList contains a list of DeviceStatus
type DeviceStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeviceStatus `json:"items"`
}
