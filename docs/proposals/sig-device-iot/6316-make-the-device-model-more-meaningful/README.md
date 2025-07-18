---
title: Make the device model more meaningful
status: WIP
authors:
  - "@rumor-sourse"
creation-date: 2025-07-13
last-updated: 2025-07-16
---

## Motivation

Currently, KubeEdge's definition of the device model is relatively simplistic, with limited practical effectiveness, and its design is prone to confusing users. In traditional IoT, devices are typically designed with a three-tier structure: thing model, product, and device instance. Due to historical reasons, the cost of splitting the model into three independent objects is high, and the significance of fine-grained abstraction is limited. Therefore, we propose defining the model as the concept of "real device product" (i.e., a combination of the thing model and the product), which is used to describe the specifications, connection protocols, attribute acquisition methods, etc., of a type of device product. In this way, device instances can share these configurations, and only need to be configured with different connection addresses for different devices. This design can realize a certain degree of reuse of configuration information and make the positioning more clear.

当前 KubeEdge 对设备模型的定义较为简单，实际作用有限，且设计容易让使用者产生困扰。在传统 IoT 中，设备通常被设计为物模型、产品、设备实例三层结构。由于历史原因，将模型拆分为三个独立对象的成本较高，且细粒度抽象的意义不大。因此，我们提出将模型定义为"现实设备产品"的概念（即物模型与产品的结合），用于描述一种设备产品的规格、连接协议、属性获取方式等。这样，设备实例可以共享这些配置，只需针对不同设备配置不同的连接地址。这种设计能够一定程度复用配置信息，并使定位更加清晰。

## Goals

1. Merge the device model and product concept to form a "real device product" model.
2. Use this model to describe the specifications, connection protocols, and attribute acquisition methods of device products.
3. Device instances share the model configuration and only need to configure different connection addresses.
4. Reduce the cost of model splitting and improve configuration reusability.

1.将设备模型与产品概念合并，形成"现实设备产品"模型。
2.通过该模型描述设备产品的规格、连接协议、属性获取方式等。
3.设备实例共享该模型配置，仅需配置不同的连接地址。
4.降低模型拆分的成本，提高配置复用性。

## Proposal Design

### Model Structure

The new "real device product" model will contain the following main fields: 
1. Name: The unique identifier of the device product. 
2. Specification: Describes the specifications of the device product, including supported properties, commands, etc. 
3. Connection Protocol: The connection protocol used by the device, such as MQTT, HTTP, etc. 
4. Property Access Method: Describes how to obtain device properties, such as polling, event triggering, etc. 
5. Metadata: Other metadata related to the device product.

新的"现实设备产品"模型将包含以下主要字段：
1.名称：设备产品的唯一标识。
2.规格：描述设备产品的规格，包括支持的属性、命令等。
3.连接协议：设备使用的连接协议，如 MQTT、HTTP 等。
4.属性获取方式：描述如何获取设备属性，例如轮询、事件触发等。
5.元数据：其他与设备产品相关的元数据。

### Relationship between Device Instance and Model

A device instance will reference the "real device product" model and share its specifications, connection protocol, property access methods, and other configurations. The device instance only needs to configure its specific connection address, while all other configurations are inherited from the model.

设备实例与模型的关系
设备实例将引用"现实设备产品"模型，共享其规格、连接协议、属性获取方式等配置。设备实例只需配置特定的连接地址，其他配置均从模型中继承。

### CRD Design

#### Real Device Product CRD

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: deviceproducts.devices.kubeedge.io
spec:
  group: devices.kubeedge.io
  names:
    kind: DeviceProduct
    plural: deviceproducts
    singular: deviceproduct
    shortNames:
      - dp
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                name:
                  type: string
                specification:
                  type: object
                  properties:
                    properties:
                      type: array
                      items:
                        type: object
                        properties:
                          name:
                            type: string
                          type:
                            type: string
                    commands:
                      type: array
                      items:
                        type: object
                        properties:
                          name:
                            type: string
                          parameters:
                            type: array
                            items:
                              type: object
                              properties:
                                name:
                                  type: string
                                type:
                                  type: string
                connectionProtocol:
                  type: string
                propertyAccessMethod:
                  type: string
                metadata:
                  type: object
```

#### Device Instance CRD

The Device Instance CRD will reference the Real Device Product Model and configure specific connection addresses.

设备实例 CRD 将引用现实设备产品模型，并配置特定的连接地址。

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: deviceinstances.devices.kubeedge.io
spec:
  group: devices.kubeedge.io
  names:
    kind: DeviceInstance
    plural: deviceinstances
    singular: deviceinstance
    shortNames:
      - di
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                name:
                  type: string
                deviceProductRef:
                  type: string
                connectionAddress:
                  type: string
```
### Device Provisioning Improvement

1. Device Product Provisioning
- Distribute the Real Device Product Model to edge nodes.
- Edge nodes receive the device product model and parse its configurations including specifications, protocols, properties, and commands.
- Edge nodes configure the device product's specifications, connection protocols, and attribute acquisition methods based on the device product model.

1、设备产品下发：
（1）将现实设备产品模型下发到边缘节点。
（2）边缘节点接收设备产品模型，解析其规格、协议、属性、命令等配置。
（3）边缘节点根据设备产品模型，配置设备产品的规格、连接协议、属性获取方式等。

2. Device Instance Provisioning
- Distribute device instances to edge nodes.
- Edge nodes receive device instances and resolve their referenced device product models.
- Edge nodes configure the connection addresses for device instances based on the referenced device product models.

2、设备实例下发：
（1）将设备实例下发到边缘节点。
（2）边缘节点接收设备实例，解析其引用的设备产品模型。
（3）边缘节点根据设备实例引用的设备产品模型，配置设备实例的连接地址。

### DMI Compatibility Improvement

DMI (Device Mapper Interface) is the interface for device-system interaction and needs to be compatible with the new device product model and device instance model. DMI compatibility design includes the following:

DMI（Device Mapper Interface）是设备与系统交互的接口，需要兼容新的设备产品模型和设备实例模型。DMI 兼容设计包括以下内容：

1. DMI Interface Adjustments
- Add support for device product model and device instance model in the DMI interface.
- Ensure the DMI interface can handle device product model configurations such as specifications, connection protocols, property access methods, etc.
- Ensure the DMI interface can handle device instance connection addresses.

2. Compatibility Handling
Ensure existing DMI interfaces can be compatible with the new device product model and device instance model. Additionally, add parsing and application logic for device product models and device instance models in the DMI interface.

1、 DMI 接口调整
（1）在 DMI 接口中，增加对设备产品模型和设备实例模型的支持。
（2）确保 DMI 接口能够处理设备产品模型的规格、连接协议、属性获取方式等配置。
（3）确保 DMI 接口能够处理设备实例的连接地址。
2、 兼容性处理
确保现有 DMI 接口能够兼容新的设备产品模型和设备实例模型。并且在 DMI 接口中，增加对设备产品模型和设备实例模型的解析和应用逻辑。

