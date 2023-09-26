# How to implement mapper
If you don't need additional customization features, just change the following three files:
## 1. config.yaml
`common.protocol` needs to be defined and to be the same as the definition in `instance.yaml`, for example:
```yaml
common:
  protocol: foo
```

## 2. devicetype.go
### ProtocolConfig And ProtocolCommonConfig
For fields `ProtocolConfig` and `ProtocolCommonConfig`, it is necessary to fill them in according to the definition of CRD.

#### example
Instance is defined as follows:
```yaml
  protocol:
    customizedProtocol:
      protocolName: foo
      configData:
        deviceID: 1
    common:
      com:
        serialPort: '/dev/ttyS0'
        baudRate: 9600
        dataBits: 8
        parity: even
        stopBits: 1
      customizedValues:
        protocolID: 1
```
So the `devicetype.go` is defined as follows:
```go
type ProtocolConfigData struct {
	DeviceID int `json:"deviceID"`
}

type CustomizedDeviceProtocolCommonConfig struct {
	Com                    `json:"com"`
	CommonCustomizedValues `json:"customizedValues"`
}

type Com struct {
	SerialPort string `json:"serialPort"`
	DataBits   int    `json:"dataBits"`
	BaudRate   int    `json:"baudRate"`
	Parity     string `json:"parity"`
	StopBits   int    `json:"stopBits"`
}

type CommonCustomizedValues struct {
	ProtocolID int `json:"protocolID"`
}
```

### VisitorConfigData
For field `VisitorConfigData`, it may have multiple definitions. So, when filling in it, the `omitempty` tag needs to be added.

#### example
Instance is defined as follows:
```yaml
  propertyVisitors:
    - propertyName: foo
      customizedProtocol:
        protocolName: fooProtocol
        configData:
          keyFoo: value1
    - propertyName: bar
      customizedProtocol:
        protocolName: barProtocol
        configData:
          keyBar: value2
```
So the `devicetype.go` is defined as follows:
```go
type VisitorConfigData struct {
	KeyFoo string `json:"keyFoo,omitempty"`
    KeyBar string `json:"keyBar,omitempty"`
}
```
## 3. driver.go
You can obtain the attribute values defined in `devicetype.go`. And use these values to implement 4 functions.

### InitDevice
When you add a device to the mapper, the `InitDevice` function will be called once.

### GetDeviceData
`GetDeviceData` will be called periodically based on `collectCycle (default 1 second)`.

#### <div id = "example">Example<div>
If you have two property, temperature and humidity. The instance is defined as follows:
```yaml
  propertyVisitors:
    - propertyName: temperature
      customizedProtocol:
        protocolName: example
        configData:
          Register: 1
    - propertyName: humidity
      customizedProtocol:
        protocolName: example
        configData:
          Register: 2
```
The `GetDeviceData` function should be defined as follows:
```go
func (c *CustomizedClient) GetDeviceData(visitor *VisitorConfig) (interface{}, error) {
	if visitor.Register == 1{
		// TODO do something to get temperature and return it
		return "temperature",nil
	}else if visitor.Register == 2{
		// TODO do something to get humidity and return it
		return "humidity",nil
	}else{
		return nil,fmt.Errorf("the register fail to recognize")
	}
}
```
### SetDeviceData
When the cloud modifies the device's `readwrite` value, SetDeviceData will be called.
You can refer to the example of [GetDeviceData](#example).



### StopDevice
When the device is removed from the mapper, the `StopDevice` function will be called once.