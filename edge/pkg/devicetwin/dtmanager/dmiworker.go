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

package dtmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	pb "github.com/kubeedge/api/apis/dmi/v1beta1"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiclient"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dmiserver"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcommon"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dtcontext"
	"github.com/kubeedge/kubeedge/edge/pkg/devicetwin/dttype"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// DMIWorker deal dmi event
type DMIWorker struct {
	Worker
	Group    string
	dmiCache *dmiserver.DMICache
	//dmiActionCallBack map for action to callback
	dmiActionCallBack map[string]CallBack
}

func (dw *DMIWorker) init() {
	dw.dmiCache = &dmiserver.DMICache{
		MapperMu:        &sync.Mutex{},
		DeviceMu:        &sync.Mutex{},
		DeviceModelMu:   &sync.Mutex{},
		MapperList:      make(map[string]*pb.MapperInfo),
		DeviceList:      make(map[string]*v1beta1.Device),
		DeviceModelList: make(map[string]*v1beta1.DeviceModel),
	}

	dw.initDMIActionCallBack()
	dw.initDeviceModelInfoFromDB()
	dw.initDeviceInfoFromDB()
	dw.initDeviceMapperInfoFromDB()
}

// Start worker
func (dw DMIWorker) Start() {
	klog.Infoln("dmi worker start")
	dw.init()

	go dmiserver.StartDMIServer(dw.dmiCache)

	for {
		select {
		case msg, ok := <-dw.ReceiverChan:
			if !ok {
				return
			}

			if dtMsg, isDTMessage := msg.(*dttype.DTMessage); isDTMessage {
				if fn, exist := dw.dmiActionCallBack[dtMsg.Action]; exist {
					err := fn(dw.DTContexts, dtMsg.Identity, dtMsg.Msg)
					if err != nil {
						klog.Errorf("DMIModule deal %s event failed: %v", dtMsg.Action, err)
					}
				} else {
					klog.Errorf("DMIModule deal %s event failed, not found callback", dtMsg.Action)
				}
			}

		case v, ok := <-dw.HeartBeatChan:
			if !ok {
				return
			}
			if err := dw.DTContexts.HeartBeat(dw.Group, v); err != nil {
				return
			}
		}
	}
}

func (dw *DMIWorker) initDMIActionCallBack() {
	dw.dmiActionCallBack = make(map[string]CallBack)
	dw.dmiActionCallBack[dtcommon.MetaDeviceOperation] = dw.dealMetaDeviceOperation
}

// overrideDeviceInstanceConfig overrides device instance configuration with model defaults
func (dw *DMIWorker) overrideDeviceInstanceConfig(device *v1beta1.Device) error {
	if device.Spec.DeviceModelRef == nil {
		return fmt.Errorf("device %s has no device model reference", device.Name)
	}

	// Get device model from cache
	deviceModelID := util.GetResourceID(device.Namespace, device.Spec.DeviceModelRef.Name)
	dw.dmiCache.DeviceModelMu.Lock()
	deviceModel, ok := dw.dmiCache.DeviceModelList[deviceModelID]
	if !ok {
		return fmt.Errorf("device model %s not found in cache for device %s", device.Spec.DeviceModelRef.Name, device.Name)
	}
	dw.dmiCache.DeviceModelMu.Unlock()

	klog.Infof("Overriding device properties for device %s using model %s", device.Name, deviceModel.Name)

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
				klog.Infof("Applied model visitors to property %s of device %s", deviceProp.Name, device.Name)
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
				klog.Infof("Merged instance visitors for property %s of device %s", deviceProp.Name, device.Name)
			} else {
				// No model visitors config data, use instance visitors config data directly
				deviceProp.Visitors.ConfigData = originalProp.Visitors.ConfigData
				klog.Infof("Used instance visitors config data for property %s of device %s", deviceProp.Name, device.Name)
			}
		}
		// If original property has no visitors config, keep the model visitors (already applied above)
	}

	// Handle protocol config data
	// Store original protocol config
	originalProtocolConfig := device.Spec.Protocol.ConfigData

	// Apply model protocol config (may be nil)
	device.Spec.Protocol.ConfigData = deviceModel.Spec.ProtocolConfigData
	if deviceModel.Spec.ProtocolConfigData != nil {
		klog.Infof("Applied model protocol config data to device %s", device.Name)
	}

	// Merge with instance protocol config if exists (instance data takes precedence)
	if originalProtocolConfig != nil && originalProtocolConfig.Data != nil {
		if device.Spec.Protocol.ConfigData != nil && device.Spec.Protocol.ConfigData.Data != nil {
			// Merge: instance data overrides model data
			for key, value := range originalProtocolConfig.Data {
				device.Spec.Protocol.ConfigData.Data[key] = value
			}
			klog.Infof("Merged instance protocol config data for device %s", device.Name)
		} else {
			// No model config data, use instance config directly
			device.Spec.Protocol.ConfigData = originalProtocolConfig
			klog.Infof("Used instance protocol config data for device %s", device.Name)
		}
	}

	return nil
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

func (dw *DMIWorker) dealMetaDeviceOperation(_ *dtcontext.DTContext, _ string, msg interface{}) error {
	message, ok := msg.(*model.Message)
	if !ok {
		return errors.New("msg not Message type")
	}
	resources := strings.Split(message.Router.Resource, "/")
	if len(resources) != 3 {
		return fmt.Errorf("wrong resources %s", message.Router.Resource)
	}
	var device v1beta1.Device
	var dm v1beta1.DeviceModel
	switch resources[1] {
	case constants.ResourceTypeDevice:
		err := json.Unmarshal(message.Content.([]byte), &device)
		if err != nil {
			return fmt.Errorf("invalid message content with err: %+v", err)
		}
		deviceID := util.GetResourceID(device.Namespace, device.Name)
		switch message.GetOperation() {
		case model.InsertOperation:
			// Override device instance config with model defaults before registering
			err = dw.overrideDeviceInstanceConfig(&device)
			if err != nil {
				klog.Errorf("override device instance config failed for device %s: %v", device.Name, err)
				return err
			}
			dw.dmiCache.DeviceMu.Lock()
			dw.dmiCache.DeviceList[deviceID] = &device
			dw.dmiCache.DeviceMu.Unlock()
			err = dmiclient.DMIClientsImp.RegisterDevice(&device)
			if err != nil {
				klog.Errorf("add device %s failed with err: %v", device.Name, err)
				return err
			}
		case model.DeleteOperation:
			err = dmiclient.DMIClientsImp.RemoveDevice(&device)
			if err != nil {
				klog.Errorf("delete device %s failed with err: %v", device.Name, err)
				return err
			}
			dw.dmiCache.DeviceMu.Lock()
			delete(dw.dmiCache.DeviceList, deviceID)
			dw.dmiCache.DeviceMu.Unlock()
		case model.UpdateOperation:
			// Override device instance config with model defaults before updating
			err = dw.overrideDeviceInstanceConfig(&device)
			if err != nil {
				klog.Errorf("override device instance config failed for device %s: %v", device.Name, err)
				return err
			}
			dw.dmiCache.DeviceMu.Lock()
			dw.dmiCache.DeviceList[deviceID] = &device
			dw.dmiCache.DeviceMu.Unlock()
			err = dmiclient.DMIClientsImp.UpdateDevice(&device)
			if err != nil {
				klog.Errorf("update device %s failed with err: %v", device.Name, err)
				return err
			}
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}
	case constants.ResourceTypeDeviceModel:
		err := json.Unmarshal(message.Content.([]byte), &dm)
		if err != nil {
			return fmt.Errorf("invalid message content with err: %+v", err)
		}
		dmID := util.GetResourceID(dm.Namespace, dm.Name)
		switch message.GetOperation() {
		case model.InsertOperation:
			dw.dmiCache.DeviceModelMu.Lock()
			dw.dmiCache.DeviceModelList[dmID] = &dm
			dw.dmiCache.DeviceModelMu.Unlock()
			err = dmiclient.DMIClientsImp.CreateDeviceModel(&dm)
			if err != nil {
				klog.Errorf("add device model %s failed with err: %v", dm.Name, err)
				return err
			}
		case model.DeleteOperation:
			err = dmiclient.DMIClientsImp.RemoveDeviceModel(&dm)
			if err != nil {
				klog.Errorf("delete device model %s failed with err: %v", dm.Name, err)
				return err
			}
			dw.dmiCache.DeviceModelMu.Lock()
			delete(dw.dmiCache.DeviceModelList, dmID)
			dw.dmiCache.DeviceModelMu.Unlock()
		case model.UpdateOperation:
			dw.dmiCache.DeviceModelMu.Lock()
			dw.dmiCache.DeviceModelList[dmID] = &dm
			dw.dmiCache.DeviceModelMu.Unlock()
			err = dmiclient.DMIClientsImp.UpdateDeviceModel(&dm)
			if err != nil {
				klog.Errorf("update device model %s failed with err: %v", dm.Name, err)
				return err
			}
		default:
			klog.Warningf("unsupported operation %s", message.GetOperation())
		}

	default:
		klog.Warningf("unsupported resource type %s", resources[3])
	}

	return nil
}

func (dw *DMIWorker) initDeviceModelInfoFromDB() {
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDeviceModel)
	if err != nil {
		klog.Errorf("fail to init device model info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		deviceModel := v1beta1.DeviceModel{}
		if err := json.Unmarshal([]byte(meta), &deviceModel); err != nil {
			klog.Errorf("fail to unmarshal device model info from db with err: %v", err)
			return
		}
		deviceModelID := util.GetResourceID(deviceModel.Namespace, deviceModel.Name)
		dw.dmiCache.DeviceModelMu.Lock()
		dw.dmiCache.DeviceModelList[deviceModelID] = &deviceModel
		dw.dmiCache.DeviceModelMu.Unlock()
	}
	klog.Infoln("success to init device model info from db")
}

func (dw *DMIWorker) initDeviceInfoFromDB() {
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDevice)
	if err != nil {
		klog.Errorf("fail to init device info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		device := v1beta1.Device{}
		if err := json.Unmarshal([]byte(meta), &device); err != nil {
			klog.Errorf("fail to unmarshal device info from db with err: %v", err)
			return
		}
		deviceID := util.GetResourceID(device.Namespace, device.Name)
		dw.dmiCache.DeviceMu.Lock()
		dw.dmiCache.DeviceList[deviceID] = &device
		dw.dmiCache.DeviceMu.Unlock()
	}
	klog.Infoln("success to init device info from db")
}

func (dw *DMIWorker) initDeviceMapperInfoFromDB() {
	metas, err := dao.QueryMeta("type", constants.ResourceTypeDeviceMapper)
	if err != nil {
		klog.Errorf("fail to init device mapper info from db with err: %v", err)
		return
	}

	for _, meta := range *metas {
		deviceMapper := pb.MapperInfo{}
		if err := json.Unmarshal([]byte(meta), &deviceMapper); err != nil {
			klog.Errorf("fail to unmarshal device mapper info from db with err: %v", err)
			return
		}
		dw.dmiCache.MapperMu.Lock()
		dw.dmiCache.MapperList[deviceMapper.Name] = &deviceMapper
		dw.dmiCache.MapperMu.Unlock()
	}
	klog.Infoln("success to init device mapper info from db")
}
