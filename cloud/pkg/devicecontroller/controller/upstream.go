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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	commonmodel "github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common/model"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1beta1"
	crdClientset "github.com/kubeedge/kubeedge/pkg/client/clientset/versioned"
)

// DeviceStatus is structure to patch device status
type DeviceStatus struct {
	Status v1beta1.DeviceStatus `json:"status"`
}

const (
	// MergePatchType is patch type
	MergePatchType = "application/merge-patch+json"
	// ResourceTypeDevices is plural of device resource in apiserver
	ResourceTypeDevices = "devices"
)

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	crdClient    crdClientset.Interface
	messageLayer messagelayer.MessageLayer
	// message channel
	deviceStatusChan chan model.Message
	// message channel for mapper
	mapperStatusChan chan model.Message

	// downstream controller to update device status in cache
	dc *DownstreamController
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start upstream devicecontroller")

	uc.deviceStatusChan = make(chan model.Message, config.Config.Buffer.UpdateDeviceStatus)
	uc.mapperStatusChan = make(chan model.Message, config.Config.Buffer.UpdateMapperStatus)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.UpdateDeviceStatusWorkers); i++ {
		go uc.updateDeviceStatus()
	}
	for i := 0; i < int(config.Config.Load.UpdateMapperStatusWorkers); i++ {
		go uc.updateMapperStatus()
	}
	return nil
}

func (uc *UpstreamController) dispatchMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop dispatchMessage")
			return
		default:
		}
		msg, err := uc.messageLayer.Receive()
		if err != nil {
			klog.Warningf("Receive message failed, %s", err)
			continue
		}

		klog.Infof("Dispatch message: %s", msg.GetID())
		resourceType, err := messagelayer.GetResourceTypeForDevice(msg.GetResource())
		if err != nil {
			klog.Warningf("Parse message: %s resource type with error: %s", msg.GetID(), err)
			continue
		}
		klog.Infof("Message: %s, resource type is: %s", msg.GetID(), resourceType)

		switch resourceType {
		case constants.ResourceTypeDeviceConnected, constants.ResourceTypeDeviceMigrate, constants.ResourceTypeTwinEdgeUpdated:
			uc.deviceStatusChan <- msg
		case constants.ResourceTypeMapperConnected:
			uc.mapperStatusChan <- msg
		case constants.ResourceTypeMembershipDetail:
		default:
			klog.Warningf("Message: %s, with resource type: %s not intended for device controller", msg.GetID(), resourceType)
		}
	}
}

func (uc *UpstreamController) updateDeviceStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop updateDeviceStatus")
			return
		case msg := <-uc.deviceStatusChan:
			klog.Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			resource := msg.GetResource()
			if strings.Contains(resource, constants.ResourceTypeDeviceMigrate) {
				// node/nodeID/device/migrate
				hubInfo := commonmodel.HubInfo{}
				err := json.Unmarshal(msg.Content.([]byte), &hubInfo)
				if err != nil {
					klog.Warningf("Failed to unmarshal hub info with error %v", err)
					continue
				}
				nodeID := hubInfo.NodeID
				// delete mapper record in node
				err = uc.dc.mapperMigrated(nodeID)
				if err != nil {
					klog.Warningf("Failed to delete mapper record about node %s", nodeID)
				}
				// delete device which needs to be migrated when node becomes offline
				err = uc.dc.deviceMigrated(nodeID)
				if err != nil {
					klog.Warningf("Failed to migrate devices on node %s", nodeID)
				}
				continue
			} else if strings.Contains(resource, constants.ResourceTypeDeviceConnected) {
				// node/nodeID/device/connect_successfully
				nodeID, err := messagelayer.GetNodeID(msg)
				if err != nil {
					klog.Warningf("Message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
					continue
				}
				deviceName := msg.GetContent()
				value, ok := uc.dc.deviceManager.DeployedDevice.Load(deviceName)
				if !ok {
					klog.Warningf("Device %s does not exist in DeployedDevice map", deviceName)
					continue
				}
				device := value.(*v1beta1.Device)
				// update device status
				device.Status.CurrentNode = nodeID
				uc.dc.deviceManager.DeployedDevice.Store(deviceName, device)
				body, err := json.Marshal(device.Status)
				if err != nil {
					klog.Errorf("Failed to marshal device status %v", device.Status)
					continue
				}
				err = uc.crdClient.DevicesV1beta1().RESTClient().Patch(MergePatchType).Namespace(device.Namespace).Resource(ResourceTypeDevices).Name(device.Name).Body(body).Do(context.Background()).Error()
				if err != nil {
					klog.Errorf("Failed to patch device status %v of device %v in namespace %v, err: %v", device.Status, device.Name, device.Namespace, err)
					continue
				}
				continue
			}
			msgTwin, err := uc.unmarshalDeviceStatusMessage(msg)
			if err != nil {
				klog.Warningf("Unmarshall failed due to error %v", err)
				continue
			}
			deviceID, err := messagelayer.GetDeviceID(msg.GetResource())
			if err != nil {
				klog.Warning("Failed to get device id")
				continue
			}
			device, ok := uc.dc.deviceManager.Device.Load(deviceID)
			if !ok {
				klog.Warningf("Device %s does not exist in downstream controller", deviceID)
				continue
			}
			cacheDevice, ok := device.(*v1beta1.Device)
			if !ok {
				klog.Warning("Failed to assert to CacheDevice type")
				continue
			}
			deviceStatus := &DeviceStatus{Status: cacheDevice.Status}
			for twinName, twin := range msgTwin.Twin {
				for i, cacheTwin := range deviceStatus.Status.Twins {
					if twinName == cacheTwin.PropertyName && twin.Actual != nil && twin.Actual.Value != nil {
						reported := v1beta1.TwinProperty{}
						reported.Value = *twin.Actual.Value
						reported.Metadata = make(map[string]string)
						if twin.Actual.Metadata != nil {
							reported.Metadata["timestamp"] = strconv.FormatInt(twin.Actual.Metadata.Timestamp, 10)
						}
						if twin.Metadata != nil {
							reported.Metadata["type"] = twin.Metadata.Type
						}
						deviceStatus.Status.Twins[i].Reported = reported
						break
					}
				}
			}

			// Store the status in cache so that when update is received by informer, it is not processed by downstream controller
			cacheDevice.Status = deviceStatus.Status
			uc.dc.deviceManager.Device.Store(deviceID, cacheDevice)

			body, err := json.Marshal(deviceStatus)
			if err != nil {
				klog.Errorf("Failed to marshal device status %v", deviceStatus)
				continue
			}
			err = uc.crdClient.DevicesV1beta1().RESTClient().Patch(MergePatchType).Namespace(cacheDevice.Namespace).Resource(ResourceTypeDevices).Name(deviceID).Body(body).Do(context.Background()).Error()
			if err != nil {
				klog.Errorf("Failed to patch device status %v of device %v in namespace %v, err: %v", deviceStatus, deviceID, cacheDevice.Namespace, err)
				continue
			}
			//send confirm message to edge twin
			resMsg := model.NewMessage(msg.GetID())
			nodeID, err := messagelayer.GetNodeID(msg)
			if err != nil {
				klog.Warningf("Message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
				continue
			}
			resource, err = messagelayer.BuildResourceForDevice(nodeID, "twin", "")
			if err != nil {
				klog.Warningf("Message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
				continue
			}
			resMsg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.ResponseOperation)
			resMsg.Content = commonconst.MessageSuccessfulContent
			err = uc.messageLayer.Response(*resMsg)
			if err != nil {
				klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
				continue
			}
			klog.Infof("Message: %s process successfully", msg.GetID())
		}
	}
}

func (uc *UpstreamController) updateMapperStatus() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop updateMapperStatus")
			return
		case msg := <-uc.mapperStatusChan:
			fmt.Printf("Message: %s, operation is: %s, and resource is: %s\n", msg.GetID(), msg.GetOperation(), msg.GetResource())
			if strings.Contains(msg.GetResource(), constants.ResourceTypeMapperConnected) {
				// node/nodeID/mapper/connect_successfully
				// when mapper is connected successfully, register mapper and deploy device corresponding to mapper
				err := uc.registerMapper(msg)
				if err != nil {
					klog.Warningf("Mapper registration failed due to error %v", err)
					continue
				}
			}
		}
	}
}

// registerMapper store mapper record in mapper2NodeMap and deploy device whose mapperRef equals to mapperName to node
func (uc *UpstreamController) registerMapper(msg model.Message) error {
	nodeID, err := messagelayer.GetNodeID(msg)
	if err != nil {
		klog.Warning("Failed to get node id")
		return err
	}

	// get mapper info from message content
	mapperInfo := types.Mapper{}
	content, err := msg.GetContentData()
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &mapperInfo)
	if err != nil {
		return err
	}
	uc.dc.mapperManager.Mapper2NodeMap.Store(mapperInfo.Name, nodeID)
	value, ok := uc.dc.mapperManager.NodeMapperList.Load(nodeID)
	// store mapper info in NodeMapperList
	if ok {
		mapperList := value.([]*types.Mapper)
		mapperList = append(mapperList, &mapperInfo)
		uc.dc.mapperManager.NodeMapperList.Store(nodeID, mapperList)
	} else {
		mapperList := make([]*types.Mapper, 0)
		mapperList = append(mapperList, &mapperInfo)
		uc.dc.mapperManager.NodeMapperList.Store(nodeID, mapperList)
	}
	err = uc.dc.deviceDeployed(mapperInfo.Name)
	if err != nil {
		klog.Warning("Failed to deployed device whose mapperRef is %s to %s", mapperInfo.Name, nodeID)
		return err
	}
	return nil
}

func (uc *UpstreamController) unmarshalDeviceStatusMessage(msg model.Message) (*types.DeviceTwinUpdate, error) {
	contentData, err := msg.GetContentData()
	if err != nil {
		return nil, err
	}

	twinUpdate := &types.DeviceTwinUpdate{}
	if err := json.Unmarshal(contentData, twinUpdate); err != nil {
		return nil, err
	}
	return twinUpdate, nil
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(dc *DownstreamController) (*UpstreamController, error) {
	uc := &UpstreamController{
		crdClient:    keclient.GetCRDClient(),
		messageLayer: messagelayer.DeviceControllerMessageLayer(),
		dc:           dc,
	}
	return uc, nil
}
