# How to implement mapper
If you don't need additional customization features, just change the following three files:
## 1. config.yaml
`common.protocol` needs to be defined and to be the same as the definition in `instance.yaml`, for example:
```yaml
common:
  protocol: foo
```

## 2. devicetype.go
### ProtocolConfig
For fields `ProtocolConfig`, it is necessary to fill it in according to the definition of CRD.

### VisitorConfigData
For field `VisitorConfigData`, it may have multiple definitions. So, when filling in it, the `omitempty` tag needs to be added.

## 3. driver.go
You can obtain the attribute values defined in `devicetype.go`. And use these values to implement 4 functions.

### InitDevice
When you add a device to the mapper, the `InitDevice` function will be called once.

### GetDeviceData
`GetDeviceData` will be called periodically based on `collectCycle (default 1 second)`.

### SetDeviceData
When the cloud modifies the device's `readwrite` value, SetDeviceData will be called.

### StopDevice
When the device is removed from the mapper, the `StopDevice` function will be called once.