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
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/kubeedge/api/apis/devices/v1beta1"
	crdinformers "github.com/kubeedge/api/client/informers/externalversions"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
	"github.com/kubeedge/kubeedge/pkg/util"
)

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient   kubernetes.Interface
	messageLayer messagelayer.MessageLayer

	deviceManager      *manager.DeviceManager
	deviceModelManager *manager.DeviceModelManager
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
	deviceModelID := util.GetResourceID(deviceModel.Namespace, deviceModel.Name)
	dc.deviceModelManager.DeviceModel.Store(deviceModelID, deviceModel)
}

// deviceModelUpdated is function to process updated deviceModel
func (dc *DownstreamController) deviceModelUpdated(deviceModel *v1beta1.DeviceModel) {
	// nothing to do when deviceModel updated, only add in map
	deviceModelID := util.GetResourceID(deviceModel.Namespace, deviceModel.Name)
	dc.deviceModelManager.DeviceModel.Store(deviceModelID, deviceModel)
}

// deviceModelDeleted is function to process deleted deviceModel
func (dc *DownstreamController) deviceModelDeleted(deviceModel *v1beta1.DeviceModel) {
	// TODO: Need to use finalizer like method to delete all devices referring to this model. Need to come up with a design.
	deviceModelID := util.GetResourceID(deviceModel.Namespace, deviceModel.Name)
	dc.deviceModelManager.DeviceModel.Delete(deviceModelID)
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
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	dc.deviceManager.Device.Store(deviceID, device)
	if device.Spec.NodeName != "" {
		edgeDevice := createDevice(device)
		msg := model.NewMessage("")

		resource, err := messagelayer.BuildResourceForDevice(device.Spec.NodeName, "membership", "")
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}
		msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)

		content := types.MembershipUpdate{AddDevices: []types.Device{
			edgeDevice,
		}}
		content.EventID = uuid.New().String()
		content.Timestamp = time.Now().UnixNano() / 1e6
		msg.Content = content

		err = dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", msg, err)
		}

		if !isExistModel(&dc.deviceManager.Device, device) {
			dc.sendDeviceModelMsg(device, model.InsertOperation)
		}
		dc.sendDeviceMsg(device, model.InsertOperation)
	}
}

// createDevice creates a device from CRD
func createDevice(device *v1beta1.Device) types.Device {
	edgeDevice := types.Device{
		// ID and name can be used as ID as we are using CRD and name(key in ETCD) will always be unique
		ID:   util.GetResourceID(device.Namespace, device.Name),
		Name: device.Name,
	}

	description, ok := device.Labels["description"]
	if ok {
		edgeDevice.Description = description
	}

	return edgeDevice
}

// isExistModel check if the target node already has the model.
func isExistModel(deviceMap *sync.Map, device *v1beta1.Device) bool {
	var res bool
	targetNode := device.Spec.NodeName
	modelName := device.Spec.DeviceModelRef.Name
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	// To find another device in deviceMap that uses the same deviceModel with exclude current device
	deviceMap.Range(func(k, v interface{}) bool {
		if k == deviceID {
			return true
		}
		deviceItem, ok := v.(*v1beta1.Device)
		if !ok {
			return true
		}
		if deviceItem.Spec.NodeName == "" {
			return true
		}
		if deviceItem.Spec.NodeName == targetNode && deviceItem.Namespace == device.Namespace &&
			deviceItem.Spec.DeviceModelRef.Name == modelName {
			res = true
			return false
		}
		return true
	})
	return res
}

// deviceUpdated updates the map, check if device is actually updated.
// If NodeName is updated, call add device for newNode, deleteDevice for old Node.
// If Spec is updated, send update message to edge
func (dc *DownstreamController) deviceUpdated(device *v1beta1.Device) {
	if len(device.Status.Twins) > 0 {
		removeTwinWithNameChanged(device)
	}
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	value, ok := dc.deviceManager.Device.Load(deviceID)
	dc.deviceManager.Device.Store(deviceID, device)
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
	deviceID := util.GetResourceID(device.Namespace, device.Name)
	dc.deviceManager.Device.Delete(deviceID)

	if device.Spec.NodeName != "" {
		edgeDevice := createDevice(device)
		msg := model.NewMessage("")

		resource, err := messagelayer.BuildResourceForDevice(device.Spec.NodeName, "membership", "")
		msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)

		content := types.MembershipUpdate{RemoveDevices: []types.Device{
			edgeDevice,
		}}
		content.EventID = uuid.New().String()
		content.Timestamp = time.Now().UnixNano() / 1e6
		msg.Content = content
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}
		err = dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", msg, err)
		}
		if !isExistModel(&dc.deviceManager.Device, device) {
			dc.sendDeviceModelMsg(device, model.DeleteOperation)
		}
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
	deviceModelID := util.GetResourceID(device.Namespace, device.Spec.DeviceModelRef.Name)
	var edgeDeviceModel any
	var ok bool
	err := retry.Do(
		func() error {
			edgeDeviceModel, ok = dc.deviceModelManager.DeviceModel.Load(deviceModelID)
			if !ok {
				return fmt.Errorf("not found device model for device: %s, operation: %s", device.Name, operation)
			}
			return nil
		},
		retry.Delay(1*time.Second),
		retry.Attempts(10),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		klog.Warning(err.Error())
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
		messageLayer:       messagelayer.DeviceControllerMessageLayer(),
	}
	return dc, nil
}

// Remove twin with changed attribute names.
func removeTwinWithNameChanged(device *v1beta1.Device) {
	properties := device.Spec.Properties
	twins := device.Status.Twins
	newTwins := make([]v1beta1.Twin, 0, len(properties))
	for _, twin := range twins {
		twinName := twin.PropertyName
		for _, property := range properties {
			if property.Name == twinName {
				newTwins = append(newTwins, twin)
				break
			}
		}
	}
	device.Status.Twins = newTwins
}
