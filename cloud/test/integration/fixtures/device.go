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
		case deviceProtocolOPCUA:
			op.device.Spec.Protocol = v1alpha1.ProtocolConfig{
				OpcUA: &v1alpha1.ProtocolConfigOpcUA{},
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

func NewDeviceNoModelReference(name string, namespace string) v1alpha1.Device {
	deviceInstanceOp := newDeviceOp(withDeviceName(name), withDeviceNamespace(namespace), withProtocolConfig(deviceProtocolOPCUA))
	return deviceInstanceOp.device
}
