package utils

import (
	"encoding/json"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
)

func NewLedDeviceModel() v1alpha1.DeviceModel {
	deviceProperty1 := v1alpha1.DeviceProperty{
		Name:        "power-status",
		Description: "Indicates whether the led light is ON/OFF",
		Type: v1alpha1.PropertyType{String: &v1alpha1.PropertyTypeString{
			AccessMode:   "ReadWrite",
			DefaultValue: "OFF",
		}},
	}
	deviceProperty2 := v1alpha1.DeviceProperty{
		Name:        "gpio-pin-number",
		Description: "Indicates the GPIO pin to which LED is connected",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadOnly",
			DefaultValue: 18,
		}},
	}
	properties := []v1alpha1.DeviceProperty{deviceProperty1, deviceProperty2}
	newDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light",
			Namespace: Namespace,
		},
		Spec: v1alpha1.DeviceModelSpec{
			Properties: properties,
		},
	}
	return newDeviceModel
}

func NewModbusDeviceModel() v1alpha1.DeviceModel {
	deviceProperty1 := v1alpha1.DeviceProperty{
		Name:        "temperature",
		Description: "temperature in degree celsius",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode: "ReadWrite",
			Maximum:    100,
			Unit:       "degree celsius",
		}},
	}
	deviceProperty2 := v1alpha1.DeviceProperty{
		Name:        "temperature-enable",
		Description: "enable data collection of temperature sensor",
		Type: v1alpha1.PropertyType{String: &v1alpha1.PropertyTypeString{
			AccessMode:   "ReadWrite",
			DefaultValue: "OFF",
		}},
	}
	properties := []v1alpha1.DeviceProperty{deviceProperty1, deviceProperty2}
	devicePropertyVisitor1 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature",
		VisitorConfig: v1alpha1.VisitorConfig{
			Modbus: &v1alpha1.VisitorConfigModbus{
				Register:       "CoilRegister",
				Offset:         2,
				Limit:          1,
				Scale:          1,
				IsSwap:         true,
				IsRegisterSwap: true,
			},
		},
	}
	devicePropertyVisitor2 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature-enable",
		VisitorConfig: v1alpha1.VisitorConfig{
			Modbus: &v1alpha1.VisitorConfigModbus{
				Register:       "DiscreteInputRegister",
				Offset:         3,
				Limit:          1,
				Scale:          1.0,
				IsSwap:         true,
				IsRegisterSwap: true,
			},
		},
	}
	propertyVisitors := []v1alpha1.DevicePropertyVisitor{devicePropertyVisitor1, devicePropertyVisitor2}
	newDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-model",
			Namespace: Namespace,
		},
		Spec: v1alpha1.DeviceModelSpec{
			Properties:       properties,
			PropertyVisitors: propertyVisitors,
		},
	}
	return newDeviceModel
}

func NewBluetoothDeviceModel() v1alpha1.DeviceModel {
	deviceProperty1 := v1alpha1.DeviceProperty{
		Name:        "temperature",
		Description: "temperature in degree celsius",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode: "ReadOnly",
			Maximum:    100,
			Unit:       "degree celsius",
		}},
	}
	deviceProperty2 := v1alpha1.DeviceProperty{
		Name:        "temperature-enable",
		Description: "enable data collection of temperature sensor",
		Type: v1alpha1.PropertyType{String: &v1alpha1.PropertyTypeString{
			AccessMode:   "ReadWrite",
			DefaultValue: "ON",
		}},
	}
	deviceProperty3 := v1alpha1.DeviceProperty{
		Name:        "io-config-initialize",
		Description: "initialize io-config with value 0",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 0,
		}},
	}
	deviceProperty4 := v1alpha1.DeviceProperty{
		Name:        "io-data-initialize",
		Description: "initialize io-data with value 0",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 0,
		}},
	}
	deviceProperty5 := v1alpha1.DeviceProperty{
		Name:        "io-config",
		Description: "register activation of io-config",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 1,
		}},
	}
	deviceProperty6 := v1alpha1.DeviceProperty{
		Name:        "io-data",
		Description: "data field to control io-control",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 0,
		}},
	}
	properties := []v1alpha1.DeviceProperty{deviceProperty1, deviceProperty2, deviceProperty3, deviceProperty4, deviceProperty5, deviceProperty6}
	devicePropertyVisitor1 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa0104514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 2,
					EndIndex:   1,
					ShiftRight: 2,
					OrderOfOperations: []v1alpha1.BluetoothOperations{
						{
							BluetoothOperationType:  "Multiply",
							BluetoothOperationValue: 0.03125,
						},
					},
				},
			},
		},
	}
	devicePropertyVisitor2 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature-enable",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa0204514000b000000000000000",
				DataWriteToBluetooth: map[string][]byte{
					"ON":  {1},
					"OFF": {0},
				},
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor3 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-config-initialize",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6604514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor4 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-data-initialize",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6504514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor5 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-config",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6604514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor6 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-data",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6504514000b000000000000000",
				DataWriteToBluetooth: map[string][]byte{
					"Red":            {1},
					"Green":          {2},
					"RedGreen":       {3},
					"Buzzer":         {4},
					"BuzzerRed":      {5},
					"BuzzerGreen":    {6},
					"BuzzerRedGreen": {7},
				},
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	propertyVisitors := []v1alpha1.DevicePropertyVisitor{devicePropertyVisitor1, devicePropertyVisitor2, devicePropertyVisitor3, devicePropertyVisitor4, devicePropertyVisitor5, devicePropertyVisitor6}
	newDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "cc2650-sensortag",
			Namespace: Namespace,
		},
		Spec: v1alpha1.DeviceModelSpec{
			Properties:       properties,
			PropertyVisitors: propertyVisitors,
		},
	}
	return newDeviceModel
}

func UpdatedLedDeviceModel() v1alpha1.DeviceModel {
	deviceProperty1 := v1alpha1.DeviceProperty{
		Name:        "power-status",
		Description: "Indicates whether the led light is ON/OFF",
		Type: v1alpha1.PropertyType{String: &v1alpha1.PropertyTypeString{
			AccessMode:   "ReadWrite",
			DefaultValue: "ON",
		}},
	}
	deviceProperty2 := v1alpha1.DeviceProperty{
		Name:        "gpio-pin-number",
		Description: "Indicates the GPIO pin to which LED is connected",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 17,
		}},
	}
	properties := []v1alpha1.DeviceProperty{deviceProperty1, deviceProperty2}
	updatedDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light",
			Namespace: Namespace,
		},
		Spec: v1alpha1.DeviceModelSpec{
			Properties: properties,
		},
	}
	return updatedDeviceModel
}

func UpdatedModbusDeviceModel() v1alpha1.DeviceModel {
	deviceProperty1 := v1alpha1.DeviceProperty{
		Name:        "temperature",
		Description: "temperature in degree",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode: "ReadOnly",
			Maximum:    200,
			Unit:       "celsius",
		}},
	}
	deviceProperty2 := v1alpha1.DeviceProperty{
		Name:        "temperature-enable",
		Description: "enable data collection of temperature sensor",
		Type: v1alpha1.PropertyType{String: &v1alpha1.PropertyTypeString{
			AccessMode:   "ReadWrite",
			DefaultValue: "ON",
		}},
	}
	properties := []v1alpha1.DeviceProperty{deviceProperty1, deviceProperty2}
	devicePropertyVisitor1 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature",
		VisitorConfig: v1alpha1.VisitorConfig{
			Modbus: &v1alpha1.VisitorConfigModbus{
				Register:       "CoilRegister",
				Offset:         2,
				Limit:          1,
				Scale:          2,
				IsSwap:         false,
				IsRegisterSwap: true,
			},
		},
	}
	devicePropertyVisitor2 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature-enable",
		VisitorConfig: v1alpha1.VisitorConfig{
			Modbus: &v1alpha1.VisitorConfigModbus{
				Register:       "DiscreteInputRegister",
				Offset:         1,
				Limit:          1,
				Scale:          1.0,
				IsSwap:         true,
				IsRegisterSwap: false,
			},
		},
	}
	propertyVisitors := []v1alpha1.DevicePropertyVisitor{devicePropertyVisitor1, devicePropertyVisitor2}
	newDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-model",
			Namespace: Namespace,
		},
		Spec: v1alpha1.DeviceModelSpec{
			Properties:       properties,
			PropertyVisitors: propertyVisitors,
		},
	}
	return newDeviceModel
}

func UpdatedBluetoothDeviceModel() v1alpha1.DeviceModel {
	deviceProperty1 := v1alpha1.DeviceProperty{
		Name:        "temperature",
		Description: "temperature in degree",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode: "ReadOnly",
			Maximum:    200,
			Unit:       "degree",
		}},
	}
	deviceProperty2 := v1alpha1.DeviceProperty{
		Name:        "temperature-enable",
		Description: "enable data collection of temperature sensor",
		Type: v1alpha1.PropertyType{String: &v1alpha1.PropertyTypeString{
			AccessMode:   "ReadWrite",
			DefaultValue: "OFF",
		}},
	}
	deviceProperty3 := v1alpha1.DeviceProperty{
		Name:        "io-config-initialize",
		Description: "initialize io-config with value 0",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 0,
		}},
	}
	deviceProperty4 := v1alpha1.DeviceProperty{
		Name:        "io-data-initialize",
		Description: "initialize io-data with value 0",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 0,
		}},
	}
	deviceProperty5 := v1alpha1.DeviceProperty{
		Name:        "io-config",
		Description: "register activation of io-config",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 1,
		}},
	}
	deviceProperty6 := v1alpha1.DeviceProperty{
		Name:        "io-data",
		Description: "data field to control io-control",
		Type: v1alpha1.PropertyType{Int: &v1alpha1.PropertyTypeInt64{
			AccessMode:   "ReadWrite",
			DefaultValue: 0,
		}},
	}
	properties := []v1alpha1.DeviceProperty{deviceProperty1, deviceProperty2, deviceProperty3, deviceProperty4, deviceProperty5, deviceProperty6}
	devicePropertyVisitor1 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa0104514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   3,
					ShiftRight: 1,
					OrderOfOperations: []v1alpha1.BluetoothOperations{
						{
							BluetoothOperationType:  "Multiply",
							BluetoothOperationValue: 0.05,
						},
					},
				},
			},
		},
	}
	devicePropertyVisitor2 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "temperature-enable",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa0204514000b000000000000000",
				DataWriteToBluetooth: map[string][]byte{
					"On":  {1},
					"OFF": {0},
				},
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor3 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-config-initialize",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6604514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor4 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-data-initialize",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6504514000b000000000000001",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor5 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-config",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6604514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	devicePropertyVisitor6 := v1alpha1.DevicePropertyVisitor{
		PropertyName: "io-data",
		VisitorConfig: v1alpha1.VisitorConfig{
			Bluetooth: &v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6504514000b000000000000000",
				DataWriteToBluetooth: map[string][]byte{
					"Red":            {2},
					"Green":          {3},
					"RedGreen":       {4},
					"Buzzer":         {5},
					"BuzzerRed":      {6},
					"BuzzerGreen":    {7},
					"BuzzerRedGreen": {8},
				},
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}
	propertyVisitors := []v1alpha1.DevicePropertyVisitor{devicePropertyVisitor1, devicePropertyVisitor2, devicePropertyVisitor3, devicePropertyVisitor4, devicePropertyVisitor5, devicePropertyVisitor6}
	newDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "cc2650-sensortag",
			Namespace: Namespace,
		},
		Spec: v1alpha1.DeviceModelSpec{
			Properties:       properties,
			PropertyVisitors: propertyVisitors,
		},
	}
	return newDeviceModel
}

func NewLedDeviceInstance(nodeSelector string) v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light-instance-01",
			Namespace: Namespace,
			Labels: map[string]string{
				"description": "LEDLight",
				"model":       "led-light",
			},
		},
		Spec: v1alpha1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "led-light",
			},
			NodeSelector: &v12.NodeSelector{
				NodeSelectorTerms: []v12.NodeSelectorTerm{
					{
						MatchExpressions: []v12.NodeSelectorRequirement{
							{
								Key:      "",
								Operator: v12.NodeSelectorOpIn,
								Values:   []string{nodeSelector},
							},
						},
					},
				},
			},
		},
		Status: v1alpha1.DeviceStatus{
			Twins: []v1alpha1.Twin{
				{
					PropertyName: "power-status",
					Desired: v1alpha1.TwinProperty{
						Value: "ON",
						Metadata: map[string]string{
							"type": "string",
						},
					},
					Reported: v1alpha1.TwinProperty{
						Value: "unknown",
					},
				},
			},
		},
	}

	return deviceInstance
}

func NewModbusDeviceInstance(nodeSelector string) v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-instance-02",
			Namespace: Namespace,
			Labels: map[string]string{
				"description":  "TISimplelinkSensorTag",
				"manufacturer": "TexasInstruments",
				"model":        "CC2650",
			},
		},
		Spec: v1alpha1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "sensor-tag-model",
			},
			NodeSelector: &v12.NodeSelector{
				NodeSelectorTerms: []v12.NodeSelectorTerm{
					{
						MatchExpressions: []v12.NodeSelectorRequirement{
							{
								Key:      "",
								Operator: v12.NodeSelectorOpIn,
								Values:   []string{nodeSelector},
							},
						},
					},
				},
			},
		},
		Status: v1alpha1.DeviceStatus{
			Twins: []v1alpha1.Twin{
				{
					PropertyName: "temperature-enable",
					Desired: v1alpha1.TwinProperty{
						Value: "OFF",
						Metadata: map[string]string{
							"type": "string",
						},
					},
					Reported: v1alpha1.TwinProperty{
						Value: "unknown",
					},
				},
			},
		},
	}
	return deviceInstance
}

func NewBluetoothDeviceInstance(nodeSelector string) v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-instance-01",
			Namespace: Namespace,
			Labels: map[string]string{
				"description":  "TISimplelinkSensorTag",
				"manufacturer": "TexasInstruments",
				"model":        "cc2650-sensortag",
			},
		},
		Spec: v1alpha1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "cc2650-sensortag",
			},
			NodeSelector: &v12.NodeSelector{
				NodeSelectorTerms: []v12.NodeSelectorTerm{
					{
						MatchExpressions: []v12.NodeSelectorRequirement{
							{
								Key:      "",
								Operator: v12.NodeSelectorOpIn,
								Values:   []string{nodeSelector},
							},
						},
					},
				},
			},
			Protocol: v1alpha1.ProtocolConfig{
				Bluetooth: &v1alpha1.ProtocolConfigBluetooth{
					MACAddress: "BC:6A:29:AE:CC:96",
				},
			},
		},
		Status: v1alpha1.DeviceStatus{
			Twins: []v1alpha1.Twin{
				{
					PropertyName: "io-data",
					Desired: v1alpha1.TwinProperty{
						Value: "1",
						Metadata: map[string]string{
							"type": "int",
						},
					},
					Reported: v1alpha1.TwinProperty{
						Value: "unknown",
					},
				},
			},
		},
	}
	return deviceInstance
}

func UpdatedLedDeviceInstance(nodeSelector string) v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light-instance-01",
			Namespace: Namespace,
			Labels: map[string]string{
				"description": "LEDLight-1",
				"model":       "led-light-1",
			},
		},
		Spec: v1alpha1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "led-light",
			},
			NodeSelector: &v12.NodeSelector{
				NodeSelectorTerms: []v12.NodeSelectorTerm{
					{
						MatchExpressions: []v12.NodeSelectorRequirement{
							{
								Key:      "",
								Operator: v12.NodeSelectorOpIn,
								Values:   []string{nodeSelector},
							},
						},
					},
				},
			},
		},
		Status: v1alpha1.DeviceStatus{
			Twins: []v1alpha1.Twin{
				{
					PropertyName: "power-status",
					Desired: v1alpha1.TwinProperty{
						Value: "OFF",
						Metadata: map[string]string{
							"type": "string",
						},
					},
					Reported: v1alpha1.TwinProperty{
						Value: "unknown",
					},
				},
			},
		},
	}
	return deviceInstance
}

func UpdatedModbusDeviceInstance(nodeSelector string) v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-instance-02",
			Namespace: Namespace,
			Labels: map[string]string{
				"description":  "TISensorTag",
				"manufacturer": "TexasInstruments-TI",
				"model":        "CC2650-sensorTag",
			},
		},
		Spec: v1alpha1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "sensor-tag-model",
			},
			NodeSelector: &v12.NodeSelector{
				NodeSelectorTerms: []v12.NodeSelectorTerm{
					{
						MatchExpressions: []v12.NodeSelectorRequirement{
							{
								Key:      "",
								Operator: v12.NodeSelectorOpIn,
								Values:   []string{nodeSelector},
							},
						},
					},
				},
			},
		},
		Status: v1alpha1.DeviceStatus{
			Twins: []v1alpha1.Twin{
				{
					PropertyName: "temperature-enable",
					Desired: v1alpha1.TwinProperty{
						Value: "ON",
						Metadata: map[string]string{
							"type": "string",
						},
					},
					Reported: v1alpha1.TwinProperty{
						Value: "unknown",
					},
				},
			},
		},
	}
	return deviceInstance
}

func UpdatedBluetoothDeviceInstance(nodeSelector string) v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-instance-01",
			Namespace: Namespace,
			Labels: map[string]string{
				"description":  "TISensorTag",
				"manufacturer": "TexasInstruments-TI",
				"model":        "cc2650-sensor-tag",
			},
		},
		Spec: v1alpha1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "cc2650-sensortag",
			},
			NodeSelector: &v12.NodeSelector{
				NodeSelectorTerms: []v12.NodeSelectorTerm{
					{
						MatchExpressions: []v12.NodeSelectorRequirement{
							{
								Key:      "",
								Operator: v12.NodeSelectorOpIn,
								Values:   []string{nodeSelector},
							},
						},
					},
				},
			},
			Protocol: v1alpha1.ProtocolConfig{
				Bluetooth: &v1alpha1.ProtocolConfigBluetooth{
					MACAddress: "BC:6A:29:AE:CC:69",
				},
			},
		},
		Status: v1alpha1.DeviceStatus{
			Twins: []v1alpha1.Twin{
				{
					PropertyName: "io-data",
					Desired: v1alpha1.TwinProperty{
						Value: "1",
						Metadata: map[string]string{
							"type": "int",
						},
					},
					Reported: v1alpha1.TwinProperty{
						Value: "unknown",
					},
				},
			},
		},
	}
	return deviceInstance
}

func IncorrectDeviceModel() v1alpha1.DeviceModel {
	newDeviceModel := v1alpha1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "device-model",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light",
			Namespace: Namespace,
		},
	}
	return newDeviceModel
}

func IncorrectDeviceInstance() v1alpha1.Device {
	deviceInstance := v1alpha1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "device",
			APIVersion: "devices.kubeedge.io/v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light-instance-01",
			Namespace: Namespace,
			Labels: map[string]string{
				"description": "LEDLight",
				"model":       "led-light",
			},
		},
	}
	return deviceInstance
}

func NewConfigMapLED(nodeSelector string) v12.ConfigMap {
	configMap := v12.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "device-profile-config-" + nodeSelector,
			Namespace: Namespace,
		},
	}
	configMap.Data = make(map[string]string)

	deviceProfile := &types.DeviceProfile{}
	deviceProfile.DeviceInstances = []*types.DeviceInstance{
		{
			Name:  "led-light-instance-01",
			ID:    "led-light-instance-01",
			Model: "led-light",
		},
	}
	deviceProfile.DeviceModels = []*types.DeviceModel{
		{
			Name: "led-light",
			Properties: []*types.Property{
				{
					Name:         "power-status",
					DataType:     "string",
					Description:  "Indicates whether the led light is ON/OFF",
					AccessMode:   "ReadWrite",
					DefaultValue: "OFF",
				},
				{
					Name:         "gpio-pin-number",
					DataType:     "int",
					Description:  "Indicates the GPIO pin to which LED is connected",
					AccessMode:   "ReadOnly",
					DefaultValue: 18,
				},
			},
		},
	}
	deviceProfile.Protocols = []*types.Protocol{
		{
			ProtocolConfig: nil,
		},
	}

	bytes, err := json.Marshal(deviceProfile)
	if err != nil {
		Err("Failed to marshal deviceprofile: %v", deviceProfile)
	}
	configMap.Data["deviceProfile.json"] = string(bytes)

	return configMap
}

func NewConfigMapBluetooth(nodeSelector string) v12.ConfigMap {
	configMap := v12.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "device-profile-config-" + nodeSelector,
			Namespace: Namespace,
		},
	}
	configMap.Data = make(map[string]string)

	deviceProfile := &types.DeviceProfile{}
	deviceProfile.DeviceInstances = []*types.DeviceInstance{
		{
			Name:     "sensor-tag-instance-01",
			ID:       "sensor-tag-instance-01",
			Protocol: "bluetooth-sensor-tag-instance-01",
			Model:    "cc2650-sensortag",
		},
	}
	deviceProfile.DeviceModels = []*types.DeviceModel{
		{
			Name: "cc2650-sensortag",
			Properties: []*types.Property{
				{
					Name:         "temperature",
					DataType:     "int",
					Description:  "temperature in degree celsius",
					AccessMode:   "ReadOnly",
					DefaultValue: 0,
					Maximum:      100,
					Unit:         "degree celsius",
				},
				{
					Name:         "temperature-enable",
					DataType:     "string",
					Description:  "enable data collection of temperature sensor",
					AccessMode:   "ReadWrite",
					DefaultValue: "ON",
				},
				{
					Name:         "io-config-initialize",
					DataType:     "int",
					Description:  "initialize io-config with value 0",
					AccessMode:   "ReadWrite",
					DefaultValue: 0,
				},
				{
					Name:         "io-data-initialize",
					DataType:     "int",
					Description:  "initialize io-data with value 0",
					AccessMode:   "ReadWrite",
					DefaultValue: 0,
				},
				{
					Name:         "io-config",
					DataType:     "int",
					Description:  "register activation of io-config",
					AccessMode:   "ReadWrite",
					DefaultValue: 1,
				}, {
					Name:         "io-data",
					DataType:     "int",
					Description:  "data field to control io-control",
					AccessMode:   "ReadWrite",
					DefaultValue: 0,
				},
			},
		},
	}
	deviceProfile.Protocols = []*types.Protocol{
		{
			Name:     "bluetooth-sensor-tag-instance-01",
			Protocol: "bluetooth",
			ProtocolConfig: v1alpha1.ProtocolConfigBluetooth{
				MACAddress: "BC:6A:29:AE:CC:96",
			},
		},
	}

	deviceProfile.PropertyVisitors = []*types.PropertyVisitor{
		{
			Name:         "temperature",
			PropertyName: "temperature",
			ModelName:    "cc2650-sensortag",
			Protocol:     "bluetooth",
			VisitorConfig: v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa0104514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					OrderOfOperations: []v1alpha1.BluetoothOperations{
						{
							BluetoothOperationType:  "Multiply",
							BluetoothOperationValue: 0.03125,
						},
					},
					ShiftRight: 2,
					StartIndex: 2,
					EndIndex:1,
				},
			},
		},
		{
			Name:         "temperature-enable",
			PropertyName: "temperature-enable",
			ModelName:    "cc2650-sensortag",
			Protocol:     "bluetooth",
			VisitorConfig: v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa0204514000b000000000000000",
				DataWriteToBluetooth: map[string][]byte{
					"ON":  {1},
					"OFF": {0},
				},
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
		{
			Name:         "io-config-initialize",
			PropertyName: "io-config-initialize",
			ModelName:    "cc2650-sensortag",
			Protocol:     "bluetooth",
			VisitorConfig: v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6604514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
		{
			Name:         "io-data-initialize",
			PropertyName: "io-data-initialize",
			ModelName:    "cc2650-sensortag",
			Protocol:     "bluetooth",
			VisitorConfig: v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6504514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
		{
			Name:         "io-config",
			PropertyName: "io-config",
			ModelName:    "cc2650-sensortag",
			Protocol:     "bluetooth",
			VisitorConfig: v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6604514000b000000000000000",
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
		{
			Name:         "io-data",
			PropertyName: "io-data",
			ModelName:    "cc2650-sensortag",
			Protocol:     "bluetooth",
			VisitorConfig: v1alpha1.VisitorConfigBluetooth{
				CharacteristicUUID: "f000aa6504514000b000000000000000",
				DataWriteToBluetooth: map[string][]byte{
					"Red":            {1},
					"Green":          {2},
					"RedGreen":       {3},
					"Buzzer":         {4},
					"BuzzerRed":      {5},
					"BuzzerGreen":    {6},
					"BuzzerRedGreen": {7},
				},
				BluetoothDataConverter: v1alpha1.BluetoothReadConverter{
					StartIndex: 1,
					EndIndex:   1,
				},
			},
		},
	}

	bytes, err := json.Marshal(deviceProfile)
	if err != nil {
		Err("Failed to marshal deviceprofile: %v", deviceProfile)
	}
	configMap.Data["deviceProfile.json"] = string(bytes)

	return configMap
}

func NewConfigMapModbus(nodeSelector string) v12.ConfigMap {
	configMap := v12.ConfigMap{
		TypeMeta: v1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "device-profile-config-" + nodeSelector,
			Namespace: Namespace,
		},
	}
	configMap.Data = make(map[string]string)

	deviceProfile := &types.DeviceProfile{}
	deviceProfile.DeviceInstances = []*types.DeviceInstance{
		{
			Name:  "sensor-tag-instance-02",
			ID:    "sensor-tag-instance-02",
			Model: "sensor-tag-model",
		},
	}
	deviceProfile.DeviceModels = []*types.DeviceModel{
		{
			Name: "sensor-tag-model",
			Properties: []*types.Property{

				{
					Name:         "temperature",
					DataType:     "int",
					Description:  "temperature in degree celsius",
					AccessMode:   "ReadWrite",
					DefaultValue: 0,
					Maximum:      100,
					Unit:         "degree celsius",
				},
				{
					Name:         "temperature-enable",
					DataType:     "string",
					Description:  "enable data collection of temperature sensor",
					AccessMode:   "ReadWrite",
					DefaultValue: "OFF",
				},
			},
		},
	}
	deviceProfile.Protocols = []*types.Protocol{
		{
			ProtocolConfig: nil,
		},
	}
	deviceProfile.PropertyVisitors = []*types.PropertyVisitor{
		{
			Name:         "temperature",
			PropertyName: "temperature",
			ModelName:    "sensor-tag-model",
			Protocol:     "modbus",
			VisitorConfig: v1alpha1.VisitorConfigModbus{
				Register:       "CoilRegister",
				Offset:         2,
				Limit:          1,
				Scale:          1,
				IsSwap:         true,
				IsRegisterSwap: true,
			},
		},
		{
			Name:         "temperature-enable",
			PropertyName: "temperature-enable",
			ModelName:    "sensor-tag-model",
			Protocol:     "modbus",
			VisitorConfig: v1alpha1.VisitorConfigModbus{
				Register:       "DiscreteInputRegister",
				Offset:         3,
				Limit:          1,
				Scale:          1,
				IsSwap:         true,
				IsRegisterSwap: true,
			},
		},
	}

	bytes, err := json.Marshal(deviceProfile)
	if err != nil {
		Err("Failed to marshal deviceprofile: %v", deviceProfile)
	}
	configMap.Data["deviceProfile.json"] = string(bytes)

	return configMap
}
