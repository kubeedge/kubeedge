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
	"reflect"
	"strings"

	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	crdClientset "github.com/kubeedge/kubeedge/cloud/pkg/client/clientset/versioned"
	keclient "github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/messagelayer"
)

// DeviceStatus is structure to patch device status
type DeviceStatus struct {
	Status v1alpha2.DeviceStatus `json:"status"`
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

	// downstream controller to update device status in cache
	dc *DownstreamController
}

// Start UpstreamController
func (uc *UpstreamController) Start() error {
	klog.Info("Start upstream devicecontroller")

	uc.deviceStatusChan = make(chan model.Message, config.Config.Buffer.UpdateDeviceStatus)
	go uc.dispatchMessage()

	for i := 0; i < int(config.Config.Buffer.UpdateDeviceStatus); i++ {
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

		resourceType, err := messagelayer.GetResourceType(msg.GetResource())
		if err != nil {
			klog.Warningf("Parse message: %s resource type with error: %s, resource is %s", msg.GetID(), err, msg.GetResource())
			continue
		}
		klog.Infof("Message: %s, resource type is: %s", msg.GetID(), resourceType)

		switch resourceType {
		case constants.ResourceTypeTwinEdgeUpdated:
			uc.deviceStatusChan <- msg
		default:
			klog.Warningf("Message: %s, with resource: %s not intended for device controller", msg.GetID(), msg.GetResource())
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

			updatedDevice, err := uc.unmarshalDeviceMessage(msg)
			if err != nil {
				klog.Warningf("Unmarshall failed due to error %v", err)
				continue
			}

			deviceID, err := messagelayer.GetDeviceID(msg.GetResource())
			if err != nil {
				klog.Warningf("Failed to get device id, msg resource is %s", msg.GetResource())
				continue
			}
			s := strings.Split(deviceID, "/")
			if len(s) < 2 {
				continue
			}
			deviceName := s[1]
			deviceNamespace := s[0]

			klog.Infof("device ID is %s", deviceID)
			device, ok := uc.dc.deviceManager.Device.Load(deviceID)
			if !ok {
				klog.Warningf("Device %s does not exist in downstream controller", deviceID)
				continue
			}
			cacheDevice, ok := device.(*v1alpha2.Device)
			if !ok {
				klog.Warning("Failed to assert to CacheDevice type")
				continue
			}

			deviceStatus := &DeviceStatus{Status: cacheDevice.Status}
			for _, twin := range updatedDevice.Status.Twins {
				for i, cacheTwin := range deviceStatus.Status.Twins {
					if twin.PropertyName == cacheTwin.PropertyName && !reflect.DeepEqual(twin.Reported, v1alpha2.TwinProperty{}) && twin.Reported.Value != "" {
						reported := twin.Reported
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
			err = uc.crdClient.DevicesV1alpha2().RESTClient().Patch(MergePatchType).Namespace(deviceNamespace).Resource(ResourceTypeDevices).Name(deviceName).Body(body).Do(context.Background()).Error()
			if err != nil {
				klog.Errorf("Failed to patch device status %v of device %v in namespace %v, err: %v", deviceStatus, deviceName, deviceNamespace, err)
				continue
			}
			//send confirm message to edge twin
			resMsg := model.NewMessage(msg.GetID())
			nodeID, err := messagelayer.GetNodeID(msg)
			if err != nil {
				klog.Warningf("Message: %s process failure, get node id failed with error: %s", msg.GetID(), err)
				continue
			}
			resource, err := messagelayer.BuildResource(nodeID, "twin", "")
			if err != nil {
				klog.Warningf("Message: %s process failure, build message resource failed with error: %s", msg.GetID(), err)
				continue
			}
			resMsg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.ResponseOperation)
			resMsg.Content = "OK"
			err = uc.messageLayer.Response(*resMsg)
			if err != nil {
				klog.Warningf("Message: %s process failure, response failed with error: %s", msg.GetID(), err)
				continue
			}
			klog.Infof("Message: %s process successfully", msg.GetID())
		}
	}
}

func (uc *UpstreamController) unmarshalDeviceMessage(msg model.Message) (*v1alpha2.Device, error) {
	contentData, err := msg.GetContentData()
	if err != nil {
		return nil, err
	}

	device := &v1alpha2.Device{}
	if err := json.Unmarshal(contentData, device); err != nil {
		return nil, err
	}
	return device, nil
}

// NewUpstreamController create UpstreamController from config
func NewUpstreamController(dc *DownstreamController) (*UpstreamController, error) {
	uc := &UpstreamController{
		crdClient:    keclient.GetCRDClient(),
		messageLayer: messagelayer.NewContextMessageLayer(),
		dc:           dc,
	}
	return uc, nil
}
