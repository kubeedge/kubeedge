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

	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
)

type DevicePropertyOp struct {
	deviceProperty v1alpha2.DeviceProperty
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

func withStringType(accessMode v1alpha2.PropertyAccessMode, defaultValue string) DevicePropertyOption {
	return func(op *DevicePropertyOp) {
		stringType := &v1alpha2.PropertyTypeString{
			DefaultValue: defaultValue,
		}
		stringType.AccessMode = accessMode
		op.deviceProperty.Type = v1alpha2.PropertyType{
			String: stringType,
		}
	}
}

func withIntType(accessMode v1alpha2.PropertyAccessMode, defaultValue int64, minimum int64, maximum int64, unit string) DevicePropertyOption {
	return func(op *DevicePropertyOp) {
		intType := &v1alpha2.PropertyTypeInt64{
			DefaultValue: defaultValue,
			Minimum:      minimum,
			Maximum:      maximum,
			Unit:         unit,
		}
		intType.AccessMode = accessMode
		op.deviceProperty.Type = v1alpha2.PropertyType{
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
		deviceProperty: v1alpha2.DeviceProperty{},
	}
	op.applyDevicePropertyOpts(opts)
	return op
}

func newDeviceModel(name string, namespace string) *v1alpha2.DeviceModel {
	spec := v1alpha2.DeviceModelSpec{}
	deviceModel := &v1alpha2.DeviceModel{
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

func DeviceModelWithPropertyNoName(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withDescription(devicePropertyTemperatureDesc),
		withStringType(v1alpha2.PropertyAccessMode(v1alpha2.ReadOnly), ""))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)
	return deviceModel
}

func DeviceModelWithPropertyNoType(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)
	return deviceModel
}

func DeviceModelWithPropertyBadAccessMode(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withStringType("", ""))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	return deviceModel
}

func NewDeviceModelBluetooth(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha2.PropertyAccessMode(v1alpha2.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	return deviceModel
}

func NewDeviceModelModbus(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha2.PropertyAccessMode(v1alpha2.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	return deviceModel
}

func NewDeviceModelOpcUA(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha2.PropertyAccessMode(v1alpha2.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	return deviceModel
}

func NewDeviceModelCustomized(name string, namespace string) *v1alpha2.DeviceModel {
	deviceModel := newDeviceModel(name, namespace)
	devicePropertyOp := newDevicePropertyOp(withName(devicePropertyTemperature), withDescription(devicePropertyTemperatureDesc),
		withIntType(v1alpha2.PropertyAccessMode(v1alpha2.ReadOnly), 0, minimum, maximum, devicePropertyUnit))
	deviceModel.Spec.Properties = append(deviceModel.Spec.Properties, devicePropertyOp.deviceProperty)

	return deviceModel
}
