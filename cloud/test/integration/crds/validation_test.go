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
	"context"
	"encoding/json"
	"os"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	deviceutils "github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/utils"
	"github.com/kubeedge/kubeedge/cloud/test/integration/fixtures"
)

func buildCrdClient(t *testing.T) *rest.RESTClient {
	kubeConfigPath := os.Getenv("KUBE_CONFIG")
	kubeAPIServerURL := os.Getenv("KUBE_APISERVER_URL")
	kubeConfig, err := clientcmd.BuildConfigFromFlags(kubeAPIServerURL, kubeConfigPath)
	if err != nil {
		t.Fatalf("Failed to build device CRD client, error is %v", err)
		return nil
	}
	crdClient, err := deviceutils.NewCRDClient(kubeConfig)
	if err != nil {
		t.Fatalf("Failed to build device CRD client, error is %v", err)
		return nil
	}
	return crdClient
}

func TestValidDeviceModel(t *testing.T) {
	testNamespace := os.Getenv("TESTNS")
	tests := map[string]struct {
		deviceModelFn func() *v1alpha2.DeviceModel
	}{
		"valid bluetooth device model": {
			deviceModelFn: func() *v1alpha2.DeviceModel {
				deviceModel := fixtures.NewDeviceModelBluetooth("bluetooth-device-model", testNamespace)
				return deviceModel
			},
		},
		"valid modbus rtu device model": {
			deviceModelFn: func() *v1alpha2.DeviceModel {
				deviceModel := fixtures.NewDeviceModelModbus("modbus-device-model", testNamespace)
				return deviceModel
			},
		},
		"valid opc ua device model": {
			deviceModelFn: func() *v1alpha2.DeviceModel {
				deviceModel := fixtures.NewDeviceModelOpcUA("opcua-device-model", testNamespace)
				return deviceModel
			},
		},
		"valid customized protocol device model": {
			deviceModelFn: func() *v1alpha2.DeviceModel {
				deviceModel := fixtures.NewDeviceModelCustomized("customized-device-model", testNamespace)
				return deviceModel
			},
		},
	}

	crdClient := buildCrdClient(t)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			deviceModel := tc.deviceModelFn()
			respBytes, err := json.Marshal(deviceModel)
			if err != nil {
				t.Fatalf("%s : json marshal error : %v", name, err)
			}
			result := crdClient.Post().Name(deviceModel.Name).Namespace(deviceModel.Namespace).Resource(fixtures.ResourceDeviceModel).Body(respBytes).Do(context.Background())
			if result.Error() != nil {
				t.Fatalf("%s: expected nil err , got %v", name, result.Error())
			}
		})
	}
}

func TestInvalidDeviceModel(t *testing.T) {
	testNamespace := os.Getenv("TESTNS")
	tests := map[string]struct {
		deviceModelFn func() *v1alpha2.DeviceModel
	}{
		"device model with property no name": {
			deviceModelFn: func() *v1alpha2.DeviceModel {
				deviceModel := fixtures.DeviceModelWithPropertyNoName("device-model-property-no-name", testNamespace)
				return deviceModel
			},
		},

		"device model with property bad access mode": {
			deviceModelFn: func() *v1alpha2.DeviceModel {
				deviceModel := fixtures.DeviceModelWithPropertyBadAccessMode("model-property-bad-access-mode", testNamespace)
				return deviceModel
			},
		},
	}

	crdClient := buildCrdClient(t)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			deviceModel := tc.deviceModelFn()
			deviceModelJSON, err := json.Marshal(deviceModel)
			if err != nil {
				t.Fatalf("%s : error while marshalling device model : %v", name, err)
			}
			result := crdClient.Post().Name(deviceModel.Name).Namespace(deviceModel.Namespace).Resource(fixtures.ResourceDeviceModel).
				Body(deviceModelJSON).Do(context.Background())
			if result.Error() == nil {
				t.Fatalf("%s: expected error", name)
			}
		})
	}
}

func TestValidDevice(t *testing.T) {
	testNamespace := os.Getenv("TESTNS")
	tests := map[string]struct {
		deviceInstanceFn func() v1alpha2.Device
	}{
		"valid device with modbus rtu protocol": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTU("device-modbus-rtu", testNamespace)
				return deviceInstance
			},
		},
		"valid device with modbus tcp protocol": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusTCP("device-modbus-tcp", testNamespace)
				return deviceInstance
			},
		},
		"valid device with opc ua protocol": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceOpcUA("device-opcua", testNamespace)
				return deviceInstance
			},
		},
		"valid device with customized protocol": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceCustomized("device-customized", testNamespace)
				return deviceInstance
			},
		},
	}

	crdClient := buildCrdClient(t)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			device := tc.deviceInstanceFn()
			deviceJSON, err := json.Marshal(device)
			if err != nil {
				t.Fatalf("%s : error while marshalling device : %v", name, err)
			}
			result := crdClient.Post().Name(device.Name).Namespace(device.Namespace).Resource(fixtures.ResourceDevice).Body(deviceJSON).Do(context.Background())
			if result.Error() != nil {
				t.Fatalf("%s expected nil err , got %v", name, result.Error())
			}
		})
	}
}

func TestInvalidDevice(t *testing.T) {
	testNamespace := os.Getenv("TESTNS")
	tests := map[string]struct {
		deviceInstanceFn func() v1alpha2.Device
	}{
		"device modbus rtu no baud rate": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUNoBaudRate("device-modbus-rtu-no-baud-rate", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu bad baud rate": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUBadBaudRate("device-modbus-rtu-bad-baud-rate", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu no data bits": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUNoDataBits("device-modbus-rtu-no-data-bits", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu bad data bits": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUBadDataBits("device-modbus-rtu-bad-data-bits", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu no parity": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUNoParity("device-modbus-rtu-no-parity", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu bad parity": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUBadParity("device-modbus-rtu-bad-parity", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu no serial port": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUNoSerialPort("device-modbus-rtu-no-serial-port", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu no slave id": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUNoSlaveID("device-modbus-rtu-no-slaveID", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu bad slave id": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUBadSlaveID("device-modbus-bad-slaveID", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu no stop bits": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUNoStopBits("device-modbus-rtu-no-stopbits", testNamespace)
				return deviceInstance
			},
		},
		"device modbus rtu bad_stop_bits": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusRTUBadStopBits("device-modbus-rtu-bad-stopbits", testNamespace)
				return deviceInstance
			},
		},
		"device modbus tcp no IP": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusTCPNoIP("device-modbus-tcp-no-IP", testNamespace)
				return deviceInstance
			},
		},
		"device modbus tcp no port": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusTCPNoPort("device-modbus-tcp-no-port", testNamespace)
				return deviceInstance
			},
		},
		"device modbus tcp no slaveID": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusTCPNoSlaveID("device-modbus-tcp-no-slaveID", testNamespace)
				return deviceInstance
			},
		},
		"device opcua no url": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceOpcUANoURL("device-opcua-no-url", testNamespace)
				return deviceInstance
			},
		},
		"device customized no protocol name": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceCustomizedNoName("device-customized-no-name", testNamespace)
				return deviceInstance
			},
		},
		"device no model reference": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceNoModelReference("device-no-model-ref", "default")
				return deviceInstance
			},
		},
		"device with ble protocol property bad operation type": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceBluetoothBadOperationType("device-bluetooth-bad-operation-type", testNamespace)
				return deviceInstance
			},
		},
		"device with ble protocol property no start index": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceBluetoothNoStartIndex("device-bluetooth-no-start-index", testNamespace)
				return deviceInstance
			},
		},
		"device with ble protocol property no end index": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceBluetoothNoEndIndex("device-bluetooth-bad-operation-type", testNamespace)
				return deviceInstance
			},
		},
		"device with ble protocol property no characteristic UUID": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceBluetoothNoCharacteristicUUID("device-bluetooth-no-char-uuid", testNamespace)
				return deviceInstance
			},
		},
		"device with modbus protocol property bad register": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusBadRegister("device-modbus-bad-register", testNamespace)
				return deviceInstance
			},
		},
		"device with modbus protocol property no register": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusNoRegister("device-modbus-no-register", testNamespace)
				return deviceInstance
			},
		},
		"device with modbus protocol property no limit": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusNoLimit("device-modbus-no-limit", testNamespace)
				return deviceInstance
			},
		},
		"device with ble protocol with no offset": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceModbusNoOffset("device-modbus-no-offset", testNamespace)
				return deviceInstance
			},
		},
		"device with opc ua property no nodeID": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceOpcUANoNodeID("device-modbus-no-nodeID", testNamespace)
				return deviceInstance
			},
		},
		"device with customized protocol no configData": {
			deviceInstanceFn: func() v1alpha2.Device {
				deviceInstance := fixtures.NewDeviceCustomizedNoConfigData("device-customized-no-configdata", testNamespace)
				return deviceInstance
			},
		},
	}

	crdClient := buildCrdClient(t)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			device := tc.deviceInstanceFn()
			deviceJSON, err := json.Marshal(device)
			if err != nil {
				t.Fatalf("%s : error while marshalling device : %v", name, err)
			}
			result := crdClient.Post().Name(device.Name).Namespace(device.Namespace).Resource(fixtures.ResourceDevice).Body(deviceJSON).Do(context.Background())
			if result.Error() == nil {
				t.Fatalf("%s : expected error", name)
			}
		})
	}
}
