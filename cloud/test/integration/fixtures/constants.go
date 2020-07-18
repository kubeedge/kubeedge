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

package fixtures

// CRD API Constants
const (
	apiVersion      = "devices.kubeedge.io/v1alpha2"
	deviceModelKind = "DeviceModel"
	deviceKind      = "Device"
)

const (
	ResourceDeviceModel = "devicemodels"
	ResourceDevice      = "devices"
)

// Device Model Constants
const (
	devicePropertyTemperature     = "temperature"
	devicePropertyTemperatureDesc = "temperature in degree celsius"
	devicePropertyUnit            = "degree celsius"
)

// Property Vistor Constants
const (
	// time.Duration, nanosecond
	reportCycle  = 1000000000
	collectCycle = 500000000
)

// Device instance constants
const (
	DefaultNamespace = "default"
	DeviceModelRef   = "sensor-tag-model"
)

type deviceProtocol string

// Supported protocols constants
const (
	deviceProtocolBluetooth  deviceProtocol = "bluetooth"
	deviceProtocolModbus     deviceProtocol = "modbus"
	deviceProtocolModbusRTU  deviceProtocol = "modbusRTU"
	deviceProtocolModbusTCP  deviceProtocol = "modbusTCP"
	deviceProtocolOPCUA      deviceProtocol = "opcua"
	deviceProtocolCustomized deviceProtocol = "customizedProtocol"
)

// integer property type constants
const (
	minimum = 0
	maximum = 100
)

// Bluetooth Protocol Constants
const (
	characteristicUUID = "f000aa0104514000b000000000000000"
	startIndex         = 1
	endIndex           = 2
	operationValue     = 0.03125
	offset             = 2
	limit              = 1
)

// Modbus RTU Protocol Constants
const (
	baudRate   = 110
	dataBits   = 8
	stopBits   = 1
	serialPort = "1"
	parity     = "none"
	slaveID    = 100
)
