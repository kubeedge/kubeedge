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

package crds

import (
	"os"
	"testing"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	cloudcoreConfig "github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

func buildCrdClient(t *testing.T) crdClientset.Interface {
	kubeConfigPath := os.Getenv("KUBE_CONFIG")
	kubeAPIServerURL := os.Getenv("KUBE_APISERVER_URL")

	client.InitKubeEdgeClient(&cloudcoreConfig.KubeAPIConfig{KubeConfig: kubeConfigPath, Master: kubeAPIServerURL})

	return client.GetCRDClient()
}

//func TestValidDevice(t *testing.T) {
//	testNamespace := os.Getenv("TESTNS")
//	tests := map[string]struct {
//		deviceInstanceFn func() v1alpha2.Device
//	}{
//		"valid device with modbus rtu protocol": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTU("device-modbus-rtu", testNamespace)
//				return deviceInstance
//			},
//		},
//		"valid device with modbus tcp protocol": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusTCP("device-modbus-tcp", testNamespace)
//				return deviceInstance
//			},
//		},
//		"valid device with opc ua protocol": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceOpcUA("device-opcua", testNamespace)
//				return deviceInstance
//			},
//		},
//		"valid device with customized protocol": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceCustomized("device-customized", testNamespace)
//				return deviceInstance
//			},
//		},
//	}
//
//	crdClient := buildCrdClient(t)
//
//	for name, tc := range tests {
//		t.Run(name, func(t *testing.T) {
//			device := tc.deviceInstanceFn()
//			_, err := crdClient.DevicesV1alpha2().Devices(device.Namespace).Create(context.TODO(), &device, v1.CreateOptions{})
//			if err != nil {
//				t.Fatalf("%s expected nil err , got %v", name, err)
//			}
//		})
//	}
//}
//
//func TestInvalidDevice(t *testing.T) {
//	testNamespace := os.Getenv("TESTNS")
//	tests := map[string]struct {
//		deviceInstanceFn func() v1alpha2.Device
//	}{
//		"device modbus rtu no baud rate": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUNoBaudRate("device-modbus-rtu-no-baud-rate", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu bad baud rate": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUBadBaudRate("device-modbus-rtu-bad-baud-rate", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu no data bits": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUNoDataBits("device-modbus-rtu-no-data-bits", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu bad data bits": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUBadDataBits("device-modbus-rtu-bad-data-bits", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu no parity": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUNoParity("device-modbus-rtu-no-parity", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu bad parity": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUBadParity("device-modbus-rtu-bad-parity", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu no serial port": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUNoSerialPort("device-modbus-rtu-no-serial-port", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu no slave id": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUNoSlaveID("device-modbus-rtu-no-slaveID", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu bad slave id": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUBadSlaveID("device-modbus-bad-slaveID", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu no stop bits": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUNoStopBits("device-modbus-rtu-no-stopbits", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus rtu bad_stop_bits": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusRTUBadStopBits("device-modbus-rtu-bad-stopbits", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus tcp no IP": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusTCPNoIP("device-modbus-tcp-no-IP", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus tcp no port": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusTCPNoPort("device-modbus-tcp-no-port", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device modbus tcp no slaveID": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusTCPNoSlaveID("device-modbus-tcp-no-slaveID", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device opcua no url": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceOpcUANoURL("device-opcua-no-url", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device customized no protocol name": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceCustomizedNoName("device-customized-no-name", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device no model reference": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceNoModelReference("device-no-model-ref", "default")
//				return deviceInstance
//			},
//		},
//		"device with ble protocol property bad operation type": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceBluetoothBadOperationType("device-bluetooth-bad-operation-type", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with ble protocol property no start index": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceBluetoothNoStartIndex("device-bluetooth-no-start-index", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with ble protocol property no end index": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceBluetoothNoEndIndex("device-bluetooth-bad-operation-type", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with ble protocol property no characteristic UUID": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceBluetoothNoCharacteristicUUID("device-bluetooth-no-char-uuid", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with modbus protocol property bad register": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusBadRegister("device-modbus-bad-register", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with modbus protocol property no register": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusNoRegister("device-modbus-no-register", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with modbus protocol property no limit": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusNoLimit("device-modbus-no-limit", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with ble protocol with no offset": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceModbusNoOffset("device-modbus-no-offset", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with opc ua property no nodeID": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceOpcUANoNodeID("device-modbus-no-nodeID", testNamespace)
//				return deviceInstance
//			},
//		},
//		"device with customized protocol no configData": {
//			deviceInstanceFn: func() v1alpha2.Device {
//				deviceInstance := fixtures.NewDeviceCustomizedNoConfigData("device-customized-no-configdata", testNamespace)
//				return deviceInstance
//			},
//		},
//	}
//
//	crdClient := buildCrdClient(t)
//
//	for name, tc := range tests {
//		t.Run(name, func(t *testing.T) {
//			device := tc.deviceInstanceFn()
//			_, err := crdClient.DevicesV1alpha2().Devices(device.Namespace).Create(context.TODO(), &device, v1.CreateOptions{})
//			if err == nil {
//				t.Fatalf("%s : expected error", name)
//			}
//		})
//	}
//}
