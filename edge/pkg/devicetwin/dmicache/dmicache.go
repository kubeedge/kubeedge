/*
Copyright 2025 The KubeEdge Authors.

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

package dmicache

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/api/apis/util"
)

const dmiCacheLoggerName = "dmiCache"

type DMICache struct {
	mapperMu        *sync.RWMutex
	deviceMu        *sync.RWMutex
	deviceModelMu   *sync.RWMutex
	mapperList      map[string]*pb.MapperInfo
	deviceModelList map[string]*v1beta1.DeviceModel
	deviceList      map[string]*v1beta1.Device
	logger          klog.Logger
}

func NewDMICache() *DMICache {
	return &DMICache{
		mapperMu:        &sync.RWMutex{},
		deviceMu:        &sync.RWMutex{},
		deviceModelMu:   &sync.RWMutex{},
		mapperList:      make(map[string]*pb.MapperInfo),
		deviceList:      make(map[string]*v1beta1.Device),
		deviceModelList: make(map[string]*v1beta1.DeviceModel),
		logger:          klog.Background().WithName(dmiCacheLoggerName),
	}
}

// PutMapper puts a mapper in the cache
func (dmiCache *DMICache) PutMapper(mapper *pb.MapperInfo) {
	dmiCache.mapperMu.Lock()
	defer dmiCache.mapperMu.Unlock()
	dmiCache.mapperList[mapper.Name] = mapper
}

// GetMapper gets a mapper from the cache
func (dmiCache *DMICache) GetMapper(name string) (*pb.MapperInfo, bool) {
	dmiCache.mapperMu.RLock()
	defer dmiCache.mapperMu.RUnlock()
	mapper, exists := dmiCache.mapperList[name]
	return mapper, exists
}

// RemoveMapper removes a mapper from the cache
func (dmiCache *DMICache) RemoveMapper(name string) {
	dmiCache.mapperMu.Lock()
	defer dmiCache.mapperMu.Unlock()
	delete(dmiCache.mapperList, name)
}

// PutDeviceModel puts a device model in the cache
func (dmiCache *DMICache) PutDeviceModel(deviceModel *v1beta1.DeviceModel) {
	dmiCache.deviceModelMu.Lock()
	defer dmiCache.deviceModelMu.Unlock()
	deviceModelID := util.GetResourceID(deviceModel.Namespace, deviceModel.Name)
	dmiCache.deviceModelList[deviceModelID] = deviceModel
}

// GetDeviceModel gets a device model from the cache
func (dmiCache *DMICache) GetDeviceModel(namespace, name string) (*v1beta1.DeviceModel, bool) {
	dmiCache.deviceModelMu.RLock()
	defer dmiCache.deviceModelMu.RUnlock()
	deviceModelID := util.GetResourceID(namespace, name)
	deviceModel, exists := dmiCache.deviceModelList[deviceModelID]
	return deviceModel, exists
}

// RemoveDeviceModel removes a device model from the cache
func (dmiCache *DMICache) RemoveDeviceModel(namespace, name string) {
	dmiCache.deviceModelMu.Lock()
	defer dmiCache.deviceModelMu.Unlock()
	deviceModelID := util.GetResourceID(namespace, name)
	delete(dmiCache.deviceModelList, deviceModelID)
}

// PutDevice puts a device in the cache
func (dmiCache *DMICache) PutDevice(device *v1beta1.Device) {
	dmiCache.deviceMu.Lock()
	defer dmiCache.deviceMu.Unlock()
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	dmiCache.deviceList[deviceID] = device
}

// GetOverriddenDevice gets an overridden device from the cache
func (dmiCache *DMICache) GetOverriddenDevice(namespace, name string) (*v1beta1.Device, *v1beta1.DeviceModel, error) {
	deviceID := util.GetResourceID(namespace, name)
	dmiCache.deviceMu.RLock()
	device, exists := dmiCache.deviceList[deviceID]
	dmiCache.deviceMu.RUnlock()
	if !exists {
		return nil, nil, fmt.Errorf("device %s not found in cache", name)
	}

	deviceCopy := device.DeepCopy()
	// use deepCopyCustomizedValue instead of DeepCopy defined on *CustomizedValue to prevent data type loss
	// before and after json marshal/unmarshal
	deviceCopy.Spec.Protocol.ConfigData = deepCopyCustomizedValue(device.Spec.Protocol.ConfigData)
	for i := range deviceCopy.Spec.Properties {
		deviceCopy.Spec.Properties[i].Visitors.ConfigData = deepCopyCustomizedValue(device.Spec.Properties[i].Visitors.ConfigData)
	}

	deviceCopy, deviceModel, err := dmiCache.overrideDeviceInstanceConfig(deviceCopy)
	if err != nil {
		return nil, nil, fmt.Errorf("override device instance config failed for device %s: %v", name, err)
	}

	return deviceCopy, deviceModel, nil
}

func (dmiCache *DMICache) DeviceIds() []string {
	dmiCache.deviceMu.RLock()
	defer dmiCache.deviceMu.RUnlock()
	deviceIDs := make([]string, 0, len(dmiCache.deviceList))
	for deviceID := range dmiCache.deviceList {
		deviceIDs = append(deviceIDs, deviceID)
	}
	return deviceIDs
}

// RemoveDevice removes a device from the cache
func (dmiCache *DMICache) RemoveDevice(namespace, name string) {
	dmiCache.deviceMu.Lock()
	defer dmiCache.deviceMu.Unlock()
	deviceID := util.GetResourceID(namespace, name)
	delete(dmiCache.deviceList, deviceID)
}

// CompareDeviceSpecHasChanged checks whether the device spec has changed
func (dmiCache *DMICache) CompareDeviceSpecHasChanged(device *v1beta1.Device) bool {
	dmiCache.deviceMu.Lock()
	defer dmiCache.deviceMu.Unlock()
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	oldDevice, ok := dmiCache.deviceList[deviceID]

	if ok && oldDevice != nil {
		newSpec := device.Spec.DeepCopy()
		oldSpec := oldDevice.Spec.DeepCopy()

		sort.Slice(newSpec.Properties, func(i, j int) bool {
			return newSpec.Properties[i].Name < newSpec.Properties[j].Name
		})
		sort.Slice(oldSpec.Properties, func(i, j int) bool {
			return oldSpec.Properties[i].Name < oldSpec.Properties[j].Name
		})

		sort.Slice(newSpec.Methods, func(i, j int) bool {
			return newSpec.Methods[i].Name < newSpec.Methods[j].Name
		})
		sort.Slice(oldSpec.Methods, func(i, j int) bool {
			return oldSpec.Methods[i].Name < oldSpec.Methods[j].Name
		})

		if reflect.DeepEqual(oldSpec, newSpec) {
			dmiCache.logger.Info("device unchanged, skip updateDevice", "device", device.Name)
			oldDevice.Status = device.Status
			return false
		}
	}

	return true
}

// overrideDeviceInstanceConfig overrides device instance configuration with model defaults
func (dmiCache *DMICache) overrideDeviceInstanceConfig(device *v1beta1.Device) (*v1beta1.Device, *v1beta1.DeviceModel, error) {
	if device.Spec.DeviceModelRef == nil {
		return nil, nil, fmt.Errorf("device %s has no device model reference", device.Name)
	}

	// Get device model from cache
	deviceModelID := util.GetResourceID(device.Namespace, device.Spec.DeviceModelRef.Name)
	dmiCache.deviceModelMu.RLock()
	deviceModel, ok := dmiCache.deviceModelList[deviceModelID]
	dmiCache.deviceModelMu.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("device model %s not found in cache for device %s", device.Spec.DeviceModelRef.Name, device.Name)
	}

	dmiCache.logger.Info("overriding device properties", "device", device.Name, "model", deviceModel.Name)

	// Store original device properties in temporary variables
	originalProperties := make([]v1beta1.DeviceProperty, len(device.Spec.Properties))
	copy(originalProperties, device.Spec.Properties)

	// Apply model visitors
	for i := range device.Spec.Properties {
		deviceProp := &device.Spec.Properties[i]

		// Find corresponding model property
		if modelProp := findModelProperty(deviceModel.Spec.Properties, deviceProp.Name); modelProp != nil {
			// Apply model visitors
			if modelProp.Visitors != nil {
				deviceProp.Visitors = *modelProp.Visitors
				deviceProp.Visitors.ConfigData = deepCopyCustomizedValue(modelProp.Visitors.ConfigData)
				dmiCache.logger.V(4).Info("applied model visitors to property", "property", deviceProp.Name, "device", device.Name)
			}
		}
	}

	// Merge with original instance data (instance data takes precedence)
	for i, originalProp := range originalProperties {
		deviceProp := &device.Spec.Properties[i]

		// Merge visitors: keep model defaults but override with instance values
		if originalProp.Visitors.ConfigData != nil && originalProp.Visitors.ConfigData.Data != nil {
			// If device property has model visitors, merge the config data
			if deviceProp.Visitors.ConfigData != nil && deviceProp.Visitors.ConfigData.Data != nil {
				// Merge: instance data overrides model data
				for key, value := range originalProp.Visitors.ConfigData.Data {
					deviceProp.Visitors.ConfigData.Data[key] = value
				}
				dmiCache.logger.V(4).Info("merged instance visitors for property", "property", deviceProp.Name, "device", device.Name)
			} else {
				// No model visitors config data, use instance visitors config data directly
				deviceProp.Visitors.ConfigData = originalProp.Visitors.ConfigData
				dmiCache.logger.V(4).Info("used instance visitors config data for property", "property", deviceProp.Name, "device", device.Name)
			}
		}
		// If original property has no visitors config, keep the model visitors (already applied above)
	}

	// Handle protocol config data
	// Store original protocol config
	originalProtocolConfig := device.Spec.Protocol.ConfigData

	// Apply model protocol config (may be nil)
	device.Spec.Protocol.ConfigData = deepCopyCustomizedValue(deviceModel.Spec.ProtocolConfigData)
	if deviceModel.Spec.ProtocolConfigData != nil {
		dmiCache.logger.V(4).Info("applied model protocol config data", "device", device.Name)
	}

	// Merge with instance protocol config if exists (instance data takes precedence)
	if originalProtocolConfig != nil && originalProtocolConfig.Data != nil {
		if device.Spec.Protocol.ConfigData != nil && device.Spec.Protocol.ConfigData.Data != nil {
			// Merge: instance data overrides model data
			for key, value := range originalProtocolConfig.Data {
				device.Spec.Protocol.ConfigData.Data[key] = value
			}
			dmiCache.logger.V(4).Info("merged instance protocol config data", "device", device.Name)
		} else {
			// No model config data, use instance config directly
			device.Spec.Protocol.ConfigData = originalProtocolConfig
			dmiCache.logger.V(4).Info("used instance protocol config data", "device", device.Name)
		}
	}

	return device, deviceModel, nil
}

// findModelProperty finds a model property by name
func findModelProperty(properties []v1beta1.ModelProperty, name string) *v1beta1.ModelProperty {
	for i := range properties {
		if properties[i].Name == name {
			return &properties[i]
		}
	}
	return nil
}

// deepCopyValue is a recursive function to deep copy interface{} values
// as long as they are composed of maps, slices, and basic types.
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{})
		for k, v2 := range val {
			newMap[k] = deepCopyValue(v2)
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(val))
		for i, v2 := range val {
			newSlice[i] = deepCopyValue(v2)
		}
		return newSlice
	default:
		return val
	}
}

// deepCopyCustomizedValue creates a deep copy of CustomizedValue
func deepCopyCustomizedValue(src *v1beta1.CustomizedValue) *v1beta1.CustomizedValue {
	if src == nil {
		return nil
	}
	cp := &v1beta1.CustomizedValue{}
	if src.Data == nil {
		return cp
	}

	cp.Data = make(map[string]interface{})
	for key, value := range src.Data {
		cp.Data[key] = deepCopyValue(value)
	}
	return cp
}
