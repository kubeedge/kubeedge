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
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DevicePropertyOp struct {
	deviceProperty v1alpha1.DeviceProperty
}

type DevicePropertyOption func(*DevicePropertyOp)

func withName(name string) DevicePropertyOption {
	return func(op *DevicePropertyOp) {
		op.deviceProperty.Name = name
	}
}

func withDescription(description string) DevicePropertyOption {
	return func(op *DevicePropertyOp) {
		op.deviceProperty.Description = description
	}
}

func withStringType(accessMode v1alpha1.PropertyAccessMode, defaultValue string) DevicePropertyOption {
	return func(op *DevicePropertyOp) {
		stringType := &v1alpha1.PropertyTypeString{
			DefaultValue: defaultValue,
		}
		stringType.AccessMode = accessMode
		op.deviceProperty.Type = v1alpha1.PropertyType{
			String: stringType,
		}
	}
}

func withIntType(accessMode v1alpha1.PropertyAccessMode, defaultValue int64, minimum int64, maximum int64, unit string) DevicePropertyOption {
	return func(op *DevicePropertyOp) {
		intType := &v1alpha1.PropertyTypeInt64{
			DefaultValue: defaultValue,
			Minimum:      minimum,
			Maximum:      maximum,
			Unit:         unit,
		}
		intType.AccessMode = accessMode
		op.deviceProperty.Type = v1alpha1.PropertyType{
			Int: intType,
		}
	}
}

func (op *DevicePropertyOp) applyDevicePropertyOpts(opts []DevicePropertyOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func newDevicePropertyOp(opts ...DevicePropertyOption) *DevicePropertyOp {
	op := &DevicePropertyOp{
		deviceProperty: v1alpha1.DeviceProperty{},
	}
	op.applyDevicePropertyOpts(opts)
	return op
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

func newDeviceModel(name string, namespace string) *v1alpha1.DeviceModel {
	spec := v1alpha1.DeviceModelSpec{}
	deviceModel := &v1alpha1.DeviceModel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       deviceModelKind,
		},
		Spec: spec,
	}
	return deviceModel
}

func DeviceModelWithPropertyNoName(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withDescription(devicePropertyTemperatureDesc),
		withStringType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), ""))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)
	return deviceModel
}

func DeviceModelWithPropertyNoType(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)
	return deviceModel
}

func DeviceModelWithPropertyBadAccessMode(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withStringType("", ""))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	return deviceModel
}

func NewDeviceModelBluetooth(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withEndIndex(endIndex),
		withOperation(v1alpha1.BluetoothAdd, operationValue))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelBluetoothBadOperationType(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withEndIndex(endIndex),
		withOperation("modulo", operationValue))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelBluetoothNoStartIndex(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withEndIndex(endIndex), withOperation(v1alpha1.BluetoothAdd, operationValue))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelBluetoothNoEndIndex(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withCharacteristicUUID(characteristicUUID), withStartIndex(startIndex), withOperation(v1alpha1.BluetoothMultiply, operationValue))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelBluetoothNoCharacteristicUUID(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolBluetooth),
		withStartIndex(startIndex), withEndIndex(endIndex), withOperation(v1alpha1.BluetoothAdd, operationValue))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelModbus(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1alpha1.ModbusRegisterTypeCoilRegister), withLimit(limit), withOffset(offset))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelModbusBadRegister(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister("test-register"), withLimit(limit), withOffset(offset))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelModbusNoLimit(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1alpha1.ModbusRegisterTypeCoilRegister), withOffset(offset))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)
	return deviceModel
}

func NewDeviceModelModbusNoOffset(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withRegister(v1alpha1.ModbusRegisterTypeCoilRegister), withLimit(limit))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelModbusNoRegister(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolModbus),
		withLimit(limit), withOffset(offset))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)

	return deviceModel
}

func NewDeviceModelOpcUA(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolOPCUA),
		withBrowseName("test"), withNodeID("node1"))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)
	return deviceModel
}

func NewDeviceModelOpcUANoNodeID(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolOPCUA),
		withBrowseName("test"))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)
	return deviceModel
}

func NewDeviceModelCustomized(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	definition := &v1alpha1.CustomizedValue{"config1": "config-val1", "config2": "config-val2"}
	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolCustomized),
		withProtocolName("test"), withProtocolDefinition(definition))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)
	return deviceModel
}

func NewDeviceModelCustomizedNoDefinition(name string, namespace string) *v1alpha1.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha1.PropertyAccessMode(v1alpha1.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	devicePropertyVisitorOp := newDevicePropVisitorOp(withVisitorName(devicePropertyTemperature),
		withVisitorCollectCycle(collectCycle),
		withVisitorReportCycle(reportCycle),
		withVisitorConfig(deviceProtocolCustomized),
		withProtocolName("test"))
	deviceModel.Spec.PropertyVisitors = append(deviceModel.Spec.PropertyVisitors, devicePropertyVisitorOp.devicePropertyVisitor)
	return deviceModel
}
