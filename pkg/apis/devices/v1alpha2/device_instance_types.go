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

package v1alpha2

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeviceSpec represents a single device instance. It is an instantation of a device model.
type DeviceSpec struct {
	// Required: DeviceModelRef is reference to the device model used as a template
	// to create the device instance.
	DeviceModelRef *v1.LocalObjectReference `json:"deviceModelRef,omitempty"`
	// Required: The protocol configuration used to connect to the device.
	Protocol ProtocolConfig `json:"protocol,omitempty"`
	// List of property visitors which describe how to access the device properties.
	// PropertyVisitors must unique by propertyVisitor.propertyName.
	// +optional
	PropertyVisitors []DevicePropertyVisitor `json:"propertyVisitors,omitempty"`
	// Data section describe a list of time-series properties which should be processed
	// on edge node.
	// +optional
	Data DeviceData `json:"data,omitempty"`
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
	// Configuration for protocol common part
	// +optional
	Common *ProtocolConfigCommon `json:"common,omitempty"`
	// Configuration for customized protocol
	// +optional
	CustomizedProtocol *ProtocolConfigCustomized `json:"customizedProtocol,omitempty"`
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
	// Required. 0-255
	SlaveID *int64 `json:"slaveID,omitempty"`
}

// Only one of COM or TCP may be specified.
type ProtocolConfigCommon struct {
	// +optional
	COM *ProtocolConfigCOM `json:"com,omitempty"`
	// +optional
	TCP *ProtocolConfigTCP `json:"tcp,omitempty"`
	// Communication type, like tcp client, tcp server or COM
	// +optional
	CommType string `json:"commType,omitempty"`
	// Reconnection timeout
	// +optional
	ReconnTimeout int64 `json:"reconnTimeout,omitempty"`
	// Reconnecting retry times
	// +optional
	ReconnRetryTimes int64 `json:"reconnRetryTimes,omitempty"`
	// Define timeout of mapper collect from device.
	// +optional
	CollectTimeout int64 `json:"collectTimeout,omitempty"`
	// Define retry times of mapper will collect from device.
	// +optional
	CollectRetryTimes int64 `json:"collectRetryTimes,omitempty"`
	// Define collect type, sync or async.
	// +optional
	// +kubebuilder:validation:Enum=sync;async
	CollectType string `json:"collectType,omitempty"`
	// Customized values for provided protocol
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	CustomizedValues *CustomizedValue `json:"customizedValues,omitempty"`
}

type ProtocolConfigTCP struct {
	// Required.
	IP string `json:"ip,omitempty"`
	// Required.
	Port int64 `json:"port,omitempty"`
}

type ProtocolConfigCOM struct {
	// Required.
	SerialPort string `json:"serialPort,omitempty"`
	// Required. BaudRate 115200|57600|38400|19200|9600|4800|2400|1800|1200|600|300|200|150|134|110|75|50
	// +kubebuilder:validation:Enum=115200;57600;38400;19200;9600;4800;2400;1800;1200;600;300;200;150;134;110;75;50
	BaudRate int64 `json:"baudRate,omitempty"`
	// Required. Valid values are 8, 7, 6, 5.
	// +kubebuilder:validation:Enum=8;7;6;5
	DataBits int64 `json:"dataBits,omitempty"`
	// Required. Valid options are "none", "even", "odd". Defaults to "none".
	// +kubebuilder:validation:Enum=none;even;odd
	Parity string `json:"parity,omitempty"`
	// Required. Bit that stops 1|2
	// +kubebuilder:validation:Enum=1;2
	StopBits int64 `json:"stopBits,omitempty"`
}

type ProtocolConfigBluetooth struct {
	// Unique identifier assigned to the device.
	// +optional
	MACAddress string `json:"macAddress,omitempty"`
}

type ProtocolConfigCustomized struct {
	// Unique protocol name
	// Required.
	ProtocolName string `json:"protocolName,omitempty"`
	// Any config data
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	ConfigData *CustomizedValue `json:"configData,omitempty"`
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

// DeviceData reports the device's time-series data to edge MQTT broker.
// These data should not be processed by edgecore. Instead, they can be process by
// third-party data-processing apps like EMQX kuiper.
type DeviceData struct {
	// Required: A list of data properties, which are not required to be processed by edgecore
	DataProperties []DataProperty `json:"dataProperties,omitempty"`
	// Topic used by mapper, all data collected from dataProperties
	// should be published to this topic,
	// the default value is $ke/events/device/+/data/update
	// +optional
	DataTopic string `json:"dataTopic,omitempty"`
}

// DataProperty represents the device property for external use.
type DataProperty struct {
	// Required: The property name for which should be processed by external apps.
	// This property should be present in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Additional metadata like timestamp when the value was reported etc.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// DevicePropertyVisitor describes the specifics of accessing a particular device
// property. Visitors are intended to be consumed by device mappers which connect to devices
// and collect data / perform actions on the device.
type DevicePropertyVisitor struct {
	// Required: The device property name to be accessed. This should refer to one of the
	// device properties defined in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Define how frequent mapper will report the value.
	// +optional
	ReportCycle int64 `json:"reportCycle,omitempty"`
	// Define how frequent mapper will collect from device.
	// +optional
	CollectCycle int64 `json:"collectCycle,omitempty"`
	// Customized values for visitor of provided protocols
	// +optional
	// +kubebuilder:validation:XPreserveUnknownFields
	CustomizedValues *CustomizedValue `json:"customizedValues,omitempty"`
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
	// CustomizedProtocol represents a set of visitor config fields of bluetooth protocol.
	// +optional
	CustomizedProtocol *VisitorConfigCustomized `json:"customizedProtocol,omitempty"`
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
	ShiftLeft uint `json:"shiftLeft,omitempty"`
	// Refers to the number of bits to shift right, if right-shift operation is necessary for conversion
	// +optional
	ShiftRight uint `json:"shiftRight,omitempty"`
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
// +kubebuilder:validation:Enum:Add;Subtract;Multiply;Divide
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
	Offset *int64 `json:"offset,omitempty"`
	// Required: Limit number of registers to read/write.
	Limit *int64 `json:"limit,omitempty"`
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
// +kubebuilder:validation:Enum=CoilRegister;DiscreteInputRegister;InputRegister;HoldingRegister
type ModbusRegisterType string

// Modbus protocol register types
const (
	ModbusRegisterTypeCoilRegister          ModbusRegisterType = "CoilRegister"
	ModbusRegisterTypeDiscreteInputRegister ModbusRegisterType = "DiscreteInputRegister"
	ModbusRegisterTypeInputRegister         ModbusRegisterType = "InputRegister"
	ModbusRegisterTypeHoldingRegister       ModbusRegisterType = "HoldingRegister"
)

// Common visitor configurations for customized protocol
type VisitorConfigCustomized struct {
	// Required: name of customized protocol
	ProtocolName string `json:"protocolName,omitempty"`
	// Required: The configData of customized protocol
	// +kubebuilder:validation:XPreserveUnknownFields
	ConfigData *CustomizedValue `json:"configData,omitempty"`
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

// CustomizedValue contains a map type data
// +kubebuilder:validation:Type=object
type CustomizedValue struct {
	Data map[string]interface{} `json:"-"`
}

// MarshalJSON implements the Marshaler interface.
func (in *CustomizedValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(in.Data)
}

// UnmarshalJSON implements the Unmarshaler interface.
func (in *CustomizedValue) UnmarshalJSON(data []byte) error {
	var out map[string]interface{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	in.Data = out
	return nil
}

// DeepCopyInto implements the DeepCopyInto interface.
func (in *CustomizedValue) DeepCopyInto(out *CustomizedValue) {
	bytes, err := json.Marshal(*in)
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
func (in *CustomizedValue) DeepCopy() *CustomizedValue {
	if in == nil {
		return nil
	}
	out := new(CustomizedValue)
	in.DeepCopyInto(out)
	return out
}
