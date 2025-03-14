/*
Copyright 2022 The KubeEdge Authors.

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
	"encoding/json"
	"errors"
	"io/ioutil"

	"k8s.io/klog/v2"

	"github.com/kubeedge/mapper-generator/pkg/common"
	"github.com/kubeedge/mapper-generator/pkg/config"
	"github.com/kubeedge/mapper-generator/pkg/util/grpcclient"
)

var ErrEmptyData error = errors.New("device or device model list is empty")

// Parse the configmap.
func Parse(path string,
	devices map[string]*common.DeviceInstance,
	dms map[string]common.DeviceModel,
	protocols map[string]common.Protocol) error {
	var deviceProfile common.DeviceProfile
	jsonFile, err := ioutil.ReadFile(path)
	if err != nil {
		err = errors.New("failed to read " + path + " file")
		return err
	}
	//Parse the JSON file and convert it into the data structure of DeviceProfile
	if err = json.Unmarshal(jsonFile, &deviceProfile); err != nil {
		return err
	}
	// loop instIndex : judge whether the configmap definition is correct, and initialize the device instance
	for instIndex := 0; instIndex < len(deviceProfile.DeviceInstances); instIndex++ {
		instance := deviceProfile.DeviceInstances[instIndex]
		// loop protoIndex : judge whether the device's protocol is correct, and initialize the device protocol
		protoIndex := 0
		for protoIndex = 0; protoIndex < len(deviceProfile.Protocols); protoIndex++ {
			if instance.ProtocolName == deviceProfile.Protocols[protoIndex].Name {
				// Verify that the protocols match
				protocolConfig := make(map[string]interface{})
				err := json.Unmarshal(deviceProfile.Protocols[protoIndex].ProtocolConfigs, &protocolConfig)
				if err != nil {
					err = errors.New("failed to parse " + deviceProfile.Protocols[protoIndex].Name)
					return err
				}
				protocols[deviceProfile.Protocols[protoIndex].Name] = deviceProfile.Protocols[protoIndex]
				instance.PProtocol = deviceProfile.Protocols[protoIndex]
			}
		}
		// loop propertyIndex : find the device model's properties for each device instance's propertyVisitor
		for propertyIndex := 0; propertyIndex < len(instance.PropertyVisitors); propertyIndex++ {
			modelName := instance.PropertyVisitors[propertyIndex].ModelName
			propertyName := instance.PropertyVisitors[propertyIndex].PropertyName
			modelIndex := 0
			// loop modelIndex : find a matching device model, and initialize the device model
			for modelIndex = 0; modelIndex < len(deviceProfile.DeviceModels); modelIndex++ {
				if modelName == deviceProfile.DeviceModels[modelIndex].Name {
					dms[deviceProfile.DeviceModels[modelIndex].Name] = deviceProfile.DeviceModels[modelIndex]
					m := 0
					// loop m :  find a matching device model's properties
					for m = 0; m < len(deviceProfile.DeviceModels[modelIndex].Properties); m++ {
						if propertyName == deviceProfile.DeviceModels[modelIndex].Properties[m].Name {
							instance.PropertyVisitors[propertyIndex].PProperty = deviceProfile.DeviceModels[modelIndex].Properties[m]
							break
						}
					}
					if m == len(deviceProfile.DeviceModels[modelIndex].Properties) {
						err = errors.New("property mismatch")
						return err
					}
					break
				}
			}
			if modelIndex == len(deviceProfile.DeviceModels) {
				err = errors.New("device model mismatch")
				return err
			}
		}
		// loop propertyIndex : find propertyVisitors for each instance's twin
		for propertyIndex := 0; propertyIndex < len(instance.Twins); propertyIndex++ {
			name := instance.Twins[propertyIndex].PropertyName
			l := 0
			// loop l : find a matching propertyName
			for l = 0; l < len(instance.PropertyVisitors); l++ {
				if name == instance.PropertyVisitors[l].PropertyName {
					instance.Twins[propertyIndex].PVisitor = &instance.PropertyVisitors[l]
					break
				}
			}
			if l == len(instance.PropertyVisitors) {
				err = errors.New("propertyVisitor mismatch")
				return err
			}
		}
		// loop propertyIndex : find propertyVisitors for each instance's property
		for propertyIndex := 0; propertyIndex < len(instance.Datas.Properties); propertyIndex++ {
			name := instance.Datas.Properties[propertyIndex].PropertyName
			l := 0
			// loop l : find a matching propertyName
			for l = 0; l < len(instance.PropertyVisitors); l++ {
				if name == instance.PropertyVisitors[l].PropertyName {
					instance.Datas.Properties[propertyIndex].PVisitor = &instance.PropertyVisitors[l]
					break
				}
			}
			if l == len(instance.PropertyVisitors) {
				err = errors.New("propertyVisitor mismatch")
				return err
			}
		}
		devices[instance.ID] = new(common.DeviceInstance)
		devices[instance.ID] = &instance
		klog.V(4).Infof("Instance:%s Successfully registered", instance.ID)
	}
	return nil
}

func ParseByUsingRegister(cfg *config.Config,
	devices map[string]*common.DeviceInstance,
	dms map[string]common.DeviceModel,
	protocols map[string]common.Protocol) error {
	deviceList, deviceModelList, err := grpcclient.RegisterMapper(cfg, true)
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
