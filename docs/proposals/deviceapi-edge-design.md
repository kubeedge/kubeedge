| title              | authors | approvers | creation-date | last-updated | status          |
| ------------------ | ------- | --------- | ------------- | ------------ | --------------- |
| Device API at Edge | @kkBill |           | 2020-11-10    | 2020-11-10   | under designing |

# Device API at Edge(alpha)

## Motivation

At present, we cannot use Device API at edge directly, which is inconvenient for developing IoT applications. This proposal addresses this problem thus facilitates the development of IoT applications and simplifies the code of edge components(such as Mapper and DeviceTwin).

### Goals

* Use Device API at edge directly

### Non-goals

* For alpha version, it is not related to List-Watch for edge   // ???

## Proposal

### Background

A brief introduction of how we use Device(or get device info) at edge currently.

### Use Cases

* Simplify the  implementation for mapper components 



---



## Proposal Discussion(draft only~)

![image-20201111165200245](../../../Library/Application Support/typora-user-images/image-20201111165200245.png)

```go
// Message struct
type Message struct {
	Header  MessageHeader `json:"header"`
	Router  MessageRoute  `json:"route,omitempty"`
  // Content 是对k8s API对象的封装
  // 对于内置资源类型而言（比如Pod、Service），把整个资源对象都封装在Content中
  // 对于Device而言，则只是把部分信息封装在Content中，从而在边缘端无法完整的解析出Device对象
	Content interface{}   `json:"content"`
}
```



在`cloud/pkg/devicecontroller/types/device.go`和`cloud/pkg/apis/devices/v1alpha2/device_instance_types.go`中重复定义Device，并且两者的定义**在部分字段的命名上有歧义、或是重复定义**。云端在下发指令给边缘端的设备时，是通过这个文件（`cloud/pkg/devicecontroller/types/device.go`）里定义的各种结构体来组织信息的，再封装进`model.Message`里下发至边缘端。为什么不把这部分信息（device.go）整合到Device CRD对象里（device_instance_types.go）。这里的做法和Pod、Service等内置的资源对象不一样，当初这么设计的初衷是什么？



For example：

![edgecore](images/device_API.png)



### Before

```go
// DeviceSpec represents a single device instance. It is an instantation of a device model.
type DeviceSpec struct {
	// Required: DeviceModelRef is reference to the device model used as a template
	// to create the device instance.
	DeviceModelRef *v1.LocalObjectReference `json:"deviceModelRef,omitempty"`
	// Required: The protocol configuration used to connect to the device.
	Protocol ProtocolConfig `json:"protocol,omitempty"`
	// List of property visitors which describe how to access the device properties.
	// PropertyVisitors must unique by propertyVisitor.propertyName.
	// +optional
	PropertyVisitors []DevicePropertyVisitor `json:"propertyVisitors,omitempty"`
	// Data section describe a list of time-series properties which should be processed
	// on edge node.
	// +optional
	Data DeviceData `json:"data,omitempty"`
	// NodeSelector indicates the binding preferences between devices and nodes.
	// Refer to k8s.io/kubernetes/pkg/apis/core NodeSelector for more details
	// +optional
	NodeSelector *v1.NodeSelector `json:"nodeSelector,omitempty"`
}

// DeviceStatus reports the device state and the desired/reported values of twin attributes.
type DeviceStatus struct {
	// A list of device twins containing desired/reported desired/reported values of twin properties..
	// Optional: A passive device won't have twin properties and this list could be empty.
	// +optional
	Twins []Twin `json:"twins,omitempty"`	
}
```



### After

```go
// DeviceSpec represents a single device instance. It is an instantation of a device model.
type DeviceSpec struct {
	// Required: DeviceModelRef is reference to the device model used as a template
	// to create the device instance.
	DeviceModelRef *v1.LocalObjectReference `json:"deviceModelRef,omitempty"`
	// Required: The protocol configuration used to connect to the device.
	Protocol ProtocolConfig `json:"protocol,omitempty"`
	// List of property visitors which describe how to access the device properties.
	// PropertyVisitors must unique by propertyVisitor.propertyName.
	// +optional
	PropertyVisitors []DevicePropertyVisitor `json:"propertyVisitors,omitempty"`
	// Data section describe a list of time-series properties which should be processed
	// on edge node.
	// +optional
	Data DeviceData `json:"data,omitempty"`
	// NodeSelector indicates the binding preferences between devices and nodes.
	// Refer to k8s.io/kubernetes/pkg/apis/core NodeSelector for more details
	// +optional
	NodeSelector *v1.NodeSelector `json:"nodeSelector,omitempty"`

	// added by @kkBill below, may need to add something more
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
}

// DeviceStatus reports the device state and the desired/reported values of twin attributes.
type DeviceStatus struct {
	// A list of device twins containing desired/reported desired/reported values of twin properties..
	// Optional: A passive device won't have twin properties and this list could be empty.
	// +optional
	Twins []Twin `json:"twins,omitempty"`

	// added by @kkBill below, may need to add something more
	Attributes  map[string]Attr     `json:"attributes,omitempty"`
	State       string              `json:"state,omitempty"`
	LastOnline  string              `json:"last_online,omitempty"`
}
```



