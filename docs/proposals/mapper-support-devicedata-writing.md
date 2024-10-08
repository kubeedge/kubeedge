---
title: Mapper Support Device Data Writing
authors:
- "@wbc6080"
  approvers:
  creation-date: 2024-6-11
  last-updated: 2024-6-11
  status: implementable
---

# Mapper Support Device Data Writing

## Motivation

Now the Mapper Framework has realized the ability to collect and report edge device data to user applications and user databases. but it cannot well support 
the scenario of device data writing. In actual use, users may need to write data to certain registers of the managed device and finally complete the control of the device behavior.

## Goals

- Mapper can complete data writing to the managed edge device.
- Users can initiate device write instructions via http method.

## Design Details

### Architecture

In traditional way, we can send device data writing instructions on the cloud side through the yaml file of the device instance.
When device property field in the device CRD is modified, the `DeviceController` on the cloud side will send the updated
device configuration file to the `DeviceTwin` on the edge side through the cloud edge channel. Then `DeviceTwin` sends the
device update information to Mapper, executes the `updatedev` function, and completes the update and writing of device property.
Although this method conforms to the native definition of Kubernetes, in edge scenarios, the cloud-edge network is unstable,
which may cause the device write control command to be unable to send to the edge. At the same time, if the device write
request volume is large, may cause the cloud edge channel to be blocked.

Therefore, this solution shown bellow refers to the way the Mapper Framework data plane processes device data and provides a
device writing interface on the data plane. With this interface, users can directly send device write commands
to the mapper data plane to prevent cloud edge channel blocking.

<img src="../images/mapper/device-write.png">

### Device Data Writing API

The parameters of the object model can be divided into three categories according to the type of functions described: `properties`, `methods` and `events`.
In the v1beta1 version of device CRD proposed in version 1.15, the device `properties `field has been defined to describe the specific information and status 
of the device when it is running. In order to realize the device data writing function, the `methods` field can be added to device CRD.

In the object model, methods refer to the capabilities or methods of a device that can be called externally. The value of a device property can be set 
through a defined method, such as the device property that controls a "light switch". The device method CRD field is defined as follows:

```go

// DeviceMethod describes the specifics all the methods of the device.
type DeviceMethod struct {
    // Required: The device method name to be accessed. It must be unique.
    Name string `json:"name,omitempty"`
    // Define the description of device method.
    // +optional
    Description string `json:"description,omitempty"`
    // PropertyNames are list of device properties that device methods can control.
    // Required: A device method can control multiple device properties.
    PropertyNames []string `json:"propertyNames,omitempty"`
}

```

### Mapper Device Writing Interface

According to the architecture diagram, in the yaml file of the device instance, the user only needs to define the method framework 
owned by the device, such as the name of the method and the parameterName that need to be input. There is no need to implement 
specific code here. The specific functions of this part are implemented in mapper data plane. At the same time, mapper needs 
to expose the corresponding **api port** according to the user-defined method, and can send device data writing commands through 
the corresponding api port. The specific process is as follows:

1.Define the device method parameters in the CRD file of the device instance, then submit the CRD file to the KubeEdge cluster.

2.After receiving the device information, mapper can identify the method parameter in the device information and exposes the port, 
starting monitoring the device write command of the corresponding port.

3.When users need to write to the device, users can access the ports exposed by the edge mapper through EdgeMesh 
or other components in the cloud, or directly access the corresponding ports on the edge nodes and send control commands.

4.Users can implement the function of writing data in the device driver by themselves. The device driver template in mapper-framework is as follows:

```go

func (c *CustomizedClient) GetDeviceData(visitor *VisitorConfig) (interface{}, error) {
    // TODO: add the code to get device's data
    // you can use c.ProtocolConfig and visitor
    return nil, nil
}

func (c *CustomizedClient) DeviceDataWrite(visitor *VisitorConfig, deviceMethodName string, propertyName string, data interface{}) error {
    // TODO: add the code to write device's data
    // you can use c.ProtocolConfig and visitor
    return nil
}

```

## Plan

In version 1.19
- Implement the feature development of mapper that supports device writing according to the above plan.
- After completing feature development, add some mapper examples that support device writing to the mappers-go repository.

