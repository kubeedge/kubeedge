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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeviceSpec represents a single device instance. It is an instantation of a device model.
type DeviceSpec struct {
	// Required: DeviceModelRef is reference to the device model used as a template
	// to create the device instance.
	DeviceModelRef *v1.LocalObjectReference `json:"deviceModelRef,omitempty"`
	// Required: The protocol configuration used to connect to the device.
	Protocol ProtocolConfig `json:"protocol,omitempty"`
	// NodeSelector indicates the binding preferences between devices and nodes.
	// Refer to k8s.io/kubernetes/pkg/apis/core NodeSelector for more details
	// +optional
	NodeSelector *v1.NodeSelector `json:"nodeSelector,omitempty"`
}

// Only one of its members may be specified.
type ProtocolConfig struct {
	// Protocol configuration for opc-ua
	// +optional
	OpcUA *ProtocolConfigOpcUA `json:"opcua,omitempty"`
	// Protocol configuration for modbus
	// +optional
	Modbus *ProtocolConfigModbus `json:"modbus,omitempty"`
	// Protocol configuration for bluetooth
	// +optional
	Bluetooth *ProtocolConfigBluetooth `json:"bluetooth,omitempty"`
}

type ProtocolConfigOpcUA struct {
	// Required: The URL for opc server endpoint.
	URL string `json:"url,omitempty"`
	// Username for access opc server.
	// +optional
	UserName string `json:"userName,omitempty"`
	// Password for access opc server.
	// +optional
	Password string `json:"password,omitempty"`
	// Defaults to "none".
	// +optional
	SecurityPolicy string `json:"securityPolicy,omitempty"`
	// Defaults to "none".
	// +optional
	SecurityMode string `json:"securityMode,omitempty"`
	// Certificate for access opc server.
	// +optional
	Certificate string `json:"certificate,omitempty"`
	// PrivateKey for access opc server.
	// +optional
	PrivateKey string `json:"privateKey,omitempty"`
	// Timeout seconds for the opc server connection.???
	// +optional
	Timeout int64 `json:"timeout,omitempty"`
}

// Only one of its members may be specified.
type ProtocolConfigModbus struct {
	// +optional
	RTU *ProtocolConfigModbusRTU `json:"rtu,omitempty"`
	// +optional
	TCP *ProtocolConfigModbusTCP `json:"tcp,omitempty"`
}

type ProtocolConfigModbusTCP struct {
	// Required.
	IP string `json:"ip,omitempty"`
	// Required.
	Port int64 `json:"port,omitempty"`
	// Required.
	SlaveID string `json:"slaveID,omitempty"`
}

type ProtocolConfigModbusRTU struct {
	// Required.
	SerialPort string `json:"serialPort,omitempty"`
	// Required. BaudRate 115200|57600|38400|19200|9600|4800|2400|1800|1200|600|300|200|150|134|110|75|50
	BaudRate int64 `json:"baudRate,omitempty"`
	// Required. Valid values are 8, 7, 6, 5.
	DataBits int64 `json:"dataBits,omitempty"`
	// Required. Valid options are "none", "even", "odd". Defaults to "none".
	Parity string `json:"parity,omitempty"`
	// Required. Bit that stops 1|2
	StopBits int64 `json:"stopBits,omitempty"`
	// Required. 0-255
	SlaveID int64 `json:"slaveID,omitempty"`
}

type ProtocolConfigBluetooth struct {
	// Unique identifier assigned to the device.
	// +optional
	MACAddress string `json:"macAddress,omitempty"`
}

// DeviceStatus reports the device state and the desired/reported values of twin attributes.
type DeviceStatus struct {
	// A list of device twins containing desired/reported desired/reported values of twin properties..
	// Optional: A passive device won't have twin properties and this list could be empty.
	// +optional
	Twins []Twin `json:"twins,omitempty"`
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
	// Required: the desired property value.
	Desired TwinProperty `json:"desired,omitempty"`
	// Required: the reported property value.
	Reported TwinProperty `json:"reported,omitempty"`
}

// TwinProperty represents the device property for which an Expected/Actual state can be defined.
type TwinProperty struct {
	// Required: The value for this property.
	Value string `json:"value,"`
	// Additional metadata like timestamp when the value was reported etc.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Device is the Schema for the devices API
// +k8s:openapi-gen=true
type Device struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceSpec   `json:"spec,omitempty"`
	Status DeviceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DeviceList contains a list of Device
type DeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Device `json:"items"`
}
