package utils

import (
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeedge/api/apis/devices/v1beta1"
)

func NewModbusDeviceModel() v1beta1.DeviceModel {
	modelProperty1 := v1beta1.ModelProperty{
		Name:        "temperature",
		Description: "temperature in degree celsius",
		Type:        v1beta1.INT,
		AccessMode:  "ReadWrite",
		Maximum:     "100",
		Unit:        "degree celsius",
	}

	newDeviceModel := v1beta1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-model",
			Namespace: Namespace,
		},
		Spec: v1beta1.DeviceModelSpec{
			Properties: []v1beta1.ModelProperty{modelProperty1},
		},
	}
	return newDeviceModel
}

func UpdatedModbusDeviceModel() v1beta1.DeviceModel {
	modelProperty1 := v1beta1.ModelProperty{
		Name:        "temperature",
		Description: "temperature in degree celsius",
		Type:        v1beta1.INT,
		AccessMode:  "ReadWrite",
		Maximum:     "200",
		Unit:        "degree celsius",
	}
	newDeviceModel := v1beta1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "DeviceModel",
			APIVersion: "devices.kubeedge.io/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "sensor-tag-model",
			Namespace: Namespace,
		},
		Spec: v1beta1.DeviceModelSpec{
			Properties: []v1beta1.ModelProperty{modelProperty1},
		},
	}
	return newDeviceModel
}

func NewModbusDeviceInstance(nodeName string) v1beta1.Device {
	property := v1beta1.DeviceProperty{
		Name: "temperature",
		Desired: v1beta1.TwinProperty{
			Value: "20",
		},
		CollectCycle:  1000,
		ReportCycle:   1000,
		ReportToCloud: true,
		Visitors: v1beta1.VisitorConfig{
			ProtocolName: "modbus",
			ConfigData: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"Register":       "CoilRegister",
					"Offset":         "2",
					"Limit":          "1",
					"Scale":          "1",
					"IsSwap":         "true",
					"IsRegisterSwap": "true",
				},
			},
		},
	}

	deviceInstance := v1beta1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1beta1",
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
		Spec: v1beta1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "sensor-tag-model",
			},
			NodeName:   nodeName,
			Properties: []v1beta1.DeviceProperty{property},
		},
	}
	return deviceInstance
}

func UpdatedModbusDeviceInstance(nodeName string) v1beta1.Device {
	property := v1beta1.DeviceProperty{
		Name: "temperature",
		Desired: v1beta1.TwinProperty{
			Value: "20",
		},
		CollectCycle:  1000,
		ReportCycle:   1000,
		ReportToCloud: true,
		Visitors: v1beta1.VisitorConfig{
			ProtocolName: "modbus",
			ConfigData: &v1beta1.CustomizedValue{
				Data: map[string]interface{}{
					"Register":       "CoilRegister",
					"Offset":         "2",
					"Limit":          "1",
					"Scale":          "1",
					"IsSwap":         "true",
					"IsRegisterSwap": "true",
				},
			},
		},
	}

	deviceInstance := v1beta1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "Device",
			APIVersion: "devices.kubeedge.io/v1beta1",
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
		Spec: v1beta1.DeviceSpec{
			DeviceModelRef: &v12.LocalObjectReference{
				Name: "sensor-tag-model",
			},
			NodeName:   nodeName,
			Properties: []v1beta1.DeviceProperty{property},
			Protocol: v1beta1.ProtocolConfig{
				ProtocolName: "modbus",
				ConfigData: &v1beta1.CustomizedValue{Data: map[string]interface{}{
					"SerialPort": "/dev/ttyS0",
					"BaudRate":   "9600",
					"DataBits":   "8",
					"Parity":     "even",
					"StopBits":   "1",
				}},
			},
		},
	}
	return deviceInstance
}

func IncorrectDeviceModel() v1beta1.DeviceModel {
	newDeviceModel := v1beta1.DeviceModel{
		TypeMeta: v1.TypeMeta{
			Kind:       "device-model",
			APIVersion: "devices.kubeedge.io/v1beta1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "led-light",
			Namespace: Namespace,
		},
	}
	return newDeviceModel
}

func IncorrectDeviceInstance() v1beta1.Device {
	deviceInstance := v1beta1.Device{
		TypeMeta: v1.TypeMeta{
			Kind:       "device",
			APIVersion: "devices.kubeedge.io/v1beta1",
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
