---
title: Device Management Enhancement
authors:
    - "@luogangyi"
approvers:
    - "@kevin-wangzefeng"
    - "@fisherxu"
creation-date: 2020-05-13
last-updated: 2020-08-13
status: implementable
---

# Device Management Enhancement
## Motivation

Device management is a key feature required for IoT use-cases in edge computing.
This proposal addresses how can we enhance device management including more flexible
protocol support, more reasonable device model and device instance design, and
additional way to handle data which is collected from device.

### Goals

* Add customized protocol support
* Add ability to allow user get and process data in edge node
* Improve device model and device instance crd design

### Non-goals

* To support streaming data from video device
* To provide specific protocols and mappers

## Proposal

We propose below modifications on current design.
* Move property visitors from device model to device instance.
* Add collectCycle and reportCycle under property visitor.
* Add data section in device spec.
* Add customized protocol config in device CRD.
* Add common part in protocol config section
* Extract common part of Modbus protocol config into an independent common part.
* Allow add any customized K-V in protocol config and property visitor section.
* Support using boolean, float, double and bytes to describe type of property in device model.



### Use Cases

* Reuse device model.
  * Considering device properties are physical attributes, but property visitors are manually configured attributes. Combining device properties and property visitors in device model decrease the reusability of device model.
  * Case 1: Same devices are connected to a central management server, eg. SCADA. In this case, devices have same properties but different property visitors.
  * Case 2: Same devices are using different industrial protocol. In this case, devices have same properties but different property visitors.
* Customized data collect cycle and report cycle
   * Users can define collect cycle and report cycle to each property. For example, a temperature property may need be collected per second, while a throughput property may need be collected per hour.
* Deal data of non-twin properties.
  * Currently, only twin properties will be sync between edge and cloud. Non-twin properties are not processed by edge-core. Time-Serial data are produced from devices and should have a way to allow user deal with these data.
* Deal various industrial protocols
  * Currently, only Modbus, OPC-UA and bluetooth are supported by KubeEdge. However there are thousands of industrial protocols. It is impossible to define all these protocols in KubeEdge. If users want to use these un-predefined protocols, we should provide a way to support.
* Customized provided protocol
  * If users want to add some special control value, such as bulk related collection, in provided protocol like Modbus, he can use the customized K-V features.

## Design Details

### Move property visitors from device model to device instance.

- move property visitors from device CRD to device instance CRD
- move property visitors from DeviceModelSpec to DeviceSpec struct.
- change device profile generating procedure

### Add collectCycle and reportCycle under property visitor.
- add collectCycle and reportCycle under property visitor in device instance CRD.
- add collectCycle and reportCycle in DevicePropertyVisitor struct.
- add customized K-V config in this part

### Add data section besides twin section.
- add data section in device instance CRD.
- add DeviceData in Device struct.
- inject data section and twin section into configmap
- add new MQTT topic to handle data from data section

### Add customized protocols support
- add 'other' under property visitor, type is object

### ADD common part in protocol config
- add common part in protocol config section
- extract common part of Modbus protocol config into this part.
- add CommType, ReconnTimeout, ReconnRetryTimes, CollectTimeout, CollectRetryTimes, CollectType in this part
- add customized K-V config in this part

### modifications on device instance types
```golang
type DeviceSpec struct {
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
	...
}

type ProtocolConfig struct {
	...
	// Protocol configuration for bluetooth
	// +optional
	Bluetooth *ProtocolConfigBluetooth `json:"bluetooth,omitempty"`
	// Configuration for protocol common part
	// +optional
	Common *ProtocolConfigCommon `json:"common,omitempty"`
	// Configuration for customized protocol
	// +optional
	CustomizedProtocol *ProtocolConfigCustomized `json:"customizedProtocol,omitempty"`
}

type ProtocolConfigModbus struct {
	// Required. 0-255
	SlaveID *int64 `json:"slaveID,omitempty"`
}

// Only one of COM or TCP may be specified.
type ProtocolConfigCommon struct {
	// +optional
	COM *ProtocolConfigCOM `json:"com,omitempty"`
	// +optional
	TCP *ProtocolConfigTCP `json:"tcp,omitempty"`
	// Communication type, like tcp client, tcp server or COM
	// +optional
	RTU *ProtocolConfigModbusRTU `json:"rtu,omitempty"`
	CommType string `json:"commType,omitempty"`
	// Reconnection timeout
	// +optional
	TCP *ProtocolConfigModbusTCP `json:"tcp,omitempty"`
	ReconnTimeout int64 `json:"reconnTimeout,omitempty"`
	// Reconnecting retry times
	// +optional
	ReconnRetryTimes int64 `json:"reconnRetryTimes,omitempty"`
	// Define timeout of mapper collect from device.
	// +optional
	CollectTimeout int64 `json:"collectTimeout,omitempty"`
	// Define retry times of mapper will collect from device.
	// +optional
	CollectRetryTimes int64 `json:"collectRetryTimes,omitempty"`
	// Define collect type, sync or async.
	// +optional
	CollectType string `json:"collectType,omitempty"`
	// Customized values for provided protocol
	// +optional
	CustomizedValues *CustomizedValue `json:"customizedValues,omitempty"`
}

type ProtocolConfigTCP struct {
	// Required.
	IP string `json:"ip,omitempty"`
	// Required.
	Port int64 `json:"port,omitempty"`
}

type ProtocolConfigCOM struct {
	...
	// Required.
	SerialPort string `json:"serialPort,omitempty"`
	// Required. BaudRate 115200|57600|38400|19200|9600|4800|2400|1800|1200|600|300|200|150|134|110|75|50
	Parity string `json:"parity,omitempty"`
	// Required. Bit that stops 1|2
	StopBits int64 `json:"stopBits,omitempty"`
}

type ProtocolConfigCustomized struct {
	// Unique protocol name
	// Required.
	ProtocolName string `json:"protocolName,omitempty"`
	// Any config data
	// +optional
	ConfigData *CustomizedValue `json:"configData,omitempty"`
}

// DeviceData reports the device's time-series data to edge MQTT broker.
// These data should not be processed by edgecore. Instead, they can be process by
// third-party data-processing apps like EMQX kuiper.
type DeviceData struct {
	// Required: A list of data properties, which are not required to be processed by edgecore
	DataProperties []DataProperty `json:"dataProperties,omitempty"`
	// Topic used by mapper, all data collected from dataProperties
	// should be published to this topic,
	// the default value is $ke/events/device/+/data/update
	// +optional
	DataTopic string `json:"dataTopic,omitempty"`
}

// DataProperty represents the device property for external use.
type DataProperty struct {
	// Required: The property name for which should be processed by external apps.
	// This property should be present in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Additional metadata like timestamp when the value was reported etc.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// DevicePropertyVisitor describes the specifics of accessing a particular device
// property. Visitors are intended to be consumed by device mappers which connect to devices
// and collect data / perform actions on the device.
type DevicePropertyVisitor struct {
	// Required: The device property name to be accessed. This should refer to one of the
	// device properties defined in the device model.
	PropertyName string `json:"propertyName,omitempty"`
	// Define how frequent mapper will report the value.
	// +optional
	ReportCycle int64 `json:"reportCycle,omitempty"`
	// Define how frequent mapper will collect from device.
	// +optional
	CollectCycle int64 `json:"collectCycle,omitempty"`
	// Customized values for visitor of provided protocols
	// +optional
	CustomizedValues *CustomizedValue `json:"customizedValues,omitempty"`
	// Required: Protocol relevant config details about the how to access the device property.
	VisitorConfig `json:",inline"`
}

type VisitorConfig struct {
	...
	// Bluetooth represents a set of additional visitor config fields of bluetooth protocol.
	// +optional
	Bluetooth *VisitorConfigBluetooth `json:"bluetooth,omitempty"`
	// CustomizedProtocol represents a set of visitor config fields of bluetooth protocol.
	// +optional
	CustomizedProtocol *VisitorConfigCustomized `json:"customizedProtocol,omitempty"`
	...
}

// Common visitor configurations for customized protocol
type VisitorConfigCustomized struct {
	// Required: name of customized protocol
	ProtocolName string `json:"protocolName,omitempty"`
	// Required: The ConfigData of customized protocol
	ConfigData *CustomizedValue `json:"configData,omitempty"`
}

type CustomizedValue map[string]interface{}
```
### modifications on device model types
```golang
// Represents the type and data validation of a property.
// Only one of its members may be specified.
type PropertyType struct {
	// +optional
	Int *PropertyTypeInt64 `json:"int,omitempty"`
	// +optional
	String *PropertyTypeString `json:"string,omitempty"`
	// +optional
	Double *PropertyTypeDouble `json:"double,omitempty"`
	// +optional
	Float *PropertyTypeFloat `json:"float,omitempty"`
	// +optional
	Boolean *PropertyTypeBoolean `json:"boolean,omitempty"`
	// +optional
	Bytes *PropertyTypeBytes `json:"bytes,omitempty"`
}

...

type PropertyTypeString struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	DefaultValue string `json:"defaultValue,omitempty"`
}

type PropertyTypeDouble struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	DefaultValue float64 `json:"defaultValue,omitempty"`
	// +optional
	Minimum float64 `json:"minimum,omitempty"`
	// +optional
	Maximum float64 `json:"maximum,omitempty"`
	// The unit of the property
	// +optional
	Unit string `json:"unit,omitempty"`
}

type PropertyTypeFloat struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	DefaultValue float32 `json:"defaultValue,omitempty"`
	// +optional
	Minimum float32 `json:"minimum,omitempty"`
	// +optional
	Maximum float32 `json:"maximum,omitempty"`
	// The unit of the property
	// +optional
	Unit string `json:"unit,omitempty"`
}

type PropertyTypeBoolean struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
	// +optional
	DefaultValue bool `json:"defaultValue,omitempty"`
}

type PropertyTypeBytes struct {
	// Required: Access mode of property, ReadWrite or ReadOnly.
	AccessMode PropertyAccessMode `json:"accessMode,omitempty"`
}
```

### modifications on device profile（used in configmap）
```golang
type DeviceInstance struct {
	...
	Protocol string `json:"protocol,omitempty"`
	// Model is deviceInstance model name
	Model string `json:"model,omitempty"`
	// A list of device twins containing desired/reported desired/reported values of twin properties..
	// Optional: A passive device won't have twin properties and this list could be empty.
	// +optional
	Twins []v1alpha2.Twin `json:"twins,omitempty"`
	// A list of data properties, which are not required to be processed by edgecore
	// +optional
	DataProperties []v1alpha2.DataProperty `json:"dataProperties,omitempty"`
	// Topic used by mapper, all data collected from dataProperties
	// should be published to this topic,
	// the default value is $ke/events/device/+/data/update
	// +optional
	DataTopic string `json:"dataTopic,omitempty"`
	// PropertyVisitors is list of all PropertyVisitors in DeviceModels
	PropertyVisitors []*PropertyVisitor `json:"propertyVisitors,omitempty"`
}

type Protocol struct {
	...
	// Protocol is protocol name defined in deviceInstance. It is generated by deviceController
	Protocol string `json:"protocol,omitempty"`
	// ProtocolConfig is protocol config
	ProtocolConfig interface{} `json:"protocolConfig"`
	// ProtocolCommonConfig is common part of protocol config
	ProtocolCommonConfig interface{} `json:"protocolCommonConfig"`
}

type PropertyVisitor struct {
	...
	ModelName string `json:"modelName,omitempty"`
	// Protocol is protocol of propertyVisitor
	Protocol string `json:"protocol,omitempty"`
	// Define how frequent mapper will report the value.
	ReportCycle int64 `json:"reportCycle,omitempty"`
	// Define how frequent mapper will collect from device.
	CollectCycle int64 `json:"collectCycle,omitempty"`
	// Customized values for visitor of provided protocols
	// +optional
	CustomizedValues interface{} `json:"customizedValues,omitempty"`
	// VisitorConfig is property visitor configuration
	VisitorConfig interface{} `json:"visitorConfig,omitempty"`
}
```

### New device model CRD sample
```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: devicemodels.devices.kubeedge.io
spec:
  group: devices.kubeedge.io
  names:
    kind: DeviceModel
    plural: devicemodels
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            properties:
              description: 'Required: List of device properties.'
              items:
                properties:
                  description:
                    description: The device property description.
                    type: string
                  name:
                    description: 'Required: The device property name.'
                    type: string
                  type:
                    description: 'Required: PropertyType represents the type and data
                      validation of the property.'
                    properties:
                      int:
                        properties:
                          accessMode:
                            description: 'Required: Access mode of property, ReadWrite
                              or ReadOnly.'
                            type: string
                            enum:
                              - ReadOnly
                              - ReadWrite
                          defaultValue:
                            format: int64
                            type: integer
                          maximum:
                            format: int64
                            type: integer
                          minimum:
                            format: int64
                            type: integer
                          unit:
                            description: The unit of the property
                            type: string
                        required:
                          - accessMode
                        type: object
                      string:
                        properties:
                          accessMode:
                            description: 'Required: Access mode of property, ReadWrite
                              or ReadOnly.'
                            type: string
                            enum:
                              - ReadOnly
                              - ReadWrite
                          defaultValue:
                            type: string
                        required:
                          - accessMode
                        type: object
                    type: object
                required:
                  - name
                  - type
                type: object
              type: array
          type: object
  version: v1alpha2
```

### New device instance CRD sample

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: devices.devices.kubeedge.io
spec:
  group: devices.kubeedge.io
  names:
    kind: Device
    plural: devices
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            deviceModelRef:
              description: 'Required: DeviceModelRef is reference to the device model
                used as a template to create the device instance.'
              type: object
            nodeSelector:
              description: NodeSelector indicates the binding preferences between
                devices and nodes. Refer to k8s.io/kubernetes/pkg/apis/core NodeSelector
                for more details
              type: object
            protocol:
              description: 'Required: The protocol configuration used to connect to
                the device.'
              properties:
                bluetooth:
                  description: Protocol configuration for bluetooth
                  properties:
                    macAddress:
                      description: Unique identifier assigned to the device.
                      type: string
                  type: object
                modbus:
                  description: Protocol configuration for modbus
                  properties:
                    slaveID:
                      description: Required. 0-255
                      format: int64
                      type: integer
                      minimum: 0
                      maximum: 255
                  required:
                    - slaveID
                  type: object
                opcua:
                  description: Protocol configuration for opc-ua
                  properties:
                    certificate:
                      description: Certificate for access opc server.
                      type: string
                    password:
                      description: Password for access opc server.
                      type: string
                    privateKey:
                      description: PrivateKey for access opc server.
                      type: string
                    securityMode:
                      description: Defaults to "none".
                      type: string
                    securityPolicy:
                      description: Defaults to "none".
                      type: string
                    timeout:
                      description: Timeout seconds for the opc server connection.???
                      format: int64
                      type: integer
                    url:
                      description: 'Required: The URL for opc server endpoint.'
                      type: string
                    userName:
                      description: Username for access opc server.
                      type: string
                  required:
                    - url
                  type: object
                common:
                  description: Common part of protocol configuration
                  properties:
                    com:
                      properties:
                        baudRate:
                          description: Required. BaudRate 115200|57600|38400|19200|9600|4800|2400|1800|1200|600|300|200|150|134|110|75|50
                          format: int64
                          type: integer
                          enum:
                            - 115200
                            - 57600
                            - 38400
                            - 19200
                            - 9600
                            - 4800
                            - 2400
                            - 1800
                            - 1200
                            - 600
                            - 300
                            - 200
                            - 150
                            - 134
                            - 110
                            - 75
                            - 50
                        dataBits:
                          description: Required. Valid values are 8, 7, 6, 5.
                          format: int64
                          type: integer
                          enum:
                            - 8
                            - 7
                            - 6
                            - 5
                        parity:
                          description: Required. Valid options are "none", "even",
                            "odd". Defaults to "none".
                          type: string
                          enum:
                            - none
                            - even
                            - odd
                        serialPort:
                          description: Required.
                          type: string
                        stopBits:
                          description: Required. Bit that stops 1|2
                          format: int64
                          type: integer
                          enum:
                            - 1
                            - 2
                      required:
                        - baudRate
                        - dataBits
                        - parity
                        - serialPort
                        - stopBits
                      type: object
                    tcp:
                      properties:
                        ip:
                          description: Required.
                          type: string
                        port:
                          description: Required.
                          format: int64
                          type: integer
                      required:
                        - ip
                        - port
                      type: object
                    commType:
                      description: Communication type, like tcp client, tcp server or COM
                      type: string
                    reconnTimeout:
                      description: Reconnection timeout
                      type: integer
                    reconnRetryTimes:
                      description: Reconnecting retry times
                      type: integer
                    collectTimeout:
                      description: 'Define timeout of mapper collect from device.'
                      format: int64
                      type: integer
                    collectRetryTimes:
                      description: 'Define retry times of mapper will collect from device.'
                      format: int64
                      type: integer
                    collectType:
                      description: 'Define collect type, sync or async.'
                      type: string
                      enum:
                        - sync
                        - async
                    customizedValues:
                      description: Customized values for provided protocol
                      type: object
                  type: object
                customizedProtocol:
                  description: Protocol configuration for customized Protocol
                  properties:
                    protocolName:
                      description: The name of protocol
                      type: string
                    configData:
                      description: customized config data
                      type: object
                  required:
                    - protocolName
                  type: object
              type: object
            propertyVisitors:
              description: 'Required: List of property visitors which describe how
                to access the device properties. PropertyVisitors must unique by propertyVisitor.propertyName.'
              items:
                properties:
                  bluetooth:
                    description: Bluetooth represents a set of additional visitor
                      config fields of bluetooth protocol.
                    properties:
                      characteristicUUID:
                        description: 'Required: Unique ID of the corresponding operation'
                        type: string
                      dataConverter:
                        description: Responsible for converting the data being read
                          from the bluetooth device into a form that is understandable
                          by the platform
                        properties:
                          endIndex:
                            description: 'Required: Specifies the end index of incoming
                              byte stream to be considered to convert the data the
                              value specified should be inclusive for example if 3
                              is specified it includes the third index'
                            format: int64
                            type: integer
                          orderOfOperations:
                            description: Specifies in what order the operations(which
                              are required to be performed to convert incoming data
                              into understandable form) are performed
                            items:
                              properties:
                                operationType:
                                  description: 'Required: Specifies the operation
                                    to be performed to convert incoming data'
                                  type: string
                                  enum:
                                    - Add
                                    - Subtract
                                    - Multiply
                                    - Divide
                                operationValue:
                                  description: 'Required: Specifies with what value
                                    the operation is to be performed'
                                  format: double
                                  type: number
                              type: object
                            type: array
                          shiftLeft:
                            description: Refers to the number of bits to shift left,
                              if left-shift operation is necessary for conversion
                            format: int64
                            type: integer
                          shiftRight:
                            description: Refers to the number of bits to shift right,
                              if right-shift operation is necessary for conversion
                            format: int64
                            type: integer
                          startIndex:
                            description: 'Required: Specifies the start index of the
                              incoming byte stream to be considered to convert the
                              data. For example: start-index:2, end-index:3 concatenates
                              the value present at second and third index of the incoming
                              byte stream. If we want to reverse the order we can
                              give it as start-index:3, end-index:2'
                            format: int64
                            type: integer
                        required:
                          - endIndex
                          - startIndex
                        type: object
                      dataWrite:
                        description: 'Responsible for converting the data coming from
                          the platform into a form that is understood by the bluetooth
                          device For example: "ON":[1], "OFF":[0]'
                        type: object
                    required:
                      - characteristicUUID
                    type: object
                  modbus:
                    description: Modbus represents a set of additional visitor config
                      fields of modbus protocol.
                    properties:
                      isRegisterSwap:
                        description: Indicates whether the high and low register swapped.
                          Defaults to false.
                        type: boolean
                      isSwap:
                        description: Indicates whether the high and low byte swapped.
                          Defaults to false.
                        type: boolean
                      limit:
                        description: 'Required: Limit number of registers to read/write.'
                        format: int64
                        type: integer
                      offset:
                        description: 'Required: Offset indicates the starting register
                          number to read/write data.'
                        format: int64
                        type: integer
                      register:
                        description: 'Required: Type of register'
                        type: string
                        enum:
                          - CoilRegister
                          - DiscreteInputRegister
                          - InputRegister
                          - HoldingRegister
                      scale:
                        description: The scale to convert raw property data into final
                          units. Defaults to 1.0
                        format: double
                        type: number
                    required:
                      - limit
                      - offset
                      - register
                    type: object
                  opcua:
                    description: Opcua represents a set of additional visitor config
                      fields of opc-ua protocol.
                    properties:
                      browseName:
                        description: The name of opc-ua node
                        type: string
                      nodeID:
                        description: 'Required: The ID of opc-ua node, e.g. "ns=1,i=1005"'
                        type: string
                    required:
                      - nodeID
                    type: object
                  customizedProtocol:
                    description: customized protocol
                    properties:
                      protocolName:
                        description: The name of protocol
                        type: string
                      configData:
                        description: customized Config Data
                        type: object
                    required:
                      - protocolName
                      - configData
                    type: object
                  propertyName:
                    description: 'Required: The device property name to be accessed.
                      This should refer to one of the device properties defined in
                      the device model.'
                    type: string
                  reportCycle:
                    description: 'Define how frequent mapper will report the value.'
                    format: int64
                    type: integer
                  collectCycle:
                    description: 'Define how frequent mapper will collect from device.'
                    format: int64
                    type: integer
                  customizedValues:
                    description: Customized values for visitor of provided protocols
                    type: object
                required:
                  - propertyName
                type: object
              type: array
            data:
              properties:
                dataTopic:
                  description: 'Topic used by mapper, all data collected from dataProperties
                    should be published to this topic,
                    the default value is $ke/events/device/+/data/update'
                  type: string
                dataProperties:
                  description: A list of data properties, which are not required to be processed by edgecore
                  items:
                    properties:
                      propertyName:
                        description: 'Required: The property name for which the desired/reported
                          values are specified. This property should be present in the
                          device model.'
                        type: string
                      metadata:
                        description: Additional metadata like filter policy, should be k-v format
                        type: object
                    required:
                      - propertyName
                    type: object
                  type: array
              type: object
          required:
            - deviceModelRef
            - nodeSelector
          type: object
        status:
          properties:
            twins:
              description: A list of device twins containing desired/reported desired/reported
                values of twin properties. A passive device won't have twin properties
                and this list could be empty.
              items:
                properties:
                  desired:
                    description: 'Required: the desired property value.'
                    properties:
                      metadata:
                        description: Additional metadata like timestamp when the value
                          was reported etc.
                        type: object
                      value:
                        description: 'Required: The value for this property.'
                        type: string
                    required:
                      - value
                    type: object
                  propertyName:
                    description: 'Required: The property name for which the desired/reported
                      values are specified. This property should be present in the
                      device model.'
                    type: string
                  reported:
                    description: 'Required: the reported property value.'
                    properties:
                      metadata:
                        description: Additional metadata like timestamp when the value
                          was reported etc.
                        type: object
                      value:
                        description: 'Required: The value for this property.'
                        type: string
                    required:
                      - value
                    type: object
                required:
                  - propertyName
                type: object
              type: array
          type: object
  version: v1alpha2
```
### New configMap sample

To avoid duplicated property visitors, we move property visitor section into device instance section.

**Note, this change requires mappers doing modifications accordingly**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: device-profile-config-01-node-1 // needs to be generated by device controller.
  namespace: foo
data:
  deviceProfile.json: |-
{
    "deviceInstances":[
        {
            "id":"sensor-tag-instance-01",
            "name":"sensor-tag-instance-01",
            "protocol":"bluetooth-sensor-tag-instance-01",
            "model":"cc2650-sensortag",
            "twins":[
                {
                    "propertyName":"io-data",
                    "desired":{
                        "value":"1",
                        "metadata":{
                            "type":"int"
                        }
                    },
                    "reported":{
                        "value":"unknown"
                    }
                }
            ],
            "data":{
                "dataProperties":[
                    {
                        "metadata":{
                            "type":"string"
                        },
                        "propertyName":"temperature"
                    }
                ],
                "dataTopic":"$ke/events/+/device/customized/update"
            },
            "propertyVisitors":[
                {
                    "name":"temperature",
                    "propertyName":"temperature",
                    "modelName":"cc2650-sensortag",
                    "protocol":"bluetooth",
                    "collectCycle":500000000,
                    "reportCycle":1000000000,
                    "visitorConfig":{
                        "characteristicUUID":"f000aa0104514000b000000000000000",
                        "dataConverter":{
                            "startIndex":2,
                            "endIndex":1,
                            "shiftRight":2,
                            "orderOfOperations":[
                                {
                                    "operationType":"Multiply",
                                    "operationValue":0.03125
                                }
                            ]
                        }
                    }
                },
                {
                    "name":"temperature-enable",
                    "propertyName":"temperature-enable",
                    "modelName":"cc2650-sensortag",
                    "protocol":"bluetooth",
                    "collectCycle":500000000,
                    "reportCycle":1000000000,
                    "visitorConfig":{
                        "characteristicUUID":"f000aa0204514000b000000000000000",
                        "dataWrite":{
                            "OFF":"AA==",
                            "ON":"AQ=="
                        },
                        "dataConverter":{
                            "startIndex":1,
                            "endIndex":1
                        }
                    }
                }
            ]
        }
    ],
    "deviceModels":[
        {
            "name":"cc2650-sensortag",
            "properties":[
                {
                    "name":"temperature",
                    "dataType":"int",
                    "description":"temperature in degree celsius",
                    "accessMode":"ReadOnly",
                    "defaultValue":0,
                    "maximum":100,
                    "unit":"degree celsius"
                },
                {
                    "name":"temperature-enable",
                    "dataType":"string",
                    "description":"enable data collection of temperature sensor",
                    "accessMode":"ReadWrite",
                    "defaultValue":"ON"
                }
            ]
        }
    ],
    "protocols":[
        {
            "name":"bluetooth-sensor-tag-instance-01",
            "protocol":"bluetooth",
            "protocolConfig":{
                "macAddress":"BC:6A:29:AE:CC:96"
            }
        }
    ]
}
```
## API Changes
Since we change device and device model CRDs, related API should be changed accordingly.
- device model api no longer need property visitors section
- device api add property visitors section
- device api add report cycle and collect cycle in property visitors section
- device api add report cycle and collect cycle in property visitors section
- device api add customized protocol config section in property visitors section
- device api add customized protocol config section in protocol section

### device model api body example
```yaml
apiVersion: devices.kubeedge.io/v1alpha1
kind: DeviceModel
metadata:
 name: cc2650-sensortag
 namespace: default
spec:
 properties:
  - name: temperature
    description: temperature in degree celsius
    type:
     int:
      accessMode: ReadOnly
      maximum: 100
      unit: degree celsius
  - name: temperature-enable
    description: enable data collection of temperature sensor
    type:
      string:
        accessMode: ReadWrite
        defaultValue: 'ON'
```

### device api body example
```yaml
apiVersion: devices.kubeedge.io/v1alpha1
kind: Device
metadata:
  name: sensor-tag-instance-01
  labels:
    description: TISimplelinkSensorTag
    manufacturer: TexasInstruments
    model: cc2650-sensortag
spec:
  deviceModelRef:
    name: cc2650-sensortag
  protocol:
    comom:
      com:
        serialPort: '1'
        baudRate: 115200
        dataBits: 8
        parity: even
        stopBits: 1
      commType: 0
    customizedProtocol:
      protocolName: MY-TEST-PROTOCOL
      configData:
        key1: val1
        key2: val2
        key3:
          innerKey1: ival1
  nodeSelector:
    nodeSelectorTerms:
    - matchExpressions:
      - key: ''
        operator: In
        values:
        - edge-node1          #pls give your edge node name
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
    - propertyName: temperature-enable
      collectCycle: 500000000
      reportCycle: 1000000000
      bluetooth:
        characteristicUUID: f000aa0204514000b000000000000000
        dataWrite:
        "ON": [1]
        "OFF": [0]
  data:
    dataTopic: "$ke/events/device/+/data/update"
    dataProperties:
      - propertyName: temperature-enable
        metadata:
          type: string
      - propertyName: temperature
        metadata:
          type: string
status:
  twins:
    - propertyName: temperature-enable
    - propertyName: io-data
```
## Open questions
- Should we split the monolithic configmap, let each mapper has its own configmap?
- Should we add a default collect cycle and report cycle in device instance?