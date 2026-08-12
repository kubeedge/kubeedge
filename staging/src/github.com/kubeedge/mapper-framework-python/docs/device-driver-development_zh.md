# Python Mapper Framework 设备驱动开发指南

## 1. 框架替你完成什么

Python Mapper Framework 对齐 KubeEdge Go `mapper-framework` 的职责划分。设备开发者只关注设备协议，不需要自己编写 DMI gRPC 和 Socket 代码。

```text
CloudCore
   │ Device / DeviceModel / desired
   ▼
EdgeCore DMI Server
   │ /etc/kubeedge/dmi.sock
   ▼
mapper_framework
   ├── MapperRegister
   ├── DeviceMapperService
   ├── Device 生命周期
   ├── collectCycle 轮询
   ├── ReportDeviceStatus / ReportDeviceStates
   └── 可选 HTTP API
          │
          ▼
      BaseDeviceDriver
          │
          ▼
      串口 / Modbus / TCP / MQTT / HTTP / 厂商 SDK
```

框架已经实现：

- EdgeCore DMI 客户端；
- Mapper Unix Socket gRPC Server；
- DeviceModel/Device 内存缓存；
- Device 注册、更新、删除和重连；
- desired 值类型转换；
- `Any`、`StringValue`、`Int32Value` 等配置解码；
- 属性采集和 Twin 上报；
- 设备状态上报；
- HTTP 查询和写入；
- 无硬件的 `PrintDriver`。

## 2. 只需要实现一个 Driver

继承 `BaseDeviceDriver`，声明自己的协议名：

```python
from typing import Any

from mapper_framework import BaseDeviceDriver, DeviceContext, PropertyContext


class SerialDeviceDriver(BaseDeviceDriver):
    protocol_name = "serial-demo"

    def __init__(self, device: DeviceContext) -> None:
        super().__init__(device)
        self.serial = None

    def init_device(self) -> None:
        port = self.device.protocol_config["port"]
        baudrate = int(self.device.protocol_config.get("baudrate", 9600))
        # import serial
        # self.serial = serial.Serial(port, baudrate=baudrate, timeout=1)
        self.serial = open(port, "r+b", buffering=0)

    def get_device_data(self, property_context: PropertyContext) -> Any:
        register = property_context.visitor_config["register"]
        # 读取 register，解析协议帧，返回 int/float/bool/string 均可。
        return read_register(self.serial, register)

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        register = property_context.visitor_config["register"]
        write_register(self.serial, register, value)

    def device_data_write(
        self,
        method_name: str,
        property_context: PropertyContext,
        value: Any,
    ) -> None:
        if method_name not in self.device.methods:
            raise ValueError(f"unsupported method: {method_name}")
        if property_context.name not in self.device.methods[method_name]:
            raise ValueError(f"unsupported property: {property_context.name}")
        self.set_device_data(value, property_context)

    def get_device_states(self) -> str:
        return '{"connected": true}'

    def stop_device(self) -> None:
        if self.serial is not None:
            self.serial.close()
            self.serial = None
```

方法职责：

| 方法 | 何时调用 | 应实现的内容 |
| --- | --- | --- |
| `init_device()` | Device 注册或更新 | 创建连接、鉴权、初始化寄存器 |
| `get_device_data(property_context)` | 周期采集或 HTTP 读 | 读取指定属性并返回 Python 值 |
| `set_device_data(value, property_context)` | 云端 desired 或 HTTP 写 | 将值编码后写入设备 |
| `device_data_write(method, property_context, value)` | HTTP method 写入 | 默认转调 `set_device_data`，复杂命令可覆盖 |
| `get_device_states()` | 状态上报周期 | 返回状态字符串或 JSON 字符串 |
| `stop_device()` | Device 删除、更新、Mapper 退出 | 停止线程并释放连接 |

框架为每个 Device 创建一个 Driver 实例，因此可以在 `self` 中保存该设备独有的串口、TCP 连接或 SDK client。

## 3. 配置从哪里来

### 3.1 Device 级协议配置

Device 的 `spec.protocol.configData` 会解码到：

```python
self.device.protocol_config
```

云端示例：

```yaml
spec:
  protocol:
    protocolName: serial-demo
    configData:
      port: /dev/ttyUSB0
      baudrate: "9600"
      timeout: "1"
```

### 3.2 属性级 Visitor 配置

Device 属性的 `visitors.configData` 会解码到：

```python
property_context.visitor_config
```

示例：

```yaml
properties:
  - name: temperature
    visitors:
      protocolName: serial-demo
      configData:
        register: "0x1000"
        encoding: float32
    collectCycle: 1000
    reportToCloud: true
```

不同属性可以使用不同寄存器或 topic，驱动只需读取 `PropertyContext`。

### 3.3 Any 类型

KubeEdge 的 `CustomizedValue.data` 是 `map<string, google.protobuf.Any>`。框架已经支持常用 wrapper：

```text
google.protobuf.StringValue
google.protobuf.BoolValue
google.protobuf.Int32Value
google.protobuf.Int64Value
google.protobuf.UInt32Value
google.protobuf.UInt64Value
google.protobuf.FloatValue
google.protobuf.DoubleValue
google.protobuf.BytesValue
```

因此驱动拿到的配置通常已经是普通 Python 值，不需要自行处理 protobuf `Any`。未知的自定义 protobuf 类型会被转换为诊断字典，不会让 gRPC 回调直接崩溃。

## 4. desired 值下发流程

当 Device 注册或更新时，框架执行：

```text
1. 找到 DeviceModel
2. 创建 Driver(device)
3. driver.init_device()
4. 遍历 Device.spec.properties
5. 打印 CLOUD_DESIRED_VALUE
6. 根据 DeviceModel.type 转换 string
7. ReadOnly 属性跳过写入
8. ReadWrite 属性调用 driver.set_device_data()
9. 启动 collectCycle/reportCycle 任务
```

模型中的常见类型会转换为：

| DeviceModel type | Driver 收到的 Python 类型 |
| --- | --- |
| `STRING`、未知类型 | `str` |
| `INT`、`INTEGER`、`INT32`、`INT64` | `int` |
| `FLOAT`、`FLOAT32`、`DOUBLE`、`FLOAT64` | `float` |
| `BOOL`、`BOOLEAN` | `bool` |

注意：desired 在 DMI protobuf 中是字符串，类型转换由框架完成。驱动不应再次把 `"false"` 当作 Python 真值直接判断。

## 5. Device Method：声明、发现和执行

### 5.1 在 Device 中声明方法

方法定义在云端 `Device` 的 `spec.methods` 中，不是在 `DeviceModel` 中。一个方法通过 `propertyNames` 声明它可以操作哪些属性：

```yaml
apiVersion: devices.kubeedge.io/v1beta1
kind: Device
metadata:
  name: python-demo-device
  namespace: default
spec:
  deviceModelRef:
    name: python-demo-model
  protocol:
    protocolName: python-demo
  properties:
    - name: message
      desired:
        value: hello
  methods:
    - name: setMessage
      description: Set message property
      propertyNames:
        - message
```

这里：

- `setMessage` 是自定义方法名，不是框架保留字；
- `message` 必须是同一个 Device 的属性；
- 一个方法可以通过 `propertyNames` 关联多个属性；
- 多个方法也可以操作同一个属性。

当前仓库的打印样例 Device 已包含 `setMessage` 方法定义，位置是：

```text
examples/cloud/python-demo-device.yaml
```

如果用户自己的 Device 没有 `spec.methods`，仍可能因为当前 HTTP 路由只检查 Device 和属性而得到 200，但这不代表方法已经正确声明。生产环境应在 Driver 中校验方法名和属性名。

### 5.2 查看 Mapper 解析到的方法

启动 Mapper 并完成 Device 注册后执行：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device
```

预期返回中包含：

```json
{
  "data": {
    "methods": [
      {
        "name": "setMessage",
        "parameters": [
          {"propertyName": "message", "valueType": "STRING"}
        ]
      }
    ]
  }
}
```

如果 `methods` 是空数组，通常表示：

- Device YAML 没有写 `spec.methods`；
- 资源还没有重新下发到 Mapper；
- Device 的更新事件没有到达 EdgeCore/Mapper。

### 5.3 调用方法

HTTP URL 格式是：

```text
GET /api/v1/devicemethod/{namespace}/{device}/{method}/{property}/{value}
```

示例：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device/setMessage/message/hello
```

框架内部调用顺序：

```text
HTTP 请求
  → HTTPServer 路由
  → DevicePanel.write_property()
  → Driver.device_data_write("setMessage", property, "hello")
  → 真实设备
```

### 5.4 在 Driver 中实现方法

`BaseDeviceDriver.device_data_write()` 默认只是调用 `set_device_data()`。如果不同方法有不同的设备动作，应覆盖它：

```python
from typing import Any

from mapper_framework import BaseDeviceDriver, DeviceContext, PropertyContext


class MyDeviceDriver(BaseDeviceDriver):
    protocol_name = "python-demo"

    def __init__(self, device: DeviceContext) -> None:
        super().__init__(device)
        self.client = connect_to_device(self.device.protocol_config)

    def device_data_write(
        self,
        method_name: str,
        property_context: PropertyContext,
        value: Any,
    ) -> None:
        if method_name != "setMessage":
            raise ValueError(f"unsupported method: {method_name}")
        if property_context.name != "message":
            raise ValueError(f"unsupported property: {property_context.name}")

        self.client.write_message(value)

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        # 云端 properties[].desired 值下发时会调用这里。
        self.client.write_message(value)

    def stop_device(self) -> None:
        self.client.close()
```

方法调用和云端 desired 下发是两个入口：

| 入口 | Driver 方法 | 适用场景 |
| --- | --- | --- |
| 云端修改 `properties[].desired.value` | `set_device_data()` | 设备属性同步 |
| HTTP method URL | `device_data_write()` | 显式执行设备动作 |

例如，HTTP 调用成功只表示 Driver 方法没有抛异常。真实设备是否写入成功，必须由 Driver 自己检查设备响应，并在失败时抛出异常。

## 6. 采集与上报

当属性同时满足：

- `reportToCloud: true`；
- Driver 实现了 `get_device_data()`；

框架会按 `collectCycle` 周期读取属性，并调用 EdgeCore 的 `ReportDeviceStatus`。上报内容包含：

```text
propertyName
observedDesired.value
reported.value
reported.metadata.type
```

驱动只需要返回设备值：

```python
def get_device_data(self, property_context: PropertyContext) -> Any:
    return self.client.read(property_context.visitor_config["register"])
```

Device 的状态上报由 `get_device_states()` 和 Device 状态中的 `reportToCloud/reportCycle` 控制：

```python
def get_device_states(self) -> str:
    return json.dumps({
        "connected": self.client.is_connected(),
        "errorCode": 0,
    })
```

当前框架没有实现数据库保存和 HTTP 推送方法；需要时可以在 Driver 或用户自己的业务线程中扩展。

## 7. 注册和启动自定义 Driver

假设文件为：

```text
my_mapper.py
```

启动参数使用 Python 的 `MODULE:CLASS` 格式：

```bash
python -m mapper_framework \
  --config config.yaml \
  --driver my_mapper:SerialDeviceDriver
```

可同时注册多个协议：

```bash
python -m mapper_framework \
  --config config.yaml \
  --driver drivers.modbus:ModbusDriver \
  --driver drivers.mqtt:MqttDriver
```

框架内置 `PrintDriver(protocol_name="python-demo")`。如果自定义 Driver 使用同一个 `protocol_name`，后注册的自定义 Driver 会覆盖默认打印 Driver。

## 8. Device 生命周期和异常处理

### RegisterDevice

`init_device()` 成功后，框架才会把 Device 放入活动设备表并启动采集。如果连接失败，框架会记录 `DEVICE_START_PENDING`，后台按 `reconnect_interval_seconds` 重试。

### UpdateDevice

框架会停止旧 Driver、创建新 Driver 并重新执行初始化和 desired 下发。这样端口、地址、认证信息和 visitor 配置变化都能生效。

### RemoveDevice

框架会停止采集线程，调用 `stop_device()`，然后清理 Device、Session 和上报缓存。`stop_device()` 应该设计为幂等操作。

### 超时和线程

硬件操作必须设置有限超时：

- 串口读设置 `timeout`；
- TCP 设置连接、读写超时；
- HTTP 设置 connect/read timeout；
- 厂商 SDK 调用放在可取消或受控线程中。

不要在驱动里使用无超时的阻塞读取，也不要让一个设备的异常退出整个 Mapper 进程。框架会捕获采集和上报线程异常，并继续运行其他设备。

## 9. HTTP API

HTTP Server 默认监听 `0.0.0.0:7777`，用于本地诊断和简单读写：

```bash
curl http://127.0.0.1:7777/api/v1/ping
curl http://127.0.0.1:7777/api/v1/device/default/python-demo-device/message
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device
curl http://127.0.0.1:7777/api/v1/meta/model/default/python-demo-device
```

如果 Device 的 `spec.methods` 中定义了 `setMessage`，可以调用：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device/setMessage/message/hello
```

这些 API 最终调用 Driver 的 `get_device_data()` 或 `device_data_write()`，因此也会访问真实设备。生产环境请在 Service、NetworkPolicy 或反向代理层增加认证和访问控制。

## 10. 容器访问实际设备

当前 Deployment 只挂载 DMI Socket：

```yaml
volumeMounts:
  - name: kubeedge-socket
    mountPath: /etc/kubeedge
```

真实串口设备还需要设备插件或最小化的 hostPath：

```yaml
volumeMounts:
  - name: serial-device
    mountPath: /dev/ttyUSB0
volumes:
  - name: serial-device
    hostPath:
      path: /dev/ttyUSB0
      type: CharDevice
```

生产环境优先使用 Kubernetes Device Plugin 和稳定的 udev 路径，避免默认使用 `privileged: true`。同一个物理设备应只由一个 Mapper 实例持有，因此 Deployment 默认 `replicas: 1`。

## 11. 开发测试清单

1. 用 `PrintDriver` 验证 DeviceModel、Device 和 protocol 过滤；
2. 用假的 Driver 验证 `init_device/set_device_data/stop_device` 调用顺序；
3. 用 fake serial/TCP server 验证帧解析；
4. 测试 desired 的字符串、整数、浮点数和布尔值；
5. 测试 ReadOnly 属性不会写入；
6. 测试 `collectCycle` 和 `reportToCloud`；
7. 模拟拔线、超时、半包和重连；
8. 测试 UpdateDevice 会释放旧连接；
9. 测试 RemoveDevice 不遗留线程和文件描述符；
10. 测试 Mapper 重启后能重新连接设备。
