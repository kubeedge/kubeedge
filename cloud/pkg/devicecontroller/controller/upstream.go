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
	"encoding/json"
	"strconv"

	"github.com/kubeedge/beehive/pkg/common/log"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/apis/devices/v1alpha1"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/utils"

	"k8s.io/client-go/rest"
)

// DeviceStatus is structure to patch device status
type DeviceStatus struct {
	Status v1alpha1.DeviceStatus `json:"status"`
}

const (
	// MergePatchType is patch type
	MergePatchType = "application/merge-patch+json"
	// ResourceTypeDevices is plural of device resource in apiserver
	ResourceTypeDevices = "devices"
)

// UpstreamController subscribe messages from edge and sync to k8s api server
type UpstreamController struct {
	crdClient    *rest.RESTClient
	messageLayer messagelayer.MessageLayer

	// stop channel
	stopDispatch           chan struct{}
	stopUpdateDeviceStatus chan struct{}

	// message channel
	deviceStatusChan chan model.Message

	// downstream controller to update device status in cache
	dc *DownstreamController
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	log.LOGGER.Infof("Start upstream device controller")
	uc.stopDispatch = make(chan struct{})
	uc.stopUpdateDeviceStatus = make(chan struct{})

	uc.deviceStatusChan = make(chan model.Message, config.UpdateDeviceStatusBuffer)

	go uc.dispatchMessage(uc.stopDispatch)

	for i := 0; i < config.UpdateDeviceStatusWorkers; i++ {
		go uc.updateDeviceStatus(uc.stopUpdateDeviceStatus)
	}

	return nil
}

func (uc *UpstreamController) dispatchMessage(stop chan struct{}) {
	running := true
	go func() {
		<-stop
		log.LOGGER.Infof("Stop dispatchMessage")
		running = false
	}()
	for running {
		msg, err := uc.messageLayer.Receive()
		if err != nil {
			log.LOGGER.Warnf("Receive message failed, %s", err)
			continue
		}

		log.LOGGER.Infof("Dispatch message: %s", msg.GetID())

		resourceType, err := messagelayer.GetResourceType(msg.GetResource())
		if err != nil {
			log.LOGGER.Warnf("Parse message: %s resource type with error: %s", msg.GetID(), err)
			continue
		}
		log.LOGGER.Infof("Message: %s, resource type is: %s", msg.GetID(), resourceType)

		switch resourceType {
		case constants.ResourceTypeTwinEdgeUpdated:
			uc.deviceStatusChan <- msg
		default:
			log.LOGGER.Warnf("Message: %s, with resource type: %s not intended for device controller", msg.GetID(), resourceType)
		}
	}
}

func (uc *UpstreamController) updateDeviceStatus(stop chan struct{}) {
	running := true
	for running {
		select {
		case msg := <-uc.deviceStatusChan:
			log.LOGGER.Infof("Message: %s, operation is: %s, and resource is: %s", msg.GetID(), msg.GetOperation(), msg.GetResource())
			msgTwin, err := uc.unmarshalDeviceStatusMessage(msg)
			if err != nil {
				log.LOGGER.Warnf("Unmarshall failed due to error %v", err)
				continue
			}
			deviceID, err := messagelayer.GetDeviceID(msg.GetResource())
			if err != nil {
				log.LOGGER.Warnf("Failed to get device id")
				continue
			}
			device, ok := uc.dc.deviceManager.Device.Load(deviceID)
			if !ok {
				log.LOGGER.Warnf("Device %s does not exist in downstream controller", deviceID)
				continue
			}
			cacheDevice, ok := device.(*CacheDevice)
			if !ok {
				log.LOGGER.Warnf("Failed to assert to CacheDevice type")
				continue
			}
			deviceStatus := &DeviceStatus{Status: cacheDevice.Status}
			for twinName, twin := range msgTwin.Twin {
				for i, cacheTwin := range deviceStatus.Status.Twins {
					if twinName == cacheTwin.PropertyName && twin.Actual != nil && twin.Actual.Value != nil {
						reported := v1alpha1.TwinProperty{}
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
				log.LOGGER.Errorf("Failed to marshal device status %v", deviceStatus)
				continue
			}
			result := uc.crdClient.Patch(MergePatchType).Namespace(cacheDevice.Namespace).Resource(ResourceTypeDevices).Name(deviceID).Body(body).Do()
			if result.Error() != nil {
				log.LOGGER.Errorf("Failed to patch device status %v of device %v in namespace %v", deviceStatus, deviceID, cacheDevice.Namespace)
				continue
			}
			log.LOGGER.Infof("Message: %s process successfully", msg.GetID())
		case <-stop:
			log.LOGGER.Infof("Stop updateDeviceStatus")
			running = false
		}
	}
}

func (uc *UpstreamController) unmarshalDeviceStatusMessage(msg model.Message) (*types.DeviceTwinUpdate, error) {
	content := msg.GetContent()
	twinUpdate := &types.DeviceTwinUpdate{}
	bytes, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, twinUpdate)
	if err != nil {
		return nil, err
	}
	return twinUpdate, nil
}

// Stop UpstreamController
func (uc *UpstreamController) Stop() error {
	log.LOGGER.Infof("Stop upstream controller")
	uc.stopDispatch <- struct{}{}

	for i := 0; i < config.UpdateDeviceStatusWorkers; i++ {
		uc.stopUpdateDeviceStatus <- struct{}{}
	}

	return nil
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(dc *DownstreamController) (*UpstreamController, error) {
	config, err := utils.KubeConfig()
	crdcli, err := utils.NewCRDClient(config)
	ml, err := messagelayer.NewMessageLayer()
	if err != nil {
		log.LOGGER.Warnf("Create message layer failed with error: %s", err)
	}
	uc := &UpstreamController{crdClient: crdcli, messageLayer: ml, dc: dc}
	return uc, nil
}
