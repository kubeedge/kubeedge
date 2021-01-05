---
title: Device API at edge
authors:
    - "@kkBill"
approvers:
    - "@kevin-wangzefeng"
    - "@fisherxu"
creation-date: 2020-11-10
last-updated: 2020-12-8
status: need review
---



# Device API at edge(alpha)

## Motivation

At present, we cannot use Device API at edge directly, which is inconvenient for developing IoT applications. This proposal addresses this problem thus facilitates the development of IoT applications and simplifies the code of edge components(such as Mapper and DeviceTwin).

### Goals

* Get Device API at edge directly

### Non-goals

* For the alpha version, we will not consider incorporating the List-Watch mechanism

## Proposal

### Background

A brief introduction of how we synchronize the Device to edge currently.

Device management in KubeEdge is implemented by making use of Kubernetes Custom Resource Definitions (CRDs) to describe device metadata/status and device controller to synchronize these device updates between edge and cloud. 

Currently, we define device info in [device_instance_types.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/apis/devices/v1alpha2/device_instance_types.go#L360)(which is a standard Kubernetes-Style API) and [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4)(which is used for message transfer between cloud and edge) respectively. When a user creates a new device instance or updates device's status, devicecontroller send the device(defined in  [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4))  instead of a complete Device API to edge. More specifically, we call [createDevice()](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/controller/downstream.go#L417) to create a device from Device API. In short, we cannot get the complete Device API at edge. And **this proposal(aplha version) focuses on how to synchronize the Device API to edge**.

Here are two examples to help us understand the difference between custom resources and built-in resources in message transfer between cloud and edge：

* Built-in Resource

Let's take [syncPod](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/edgecontroller/controller/downstream.go#L47) as an example. Here we can see that cloud send a whole `Pod` to edge. Then we parse the Pod completely so that we can use the Pod API at edge.

* Device

Let's take [syncDevice](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/controller/downstream.go#L148) as an example. When a user creates a new device Instance,  `deviceAdded()` was called, then devicecontroller will send a `MembershipUpdate` to edge instead of a standard Device API, which is different from other built-in resource objects like Pod. Therefore, users cannot get a  complete Device API at edge.

### Design Detail

* Cloud 

The device controller is the cloud component of KubeEdge which is responsible for device management, and **it is responsible for synchronizing device updates between edge and cloud.** It starts two separate goroutines called `upstream controller` and `downstream controller`(these are not separate controllers as such but named here for clarity). 

We start a new goroutine to sync Device CRD in `downstream`. 

```go
// Start DownstreamController
func (dc *DownstreamController) Start() error {
	klog.Info("Start downstream devicecontroller")
	go dc.syncDeviceModel()
	time.Sleep(1 * time.Second)
	go dc.syncDevice()
  
  // added function here
	go dc.syncDeviceCRD() // sync Device API to edge

	return nil
}
```

`syncDeviceCRD()` is as follows, the logic behind it is the same as `syncDevice()`. It uses a new resourceType ("device_crd") to build message's router and `devicetwin` at edge will classify message according to the resourceType.

```go
// syncDeviceCRD is used to get device events from informer and sync a whole Device CRD to edge
func (dc *DownstreamController) syncDeviceCRD() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop device controller downstream syncDeviceCRD loop")
			return
		case e := <-dc.deviceManager.Events():
			device, ok := e.Object.(*v1alpha2.Device)
			if !ok {
				klog.Warningf("Object type: %T unsupported", device)
				continue
			}

			if len(device.Spec.NodeSelector.NodeSelectorTerms) != 0 &&
				len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions) != 0 &&
				len(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values) != 0 {
				msg := model.NewMessage("")
				msg.SetResourceVersion(device.ResourceVersion)
				msg.Content = device

				edgeNode := device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0]
				resourceType := "device_crd"
				resource, err := messagelayer.BuildResource(edgeNode, resourceType, "")
				if err != nil {
					klog.Warningf("Build message resource failed with error: %s", err)
					return
				}
				switch e.Type {
				case watch.Added:
					msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.InsertOperation)
				case watch.Deleted:
					msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.DeleteOperation)
				case watch.Modified:
					msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)
				default:
					klog.Warningf("Device event type: %s unsupported", e.Type)
				}
				err = dc.messageLayer.Send(*msg)
				if err != nil {
					klog.Errorf("Failed to send device %v due to error %v", msg, err)
				}
			}
		}
	}
}
```



* Edge

We define a new table("DeviceMeta") at egde to store Device API metadata, which is similar to [Meta](https://github.com/kubeedge/kubeedge/blob/master/edge/pkg/metamanager/dao/meta.go#L17) defined in `metamanager`.

```go
type DeviceMeta struct {
	Key   string `orm:"column(key); size(256); pk"`
	Value string `orm:"column(value); null; type(text)"`
}
```

The edge side will identify the message according to `Msg.Router.Resource` and specify the callback function to execute based on operation type of the message.



We split this into the following stages:

* alpha

Make device API available both at cloud and edge side.

* beta

Discard configmaps store device info, make apps call Device API directly.

* graduated

Call list/watch to sync Device API at cloud and edge side.


### Use Cases

* Provide Device API at edge.
* Simplify the implementation for mapper components for users.

### Open Question

* Currently, we define device info in [device_instance_types.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/apis/devices/v1alpha2/device_instance_types.go#L360) and [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) respectively, some fields have the same meaning but different names, which is ambiguous; and some fields in [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) don't exist in Device CRD. Do we need to reconsider the Device CRD design?
* Since we cannot get a complete Device API at edge at present, we divide the processing of the device into three parts, namely Device, Device's Twin and Device's Attr. Once we can get the complete Device API, do we need to reconsider this part?
* For mapper, it gets device info by configmap. Here's the logic: when one creates/updates a device at cloud, it will creates/updates relevant configmap, then `edgecontroller` watch  configmap's event and sync the configmap to edge, mapper gets it finally. If we can get a complete Device API at edge, shall we consider discarding ConfigMap？

