---
title: Mapper Support Devicedata Writing
authors:
- "@wbc6080"
  approvers:
  creation-date: 2024-3-5
  last-updated: 2024-4-19
  status: implementable
---

# Mapper Support Devicedata Writing

## Motivation

Now the Mapper Framework has realized the ability to collect and report edge device data to user applications and user databases. but it cannot well support 
the scenario of device data writing. In actual use, users may need to write data to certain registers of the managed device and finally complete the control of the device behavior.


## Goals

- Mapper can complete data writing to the managed edge device.
- Users can initiate device write instructions via http method.

## Design Details

### Architecture

<img src="../images/mapper/device-write.png">

### Device data writing API

The parameters of the object model can be divided into three categories according to the type of functions described: properties, methods and events.
In the v1beta1 version of device CRD proposed in version 1.15, the device properties field has been defined to describe the specific information and status 
of the device when it is running. In order to realize the device data writing function, the method field can be added to device CRD.

In the object model, methods refer to the capabilities or methods of a device that can be called externally. The value of a device property can be set 
through a defined method, such as the device property that controls a "light switch". The device method crd field is defined as follows:

```go

// DeviceMethod describes the control method of the device.
type DeviceMethod struct {
    // Required: The device method name to be accessed. It must be unique.
    Name string `json:"name,omitempty"`
    // Description of this method.
    Description string `json:"description,omitempty"`
    // The list of the deviceProperty controlled by deviceMethod.
    PropertyList []string `json:"desired,omitempty"`
    // Enter parameter name.
    ParameterName string `json:"parameterName,omitempty"`
}

```


### mapper device writing interface

In traditional way, we can send device data writing instructions on the cloud side through the yaml file of the device instance. 
When device property field in the device CRD is modified, the DeviceController on the cloud side will send the updated 
device configuration file to the DeviceTwin on the edge side through the cloud edge channel. Then DeviceTwin sends the 
device update information to mapper, executes the updatedev function, and completes the update and writing of device property.
Although this method conforms to the native definition of k8s, in edge scenarios, the cloud-edge network is unstable, 
which may cause the device write control command to be unable to send to the edge. At the same time, if the device write
request volume is large, may cause the cloud edge channel to be blocked. 

Therefore, this solution refers to the way the Mapper Framework data plane processes device data and provides a 
device writing interface on the data plane. With this interface, users can directly send device write commands 
to the mapper data plane to prevent cloud edge channel blocking.

According to the architecture diagram, in the yaml file of the device instance, the user only needs to define the method framework 
owned by the device, such as the name of the method and the parameterName that need to be input. There is no need to implement 
specific code here. The specific functions of this part are implemented in mapper data plane. At the same time, mapper needs 
to expose the corresponding **api port** according to the user-defined method, and can send device data writing commands through 
the corresponding api port. The specific process is as follows:

1.Define the device method parameters in the CRD file of the device instance, then submit the CRD file to the kubeedge cluster.

2.After receiving the device information, mapper can identify the method parameter in the device information and exposes the port, 
starting monitoring the device write command of the corresponding port.

3.When users need to write to the device, they can access the ports exposed by the edge mapper through edgemesh 
and other components in the cloud, or directly access the corresponding ports on the edge nodes and send control commands.

4.Users can implement the function of writing data in the device driver by themselves

## Plan

In version 1.18
- Implement the feature development of mapper that supports device writing according to the above plan.
- After completing feature development, add some mapper examples that support device writing to the mappers-go repository.

