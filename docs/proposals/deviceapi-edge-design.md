---
title: Device API on the Edge
authors:
    - "@kkBill"
approvers:
    - "@kevin-wangzefeng"
    - "@fisherxu"
creation-date: 2020-11-10
last-updated: 2020-12-8
status: need review
---



# Device API on the Edge(alpha)

## Motivation

At present, we cannot use Device API on the edge directly, which is inconvenient for developing IoT applications. This proposal addresses this problem thus facilitates the development of IoT applications and simplifies the code of edge components(such as Mapper and DeviceTwin).

### Goals

* Use Device API on the edge directly

### Non-goals

* For the alpha version, we will not consider incorporating the List-Watch mechanism

## Proposal

### Background

A brief introduction of how we use Device(or get device info) on the edge **currently**.

Device management in KubeEdge is implemented by making use of Kubernetes Custom Resource Definitions (CRDs) to describe device metadata/status and device controller to synchronize these device updates between edge and cloud. Our focus is on message delivery between cloud and edge.

First of all, we all know that cloud and edge communicates via CloudHub and EdgeHub. Or, more specifically, they store the information to be transmitted in `Message.Content`, see below:

```go
// Message struct
type Message struct {
	Header  MessageHeader `json:"header"`
	Router  MessageRoute  `json:"route,omitempty"`
	Content interface{}   `json:"content"`
}
```

#### (1) Pod and other built-in resource object

Let's take [syncPod](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/edgecontroller/controller/downstream.go#L47) as an example. Here we can see that cloud send a whole `Pod` to edge. Then we parse the Pod completely so that we can use the Pod API on the edge.

#### (2) Device

Let's take [syncDevice](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/controller/downstream.go#L594) as an example. We send `DeviceTwinUpdate` to edge when device twin updated. Since we **DON'T distribute the entire Device to the edge**, we cannot use the Device API as easily on the edge as we do on the cloud. In addition, in order to parse out the message sent by the cloud, additional data structure(e.g.  [dttype](https://github.com/kubeedge/kubeedge/tree/master/edge/pkg/devicetwin/dttype) and [mapper type](https://github.com/kubeedge/kubeedge/blob/master/mappers/common/configmaptype.go)) is required, which increases the complexity of the code.

On the other hand, a ConfigMap is created to store the Device information when a new Device is created, which can also be optimized and simplified once we can use Device API on the edge.

### Design Detail

* Device CRD

Currently, we have defined Device in [device_instance_types.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/apis/devices/v1alpha2/device_instance_types.go#L360) and [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) respectively, some fields have the same meaning but different names, which is ambiguous. 

When the device status is updated, the "update message" is sent to the edge through the structure defined in  [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) . The example mentioned in the background section shows that when device status is updated, we send `DeviceTwinUpdate` to the edge rather than a completed `Device` object, which is different from built-in resource objects (such as Pod). It is suggested to move `DeviceTwinUpdate`, `MembershipUpdate` and other structures in  [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) to  [device_instance_types.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/apis/devices/v1alpha2/device_instance_types.go#L360) and re-define `DeviceSpec` and `DeviceStatus` , then **we send the whole `Device` object to edge like `Pod` do**. Once we get a whole `Device` on the edge, then we can store device info on the edge and use Device API on the edge. And **this is the key of this proposal**.

* DeviceController

The device controller is the cloud component of KubeEdge which is responsible for device management, and **it is responsible for synchronizing device updates between edge and cloud.** It starts two separate goroutines called `upstream controller` and `downstream controller`(these are not separate controllers as such but named here for clarity). 

Since we modify the `Device` object, then we need to modify device controller(both `downstream` and `upstream` ). The changes are as follows:

* discard configmap when new device created since we store the device properties and visitors etc in device_instance.
* send a whole `Device` update to edge instead of `membership` update or `twin` update on downstream.
* send a whole `Device` update to cloud instead of `twin` update on upstream.

At present, device controller follows the [design](https://github.com/kubeedge/kubeedge/blob/master/docs/components/cloud/device_controller.md#device-controller). 

* DeviceTwin & Mapper

Once we can retrive a whloe `Device` from cloud,  we can simplify the implementation on the edge. For now, we have to defive some types(e.g.  [dttype](https://github.com/kubeedge/kubeedge/tree/master/edge/pkg/devicetwin/dttype) and [mapper type](https://github.com/kubeedge/kubeedge/blob/master/mappers/common/configmaptype.go)) on the edge in order to unmarshal/marshal device info. Once we can use Device API on the edge, it can be optimized.

### Use Cases

* Simplify device API sync logic, improve maintainability
* Simplify the implementation for mapper components for user


