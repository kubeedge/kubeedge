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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1beta1"
)

type DeviceOp struct {
	device v1beta1.Device
}

type DeviceOption func(*DeviceOp)

func (op *DeviceOp) applyDeviceOpts(opts []DeviceOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func newDeviceOp(opts ...DeviceOption) *DeviceOp {
	op := &DeviceOp{
		device: v1beta1.Device{
			Spec:       v1beta1.DeviceSpec{},
			ObjectMeta: metav1.ObjectMeta{},
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiVersion,
				Kind:       deviceKind,
			},
		},
	}
	op.applyDeviceOpts(opts)
	return op
}

func withDeviceName(name string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.ObjectMeta.Name = name
	}
}

func withDeviceNamespace(namespace string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.ObjectMeta.Namespace = namespace
	}
}

func withProtocolConfig(protocol deviceProtocol) DeviceOption {
	return func(op *DeviceOp) {
		switch protocol {
		case deviceProtocolModbusRTU:
			op.device.Spec.Protocol = v1beta1.ProtocolConfig{
				Modbus: &v1beta1.ProtocolConfigModbus{
					SlaveID: pointer.Int64Ptr(1),
				},
				Common: &v1beta1.ProtocolConfigCommon{
					COM: &v1beta1.ProtocolConfigCOM{},
				},
			}
		case deviceProtocolModbusTCP:
			op.device.Spec.Protocol = v1beta1.ProtocolConfig{
				Modbus: &v1beta1.ProtocolConfigModbus{
					SlaveID: pointer.Int64Ptr(1),
				},
				Common: &v1beta1.ProtocolConfigCommon{
					TCP: &v1beta1.ProtocolConfigTCP{},
				},
			}
		case deviceProtocolBluetooth:
			op.device.Spec.Protocol = v1beta1.ProtocolConfig{
				Bluetooth: &v1beta1.ProtocolConfigBluetooth{},
			}
		case deviceProtocolOPCUA:
			op.device.Spec.Protocol = v1beta1.ProtocolConfig{
				OpcUA: &v1beta1.ProtocolConfigOpcUA{},
			}
		case deviceProtocolCustomized:
			op.device.Spec.Protocol = v1beta1.ProtocolConfig{
				CustomizedProtocol: &v1beta1.ProtocolConfigCustomized{},
			}
		default:
		}
	}
}

func withBaudRate(baudRate int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.COM.BaudRate = baudRate
	}
}

func withDataBits(dataBits int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.COM.DataBits = dataBits
	}
}

func withParity(parity string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.COM.Parity = parity
	}
}

func withSerialPort(serialPort string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.COM.SerialPort = serialPort
	}
}

func withStopBits(stopBits int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.COM.StopBits = stopBits
	}
}

func withSlaveID(slaveID int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.SlaveID = &slaveID
	}
}

func withTCPPort(port int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.TCP.Port = port
	}
}

func withTCPSlaveID(tcpSlaveID int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.SlaveID = &tcpSlaveID
	}
}

func withTCPServerIP(ip string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Common.TCP.IP = ip
	}
}

func withOPCUAServerURL(url string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.OpcUA.URL = url
	}
}

func withBluetoothMac(mac string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Bluetooth.MACAddress = mac
	}
}

func withCustromizedProtocolName(name string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.CustomizedProtocol.ProtocolName = name
	}
}

type DevicePropertiesOp struct {
	deviceProperty v1beta1.DeviceProperties
	//devicePropertyName string
	//reportCycle        int64
	//collectCycle       int64
	//visitor            v1beta1.VisitorConfig
}

type DevicePropertyVisitorOption func(*DevicePropertiesOp)

func withVisitorName(name string) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Name = name
	}
}

func withVisitorReportCycle(reportCycle int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.ReportCycle = reportCycle
	}
}

func withVisitorCollectCycle(collectCycle int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.CollectCycle = collectCycle
	}
}

func withVisitorConfig(protocol deviceProtocol) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		switch protocol {
		case deviceProtocolBluetooth:
			op.deviceProperty.Visitors = v1beta1.VisitorConfig{
				Bluetooth: &v1beta1.VisitorConfigBluetooth{
					CharacteristicUUID:     "",
					BluetoothDataConverter: v1beta1.BluetoothReadConverter{},
				},
			}
		case deviceProtocolModbus:
			op.deviceProperty.Visitors = v1beta1.VisitorConfig{
				Modbus: &v1beta1.VisitorConfigModbus{},
			}
		case deviceProtocolOPCUA:
			op.deviceProperty.Visitors = v1beta1.VisitorConfig{
				OpcUA: &v1beta1.VisitorConfigOPCUA{},
			}
		case deviceProtocolCustomized:
			op.deviceProperty.Visitors = v1beta1.VisitorConfig{
				CustomizedProtocol: &v1beta1.VisitorConfigCustomized{},
			}
		default:
		}
	}
}

func withCharacteristicUUID(characteristicUUID string) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.Bluetooth.CharacteristicUUID = characteristicUUID
	}
}

func withStartIndex(startIndex int) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.Bluetooth.BluetoothDataConverter.StartIndex = startIndex
	}
}

func withEndIndex(endIndex int) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.Bluetooth.BluetoothDataConverter.EndIndex = endIndex
	}
}

func withOperation(operationType v1beta1.BluetoothArithmeticOperationType, value float64) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		bluetoothOperation := v1beta1.BluetoothOperations{
			BluetoothOperationType:  operationType,
			BluetoothOperationValue: value,
		}
		op.deviceProperty.Visitors.Bluetooth.BluetoothDataConverter.OrderOfOperations =
			append(op.deviceProperty.Visitors.Bluetooth.BluetoothDataConverter.OrderOfOperations, bluetoothOperation)
	}
}

func withRegister(register v1beta1.ModbusRegisterType) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.Modbus.Register = register
	}
}

func withOffset(offset int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.Modbus.Offset = &offset
	}
}

func withLimit(limit int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.Modbus.Limit = &limit
	}
}

func withNodeID(nodeID string) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.OpcUA.NodeID = nodeID
	}
}

func withBrowseName(browseName string) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.OpcUA.BrowseName = browseName
	}
}

func withProtocolName(protocolName string) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.CustomizedProtocol.ProtocolName = protocolName
	}
}

func withProtocolConfigData(configData *v1beta1.CustomizedValue) DevicePropertyVisitorOption {
	return func(op *DevicePropertiesOp) {
		op.deviceProperty.Visitors.CustomizedProtocol.ConfigData = configData
	}
}

func (op *DevicePropertiesOp) applyDevicePropertiesOpts(opts []DevicePropertyVisitorOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func newDevicePropertiesOp(opts ...DevicePropertyVisitorOption) *DevicePropertiesOp {
	op := &DevicePropertiesOp{
		deviceProperty: v1beta1.DeviceProperties{},
	}
	op.applyDevicePropertiesOpts(opts)
	return op
}

func NewDeviceModbusRTU(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoBaudRate(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadBaudRate(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(100), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoDataBits(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadDataBits(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(10), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoParity(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(10), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadParity(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity("test"),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoSerialPort(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoSlaveID(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadSlaveID(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(300))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoStopBits(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadStopBits(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(3), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCP(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPServerIP("127.0.0.1"), withTCPPort(8080), withTCPSlaveID(1))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCPNoIP(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPPort(8080), withTCPSlaveID(1))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCPNoPort(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPServerIP("127.0.0.1"), withTCPSlaveID(1))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCPNoSlaveID(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPPort(8080), withTCPServerIP("127.0.0.1"))
	return deviceInstanceOp.device
}

func NewDeviceOpcUA(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolOPCUA), withOPCUAServerURL("http://test-opcuaserver.com"))
	return deviceInstanceOp.device
}

func NewDeviceOpcUANoURL(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolOPCUA))
	return deviceInstanceOp.device
}

func NewDeviceCustomized(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolCustomized), withCustromizedProtocolName("test-customized-protocol"))
	return deviceInstanceOp.device
}

func NewDeviceCustomizedNoName(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolCustomized))
	return deviceInstanceOp.device
}

func NewDeviceNoModelReference(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withProtocolConfig(deviceProtocolOPCUA))
	return deviceInstanceOp.device
}

func NewDeviceBluetoothBadOperationType(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withEndIndex(endIndex),
		withOperation("modulo", operationValue))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceBluetoothNoStartIndex(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withEndIndex(endIndex), withOperation(v1beta1.BluetoothAdd, operationValue))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceBluetoothNoEndIndex(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withOperation(v1beta1.BluetoothMultiply, operationValue))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceBluetoothNoCharacteristicUUID(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withStartIndex(startIndex), withEndIndex(endIndex), withOperation(v1beta1.BluetoothAdd, operationValue))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceModbusBadRegister(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister("test-register"), withLimit(limit), withOffset(offset))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceModbusNoLimit(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1beta1.ModbusRegisterTypeCoilRegister), withOffset(offset))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceModbusNoOffset(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1beta1.ModbusRegisterTypeCoilRegister), withLimit(limit))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceModbusNoRegister(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withLimit(limit), withOffset(offset))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceOpcUANoNodeID(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolOPCUA), withOPCUAServerURL("http://test-opcuaserver.com"))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolOPCUA),
		withBrowseName("test"))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}

func NewDeviceCustomizedNoConfigData(name string, namespace string) v1beta1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace),
		withProtocolConfig(deviceProtocolCustomized), withCustromizedProtocolName("test-customized-protocol"))

	devicePropertiesOp := newDevicePropertiesOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolCustomized),
		withProtocolName("test"))
	deviceInstanceOp.device.Spec.Properties = append(deviceInstanceOp.device.Spec.Properties, devicePropertiesOp.deviceProperty)

	return deviceInstanceOp.device
}
