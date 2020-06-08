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
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
)

type DeviceOp struct {
	device v1alpha1.Device
}

type DeviceOption func(*DeviceOp)

func (op *DeviceOp) applyDeviceOpts(opts []DeviceOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func newDeviceOp(opts ...DeviceOption) *DeviceOp {
	op := &DeviceOp{
		device: v1alpha1.Device{
			Spec:       v1alpha1.DeviceSpec{},
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

func withDeviceModelReference(deviceModelRef string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.DeviceModelRef = &v1.LocalObjectReference{
			Name: deviceModelRef,
		}
	}
}

func withProtocolConfig(protocol deviceProtocol) DeviceOption {
	return func(op *DeviceOp) {
		switch protocol {
		case deviceProtocolModbusRTU:
			op.device.Spec.Protocol = v1alpha1.ProtocolConfig{
				Modbus: &v1alpha1.ProtocolConfigModbus{
					RTU: &v1alpha1.ProtocolConfigModbusRTU{},
				},
			}
		case deviceProtocolModbusTCP:
			op.device.Spec.Protocol = v1alpha1.ProtocolConfig{
				Modbus: &v1alpha1.ProtocolConfigModbus{
					TCP: &v1alpha1.ProtocolConfigModbusTCP{},
				},
			}
		case deviceProtocolBluetooth:
			op.device.Spec.Protocol = v1alpha1.ProtocolConfig{
				Bluetooth: &v1alpha1.ProtocolConfigBluetooth{},
			}
		case deviceProtocolOPCUA:
			op.device.Spec.Protocol = v1alpha1.ProtocolConfig{
				OpcUA: &v1alpha1.ProtocolConfigOpcUA{},
			}
		case deviceProtocolCustomized:
			op.device.Spec.Protocol = v1alpha1.ProtocolConfig{
				CustomizedProtocol: &v1alpha1.ProtocolConfigCustomized{},
			}
		default:
		}
	}
}

func withBaudRate(baudRate int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.RTU.BaudRate = baudRate
	}
}

func withDataBits(dataBits int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.RTU.DataBits = dataBits
	}
}

func withParity(parity string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.RTU.Parity = parity
	}
}

func withSerialPort(serialPort string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.RTU.SerialPort = serialPort
	}
}

func withStopBits(stopBits int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.RTU.StopBits = stopBits
	}
}

func withSlaveID(slaveID int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.RTU.SlaveID = slaveID
	}
}

func withTCPPort(port int64) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.TCP.Port = port
	}
}

func withTCPSlaveID(tcpSlaveID string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.TCP.SlaveID = tcpSlaveID
	}
}

func withTCPServerIP(ip string) DeviceOption {
	return func(op *DeviceOp) {
		op.device.Spec.Protocol.Modbus.TCP.IP = ip
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

type DevicePropertyVisitorOp struct {
	devicePropertyVisitor v1alpha1.DevicePropertyVisitor
}

type DevicePropertyVisitorOption func(*DevicePropertyVisitorOp)

func withVisitorName(name string) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.PropertyName = name
	}
}

func withVisitorReportCycle(reportCycle int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.ReportCycle = reportCycle
	}
}

func withVisitorCollectCycle(collectCycle int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.CollectCycle = collectCycle
	}
}

func withVisitorConfig(protocol deviceProtocol) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		switch protocol {
		case deviceProtocolBluetooth:
			op.devicePropertyVisitor.VisitorConfig = v1alpha1.VisitorConfig{
				Bluetooth: &v1alpha1.VisitorConfigBluetooth{
					CharacteristicUUID:     "",
					BluetoothDataConverter: v1alpha1.BluetoothReadConverter{},
				},
			}
		case deviceProtocolModbus:
			op.devicePropertyVisitor.VisitorConfig = v1alpha1.VisitorConfig{
				Modbus: &v1alpha1.VisitorConfigModbus{},
			}
		case deviceProtocolOPCUA:
			op.devicePropertyVisitor.VisitorConfig = v1alpha1.VisitorConfig{
				OpcUA: &v1alpha1.VisitorConfigOPCUA{},
			}
		case deviceProtocolCustomized:
			op.devicePropertyVisitor.VisitorConfig = v1alpha1.VisitorConfig{
				CustomizedProtocol: &v1alpha1.VisitorConfigCustomized{},
			}
		default:
		}
	}
}

func withCharacteristicUUID(characteristicUUID string) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.Bluetooth.CharacteristicUUID = characteristicUUID
	}
}

func withStartIndex(startIndex int) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.Bluetooth.BluetoothDataConverter.StartIndex = startIndex
	}
}

func withEndIndex(endIndex int) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.Bluetooth.BluetoothDataConverter.EndIndex = endIndex
	}
}

func withOperation(operationType v1alpha1.BluetoothArithmeticOperationType, value float64) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		bluetoothOperation := v1alpha1.BluetoothOperations{
			BluetoothOperationType:  operationType,
			BluetoothOperationValue: value,
		}
		op.devicePropertyVisitor.VisitorConfig.Bluetooth.BluetoothDataConverter.OrderOfOperations =
			append(op.devicePropertyVisitor.VisitorConfig.Bluetooth.BluetoothDataConverter.OrderOfOperations, bluetoothOperation)
	}
}

func withRegister(register v1alpha1.ModbusRegisterType) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.Modbus.Register = register
	}
}

func withOffset(offset int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.Modbus.Offset = offset
	}
}

func withLimit(limit int64) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.Modbus.Limit = limit
	}
}

func withNodeID(nodeID string) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.OpcUA.NodeID = nodeID
	}
}

func withBrowseName(browseName string) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.OpcUA.BrowseName = browseName
	}
}

func withProtocolName(protocolName string) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.CustomizedProtocol.ProtocolName = protocolName
	}
}

func withProtocolDefinition(definition *v1alpha1.CustomizedValue) DevicePropertyVisitorOption {
	return func(op *DevicePropertyVisitorOp) {
		op.devicePropertyVisitor.VisitorConfig.CustomizedProtocol.Definition = definition
	}
}

func (op *DevicePropertyVisitorOp) applyDevicePropVisitorOpts(opts []DevicePropertyVisitorOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func newDevicePropVisitorOp(opts ...DevicePropertyVisitorOption) *DevicePropertyVisitorOp {
	op := &DevicePropertyVisitorOp{
		devicePropertyVisitor: v1alpha1.DevicePropertyVisitor{},
	}
	op.applyDevicePropVisitorOpts(opts)
	return op
}

func NewDeviceModbusRTU(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoBaudRate(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadBaudRate(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(100), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoDataBits(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadDataBits(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(10), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoParity(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(10), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadParity(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity("test"),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoSerialPort(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withStopBits(stopBits), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoSlaveID(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadSlaveID(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(stopBits), withSlaveID(300))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUNoStopBits(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusRTUBadStopBits(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity),
		withSerialPort(serialPort), withStopBits(3), withSlaveID(slaveID))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCP(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPServerIP("127.0.0.1"), withTCPPort(8080), withTCPSlaveID("1"))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCPNoIP(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPPort(8080), withTCPSlaveID("1"))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCPNoPort(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPServerIP("127.0.0.1"), withTCPSlaveID("1"))
	return deviceInstanceOp.device
}

func NewDeviceModbusTCPNoSlaveID(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusTCP), withTCPPort(8080), withTCPServerIP("127.0.0.1"))
	return deviceInstanceOp.device
}

func NewDeviceOpcUA(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolOPCUA), withOPCUAServerURL("http://test-opcuaserver.com"))
	return deviceInstanceOp.device
}

func NewDeviceOpcUANoURL(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolOPCUA))
	return deviceInstanceOp.device
}

func NewDeviceCustomized(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolCustomized), withCustromizedProtocolName("test-customized-protocol"))
	return deviceInstanceOp.device
}

func NewDeviceCustomizedNoName(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolCustomized))
	return deviceInstanceOp.device
}

func NewDeviceNoModelReference(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withProtocolConfig(deviceProtocolOPCUA))
	return deviceInstanceOp.device
}

func NewDeviceBluetoothBadOperationType(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withEndIndex(endIndex),
		withOperation("modulo", operationValue))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceBluetoothNoStartIndex(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withEndIndex(endIndex), withOperation(v1alpha1.BluetoothAdd, operationValue))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceBluetoothNoEndIndex(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withOperation(v1alpha1.BluetoothMultiply, operationValue))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceBluetoothNoCharacteristicUUID(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolBluetooth), withBluetoothMac("BC:6A:29:AE:CC:96"))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withStartIndex(startIndex), withEndIndex(endIndex), withOperation(v1alpha1.BluetoothAdd, operationValue))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceModbusBadRegister(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister("test-register"), withLimit(limit), withOffset(offset))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceModbusNoLimit(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1alpha1.ModbusRegisterTypeCoilRegister), withOffset(offset))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceModbusNoOffset(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1alpha1.ModbusRegisterTypeCoilRegister), withLimit(limit))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceModbusNoRegister(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolModbusRTU), withBaudRate(baudRate), withDataBits(dataBits), withParity(parity), withSerialPort(serialPort),
		withStopBits(stopBits), withSlaveID(slaveID))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withLimit(limit), withOffset(offset))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceOpcUANoNodeID(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolOPCUA), withOPCUAServerURL("http://test-opcuaserver.com"))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolOPCUA),
		withBrowseName("test"))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}

func NewDeviceCustomizedNoDefinition(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withDeviceModelReference(DeviceModelRef),
		withProtocolConfig(deviceProtocolCustomized), withCustromizedProtocolName("test-customized-protocol"))

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolCustomized),
		withProtocolName("test"))
	deviceInstanceOp.device.Spec.PropertyVisitors = append(deviceInstanceOp.device.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceInstanceOp.device
}
