# Device Management User Guide

KubeEdge supports device management with the help of Kubernetes [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) and a Device Mapper (explained below) corresponding to the device being used.
We currently manage devices from the cloud and synchronize the device updates between edge nodes and cloud, with the help of device controller and device twin modules.

## Notice
Device Management features are updated from v1alpha1 to v1alpha2 in release v1.4.
It is **not** compatible for v1alpha1 and v1alpha2.
Details can be found [device-management-enhance](/docs/proposals/device-management-enhance.md)

## Device Model

A `device model` describes the device properties such as 'temperature' or 'pressure'. A device model is like a reusable template using which many devices can be created and managed.

Details on device model definition can be found [here](/docs/proposals/device-management-enhance.md#modifications-on-device-model-types).

### Device Model Sample
A sample device model like below,
```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: DeviceModel
metadata:
 name: sensor-tag-model
 namespace: default
spec:
 properties:
  - name: temperature
    description: temperature in degree celsius
    type:
     int:
      accessMode: ReadWrite
      maximum: 100
      unit: degree celsius
  - name: temperature-enable
    description: enable data collection of temperature sensor
    type:
      string:
        accessMode: ReadWrite
        defaultValue: 'OFF'
```


## Device Instance

A `device` instance represents an actual device object. It is like an instantiation of the `device model` and references properties defined in the model which exposed by property visitors to access. The device spec is static while the device status contains dynamically changing data like the desired state of a device property and the state reported by the device.

Details on device instance definition can be found [here](/docs/proposals/device-management-enhance.md#modifications-on-device-instance-types).

### Device Instance Sample
A sample device instance like below,
```yaml
apiVersion: devices.kubeedge.io/v1alpha2
kind: Device
metadata:
  name: sensor-tag-instance-01
  labels:
    description: TISimplelinkSensorTag
    manufacturer: TexasInstruments
    model: CC2650
spec:
  deviceModelRef:
    name: sensor-tag-model
  protocol:
    modbus:
      slaveID: 1
    common:
      com:
        serialPort: '1'
        baudRate: 115200
        dataBits: 8
        parity: even
        stopBits: 1
  nodeSelector:
    nodeSelectorTerms:
    - matchExpressions:
      - key: ''
        operator: In
        values:
        - node1
  propertyVisitors:
    - propertyName: temperature
      modbus:
        register: CoilRegister
        offset: 2
        limit: 1
        scale: 1
        isSwap: true
        isRegisterSwap: true
    - propertyName: temperature-enable
      modbus:
        register: DiscreteInputRegister
        offset: 3
        limit: 1
        scale: 1.0
        isSwap: true
        isRegisterSwap: true
status:
  twins:
    - propertyName: temperature
      reported:
        metadata:
          timestamp: '1550049403598'
          type: int
        value: '10'
      desired:
        metadata:
          timestamp: '1550049403598'
          type: int
        value: '15'
```
### Customized Protocols and Customized Settings
From KubeEdge v1.4, we can support customized protocols and customized settings, samples like below

- customized protocols

```yaml
  propertyVisitors:
    - propertyName: temperature
      collectCycle: 500000000
      reportCycle: 1000000000
      customizedProtocol:
        protocolName: MY-TEST-PROTOCOL
        configData:
          def1: def1-val
          def2: def2-val
          def3:
            innerDef1: idef-val
```

- customized values

```yaml
  protocol:
    common:
      ...
      customizedValues:
        def1: def1-val
        def2: def2-val
```

### Data Topic
From KubeEdge v1.4, we add data section defined in device spec.
Data section describe a list of time-series properties which will be reported by mappers to edge MQTT broker and should be processed in edge.

```yaml
apiVersion: devices.kubeedge.io/v1alpha1
kind: Device
metadata:
    ...
spec:
  deviceModelRef:
    ...
  protocol:
    ...
  nodeSelector:
    ...
  propertyVisitors:
    ...
  data:
    dataTopic: "$ke/events/device/+/data/update"
    dataProperties:
      - propertyName: pressure
        metadata:
          type: int
      - propertyName: temperature
        metadata:
          type: int
```

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

 Mapper design details can be found [here](/docs/proposals/mapper-design.md#mapper-design)

 An example of a mapper application created to support bluetooth protocol can be found [here](https://github.com/kubeedge/kubeedge/tree/master/mappers/bluetooth_mapper#bluetooth-mapper)


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
    The name of the config map will be as follows: device-profile-config-< edge node name >. The updates of the config map is handled internally by the device controller.

3. Run the mapper application corresponding to your protocol.

4. Edit the status section of the device instance yaml created in step 2 and apply the yaml to change the state of device twin. This change will be reflected at the edge, through the device controller
 and device twin modules. Based on the updated value of device twin at the edge the mapper will be able to perform its operation on the device.

5. The reported values of the device twin are updated by the mapper application at the edge and this data is synced back to the cloud by the device controller. User can view the update at the cloud by checking his device instance object.

```shell
    Note: Sample device model and device instance for a few protocols can be found at $GOPATH/src/github.com/kubeedge/kubeedge/build/crd-samples/devices
```
