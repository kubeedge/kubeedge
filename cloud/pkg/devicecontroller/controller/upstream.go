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
	"strconv"

	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	crdClientset "github.com/kubeedge/api/client/clientset/versioned"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	utilcontext "github.com/kubeedge/kubeedge/cloud/pkg/common/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	commonconst "github.com/kubeedge/kubeedge/common/constants"
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
	// deviceTwinsChan message channel
	deviceTwinsChan chan model.Message
	// deviceStates message channel
	deviceStatesChan chan model.Message
	// downstream controller to update device status in cache
	dc *DownstreamController
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start upstream devicecontroller")

	uc.deviceTwinsChan = make(chan model.Message, config.Config.Buffer.UpdateDeviceTwins)
	uc.deviceStatesChan = make(chan model.Message, config.Config.Buffer.UpdateDeviceStates)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Load.UpdateDeviceStatusWorkers); i++ {
		go uc.updateDeviceStatus()
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
		case constants.ResourceTypeTwinEdgeUpdated:
			uc.deviceTwinsChan <- msg
		case constants.ResourceDeviceStateUpdated:
			uc.deviceStatesChan <- msg
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
		case msg := <-uc.deviceStatesChan:
			klog.Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			msgState, err := uc.unmarshalDeviceStatesMessage(msg)
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
				klog.Warningf("Device %s does not exist in upstream controller", deviceID)
				continue
			}
			cacheDevice, ok := device.(*v1beta1.Device)
			if !ok {
				klog.Warning("Failed to assert to CacheDevice type")
				continue
			}

			// Store the status in cache so that when update is received by informer, it is not processed by downstream controller
			cacheDevice.Status.State = msgState.Device.State
			cacheDevice.Status.LastOnlineTime = msgState.Device.LastOnlineTime
			uc.dc.deviceManager.Device.Store(deviceID, cacheDevice)

			body, err := json.Marshal(cacheDevice.Status)
			if err != nil {
				klog.Errorf("Failed to marshal device states %v", cacheDevice.Status)
				continue
			}
			err = uc.crdClient.DevicesV1beta1().RESTClient().Patch(MergePatchType).Namespace(cacheDevice.Namespace).Resource(ResourceTypeDevices).Name(cacheDevice.Name).Body(body).Do(context.Background()).Error()
			if err != nil {
				klog.Errorf("Failed to patch device states %v of device %v in namespace %v, err: %v", cacheDevice,
					deviceID, cacheDevice.Namespace, err)
				continue
			}

			//send confirm message to edge twin
			resMsg := model.NewMessage(msg.GetID())
			nodeID, err := messagelayer.GetNodeID(msg)
			if err != nil {
				klog.Warningf("Message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
				continue
			}
			resource, err := messagelayer.BuildResourceForDevice(nodeID, "twin", "")
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
		case msg := <-uc.deviceTwinsChan:
			klog.Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
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
				deviceTwin := findOrCreateTwinByName(twinName, cacheDevice.Spec.Properties, deviceStatus)
				if deviceTwin != nil {
					if twin.Actual != nil && twin.Actual.Value != nil {
						reported := v1beta1.TwinProperty{}
						reported.Value = *twin.Actual.Value
						reported.Metadata = make(map[string]string)
						if twin.Actual.Metadata != nil {
							reported.Metadata["timestamp"] = strconv.FormatInt(twin.Actual.Metadata.Timestamp, 10)
						}
						if twin.Metadata != nil {
							reported.Metadata["type"] = twin.Metadata.Type
						}
						deviceTwin.Reported = reported
					}

					if twin.Expected != nil && twin.Expected.Value != nil {
						observedDesired := v1beta1.TwinProperty{}
						observedDesired.Value = *twin.Expected.Value
						observedDesired.Metadata = make(map[string]string)
						if twin.Expected.Metadata != nil {
							observedDesired.Metadata["timestamp"] = strconv.FormatInt(twin.Expected.Metadata.Timestamp, 10)
						}
						if twin.Metadata != nil {
							observedDesired.Metadata["type"] = twin.Metadata.Type
						}
						deviceTwin.ObservedDesired = observedDesired
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
			err = uc.crdClient.DevicesV1beta1().RESTClient().Patch(MergePatchType).Namespace(cacheDevice.Namespace).Resource(ResourceTypeDevices).Name(cacheDevice.Name).Body(body).Do(utilcontext.FromMessage(context.Background(), msg)).Error()
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
			resource, err := messagelayer.BuildResourceForDevice(nodeID, "twin", "")
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

func (uc *UpstreamController) unmarshalDeviceStatesMessage(msg model.Message) (*types.DeviceStateUpdate, error) {
	contentData, err := msg.GetContentData()
	if err != nil {
		return nil, err
	}

	stateUpdate := &types.DeviceStateUpdate{}
	if err := json.Unmarshal(contentData, stateUpdate); err != nil {
		return nil, err
	}
	return stateUpdate, nil
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

func findOrCreateTwinByName(twinName string, properties []v1beta1.DeviceProperty, deviceStatus *DeviceStatus) *v1beta1.Twin {
	for i := range properties {
		if twinName == properties[i].Name {
			twin := findTwinByName(twinName, deviceStatus)
			if twin != nil {
				return twin
			}
			twin = &v1beta1.Twin{
				PropertyName: twinName,
			}
			deviceStatus.Status.Twins = append(deviceStatus.Status.Twins, *twin)
			return twin
		}
	}
	return nil
}

func findTwinByName(twinName string, deviceStatus *DeviceStatus) *v1beta1.Twin {
	for i := range deviceStatus.Status.Twins {
		if twinName == deviceStatus.Status.Twins[i].PropertyName {
			return &deviceStatus.Status.Twins[i]
		}
	}
	return nil
}
