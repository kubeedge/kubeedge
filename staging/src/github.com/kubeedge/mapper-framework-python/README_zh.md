# KubeEdge Python Mapper Framework

这是一个面向 KubeEdge DMI v1beta1 的 Python Mapper 运行时框架。框架负责：

- 向 EdgeCore DMI Server 注册 Mapper；
- 提供 Mapper 侧 Unix Socket gRPC Server；
- 管理 DeviceModel 和 Device 生命周期；
- 为每个 Device 创建独立的驱动实例；
- 应用云端 desired 值；
- 按 `collectCycle` 采集属性并按 DMI 接口上报；
- 上报设备状态；
- 提供可选的 HTTP 查询/写入 API；
- 处理 protobuf `Any` 和 KubeEdge 的标准 wrapper 类型。

用户只需要实现 `BaseDeviceDriver` 的硬件相关方法，然后通过 `--driver MODULE:CLASS` 启动。DMI、Socket、生命周期、轮询和上报代码不需要重复编写。

本目录参考 KubeEdge 的 Go `mapper-framework`：

```text
staging/src/github.com/kubeedge/mapper-framework
```

## 在进行下述操作前，先确保已安装 kubeedge 相关组件，并接入至少一个边缘节点。

## 快速运行：只打印云端 value

环境要求：Python 3.10+。
```bash
# 如果在 Ubuntu22.04 及以上版本上，确保先安装 Python 3.10+
apt-get update
apt-get install -y python3.10 python3.10-venv
```

```bash
cd staging/src/github.com/kubeedge/mapper-framework-python
python3.10 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --prefer-binary --timeout 60 --retries 2
python -m mapper_framework --config config.yaml
```

默认内置 `PrintDriver`，协议名是 `python-demo`。它不连接真实设备；收到云端 Device 的 desired 值时会打印：

```text
CLOUD_DESIRED_VALUE
{
  "device_id": "default/python-demo-device",
  "property": "message",
  "value": "hello from cloud"
}
```

EdgeCore 的 DMI Socket 默认是 `/etc/kubeedge/dmi.sock`，Mapper 自己的 Socket 默认是 `/etc/kubeedge/python-demo-mapper.sock`。Mapper 启动后，后一个 Socket 由框架自动创建。

## 云端验证

先把 `python-demo-device.yaml` 中的 `spec.nodeName` 改为实际运行 EdgeCore 的节点名，然后在云端执行：

```bash
kubectl apply -f examples/cloud/python-demo-devicemodel.yaml
kubectl apply -f examples/cloud/python-demo-device.yaml
```

修改云端 desired 值：

```bash
kubectl patch device python-demo-device -n default \
  --type='json' \
  -p='[{"op":"replace","path":"/spec/properties/0/desired/value","value":"value-from-cloud"}]'
```

框架会依次收到 `RegisterDevice` 或 `UpdateDevice`，打印云端 value，并调用默认驱动。

## 对接真实设备

新建一个驱动文件，例如 `my_mapper.py`：

```python
from typing import Any

from mapper_framework import BaseDeviceDriver, DeviceContext, PropertyContext


class MyDeviceDriver(BaseDeviceDriver):
    protocol_name = "my-device-protocol"

    def __init__(self, device: DeviceContext) -> None:
        super().__init__(device)
        self.client = None

    def init_device(self) -> None:
        # protocol.configData 已被解码到 self.device.protocol_config
        address = self.device.protocol_config["address"]
        self.client = connect_to_device(address)

    def get_device_data(self, property_context: PropertyContext) -> Any:
        # visitor.configData 已被解码到 property_context.visitor_config
        register = property_context.visitor_config["register"]
        return self.client.read(register)

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        register = property_context.visitor_config["register"]
        self.client.write(register, value)

    def device_data_write(self, method_name, property_context, value):
        if method_name not in self.device.methods:
            raise ValueError(f"unsupported method: {method_name}")
        if property_context.name not in self.device.methods[method_name]:
            raise ValueError(f"unsupported property: {property_context.name}")
        self.set_device_data(value, property_context)

    def get_device_states(self) -> str:
        return '{"connected": true}'

    def stop_device(self) -> None:
        if self.client is not None:
            self.client.close()
            self.client = None
```

把云端 Device/DeviceModel 的协议名改成 `my-device-protocol`，然后启动：

```bash
python -m mapper_framework \
  --config config.yaml \
  --driver my_mapper:MyDeviceDriver
```

完整开发说明见：

[`docs/device-driver-development_zh.md`](docs/device-driver-development_zh.md)

要运行用于联调的 Modbus RTU 模拟设备，请参阅：

[`examples/modbus/simulator/README.md`](examples/modbus/simulator/README.md)

## HTTP API

默认 HTTP Server 监听 `0.0.0.0:7777`，可以在 `config.yaml` 中关闭：

```yaml
http_server:
  enabled: false
```

接口：

```text
GET /api/v1/ping
GET /api/v1/device/{namespace}/{name}/{property}
GET /api/v1/devicemethod/{namespace}/{name}
GET /api/v1/devicemethod/{namespace}/{name}/{method}/{property}/{value}
GET /api/v1/meta/model/{namespace}/{name}
```

`device` 查询会调用驱动的 `get_device_data()`；method 写入会调用 `device_data_write()`。数据库接口暂未实现，访问 database 路由会返回 503。

### 如何定义和调用 Device Method

`setMessage` 不是框架内置的固定方法名，它必须先声明在 Device 的 `spec.methods` 中。声明位置是 Device，不是 DeviceModel：

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

字段含义：

- `methods[].name`：方法名，会出现在 HTTP URL 中；
- `methods[].propertyNames`：该方法允许操作的属性；
- `properties[].name`：必须与 `propertyNames` 中的属性名对应。

应用后先查看 Mapper 当前解析到的方法：

```bash
kubectl apply -f examples/cloud/python-demo-device.yaml
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device
```

调用格式为：

```text
GET /api/v1/devicemethod/{namespace}/{device}/{method}/{property}/{value}
```

例如：

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device/setMessage/message/hello
```

调用链路如下：

```text
HTTP method=setMessage, property=message, value=hello
  → DevicePanel.write_property()
  → Driver.device_data_write("setMessage", property_context, "hello")
  → 真实设备驱动
```

默认 `PrintDriver` 不连接真实设备，只会打印收到的值。真实设备 Driver 应覆盖 `device_data_write()`，并根据 `method_name` 执行对应动作：

```python
def device_data_write(self, method_name, property_context, value):
    if method_name != "setMessage":
        raise ValueError(f"unsupported method: {method_name}")
    if property_context.name != "message":
        raise ValueError(f"unsupported property: {property_context.name}")

    self.client.write_message(value)
```

当前 HTTP 层会校验 Device 和属性是否存在，但方法名最终由 Driver 处理。因此生产 Driver 应自行校验 `method_name`，不要仅依据 HTTP 返回 200 判断真实设备已经成功执行。

## 配置

```yaml
grpc_server:
  socket_path: /etc/kubeedge/python-demo-mapper.sock

common:
  name: python-demo-mapper
  version: v0.1.0
  api_version: v1beta1
  protocol: python-demo
  edgecore_sock: /etc/kubeedge/dmi.sock
  register_with_data: true
  request_timeout_seconds: 10
  reconnect_interval_seconds: 5

http_server:
  enabled: true
  host: 0.0.0.0
  port: 7777
```

`common.protocol` 是 Mapper 注册给 EdgeCore 的协议名。EdgeCore 只会把协议匹配的 Device 分配给该 Mapper；使用自定义驱动时，驱动的 `protocol_name`、Device 的 `spec.protocol.protocolName` 和这里的值必须一致。

## 容器部署

构建和部署：

```bash
docker build -t kubeedge/python-mapper-framework:dev .
kubectl apply -f deployment.yaml
```

Deployment 默认只挂载 `/etc/kubeedge`，因此适用于内置打印驱动。真实设备还需要将设备节点或 Kubernetes Device Plugin 资源暴露到容器；串口、USB、厂商 SDK 的权限和挂载方式请按具体设备补充。

## 测试

```bash
PYTHONPATH=. python -m unittest discover -s tests -v
python -m mapper_framework --help
```
