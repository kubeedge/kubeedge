/*
Copyright 2019 The KubeEdge Authors.

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

// DeviceModelSpec defines the model / template for a device.It is a blueprint which describes the device
// capabilities and access mechanism via property visitors.
type DeviceModelSpec struct {
	// Required: List of device properties.
	Properties []DeviceProperty `json:"properties,omitempty"`
	// Required: List of property visitors which describe how to access the device properties.
	// PropertyVisitors must unique by propertyVisitor.propertyName.
	PropertyVisitors []DevicePropertyVisitor `json:"propertyVisitors,omitempty"`
}

// DeviceProperty describes an individual device property / attribute like temperature / humidity etc.
type DeviceProperty struct {
	// Required: The device property name.
	Name string `json:"name,omitempty"`
	// The device property description.
	// +optional
	Description string `json:"description,omitempty"`
	// Required: PropertyType represents the type and data validation of the property.
	Type PropertyType `json:"type,omitempty"`
}

// Represents the type and data validation of a property.
// Only one of its members may be specified.
type PropertyType struct {
	// +optional
	Int PropertyTypeInt64 `json:"int,omitempty"`
	// +optional
	String PropertyTypeString `json:"string,omitempty"`
}

type PropertyTypeInt64 struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	DefaultValue int64 `json:"defaultValue,omitempty"`
	// +optional
	Minimum int64 `json:"minimum,omitempty"`
	// +optional
	Maximum int64 `json:"maximum,omitempty"`
	// The unit of the property
	// +optional
	Unit string `json:"unit,omitempty"`
}

type PropertyTypeString struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	DefaultValue string `json:"defaultValue,omitempty"`
}

// The access mode for  a device property.
type PropertyAccessMode string

// Access mode constants for a device property.
const (
	ReadWrite PropertyAccessMode = "ReadWrite"
	ReadOnly  PropertyAccessMode = "ReadOnly"
)

// DevicePropertyVisitor describes the specifics of accessing a particular device
// property. Visitors are intended to be consumed by device mappers which connect to devices
// and collect data / perform actions on the device.
type DevicePropertyVisitor struct {
	// Required: The device property name to be accessed. This should refer to one of the
	// device properties defined in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Required: Protocol relevant config details about the how to access the device property.
	VisitorConfig `json:",inline"`
}

// At least one of its members must be specified.
type VisitorConfig struct {
	// Opcua represents a set of additional visitor config fields of opc-ua protocol.
	// +optional
	OpcUA VisitorConfigOPCUA `json:"opcua,omitempty"`
	// Modbus represents a set of additional visitor config fields of modbus protocol.
	// +optional
	Modbus VisitorConfigModbus `json:"modbus,omitempty"`
}

// Common visitor configurations for opc-ua protocol
type VisitorConfigOPCUA struct {
	// Required: The ID of opc-ua node, e.g. "ns=1,i=1005"
	NodeID string `json:"nodeID,omitempty"`
	// The name of opc-ua node
	BrowseName string `json:"browseName,omitempty"`
}

// Common visitor configurations for modbus protocol
type VisitorConfigModbus struct {
	// Required: Type of register
	Register ModbusRegisterType `json:"register,omitempty"`
	// Required: Offset indicates the starting register number to read/write data.
	Offset int64 `json:"offset,omitempty"`
	// Required: Limit number of registers to read/write.
	Limit int64 `json:"limit,omitempty"`
	// The scale to convert raw property data into final units.
	// Defaults to 1.0
	// +optional
	Scale float64 `json:"scale,omitempty"`
	// Indicates whether the high and low byte swapped.
	// Defaults to false.
	// +optional
	IsSwap bool `json:"isSwap,omitempty"`
	// Indicates whether the high and low register swapped.
	// Defaults to false.
	// +optional
	IsRegisterSwap bool `json:"isRegisterSwap,omitempty"`
}

// The Modbus register type to read a device property.
type ModbusRegisterType string

// Modbus protocol register types
const (
	ModbusRegisterTypeCoilRegister          ModbusRegisterType = "CoilRegister"
	ModbusRegisterTypeDiscreteInputRegister ModbusRegisterType = "DiscreteInputRegister"
	ModbusRegisterTypeInputRegister         ModbusRegisterType = "InputRegister"
	ModbusRegisterTypeHoldingRegister       ModbusRegisterType = "HoldingRegister"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceModel is the Schema for the device model API
// +k8s:openapi-gen=true
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
