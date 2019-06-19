# Device Management User Guide

KubeEdge supports device management with the help of Kubernetes [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) and a Device Mapper (explained below) corresponding to the device being used.
We currently manage devices from the cloud and synchronize the device updates between edge nodes and cloud, with the help of device controller and device twin modules.
 
 
 ## Device Model

A `device model` describes the device properties exposed by the device and property visitors to access these properties. A device model is like a reusable template using which many devices can be created and managed.

Details on device model definition can be found [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/device-crd.md#device-model-type-definition).

A sample device model can be found [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/device-crd.md#device-model-sample)


## Device Instance 
 
A `device` instance represents an actual device object. It is like an instantiation of the `device model` and references properties defined in the model. The device spec is static while the device status contains dynamically changing data like the desired state of a device property and the state reported by the device.
 
Details on device instance definition can be found [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/device-crd.md#device-instance-type-definition).
 
A sample device model can be found [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/device-crd.md#device-instance-sample).


## Device Mapper
 
 Mapper is an application that is used to connect and and control devices. Following are the responsibilities of mapper:
 1) Scan and connect to the device.
 2) Report the actual state of twin-attributes of device.
 3) Map the expected state of device-twin to actual state of device-twin.
 4) Collect telemetry data from device.
 5) Convert readings from device to format accepted by KubeEdge.
 6) Schedule actions on the device.
 7) Check health of the device.
 
 Mapper can be specific to a protocol where standards are defined i.e Bluetooth, Zigbee, etc or specific to a device if it a custom protocol.
 
 Mapper design details can be found [here](https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/mapper-design.md#mapper-design)
 
 An example of a mapper application created to support bluetooth protocol can be found [here](https://github.com/kubeedge/kubeedge/tree/master/device/bluetooth_mapper#bluetooth-mapper) 
 
    
## Usage of Device CRD

The following are the steps to 

1. Create a device model in the cloud node.

    ```shell
            kubectl apply -f <path to device model yaml>
    ```

2. Create a device instance in the cloud node.

    ```shell
           kubectl apply -f <path to device instance yaml>
    ```

    Note: Creation of device instance will also lead to the creation of a config map which will contain information about the devices which are required by the mapper applications 
    The name of the config map will be as follows: device-profile-config-< edge node name >. The updation of the config map is handled internally by the device controller.

3. Run the mapper application corresponding to your protocol.

4. Edit the status section of the device instance yaml created in step 2 and apply the yaml to change the state of device twin. This change will be reflected at the edge, through the device controller
 and device twin modules. Based on the updated value of device twin at the edge the mapper will be able to perform its operation on the device.
 
5. The reported values of the device twin are updated by the mapper application at the edge and this data is synced back to the cloud by the device controller. User can view the update at the cloud by checking his device instance object.

```shel
    Note: Sample device model and device instance for a few protocols can be found at $GOPATH/src/github.com/kubeedge/kubeedge/build/crd-samples/devices 
```