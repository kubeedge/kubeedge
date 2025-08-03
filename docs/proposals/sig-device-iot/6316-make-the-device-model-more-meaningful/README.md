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

1. Merge the device model with the product concept to create a "Device Product" model.
2. Extract some fields from the existing device instance into the device model to establish default instance configurations.
3. Device instances share this model configuration, requiring only the configuration of different connection addresses.
4. Reduce the cost of model separation and improve configuration reusability.

1.将设备模型与产品概念合并，形成"现实设备产品"模型。
2.将现有的设备实例模型的一些字段抽取到model层中，形成默认实例配置。
3.设备实例共享该模型配置，仅需配置不同的连接地址。
4.降低模型拆分的成本，提高配置复用性。

## Proposal Design

### Property

device_model struct will add the following fields:

device_model层将新增以下字段：

```go
type DefaultInstanceProperty struct{
    Visitors *VisitorConfig `json:"visitors,omitempty"`
    
    ReportCycle *int64 `json:"reportCycle,omitempty"`
    CollectCycle *int64 `json:"collectCycle,omitempty"`
    
    ReportToCloud *bool `json:"reportToCloud,omitempty"`

    PushMethod *PushMethod `json:"pushMethod,omitempty"`
}
```

### Relationship between Device Instance and Model

A device instance will reference the "real device product" model and share its specifications, connection protocol, property access methods, and other configurations. The device instance only needs to configure its specific connection address, while all other configurations are inherited from the model.

设备实例与模型的关系
设备实例将引用"现实设备产品"模型，共享其规格、连接协议、属性获取方式等配置。设备实例只需配置其特有的属性，其他配置均从模型中继承，同时也可以根据需求重写默认的配置。

<img src="./new-device-crd.jpg">


### DMI Compatibility Improvement

DMI (Device Mapper Interface) is the interface through which devices interact with the system and needs to be compatible with the new device product model.  
The DMI compatibility design will be implemented by modifying the `dealMetaDeviceOperation` function in the file `edge\pkg\devicetwin\dtmanager\dmiworker.go`. The main purpose is to rewrite the fields of the received `device_instance`.

DMI（Device Mapper Interface）是设备与系统交互的接口，需要兼容新的设备产品模型。
DMI 兼容设计将在edge\pkg\devicetwin\dtmanager\dmiworker.go文件中的dealMetaDeviceOperation函数处做修改，主要作用是对接收到的device_instance进行字段重写

1.Field rewriting trigger conditions
- Inserting device instance
- Updating device instance

2.Field rewriting logic
- Retrieve the device model
- Check whether default configurations exist
- Rewrite the device instance properties: if a certain field is empty, replace it with the default value

1、 字段重写触发时机
（1）插入设备实例
（2）更新设备实例
2、 字段重写逻辑
（1）获取device model
（2）检查是否有默认配置
（3）device instance properties重写，即如果某个字段为空，则用默认字段代替它

```go
// mergeDevicePropertiesWithModel merges device properties with model defaults
func (dw *DMIWorker) mergeDevicePropertiesWithModel(device *v1beta1.Device) error {
	if device.Spec.DeviceModelRef == nil {
		return fmt.Errorf("device %s has no device model reference", device.Name)
	}

	// Get device model from cache
	deviceModelID := util.GetResourceID(device.Namespace, device.Spec.DeviceModelRef.Name)
	dw.dmiCache.DeviceModelMu.Lock()
	deviceModel, exists := dw.dmiCache.DeviceModelList[deviceModelID]
	dw.dmiCache.DeviceModelMu.Unlock()

	if !exists {
		return fmt.Errorf("device model %s not found for device %s", device.Spec.DeviceModelRef.Name, device.Name)
	}

	// If device model has no default instance property, no merging needed
	if deviceModel.Spec.DefaultInstanceProperty == nil {
		klog.Infof("Device model %s has no default instance property, skipping merge for device %s", deviceModel.Name, device.Name)
		return nil
	}

	klog.Infof("Merging device properties with model defaults for device %s using model %s", device.Name, deviceModel.Name)

	defaultProps := deviceModel.Spec.DefaultInstanceProperty

	// Merge properties
	for i := range device.Spec.Properties {
		deviceProp := &device.Spec.Properties[i]

		// Apply default visitors if not set (check if visitors is empty or has no protocol name)
		if (deviceProp.Visitors.ProtocolName == "" || deviceProp.Visitors.ConfigData == nil) && defaultProps.Visitors != nil {
			deviceProp.Visitors = *defaultProps.Visitors
			klog.Infof("Applied default visitors to property %s of device %s", deviceProp.Name, device.Name)
		}

		// Apply default report cycle if not set
		if deviceProp.ReportCycle == 0 && defaultProps.ReportCycle != nil {
			deviceProp.ReportCycle = *defaultProps.ReportCycle
			klog.Infof("Applied default report cycle %d to property %s of device %s", *defaultProps.ReportCycle, deviceProp.Name, device.Name)
		}

		// Apply default collect cycle if not set
		if deviceProp.CollectCycle == 0 && defaultProps.CollectCycle != nil {
			deviceProp.CollectCycle = *defaultProps.CollectCycle
			klog.Infof("Applied default collect cycle %d to property %s of device %s", *defaultProps.CollectCycle, deviceProp.Name, device.Name)
		}

		// Apply default report to cloud if not set
		if !deviceProp.ReportToCloud && defaultProps.ReportToCloud != nil {
			deviceProp.ReportToCloud = *defaultProps.ReportToCloud
			klog.Infof("Applied default report to cloud %v to property %s of device %s", *defaultProps.ReportToCloud, deviceProp.Name, device.Name)
		}

		// Apply default push method if not set
		if deviceProp.PushMethod == nil && defaultProps.PushMethod != nil {
			deviceProp.PushMethod = defaultProps.PushMethod
			klog.Infof("Applied default push method to property %s of device %s", deviceProp.Name, device.Name)
		}
	}

	return nil
}
```
