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

In the architecture diagram, we show a sample configuration file of a device instance after adding the deviceMethod field. Here we define a `setValue` device method, 
which can be used to control and assign values to the device properties like temperature, name2.

### Mapper Device Writing Interface

According to the architecture diagram, in the yaml file of the device instance, the user only needs to define the method framework 
owned by the device, such as the name of the method and the parameterNames that controlled by device method. Subsequently, the user needs to implement the corresponding 
device method in the mapper's device driver part according to the definition in the CRD. Most of these methods write values to certain registers of the device, thereby completing write control of the device.

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

At the same time, users can obtain all device methods of this device based on the mapper API port(as defined by the user in the device instance CRD). 
The information returned by the API also includes the calling command of the device methods.

1. Get all device methods of the device   
   Url: https://{mapper-pod-ip}:{mapper-api-port}/api/v1/devicemethod/{deviceNamespace}/{deviceName}
   Response:
   ```json
   {
    "Data": {
        "Methods": [
            {
                "Name": "setValue",
                "Path": "/api/v1/devicemethod/default/random-instance-01/setValue/{propertyName}/{data}",
                "Parameters": [
                    {
                        "PropertyName": "random-int",
                        "ValueType": "int"
                    }
                ]
            }
        ]
    }
    }
   ```

After getting the calling command of the device method, user can create device write request like:

2. Create device write request
   Url: https://{mapper-pod-ip}:{mapper-api-port}/api/v1/devicemethod/{deviceNamespace}/{deviceName}/{deviceMethodName}/{propertyName}/{data}
   Response:
   ```json
   {
    "statusCode": 200,
    "Message": "Write data ** to device ** successfully."
    }
   ```

To sum up, the user-defined device method to implement device writing is mainly divided into the following steps:

1.Define the device method parameters in the CRD file of the device instance, then submit the CRD file to the KubeEdge cluster.

2.Users need to implement the function of writing data in the device driver by themselves according to device CRD.

3.Users can access the mapper api port to obtain all device methods of the device and obtain the device method calling command.

4.User calls the device method command that obtained in the third step, and directly create a device write request to the mapper at the edge node, 
or forwards the device write request in the cloud through components such as edgemesh.


## Plan

In version 1.19
- Implement the feature development of mapper that supports device writing according to the above plan.
- After completing feature development, add some mapper examples that support device writing to the mappers-go repository.

In version 1.20
- Enhance security of device write requests.

