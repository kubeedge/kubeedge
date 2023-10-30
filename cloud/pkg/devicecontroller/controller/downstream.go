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
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	"github.com/kubeedge/kubeedge/pkg/apis/devices/v1beta1"
	crdinformers "github.com/kubeedge/kubeedge/pkg/client/informers/externalversions"
)

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient   kubernetes.Interface
	messageLayer messagelayer.MessageLayer

	deviceManager      *manager.DeviceManager
	deviceModelManager *manager.DeviceModelManager
	mapperManager      *manager.MapperManager
}

// syncDeviceModel is used to get events from informer
func (dc *DownstreamController) syncDeviceModel() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop syncDeviceModel")
			return
		case e := <-dc.deviceModelManager.Events():
			deviceModel, ok := e.Object.(*v1beta1.DeviceModel)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.deviceModelAdded(deviceModel)
			case watch.Modified:
				dc.deviceModelUpdated(deviceModel)
			case watch.Deleted:
				dc.deviceModelDeleted(deviceModel)
			default:
				klog.Warningf("deviceModel event type: %s unsupported", e.Type)
			}
		}
	}
}

// deviceModelAdded is function to process addition of new deviceModel in apiserver
func (dc *DownstreamController) deviceModelAdded(deviceModel *v1beta1.DeviceModel) {
	// nothing to do when deviceModel added, only add in map
	dc.deviceModelManager.DeviceModel.Store(deviceModel.Name, deviceModel)
}

// deviceModelUpdated is function to process updated deviceModel
func (dc *DownstreamController) deviceModelUpdated(deviceModel *v1beta1.DeviceModel) {
	// nothing to do when deviceModel updated, only add in map
	dc.deviceModelManager.DeviceModel.Store(deviceModel.Name, deviceModel)
}

// deviceModelDeleted is function to process deleted deviceModel
func (dc *DownstreamController) deviceModelDeleted(deviceModel *v1beta1.DeviceModel) {
	// TODO: Need to use finalizer like method to delete all devices referring to this model. Need to come up with a design.
	dc.deviceModelManager.DeviceModel.Delete(deviceModel.Name)
}

// syncDevice is used to get device events from informer
func (dc *DownstreamController) syncDevice() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop syncDevice")
			return
		case e := <-dc.deviceManager.Events():
			device, ok := e.Object.(*v1beta1.Device)
			if !ok {
				klog.Warningf("Object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.deviceAdded(device)
			case watch.Modified:
				dc.deviceUpdated(device)
			case watch.Deleted:
				dc.deviceDeleted(device)
			default:
				klog.Warningf("Device event type: %s unsupported", e.Type)
			}
		}
	}
}

// deviceAdded creates a device, adds in deviceManagers map, send a message to edge node if node selector is present.
func (dc *DownstreamController) deviceAdded(device *v1beta1.Device) {
	// Nothing to do when device added, only add in UndeployedDevice map
	dc.deviceManager.UndeployedDevice.Store(device.Name, device)
	// Determine which node the Device should deploy
	nodeID := ""
	if device.Spec.NodeName != "" {
		// If device.Spec.NodeName is present, directly define the node where the device should be deployed
		nodeID = device.Spec.NodeName
	} else {
		// If device.Spec.NodeName is not present
		// Find the node where device.MapperRef corresponds to mapper deployed
		value, ok := dc.mapperManager.Mapper2NodeMap.Load(device.Spec.MapperRef.Name)
		if ok {
			nodeID = value.(string)
			device.Spec.NodeName = nodeID
		}
	}
	if nodeID != "" {
		// Add the device to the DeloyedDevice Map
		dc.deviceManager.DeployedDevice.Store(device.Name, device)
		dc.deviceManager.UndeployedDevice.Delete(device.Name)
		// Add the device to the NodeDeviceList Map
		value, ok := dc.deviceManager.NodeDeviceList.Load(nodeID)
		if ok {
			// If the nodeID exists in the map, append the device to the device list
			deviceList := value.([]*v1beta1.Device)
			deviceList = append(deviceList, device)
			dc.deviceManager.NodeDeviceList.Store(nodeID, deviceList)
		} else {
			// If the nodeID does not exist in the map, create a new device list
			deviceList := make([]*v1beta1.Device, 0)
			deviceList = append(deviceList, device)
			dc.deviceManager.NodeDeviceList.Store(nodeID, deviceList)
		}
		devicemodelMsg := model.NewMessage("")
		resource, err := messagelayer.BuildResourceForDevice(nodeID, "devicemodel", "")
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}
		devicemodelMsg.BuildRouter("meta", constants.GroupTwin, resource, model.InsertOperation)
		value, ok = dc.deviceModelManager.DeviceModel.Load(device.Spec.DeviceModelRef.Name)
		var devicemodel *v1beta1.DeviceModel
		if ok {
			devicemodel = value.(*v1beta1.DeviceModel)
		} else {
			fmt.Println("deviceModel not found")
		}
		devicemodelMsg.Content, _ = json.Marshal(*devicemodel)
		err = dc.messageLayer.Send(*devicemodelMsg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", devicemodelMsg, err)
		}

		deviceMsg := model.NewMessage("")
		resource, err = messagelayer.BuildResourceForDevice(nodeID, "device", "")
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}
		deviceMsg.BuildRouter("meta", constants.GroupTwin, resource, model.InsertOperation)
		deviceMsg.Content, _ = json.Marshal(*device)
		err = dc.messageLayer.Send(*deviceMsg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", deviceMsg, err)
		}

		dc.sendDeviceModelMsg(device, model.InsertOperation)
		dc.sendDeviceMsg(device, model.InsertOperation)
	}
}

// createDevice creates a device from CRD
func createDevice(device *v1beta1.Device) types.Device {
	edgeDevice := types.Device{
		// ID and name can be used as ID as we are using CRD and name(key in ETCD) will always be unique
		ID:   device.Name,
		Name: device.Name,
	}

	description, ok := device.Labels["description"]
	if ok {
		edgeDevice.Description = description
	}

	return edgeDevice
}

// deviceUpdated updates the map, check if device is actually updated.
// If NodeName is updated, call add device for newNode, deleteDevice for old Node.
// If Spec is updated, send update message to edge
func (dc *DownstreamController) deviceUpdated(device *v1beta1.Device) {
	value, ok := dc.deviceManager.DeployedDevice.Load(device.Name)
	dc.deviceManager.DeployedDevice.Store(device.Name, device)
	if ok {
		cachedDevice := value.(*v1beta1.Device)
		if isDeviceUpdated(cachedDevice, device) {
			// if NodeName changed, delete from old node and create in new node
			if cachedDevice.Spec.NodeName != device.Spec.NodeName {
				deletedDevice := &v1beta1.Device{ObjectMeta: cachedDevice.ObjectMeta,
					Spec:     cachedDevice.Spec,
					Status:   cachedDevice.Status,
					TypeMeta: device.TypeMeta,
				}
				dc.deviceDeleted(deletedDevice)
				dc.deviceAdded(device)
			} else {
				dc.sendDeviceModelMsg(device, model.UpdateOperation)
				dc.sendDeviceMsg(device, model.UpdateOperation)
			}
		}
	} else {
		// If device not present in device map means it is not modified and added.
		dc.deviceAdded(device)
	}
}

// isDeviceUpdated checks if device is actually updated
func isDeviceUpdated(oldTwin *v1beta1.Device, newTwin *v1beta1.Device) bool {
	// does not care fields
	oldTwin.ObjectMeta.ResourceVersion = newTwin.ObjectMeta.ResourceVersion
	oldTwin.ObjectMeta.Generation = newTwin.ObjectMeta.Generation
	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(oldTwin.ObjectMeta, newTwin.ObjectMeta) || !reflect.DeepEqual(oldTwin.Spec, newTwin.Spec)
}

// deviceDeleted send a deleted message to the edgeNode and deletes the device from the deviceManager.Device map
func (dc *DownstreamController) deviceDeleted(device *v1beta1.Device) {
	// Init nodeID to empty string
	nodeID := ""
	if device.Status.CurrentNode != "" {
		// If device.Status.CurrentNode is present
		// it means device is deployed on a node and is registered successfully, delete from that node
		nodeID = device.Status.CurrentNode
		dc.deviceManager.DeployedDevice.Delete(device.Name)
	} else if device.Spec.NodeName != "" {
		// Else if device.Spec.NodeName is present
		// it means device is deployed on a node but has not yet been registered successfully, delete from that node
		nodeID = device.Spec.NodeName
		dc.deviceManager.DeployedDevice.Delete(device.Name)
	} else {
		// device is not deployed on any node, delete from undeployedDevice map
		dc.deviceManager.UndeployedDevice.Delete(device.Name)
	}
	if nodeID != "" {
		msg := model.NewMessage("")
		value, ok := dc.deviceManager.NodeDeviceList.Load(nodeID)
		if ok {
			// Remove device from nodeDeviceList
			deviceList := value.([]*v1beta1.Device)
			for i := range deviceList {
				if deviceList[i].Name == device.Name {
					deviceList = append(deviceList[:i], deviceList[i+1:]...)
					dc.deviceManager.NodeDeviceList.Store(nodeID, deviceList)
					break
				}
			}
		}
		resource, err := messagelayer.BuildResourceForDevice(nodeID, "device", "")
		msg.BuildRouter("meta", constants.GroupTwin, resource, model.DeleteOperation)

		msg.Content, _ = json.Marshal(*device)
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}
		err = dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", msg, err)
		}
		dc.sendDeviceModelMsg(device, model.DeleteOperation)
		dc.sendDeviceMsg(device, model.DeleteOperation)
	}
}

func (dc *DownstreamController) sendDeviceMsg(device *v1beta1.Device, operation string) {
	device.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   v1beta1.GroupName,
		Version: v1beta1.Version,
		Kind:    constants.KindTypeDevice,
	})
	modelMsg := model.NewMessage("").
		SetResourceVersion(device.ResourceVersion).
		FillBody(device)
	modelResource, err := messagelayer.BuildResource(
		device.Spec.NodeName,
		device.Namespace,
		constants.ResourceTypeDevice,
		device.Name)
	if err != nil {
		klog.Warningf("Built message resource failed for device, device: %s, operation: %s, error: %s", device.Name, operation, err)
		return
	}

	// filter operation
	switch operation {
	case model.InsertOperation:
	case model.DeleteOperation:
	case model.UpdateOperation:
	default:
		klog.Warningf("unknown operation %s for device %s when send device msg", operation, device.Name)
		return
	}
	modelMsg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupResource, modelResource, operation)

	err = dc.messageLayer.Send(*modelMsg)
	if err != nil {
		klog.Errorf("Failed to send device addition message %v, device: %s, operation: %s, error: %v",
			modelMsg, device.Name, operation, err)
	}
}

func (dc *DownstreamController) sendDeviceModelMsg(device *v1beta1.Device, operation string) {
	// send operate msg for device model
	// now it is depended on device, maybe move this code to syncDeviceModel's method
	if device == nil || device.Spec.DeviceModelRef == nil {
		return
	}
	edgeDeviceModel, ok := dc.deviceModelManager.DeviceModel.Load(device.Spec.DeviceModelRef.Name)
	if !ok {
		klog.Warningf("not found device model for device: %s, operation: %s", device.Name, operation)
		return
	}

	deviceModel, ok := edgeDeviceModel.(*v1beta1.DeviceModel)
	if !ok {
		klog.Warningf("edgeDeviceModel is not *v1beta1.DeviceModel for device: %s, operation: %s", device.Name, operation)
		return
	}

	deviceModel.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   v1beta1.GroupName,
		Version: v1beta1.Version,
		Kind:    constants.KindTypeDeviceModel,
	})
	modelMsg := model.NewMessage("").
		SetResourceVersion(deviceModel.ResourceVersion).
		FillBody(deviceModel)
	modelResource, err := messagelayer.BuildResource(
		device.Spec.NodeName,
		deviceModel.Namespace,
		constants.ResourceTypeDeviceModel,
		deviceModel.Name)
	if err != nil {
		klog.Warningf("Built message resource failed for device model, device: %s, operation: %s, error: %s", device.Name, operation, err)
		return
	}

	// filter operation
	switch operation {
	case model.InsertOperation:
	case model.DeleteOperation:
	case model.UpdateOperation:
	default:
		klog.Warningf("unknown operation %s for device %s when send device model msg", operation, device.Name)
		return
	}
	modelMsg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupResource, modelResource, operation)

	err = dc.messageLayer.Send(*modelMsg)
	if err != nil {
		klog.Errorf("Failed to send device model addition message %v, device: %s, operation: %s, error: %v",
			modelMsg, device.Name, operation, err)
	}
}

// mapperDeleted delete records about node with nodeID in mapperManager
func (dc *DownstreamController) mapperMigrated(nodeID string) error {
	// When the node is disconnected, the mapper on the node needs to be deleted
	value, ok := dc.mapperManager.NodeMapperList.Load(nodeID)
	if ok {
		mapperList := value.([]*types.Mapper)
		for _, mapper := range mapperList {
			// delete mapper in Mapper2NodeMap
			dc.mapperManager.Mapper2NodeMap.Delete(mapper.Name)
			msg := model.NewMessage("")
			resource, err := messagelayer.BuildResourceForDevice(nodeID, "mapper", "")
			if err != nil {
				klog.Errorf("Built message resource failed with error: %s", err)
			}
			msg.BuildRouter("meta", constants.GroupTwin, resource, model.DeleteOperation)
			msg.Content, err = json.Marshal(mapper)
			if err != nil {
				klog.Warningf("Failed to marshal mapper %v due to error %v", mapper, err)
			}
			err = dc.messageLayer.Send(*msg)
			if err != nil {
				klog.Errorf("Failed to send mapper deletion message %v due to error %v", msg, err)
			}
		}
		// delete nodeId in NodeMapperList
		dc.mapperManager.NodeMapperList.Delete(nodeID)
	}
	return nil
}

// deviceMigrated indicates the device to be migrated on the node with nodeID
func (dc *DownstreamController) deviceMigrated(nodeID string) error {
	value, ok := dc.deviceManager.NodeDeviceList.Load(nodeID)
	if ok {
		deviceList := value.([]*v1beta1.Device)
		for i := range deviceList {
			// if device.Spec.MigrateOnOffline is true, enable device migration
			if deviceList[i].Spec.MigrateOnOffline {
				device := deviceList[i].DeepCopy()
				device.Status.CurrentNode = ""
				device.Spec.NodeName = ""
				dc.deviceUpdated(device)
			}
		}
	}
	return nil
}

// deviceDeployed finds device whose Spec.MapperRef.Name equals to mappername,
// deletes the device from the deviceManager.UndeployedDevice map and deploy the device to node with nodeID
func (dc *DownstreamController) deviceDeployed(mappername string) error {
	// find device whose Spec.MapperRef.Name equals to mappername, deploy the device to node where the mapper is deployed
	dc.deviceManager.UndeployedDevice.Range(func(key, value interface{}) bool {
		device := value.(*v1beta1.Device)
		if device.Spec.MapperRef.Name == mappername {
			value, ok := dc.mapperManager.Mapper2NodeMap.Load(mappername)
			if ok {
				nodeID := value.(string)
				device.Spec.NodeName = nodeID
				dc.deviceAdded(device)
			}
		}
		return true
	})
	return nil
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start downstream devicecontroller")

	go dc.syncDeviceModel()

	// Wait for adding all device model
	// TODO need to think about sync
	time.Sleep(1 * time.Second)
	go dc.syncDevice()

	return nil
}

// NewDownstreamController create a DownstreamController from config
func NewDownstreamController(crdInformerFactory crdinformers.SharedInformerFactory) (*DownstreamController, error) {
	deviceManager, err := manager.NewDeviceManager(crdInformerFactory.Devices().V1beta1().Devices().Informer())
	if err != nil {
		klog.Warningf("Create device manager failed with error: %s", err)
		return nil, err
	}

	deviceModelManager, err := manager.NewDeviceModelManager(crdInformerFactory.Devices().V1beta1().DeviceModels().Informer())
	if err != nil {
		klog.Warningf("Create device manager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:         client.GetKubeClient(),
		deviceManager:      deviceManager,
		deviceModelManager: deviceModelManager,
		mapperManager:      manager.NewMapperManager(),
		messageLayer:       messagelayer.DeviceControllerMessageLayer(),
	}
	return dc, nil
}
