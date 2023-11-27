/*
Copyright 2023 The KubeEdge Authors.

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

package parse

import (
	"errors"

	"k8s.io/klog/v2"

	"github.com/kubeedge/Template/pkg/common"
	"github.com/kubeedge/Template/pkg/grpcclient"
)

var ErrEmptyData error = errors.New("device or device model list is empty")

func ParseByUsingRegister(devices map[string]*common.DeviceInstance,
	dms map[string]common.DeviceModel,
	protocols map[string]common.ProtocolConfig) error {
	deviceList, deviceModelList, err := grpcclient.RegisterMapper(true)
	if err != nil {
		return err
	}

	if len(deviceList) == 0 || len(deviceModelList) == 0 {
		return ErrEmptyData
	}
	modelMap := make(map[string]common.DeviceModel)
	for _, model := range deviceModelList {
		cur := ParseDeviceModelFromGrpc(model)
		modelMap[model.Name] = cur
	}

	for _, device := range deviceList {
		commonModel := modelMap[device.Spec.DeviceModelReference]
		protocol, err := BuildProtocolFromGrpc(device)
		if err != nil {
			return err
		}
		instance, err := ParseDeviceFromGrpc(device, &commonModel)
		if err != nil {
			return err
		}
		instance.PProtocol = protocol
		devices[instance.ID] = new(common.DeviceInstance)
		devices[instance.ID] = instance
		klog.V(4).Info("Instance: ", instance.ID)
		dms[instance.Model] = modelMap[instance.Model]
		protocols[instance.ProtocolName] = protocol
	}

	return nil
}
