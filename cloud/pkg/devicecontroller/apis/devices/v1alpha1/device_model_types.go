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
	Int *PropertyTypeInt64 `json:"int,omitempty"`
	// +optional
	String *PropertyTypeString `json:"string,omitempty"`
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
	OpcUA *VisitorConfigOPCUA `json:"opcua,omitempty"`
	// Modbus represents a set of additional visitor config fields of modbus protocol.
	// +optional
	Modbus *VisitorConfigModbus `json:"modbus,omitempty"`
	// Bluetooth represents a set of additional visitor config fields of bluetooth protocol.
	// +optional
	Bluetooth *VisitorConfigBluetooth `json:"bluetooth,omitempty"`
}

// Common visitor configurations for bluetooth protocol
type VisitorConfigBluetooth struct {
	// Required: Unique ID of the corresponding operation
	CharacteristicUUID string `json:"characteristicUUID,omitempty"`
	// Responsible for converting the data coming from the platform into a form that is understood by the bluetooth device
	// For example: "ON":[1], "OFF":[0]
	//+optional
	DataWriteToBluetooth map[string][]byte `json:"dataWrite,omitempty"`
	// Responsible for converting the data being read from the bluetooth device into a form that is understandable by the platform
	//+optional
	BluetoothDataConverter BluetoothReadConverter `json:"dataConverter,omitempty"`
}

// Specifies the operations that may need to be performed to convert the data
type BluetoothReadConverter struct {
	// Required: Specifies the start index of the incoming byte stream to be considered to convert the data.
	// For example: start-index:2, end-index:3 concatenates the value present at second and third index of the incoming byte stream. If we want to reverse the order we can give it as start-index:3, end-index:2
	StartIndex int `json:"startIndex,omitempty"`
	// Required: Specifies the end index of incoming byte stream to be considered to convert the data
	// the value specified should be inclusive for example if 3 is specified it includes the third index
	EndIndex int `json:"endIndex,omitempty"`
	// Refers to the number of bits to shift left, if left-shift operation is necessary for conversion
	// +optional
	ShiftLeft int `json:"shiftLeft,omitempty"`
	// Refers to the number of bits to shift right, if right-shift operation is necessary for conversion
	// +optional
	ShiftRight int `json:"shiftRight,omitempty"`
	// Specifies in what order the operations(which are required to be performed to convert incoming data into understandable form) are performed
	//+optional
	OrderOfOperations []BluetoothOperations `json:"orderOfOperations,omitempty"`
}

// Specify the operation that should be performed to convert incoming data into understandable form
type BluetoothOperations struct {
	// Required: Specifies the operation to be performed to convert incoming data
	BluetoothOperationType BluetoothArithmeticOperationType `json:"operationType,omitempty"`
	// Required: Specifies with what value the operation is to be performed
	BluetoothOperationValue float64 `json:"operationValue,omitempty"`
}

// Operations supported by Bluetooth protocol to convert the value being read from the device into an understandable form
type BluetoothArithmeticOperationType string

// Bluetooth Protocol Operation type
const (
	BluetoothAdd      BluetoothArithmeticOperationType = "Add"
	BluetoothSubtract BluetoothArithmeticOperationType = "Subtract"
	BluetoothMultiply BluetoothArithmeticOperationType = "Multiply"
	BluetoothDivide   BluetoothArithmeticOperationType = "Divide"
)

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
