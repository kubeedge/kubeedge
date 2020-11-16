| title              | authors | approvers | creation-date | last-updated | status          |
| ------------------ | ------- | --------- | ------------- | ------------ | --------------- |
| Device API at Edge | @kkBill |           | 2020-11-10    | 2020-11-10   | under designing |

# Device API at Edge(alpha)

## Motivation

At present, we cannot use Device API at edge directly, which is inconvenient for developing IoT applications. This proposal addresses this problem thus facilitates the development of IoT applications and simplifies the code of edge components(such as Mapper and DeviceTwin).

### Goals

* Use Device API at edge directly

### Non-goals

* ??

## Proposal

### Background

A brief introduction of how we use Device(or get device info) at edge **currently**.

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

#### For Pod and other built-in resource object

Let's take [syncPod](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/edgecontroller/controller/downstream.go#L47) as an example, as follows:

```go
func (dc *DownstreamController) syncPod() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Warning("Stop edgecontroller downstream syncPod loop")
			return
		case e := <-dc.podManager.Events():
			pod, ok := e.Object.(*v1.Pod)
			...
			msg := model.NewMessage("")
			msg.SetResourceVersion(pod.ResourceVersion)
			resource, err := messagelayer.BuildResource(pod.Spec.NodeName, pod.Namespace, model.ResourceTypePod, pod.Name)
			...
			msg.Content = pod // send pod to edge
			switch e.Type {
			case watch.Added:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.InsertOperation)
				dc.lc.AddOrUpdatePod(*pod)
			case watch.Deleted:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.DeleteOperation)
			case watch.Modified:
				msg.BuildRouter(modules.EdgeControllerModuleName, constants.GroupResource, resource, model.UpdateOperation)
				dc.lc.AddOrUpdatePod(*pod)
			default:
				klog.Warningf("pod event type: %s unsupported", e.Type)
			}
			if err := dc.messageLayer.Send(*msg); err != nil {
				...
			}
		}
	}
}
```

We send `Pod` to edge. Then we parse the Pod completely at the edge so that we can use the Pod API at the edge.

#### For Device

Let's take [syncDevice](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/controller/downstream.go#L594) as an example, as follows:

```go
func (dc *DownstreamController) deviceUpdated(device *v1alpha2.Device) {
  			...
				// update twin properties
				if isDeviceStatusUpdated(&cachedDevice.Status, &device.Status) {
					// TODO: add an else if condition to check if DeviceModelReference has changed, if yes whether deviceModelReference exists
					twin := make(map[string]*types.MsgTwin)
					addUpdatedTwins(device.Status.Twins, twin, device.ResourceVersion)
					addDeletedTwins(cachedDevice.Status.Twins, device.Status.Twins, twin, device.ResourceVersion)
					msg := model.NewMessage("")

					resource, err := messagelayer.BuildResource(device.Spec.NodeSelector.NodeSelectorTerms[0].MatchExpressions[0].Values[0], "device/"+device.Name+"/twin/cloud_updated", "")
					if err != nil {
						klog.Warningf("Built message resource failed with error: %s", err)
						return
					}
					msg.BuildRouter(modules.DeviceControllerModuleName, constants.GroupTwin, resource, model.UpdateOperation)
					content := types.DeviceTwinUpdate{Twin: twin} // send DeviceTwinUpdate to edge
					content.EventID = uuid.NewV4().String()
					content.Timestamp = time.Now().UnixNano() / 1e6
					msg.Content = content

					err = dc.messageLayer.Send(*msg)
					if err != nil {
						klog.Errorf("Failed to send deviceTwin message %v due to error %v", msg, err)
					}
				}
  	...
}
```

We send `DeviceTwinUpdate` to edge when device twin updated. Since we **DON'T** distribute the entire Device to the edge, we cannot use the Device API as easily on the edge as we do on the cloud. In addition, in order to parse out the message sent by the cloud, additional data structure(e.g.  [dttype](https://github.com/kubeedge/kubeedge/tree/master/edge/pkg/devicetwin/dttype) and [mapper type](https://github.com/kubeedge/kubeedge/blob/master/mappers/common/configmaptype.go)) is required, which increases the complexity of the code.

On the other hand, a ConfigMap is created to store the Device information when a new Device is created, which can also be optimized and simplified once we can use Device API on the edge.

### Effects on other components

* Device CRD need to adjust

Currently, we have defined Device in [device_instance_types.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/apis/devices/v1alpha2/device_instance_types.go#L360) and [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) respectively, some fields have the same meaning but different names, which is ambiguous. 

When the device status is updated, the "update message" is sent to the edge through the structure defined in  [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) . The example mentioned in the background section shows that when device status is updated, we send `DeviceTwinUpdate` to the edge rather than a completed `Device` object, which is different from built-in resource objects (such as Pod). Why not move `DeviceTwinUpdate`, `MembershipUpdate` and other structures in  [device.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/devicecontroller/types/device.go#L4) to  [device_instance_types.go](https://github.com/kubeedge/kubeedge/blob/master/cloud/pkg/apis/devices/v1alpha2/device_instance_types.go#L360) and re-define `DeviceSpec` and `DeviceStatus` , then we send the whole `Device` object to edge like `Pod` do. Once we get a whole `Device` at edge, then we can store device info at edge and use Device API at edge.

* DeviceController

The device controller is the cloud component of KubeEdge which is responsible for device management, and **it is responsible for synchronizing device updates between edge and cloud.** It starts two separate goroutines called `upstream controller` and `downstream controller`(these are not separate controllers as such but named here for clarity). 

If we do modify the `Device` object, then we need to modify device controller(both `downstream` and `upstream` ). The changes are as follows:

* discard configmap when new device created since we store the device properties and visitors etc in device_instance.
* send a whole `Device update` to edge instead of `membership update` or `twin update` on downstream.
* send a whole `Device update` to cloud instead of `twin update` on upstream.

At present, device controller follows the design [here](https://github.com/kubeedge/kubeedge/blob/master/docs/components/cloud/device_controller.md#device-controller). And I think it's implementation is kind of complicated.

* DeviceTwin & Mapper

Once we can retrive a whloe `Device` from cloud,  we can simplify the implementation at edge. For now, we have to defive some types(e.g.  [dttype](https://github.com/kubeedge/kubeedge/tree/master/edge/pkg/devicetwin/dttype) and [mapper type](https://github.com/kubeedge/kubeedge/blob/master/mappers/common/configmaptype.go)) at edge in order to unmarshal/marshal device info. If we can use Device API at edge, it can be optimized.

### Use Cases

* Simplify the implementation for mapper components 

### Design Detail

