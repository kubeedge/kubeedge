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
	"reflect"
	"strconv"
	"time"

	"github.com/google/uuid"
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

	deviceManager *manager.DeviceManager
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
			case watch.Deleted:
				dc.deviceDeleted(device)
			case watch.Modified:
				dc.deviceUpdated(device)
			default:
				klog.Warningf("Device event type: %s unsupported", e.Type)
			}
		}
	}
}

// deviceAdded creates a device, adds in deviceManagers map, send a message to edge node if node selector is present.
func (dc *DownstreamController) deviceAdded(device *v1beta1.Device) {
	dc.deviceManager.Device.Store(device.Name, device)
	if len(device.Spec.NodeName) > 0 {
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

	// TODO: optional is Always false, currently not present in CRD definition, need to add or remove from deviceTwin @ Edge
	opt := false
	optional := &opt
	twin := make(map[string]*types.MsgTwin, len(device.Status.Twins))
	for i, dtwin := range device.Status.Twins {
		expected := &types.TwinValue{}
		expected.Value = &device.Status.Twins[i].Desired.Value
		metadataType, ok := device.Status.Twins[i].Desired.Metadata["type"]
		if !ok {
			metadataType = "string"
		}
		timestamp := time.Now().UnixNano() / 1e6

		metadata := &types.ValueMetadata{Timestamp: timestamp}
		expected.Metadata = metadata

		// TODO: how to manage versioning ??
		cloudVersion, err := strconv.ParseInt(device.ResourceVersion, 10, 64)
		if err != nil {
			klog.Warningf("Failed to parse cloud version due to error %v", err)
		}
		twinVersion := &types.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
		msgTwin := &types.MsgTwin{
			Expected:        expected,
			Optional:        optional,
			Metadata:        &types.TypeMetadata{Type: metadataType},
			ExpectedVersion: twinVersion,
		}
		twin[dtwin.PropertyName] = msgTwin
	}
	edgeDevice.Twin = twin
	return edgeDevice
}

// isDeviceUpdated checks if device is actually updated
func isDeviceUpdated(oldTwin *v1beta1.Device, newTwin *v1beta1.Device) bool {
	// does not care fields
	oldTwin.ObjectMeta.ResourceVersion = newTwin.ObjectMeta.ResourceVersion
	oldTwin.ObjectMeta.Generation = newTwin.ObjectMeta.Generation
	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(oldTwin.ObjectMeta, newTwin.ObjectMeta) || !reflect.DeepEqual(oldTwin.Spec, newTwin.Spec) || !reflect.DeepEqual(oldTwin.Status, newTwin.Status)
}

// isDeviceStatusUpdated checks if DeviceStatus is updated
func isDeviceStatusUpdated(oldTwin *v1beta1.DeviceStatus, newTwin *v1beta1.DeviceStatus) bool {
	return !reflect.DeepEqual(oldTwin, newTwin)
}

// isDeviceDataUpdated checks if DeviceData is updated
func isDeviceDataUpdated(oldData *v1beta1.DeviceData, newData *v1beta1.DeviceData) bool {
	return !reflect.DeepEqual(oldData, newData)
}

// deviceUpdated updates the map, check if device is actually updated.
// If nodeSelector is updated, call add device for newNode, deleteDevice for old Node.
// If twin is updated, send twin update message to edge
func (dc *DownstreamController) deviceUpdated(device *v1beta1.Device) {
	value, ok := dc.deviceManager.Device.Load(device.Name)
	dc.deviceManager.Device.Store(device.Name, device)
	if ok {
		cachedDevice := value.(*v1beta1.Device)
		if isDeviceUpdated(cachedDevice, device) {
			// if node selector updated delete from old node and create in new node
			if cachedDevice.Spec.NodeName != device.Spec.NodeName {
				deletedDevice := &v1beta1.Device{ObjectMeta: cachedDevice.ObjectMeta,
					Spec:     cachedDevice.Spec,
					Status:   cachedDevice.Status,
					TypeMeta: device.TypeMeta,
				}
				dc.deviceDeleted(deletedDevice)
				dc.deviceAdded(device)
			} else {
				// update twin properties
				if isDeviceStatusUpdated(&cachedDevice.Status, &device.Status) {
					// TODO: add an else if condition to check if DeviceModelReference has changed, if yes whether deviceModelReference exists
					twin := make(map[string]*types.MsgTwin)
					addUpdatedTwins(device.Status.Twins, twin, device.ResourceVersion)
					addDeletedTwins(cachedDevice.Status.Twins, device.Status.Twins, twin, device.ResourceVersion)
					msg := model.NewMessage("")

					resource, err := messagelayer.BuildResourceForDevice(device.Spec.NodeName, "device/"+device.Name+"/twin/cloud_updated", "")
					if err != nil {
						klog.Warningf("Built message resource failed with error: %s", err)
						return
					}
					msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)
					content := types.DeviceTwinUpdate{Twin: twin}
					content.EventID = uuid.New().String()
					content.Timestamp = time.Now().UnixNano() / 1e6
					msg.Content = content

					err = dc.messageLayer.Send(*msg)
					if err != nil {
						klog.Errorf("Failed to send deviceTwin message %v due to error %v", msg, err)
					}
				}
				// distribute device model
				if isDeviceStatusUpdated(&cachedDevice.Status, &device.Status) ||
					isDeviceDataUpdated(&cachedDevice.Spec.Data, &device.Spec.Data) {
					dc.sendDeviceMsg(device, model.UpdateOperation)
				}
			}
		}
	} else {
		// If device not present in device map means it is not modified and added.
		dc.deviceAdded(device)
	}
}

// addDeletedTwins add deleted twins in the message
func addDeletedTwins(oldTwin []v1beta1.Twin, newTwin []v1beta1.Twin, twin map[string]*types.MsgTwin, version string) {
	opt := false
	optional := &opt
	for i, dtwin := range oldTwin {
		if !ifTwinPresent(dtwin, newTwin) {
			expected := &types.TwinValue{}
			expected.Value = &oldTwin[i].Desired.Value
			timestamp := time.Now().UnixNano() / 1e6

			metadata := &types.ValueMetadata{Timestamp: timestamp}
			expected.Metadata = metadata

			// TODO: how to manage versioning ??
			cloudVersion, err := strconv.ParseInt(version, 10, 64)
			if err != nil {
				klog.Warningf("Failed to parse cloud version due to error %v", err)
			}
			twinVersion := &types.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
			msgTwin := &types.MsgTwin{
				Expected:        expected,
				Optional:        optional,
				Metadata:        &types.TypeMetadata{Type: "deleted"},
				ExpectedVersion: twinVersion,
			}
			twin[dtwin.PropertyName] = msgTwin
		}
	}
}

// ifTwinPresent checks if twin is present in the array of twins
func ifTwinPresent(twin v1beta1.Twin, newTwins []v1beta1.Twin) bool {
	for _, dtwin := range newTwins {
		if twin.PropertyName == dtwin.PropertyName {
			return true
		}
	}
	return false
}

// addUpdatedTwins is function of add updated twins to send to edge
func addUpdatedTwins(newTwin []v1beta1.Twin, twin map[string]*types.MsgTwin, version string) {
	opt := false
	optional := &opt
	for i, dtwin := range newTwin {
		expected := &types.TwinValue{}
		expected.Value = &newTwin[i].Desired.Value
		metadataType, ok := newTwin[i].Desired.Metadata["type"]
		if !ok {
			metadataType = "string"
		}
		timestamp := time.Now().UnixNano() / 1e6

		metadata := &types.ValueMetadata{Timestamp: timestamp}
		expected.Metadata = metadata

		// TODO: how to manage versioning ??
		cloudVersion, err := strconv.ParseInt(version, 10, 64)
		if err != nil {
			klog.Warningf("Failed to parse cloud version due to error %v", err)
		}
		twinVersion := &types.TwinVersion{CloudVersion: cloudVersion, EdgeVersion: 0}
		msgTwin := &types.MsgTwin{
			Expected:        expected,
			Optional:        optional,
			Metadata:        &types.TypeMetadata{Type: metadataType},
			ExpectedVersion: twinVersion,
		}
		twin[dtwin.PropertyName] = msgTwin
	}
}

func (dc *DownstreamController) sendDeviceMsg(device *v1beta1.Device, operation string) {
	device.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   v1beta1.GroupName,
		Version: v1beta1.Version,
		Kind:    constants.KindTypeDevice,
	})
	deviceMsg := model.NewMessage("").
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
	deviceMsg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupResource, modelResource, operation)

	err = dc.messageLayer.Send(*deviceMsg)
	if err != nil {
		klog.Errorf("Failed to send device addition message %v, device: %s, operation: %s, error: %v",
			deviceMsg, device.Name, operation, err)
	}
	klog.Infof("send msg: %v", deviceMsg.Router)
}

// deviceDeleted send a deleted message to the edgeNode and deletes the device from the deviceManager.Device map
func (dc *DownstreamController) deviceDeleted(device *v1beta1.Device) {
	dc.deviceManager.Device.Delete(device.Name)
	edgeDevice := createDevice(device)
	msg := model.NewMessage("")

	if len(device.Spec.NodeName) > 0 {
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
		dc.sendDeviceMsg(device, model.DeleteOperation)
	}
}

// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start downstream devicecontroller")

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

	dc := &DownstreamController{
		kubeClient:    client.GetKubeClient(),
		deviceManager: deviceManager,
		messageLayer:  messagelayer.DeviceControllerMessageLayer(),
	}
	return dc, nil
}
