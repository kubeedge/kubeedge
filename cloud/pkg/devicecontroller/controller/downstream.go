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
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	crdinformers "github.com/kubeedge/kubeedge/cloud/pkg/client/informers/externalversions"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/constants"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/manager"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/devicecontroller/types"
)

// Constants for protocol, datatype, configmap, deviceProfile
const (
	OPCUA              = "opcua"
	Modbus             = "modbus"
	Bluetooth          = "bluetooth"
	CustomizedProtocol = "customized-protocol"

	DataTypeInt     = "int"
	DataTypeString  = "string"
	DataTypeDouble  = "double"
	DataTypeFloat   = "float"
	DataTypeBoolean = "boolean"
	DataTypeBytes   = "bytes"

	ConfigMapKind    = "ConfigMap"
	ConfigMapVersion = "v1"

	DeviceProfileConfigPrefix = "device-profile-config-"

	DeviceProfileJSON = "deviceProfile.json"
)

// DownstreamController watch kubernetes api server and send change to edge
type DownstreamController struct {
	kubeClient   kubernetes.Interface
	messageLayer messagelayer.MessageLayer

	deviceManager      *manager.DeviceManager
	deviceModelManager *manager.DeviceModelManager
	configMapManager   *manager.ConfigMapManager

	crdClient *rest.RESTClient
}

// syncDeviceModel is used to get events from informer
func (dc *DownstreamController) syncDeviceModel() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop syncDeviceModel")
			return
		case e := <-dc.deviceModelManager.Events():
			deviceModel, ok := e.Object.(*v1alpha2.DeviceModel)
			if !ok {
				klog.Warningf("object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				dc.deviceModelAdded(deviceModel)
			case watch.Deleted:
				dc.deviceModelDeleted(deviceModel)
			case watch.Modified:
				dc.deviceModelUpdated(deviceModel)
			default:
				klog.Warningf("deviceModel event type: %s unsupported", e.Type)
			}
		}
	}
}

// deviceModelAdded is function to process addition of new deviceModel in apiserver
func (dc *DownstreamController) deviceModelAdded(deviceModel *v1alpha2.DeviceModel) {
	// nothing to do when deviceModel added, only add in map
	dc.deviceModelManager.DeviceModel.Store(deviceModel.Name, deviceModel)
}

// isDeviceModelUpdated is function to check if deviceModel is actually updated
func isDeviceModelUpdated(oldTwin *v1alpha2.DeviceModel, newTwin *v1alpha2.DeviceModel) bool {
	// does not care fields
	oldTwin.ObjectMeta.ResourceVersion = newTwin.ObjectMeta.ResourceVersion
	oldTwin.ObjectMeta.Generation = newTwin.ObjectMeta.Generation

	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(oldTwin.ObjectMeta, newTwin.ObjectMeta) || !reflect.DeepEqual(oldTwin.Spec, newTwin.Spec)
}

// deviceModelUpdated is function to process updated deviceModel
func (dc *DownstreamController) deviceModelUpdated(deviceModel *v1alpha2.DeviceModel) {
	value, ok := dc.deviceModelManager.DeviceModel.Load(deviceModel.Name)
	dc.deviceModelManager.DeviceModel.Store(deviceModel.Name, deviceModel)
	if ok {
		cachedDeviceModel := value.(*v1alpha2.DeviceModel)
		if isDeviceModelUpdated(cachedDeviceModel, deviceModel) {
			dc.updateAllConfigMaps(deviceModel)
		}
	} else {
		dc.deviceModelAdded(deviceModel)
	}
}

// updateAllConfigMaps is function to update configMaps which refer to an updated deviceModel
func (dc *DownstreamController) updateAllConfigMaps(deviceModel *v1alpha2.DeviceModel) {
	//TODO: add logic to update all config maps, How to manage if a property is deleted but a device is referring that property. Need to come up with a design.
}

// deviceModelDeleted is function to process deleted deviceModel
func (dc *DownstreamController) deviceModelDeleted(deviceModel *v1alpha2.DeviceModel) {
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
			device, ok := e.Object.(*v1alpha2.Device)
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

// addToConfigMap adds device in the configmap
func (dc *DownstreamController) addToConfigMap(device *v1alpha2.Device) {
	configMap, ok := dc.configMapManager.ConfigMap.Load(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
	if !ok {
		nodeConfigMap := &v1.ConfigMap{}
		nodeConfigMap.Name = DeviceProfileConfigPrefix + device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
		nodeConfigMap.Namespace = device.Namespace
		nodeConfigMap.Data = make(map[string]string)
		// TODO: how to handle 2 device of multiple namespaces bind to same node ?
		dc.addDeviceProfile(device, nodeConfigMap)
		// store new config map
		dc.configMapManager.ConfigMap.Store(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], nodeConfigMap)

		if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Get(context.Background(), nodeConfigMap.Name, metav1.GetOptions{}); err != nil {
			if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Create(context.Background(), nodeConfigMap, metav1.CreateOptions{}); err != nil {
				klog.Errorf("Failed to create config map %v in namespace %v, error %v", nodeConfigMap, device.Namespace, err)
				return
			}
		}
		if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Update(context.Background(), nodeConfigMap, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Failed to update config map %v in namespace %v, error %v", nodeConfigMap, device.Namespace, err)
			return
		}
		return
	}
	nodeConfigMap, ok := configMap.(*v1.ConfigMap)
	if !ok {
		klog.Error("Failed to assert to configmap")
		return
	}
	dc.addDeviceProfile(device, nodeConfigMap)
	// store new config map
	dc.configMapManager.ConfigMap.Store(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], nodeConfigMap)
	if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Update(context.Background(), nodeConfigMap, metav1.UpdateOptions{}); err != nil {
		klog.Errorf("Failed to update config map %v in namespace %v", nodeConfigMap, device.Namespace)
		return
	}
}

// addDeviceProfile is function to add deviceProfile in configMap
func (dc *DownstreamController) addDeviceProfile(device *v1alpha2.Device, configMap *v1.ConfigMap) {
	deviceProfile := &types.DeviceProfile{}
	dp, ok := configMap.Data[DeviceProfileJSON]
	if !ok {
		// create deviceProfileStruct
		deviceProfile.DeviceInstances = make([]*v1alpha2.Device, 0)
		deviceProfile.DeviceModels = make([]*types.DeviceModel, 0)
		//deviceProfile.PropertyVisitors = make([]*types.PropertyVisitor, 0)
		//deviceProfile.Protocols = make([]*types.Protocol, 0)
	} else {
		err := json.Unmarshal([]byte(dp), deviceProfile)
		if err != nil {
			klog.Errorf("Failed to Unmarshal deviceprofile: %v", deviceProfile)
			return
		}
	}

	addDeviceInstanceAndProtocol(device, deviceProfile)
	dm, ok := dc.deviceModelManager.DeviceModel.Load(device.Spec.DeviceModelRef.Name)
	if !ok {
		klog.Errorf("Failed to get device model %v", device.Spec.DeviceModelRef.Name)
		return
	}
	deviceModel := dm.(*v1alpha2.DeviceModel)
	// if model already exists no need to add model and visitors
	checkModelExists := false
	for _, dm := range deviceProfile.DeviceModels {
		if dm.Name == deviceModel.Name {
			checkModelExists = true
			break
		}
	}
	if !checkModelExists {
		addDeviceModelAndVisitors(deviceModel, deviceProfile)
	}
	bytes, err := json.Marshal(deviceProfile)
	if err != nil {
		klog.Errorf("Failed to marshal deviceprofile: %v", deviceProfile)
		return
	}
	configMap.Data[DeviceProfileJSON] = string(bytes)
}

// addDeviceModelAndVisitors adds deviceModels and deviceVisitors in configMap
func addDeviceModelAndVisitors(deviceModel *v1alpha2.DeviceModel, deviceProfile *types.DeviceProfile) {
	// todo: model should support namespace, device refactor introduced
	model := &types.DeviceModel{}
	model.Name = deviceModel.Name
	model.Properties = make([]*types.Property, 0, len(deviceModel.Spec.Properties))
	for _, ppt := range deviceModel.Spec.Properties {
		property := &types.Property{}
		property.Name = ppt.Name
		property.Description = ppt.Description
		if ppt.Type.Int != nil {
			property.AccessMode = string(ppt.Type.Int.AccessMode)
			property.DataType = DataTypeInt
			property.DefaultValue = ppt.Type.Int.DefaultValue
			property.Maximum = ppt.Type.Int.Maximum
			property.Minimum = ppt.Type.Int.Minimum
			property.Unit = ppt.Type.Int.Unit
		} else if ppt.Type.String != nil {
			property.AccessMode = string(ppt.Type.String.AccessMode)
			property.DataType = DataTypeString
			property.DefaultValue = ppt.Type.String.DefaultValue
		} else if ppt.Type.Double != nil {
			property.AccessMode = string(ppt.Type.Double.AccessMode)
			property.DataType = DataTypeDouble
			property.DefaultValue = ppt.Type.Double.DefaultValue
			property.Maximum = ppt.Type.Double.Maximum
			property.Minimum = ppt.Type.Double.Minimum
			property.Unit = ppt.Type.Double.Unit
		} else if ppt.Type.Float != nil {
			property.AccessMode = string(ppt.Type.Float.AccessMode)
			property.DataType = DataTypeFloat
			property.DefaultValue = ppt.Type.Float.DefaultValue
			property.Maximum = ppt.Type.Float.Maximum
			property.Minimum = ppt.Type.Float.Minimum
			property.Unit = ppt.Type.Float.Unit
		} else if ppt.Type.Boolean != nil {
			property.AccessMode = string(ppt.Type.Boolean.AccessMode)
			property.DataType = DataTypeBoolean
			property.DefaultValue = ppt.Type.Boolean.DefaultValue
		} else if ppt.Type.Bytes != nil {
			property.AccessMode = string(ppt.Type.Bytes.AccessMode)
			property.DataType = DataTypeBytes
		}
		model.Properties = append(model.Properties, property)
	}
	deviceProfile.DeviceModels = append(deviceProfile.DeviceModels, model)
}

// addDeviceInstanceAndProtocol adds deviceInstance and protocol in configMap
func addDeviceInstanceAndProtocol(device *v1alpha2.Device, deviceProfile *types.DeviceProfile) {
	deviceProtocol := &types.Protocol{}

	if device.Spec.Protocol.Common != nil {
		deviceProtocol.ProtocolCommonConfig = device.Spec.Protocol.Common
	}
	var protocol string
	if device.Spec.Protocol.OpcUA != nil {
		// todo: need use namespace
		protocol = OPCUA + "-" + device.Namespace + "/" + device.Name
		deviceProtocol.Name = protocol
		deviceProtocol.Protocol = OPCUA
		deviceProtocol.ProtocolConfig = device.Spec.Protocol.OpcUA
	} else if device.Spec.Protocol.Modbus != nil {
		protocol = Modbus + "-" + device.Namespace + "/" + device.Name
		deviceProtocol.Name = protocol
		deviceProtocol.Protocol = Modbus
		deviceProtocol.ProtocolConfig = device.Spec.Protocol.Modbus
	} else if device.Spec.Protocol.Bluetooth != nil {
		protocol = Bluetooth + "-" + device.Namespace + "/" + device.Name
		deviceProtocol.Name = protocol
		deviceProtocol.Protocol = Bluetooth
		deviceProtocol.ProtocolConfig = device.Spec.Protocol.Bluetooth
	} else if device.Spec.Protocol.CustomizedProtocol != nil {
		protocol = CustomizedProtocol + "-" + device.Namespace + "/" + device.Name

		deviceProtocol.Name = protocol
		deviceProtocol.Protocol = CustomizedProtocol
		deviceProtocol.ProtocolConfig = device.Spec.Protocol.CustomizedProtocol
	} else {
		klog.Warning("Device doesn't support valid protocol")
	}

	deviceProfile.DeviceInstances = append(deviceProfile.DeviceInstances, device)
	deviceProfile.Protocols = append(deviceProfile.Protocols, deviceProtocol)
}

// generateDeviceKey generate a device key using namespace, name and separator "/"
func generateDeviceKey(device *v1alpha2.Device) string {
	return device.Namespace + "/" + device.Name
}

// deviceAdded creates a device, adds in deviceManagers map, send a message to edge node if node selector is present.
func (dc *DownstreamController) deviceAdded(device *v1alpha2.Device) {
	// use device namespace+name as primary key
	uniqueKey := generateDeviceKey(device)
	dc.deviceManager.Device.Store(uniqueKey, device)
	if len(device.Spec.NodeSelector.NodeSelectorTerms) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) != 0 {
		dc.addToConfigMap(device)

		// use membership/added as resource type
		resource, err := messagelayer.BuildResource(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], "membership/added", "")
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}

		msg := model.NewMessage("").BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation).FillBody(*device)

		err = dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", msg, err)
		}
	}
}

// isDeviceUpdated checks if device is actually updated
func isDeviceUpdated(oldTwin *v1alpha2.Device, newTwin *v1alpha2.Device) bool {
	// does not care fields
	oldTwin.ObjectMeta.ResourceVersion = newTwin.ObjectMeta.ResourceVersion
	oldTwin.ObjectMeta.Generation = newTwin.ObjectMeta.Generation
	// return true if ObjectMeta or Spec or Status changed, else false
	return !reflect.DeepEqual(oldTwin.ObjectMeta, newTwin.ObjectMeta) || !reflect.DeepEqual(oldTwin.Spec, newTwin.Spec) || !reflect.DeepEqual(oldTwin.Status, newTwin.Status)
}

// isNodeSelectorUpdated checks if nodeSelector is updated
func isNodeSelectorUpdated(oldTwin *v1.NodeSelector, newTwin *v1.NodeSelector) bool {
	return !reflect.DeepEqual(oldTwin.NodeSelectorTerms, newTwin.NodeSelectorTerms)
}

// isProtocolConfigUpdated checks if protocol is updated
func isProtocolConfigUpdated(oldTwin *v1alpha2.ProtocolConfig, newTwin *v1alpha2.ProtocolConfig) bool {
	return !reflect.DeepEqual(oldTwin, newTwin)
}

// isDeviceStatusUpdated checks if DeviceStatus is updated
func isDeviceStatusUpdated(oldTwin *v1alpha2.DeviceStatus, newTwin *v1alpha2.DeviceStatus) bool {
	return !reflect.DeepEqual(oldTwin, newTwin)
}

// isDeviceDataUpdated checks if DeviceData is updated
func isDeviceDataUpdated(oldData *v1alpha2.DeviceData, newData *v1alpha2.DeviceData) bool {
	return !reflect.DeepEqual(oldData, newData)
}

// isDevicePropertyVisitorsUpdated checks if DeviceProperyVisitors is updated
func isDevicePropertyVisitorsUpdated(oldPropertyVisitors *[]v1alpha2.DevicePropertyVisitor, newPropertyVisitors *[]v1alpha2.DevicePropertyVisitor) bool {
	return !reflect.DeepEqual(oldPropertyVisitors, newPropertyVisitors)
}

// updateConfigMap updates the protocol, twins and data in the deviceProfile in configmap
func (dc *DownstreamController) updateConfigMap(device *v1alpha2.Device) {
	if len(device.Spec.NodeSelector.NodeSelectorTerms) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) != 0 {
		configMap, ok := dc.configMapManager.ConfigMap.Load(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
		if !ok {
			klog.Error("Failed to load configmap")
			return
		}

		nodeConfigMap, ok := configMap.(*v1.ConfigMap)
		if !ok {
			klog.Error("Failed to assert to configmap")
			return
		}
		dp, ok := nodeConfigMap.Data[DeviceProfileJSON]
		if !ok || dp == "{}" {
			// This case should never be hit as we delete empty configmaps
			klog.Error("Failed to get deviceProfile from configmap data or deviceProfile is empty")
			return
		}

		deviceProfile := &types.DeviceProfile{}
		if err := json.Unmarshal([]byte(dp), deviceProfile); err != nil {
			klog.Errorf("Failed to unmarshal due to error: %v", err)
			return
		}
		var oldProtocol string
		for _, devInst := range deviceProfile.DeviceInstances {
			if device.Name == devInst.Name {
				oldProtocol = devInst.Spec.DeviceModelRef.Name
				break
			}
		}

		// delete the old protocol
		for i, ptcl := range deviceProfile.Protocols {
			if ptcl.Name == oldProtocol {
				deviceProfile.Protocols = append(deviceProfile.Protocols[:i], deviceProfile.Protocols[i+1:]...)
				break
			}
		}

		// add new protocol
		deviceProtocol := &types.Protocol{}
		if device.Spec.Protocol.OpcUA != nil {
			deviceProtocol = buildDeviceProtocol(OPCUA, device.Name, device.Spec.Protocol.OpcUA)
		} else if device.Spec.Protocol.Modbus != nil {
			deviceProtocol = buildDeviceProtocol(Modbus, device.Name, device.Spec.Protocol.Modbus)
		} else if device.Spec.Protocol.Bluetooth != nil {
			deviceProtocol = buildDeviceProtocol(Bluetooth, device.Name, device.Spec.Protocol.Bluetooth)
		} else if device.Spec.Protocol.CustomizedProtocol != nil {
			deviceProtocol = buildDeviceProtocol(CustomizedProtocol, device.Name, device.Spec.Protocol.CustomizedProtocol)
		} else {
			klog.Warning("Unsupported device protocol")
		}
		// add protocol common
		deviceProtocol.ProtocolCommonConfig = device.Spec.Protocol.Common
		deviceProfile.Protocols = append(deviceProfile.Protocols, deviceProtocol)

		bytes, err := json.Marshal(deviceProfile)
		if err != nil {
			klog.Errorf("Failed to marshal deviceprofile: %v, error: %v", deviceProfile, err)
			return
		}
		nodeConfigMap.Data[DeviceProfileJSON] = string(bytes)
		dc.configMapManager.ConfigMap.Store(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], nodeConfigMap)
		if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Update(context.Background(), nodeConfigMap, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Failed to update config map %v in namespace %v, error: %v", nodeConfigMap, device.Namespace, err)
			return
		}
	}
}

func buildDeviceProtocol(protocol, deviceName string, ProtocolConfig interface{}) *types.Protocol {
	var deviceProtocol types.Protocol
	deviceProtocol.Name = protocol + "-" + deviceName
	deviceProtocol.Protocol = protocol
	deviceProtocol.ProtocolConfig = ProtocolConfig
	return &deviceProtocol
}

// deviceUpdated updates the map, check if device is actually updated.
// If nodeSelector is updated, call add device for newNode, deleteDevice for old Node.
// If twin is updated, send twin update message to edge
func (dc *DownstreamController) deviceUpdated(device *v1alpha2.Device) {
	uniqueKey := generateDeviceKey(device)
	value, ok := dc.deviceManager.Device.Load(uniqueKey)
	dc.deviceManager.Device.Store(uniqueKey, device)
	if ok {
		cachedDevice := value.(*v1alpha2.Device)
		if isDeviceUpdated(cachedDevice, device) {
			// if node selector updated delete from old node and create in new node
			if isNodeSelectorUpdated(cachedDevice.Spec.NodeSelector, device.Spec.NodeSelector) {
				dc.deviceAdded(device)
				deletedDevice := &v1alpha2.Device{ObjectMeta: cachedDevice.ObjectMeta,
					Spec:     cachedDevice.Spec,
					Status:   cachedDevice.Status,
					TypeMeta: cachedDevice.TypeMeta,
				}
				dc.deviceDeleted(deletedDevice)
			} else {
				// update config map if spec, data or twins changed
				if isProtocolConfigUpdated(&cachedDevice.Spec.Protocol, &device.Spec.Protocol) ||
					isDeviceStatusUpdated(&cachedDevice.Status, &device.Status) ||
					isDeviceDataUpdated(&cachedDevice.Spec.Data, &device.Spec.Data) ||
					isDevicePropertyVisitorsUpdated(&cachedDevice.Spec.PropertyVisitors, &device.Spec.PropertyVisitors) {
					dc.updateConfigMap(device)
				}
				// update twin properties
				if isDeviceStatusUpdated(&cachedDevice.Status, &device.Status) {
					// TODO: add an else if condition to check if DeviceModelReference has changed, if yes whether deviceModelReference exists
					sendTwin := make([]v1alpha2.Twin, 0)
					// TODO: do not consider version factor
					addUpdatedTwins(device.Status.Twins, &sendTwin)
					addDeletedTwins(cachedDevice.Status.Twins, device.Status.Twins, &sendTwin)
					updatedDevice := v1alpha2.Device{}
					updatedDevice.Namespace = device.Namespace
					updatedDevice.Name = device.Name
					updatedDevice.Status.Twins = sendTwin
					resource, err := messagelayer.BuildResource(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], "device/"+uniqueKey+"/twin/cloud_updated", "")
					if err != nil {
						klog.Warningf("Built message resource failed with error: %s", err)
						return
					}
					msg := model.NewMessage("").BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation).FillBody(updatedDevice)

					err = dc.messageLayer.Send(*msg)
					if err != nil {
						klog.Errorf("Failed to send deviceTwin message %v due to error %v", msg, err)
					}
				}
			}
		}
	} else {
		// If device not present in device map means it is not modified and added.
		dc.deviceAdded(device)
	}
}

// addDeletedTwins add deleted twins in the message
func addDeletedTwins(oldTwin []v1alpha2.Twin, newTwin []v1alpha2.Twin, twin *[]v1alpha2.Twin) {
	for _, dtwin := range oldTwin {
		if !ifTwinPresent(dtwin, newTwin) {
			deletedTwin := dtwin
			deletedTwin.Desired.Metadata["type"] = "deleted"
			*twin = append(*twin, deletedTwin)
		}
	}
}

// ifTwinPresent checks if twin is present in the array of twins
func ifTwinPresent(twin v1alpha2.Twin, newTwins []v1alpha2.Twin) bool {
	for _, dtwin := range newTwins {
		if twin.PropertyName == dtwin.PropertyName {
			return true
		}
	}
	return false
}

// addUpdatedTwins is function of add updated twins to send to edge
func addUpdatedTwins(newTwin []v1alpha2.Twin, twin *[]v1alpha2.Twin) {
	for _, dtwin := range newTwin {
		updatedTwin := dtwin
		*twin = append(*twin, updatedTwin)
	}
}

// deleteFromConfigMap deletes a device from configMap
func (dc *DownstreamController) deleteFromConfigMap(device *v1alpha2.Device) {
	if len(device.Spec.NodeSelector.NodeSelectorTerms) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) != 0 {
		configMap, ok := dc.configMapManager.ConfigMap.Load(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
		if !ok {
			return
		}
		nodeConfigMap, ok := configMap.(*v1.ConfigMap)
		if !ok {
			klog.Error("Failed to assert to configmap")
			return
		}

		dc.deleteFromDeviceProfile(device, nodeConfigMap)

		// There are two cases we can delete configMap:
		// 1. no device bound to it, as Data[DeviceProfileJSON] is "{}"
		// 2. device instance created alone then removed, as Data[DeviceProfileJSON] is ""
		if nodeConfigMap.Data[DeviceProfileJSON] == "{}" || nodeConfigMap.Data[DeviceProfileJSON] == "" {
			if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Get(context.Background(), nodeConfigMap.Name, metav1.GetOptions{}); err == nil {
				if err = dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Delete(context.Background(), nodeConfigMap.Name, metav1.DeleteOptions{}); err != nil {
					klog.Errorf("failed to delete config map %s in namespace %s", nodeConfigMap.Name, device.Namespace)
					return
				}
			}
			// remove from cache
			dc.configMapManager.ConfigMap.Delete(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0])
			return
		}

		// store new config map
		dc.configMapManager.ConfigMap.Store(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], nodeConfigMap)
		if _, err := dc.kubeClient.CoreV1().ConfigMaps(device.Namespace).Update(context.Background(), nodeConfigMap, metav1.UpdateOptions{}); err != nil {
			klog.Errorf("Failed to update config map %v in namespace %v", nodeConfigMap, device.Namespace)
			return
		}
	}
}

// deleteFromDeviceProfile deletes a device from deviceProfile
func (dc *DownstreamController) deleteFromDeviceProfile(device *v1alpha2.Device, configMap *v1.ConfigMap) {
	dp, ok := configMap.Data[DeviceProfileJSON]
	if !ok {
		klog.Error("Device profile does not exist in the configmap")
		return
	}

	deviceProfile := &types.DeviceProfile{}
	err := json.Unmarshal([]byte(dp), deviceProfile)
	if err != nil {
		klog.Errorf("Failed to Unmarshal deviceprofile: %v", deviceProfile)
		return
	}
	deleteDeviceInstanceAndProtocol(device, deviceProfile)

	dm, ok := dc.deviceModelManager.DeviceModel.Load(device.Spec.DeviceModelRef.Name)
	if !ok {
		klog.Errorf("Failed to get device model %v", device.Spec.DeviceModelRef.Name)
		return
	}
	deviceModel := dm.(*v1alpha2.DeviceModel)
	// if model referenced by other devices, no need to delete the model
	checkModelReferenced := false
	for _, dvc := range deviceProfile.DeviceInstances {
		// todo: devicemodel should also support namespace, device refactoring introduced
		if dvc.Spec.DeviceModelRef.Name == deviceModel.Name {
			checkModelReferenced = true
			break
		}
	}
	if !checkModelReferenced {
		deleteDeviceModelAndVisitors(deviceModel, deviceProfile)
	}
	bytes, err := json.Marshal(deviceProfile)
	if err != nil {
		klog.Errorf("Failed to marshal deviceprofile: %v", deviceProfile)
		return
	}
	configMap.Data[DeviceProfileJSON] = string(bytes)
}

// deleteDeviceInstanceAndProtocol deletes deviceInstance and protocol from deviceProfile
func deleteDeviceInstanceAndProtocol(device *v1alpha2.Device, deviceProfile *types.DeviceProfile) {
	var protocol string
	for i, devInst := range deviceProfile.DeviceInstances {
		// using namespace+name confirm device
		if device.Name == devInst.Name && device.Namespace == devInst.Namespace {
			if devInst.Spec.Protocol.Modbus != nil {
				protocol = Modbus
			} else if devInst.Spec.Protocol.OpcUA != nil {
				protocol = OPCUA
			} else if devInst.Spec.Protocol.Bluetooth != nil {
				protocol = Bluetooth
			} else if devInst.Spec.Protocol.CustomizedProtocol != nil {
				protocol = CustomizedProtocol
			}
			deviceProfile.DeviceInstances[i] = deviceProfile.DeviceInstances[len(deviceProfile.DeviceInstances)-1]
			deviceProfile.DeviceInstances[len(deviceProfile.DeviceInstances)-1] = nil
			deviceProfile.DeviceInstances = deviceProfile.DeviceInstances[:len(deviceProfile.DeviceInstances)-1]
			break
		}
	}

	for i, ptcl := range deviceProfile.Protocols {
		if ptcl.Name == protocol {
			deviceProfile.Protocols[i] = deviceProfile.Protocols[len(deviceProfile.Protocols)-1]
			deviceProfile.Protocols[len(deviceProfile.Protocols)-1] = nil
			deviceProfile.Protocols = deviceProfile.Protocols[:len(deviceProfile.Protocols)-1]
			return
		}
	}
}

// deleteDeviceModelAndVisitors deletes deviceModel and visitor from deviceProfile
func deleteDeviceModelAndVisitors(deviceModel *v1alpha2.DeviceModel, deviceProfile *types.DeviceProfile) {
	for i, dm := range deviceProfile.DeviceModels {
		if dm.Name == deviceModel.Name {
			deviceProfile.DeviceModels[i] = deviceProfile.DeviceModels[len(deviceProfile.DeviceModels)-1]
			deviceProfile.DeviceModels[len(deviceProfile.DeviceModels)-1] = nil
			deviceProfile.DeviceModels = deviceProfile.DeviceModels[:len(deviceProfile.DeviceModels)-1]
			break
		}
	}
}

// deviceDeleted send a deleted message to the edgeNode and deletes the device from the deviceManager.Device map
func (dc *DownstreamController) deviceDeleted(device *v1alpha2.Device) {
	uniqueKey := generateDeviceKey(device)
	dc.deviceManager.Device.Delete(uniqueKey)
	dc.deleteFromConfigMap(device)

	if len(device.Spec.NodeSelector.NodeSelectorTerms) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) != 0 && len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) != 0 {
		// use membership/deleted as resource type
		resource, err := messagelayer.BuildResource(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], "membership/deleted", "")
		if err != nil {
			klog.Warningf("Built message resource failed with error: %s", err)
			return
		}
		// why use UpdateOperation but not InsertOperation
		msg := model.NewMessage("").BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation).FillBody(*device)

		err = dc.messageLayer.Send(*msg)
		if err != nil {
			klog.Errorf("Failed to send device addition message %v due to error %v", msg, err)
		}
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
	deviceManager, err := manager.NewDeviceManager(crdInformerFactory.Devices().V1alpha2().Devices().Informer())
	if err != nil {
		klog.Warningf("Create device manager failed with error: %s", err)
		return nil, err
	}

	deviceModelManager, err := manager.NewDeviceModelManager(crdInformerFactory.Devices().V1alpha2().DeviceModels().Informer())
	if err != nil {
		klog.Warningf("Create device manager failed with error: %s", err)
		return nil, err
	}

	dc := &DownstreamController{
		kubeClient:         client.GetKubeClient(),
		deviceManager:      deviceManager,
		deviceModelManager: deviceModelManager,
		messageLayer:       messagelayer.NewContextMessageLayer(),
		configMapManager:   manager.NewConfigMapManager(),
	}
	return dc, nil
}
