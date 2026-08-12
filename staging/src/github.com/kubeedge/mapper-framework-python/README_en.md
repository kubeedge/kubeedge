# KubeEdge Python Mapper Framework

This is a Python Mapper runtime framework for KubeEdge DMI v1beta1. The framework:

- Registers a Mapper with the EdgeCore DMI Server.
- Provides the Mapper-side Unix socket gRPC server.
- Manages DeviceModel and Device lifecycles.
- Creates an independent driver instance for each Device.
- Applies desired values from the cloud.
- Collects properties according to `collectCycle` and reports them through DMI.
- Reports device states.
- Optionally exposes HTTP APIs for reads and writes.
- Handles protobuf `Any` and KubeEdge standard wrapper types.

Implement only the hardware-specific methods of `BaseDeviceDriver`, then start the
Mapper with `--driver MODULE:CLASS`. You do not need to rewrite DMI, sockets,
lifecycle handling, polling, or reporting.

This directory follows KubeEdge's Go `mapper-framework`:

```text
staging/src/github.com/kubeedge/mapper-framework
```

Before proceeding, ensure KubeEdge is installed and at least one edge node is
connected.

## Quick start: print cloud values

Python 3.10 or later is required. On Ubuntu 22.04 and later, install Python 3.10
first if necessary:

```bash
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

The built-in `PrintDriver` uses the `python-demo` protocol. It does not connect to
a physical device; when it receives a Device desired value from the cloud, it
prints:

```text
CLOUD_DESIRED_VALUE
{
  "device_id": "default/python-demo-device",
  "property": "message",
  "value": "hello from cloud"
}
```

The default EdgeCore DMI socket is `/etc/kubeedge/dmi.sock`. The Mapper socket is
`/etc/kubeedge/python-demo-mapper.sock` by default and is created automatically
when the Mapper starts.

## Verify from the cloud

Change `spec.nodeName` in `python-demo-device.yaml` to the name of the node
running EdgeCore, then run on the cloud side:

```bash
kubectl apply -f examples/cloud/python-demo-devicemodel.yaml
kubectl apply -f examples/cloud/python-demo-device.yaml
```

Update a desired value:

```bash
kubectl patch device python-demo-device -n default \
  --type='json' \
  -p='[{"op":"replace","path":"/spec/properties/0/desired/value","value":"value-from-cloud"}]'
```

The framework receives `RegisterDevice` or `UpdateDevice`, prints the cloud value,
and invokes the default driver.

## Connect a physical device

Create a driver file such as `my_mapper.py`:

```python
from typing import Any

from mapper_framework import BaseDeviceDriver, DeviceContext, PropertyContext


class MyDeviceDriver(BaseDeviceDriver):
    protocol_name = "my-device-protocol"

    def __init__(self, device: DeviceContext) -> None:
        super().__init__(device)
        self.client = None

    def init_device(self) -> None:
        # protocol.configData is decoded into self.device.protocol_config.
        address = self.device.protocol_config["address"]
        self.client = connect_to_device(address)

    def get_device_data(self, property_context: PropertyContext) -> Any:
        # visitor.configData is decoded into property_context.visitor_config.
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

Set the protocol name in the cloud Device and DeviceModel to `my-device-protocol`,
then start the Mapper:

```bash
python -m mapper_framework \
  --config config.yaml \
  --driver my_mapper:MyDeviceDriver
```

See the complete development guide at
[`docs/device-driver-development_en.md`](docs/device-driver-development_en.md).

To run the Modbus RTU simulated device for integration testing, see
[`examples/modbus/simulator/README_en.md`](examples/modbus/simulator/README_en.md).

## HTTP API

The HTTP server listens on `0.0.0.0:7777` by default. Disable it in `config.yaml`:

```yaml
http_server:
  enabled: false
```

Endpoints:

```text
GET /api/v1/ping
GET /api/v1/device/{namespace}/{name}/{property}
GET /api/v1/devicemethod/{namespace}/{name}
GET /api/v1/devicemethod/{namespace}/{name}/{method}/{property}/{value}
GET /api/v1/meta/model/{namespace}/{name}
```

A `device` query invokes `get_device_data()` on the driver; a method write invokes
`device_data_write()`. The database API is not implemented yet, so database routes
return 503.

### Define and invoke a Device Method

`setMessage` is not a built-in method name. Declare it in the Device's
`spec.methods`, not in the DeviceModel:

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

- `methods[].name` is the method name used in the HTTP URL.
- `methods[].propertyNames` lists the properties the method may operate on.
- Each `properties[].name` must correspond to a name in `propertyNames`.

Apply the resource and inspect the methods parsed by the Mapper:

```bash
kubectl apply -f examples/cloud/python-demo-device.yaml
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device
```

The invocation format is:

```text
GET /api/v1/devicemethod/{namespace}/{device}/{method}/{property}/{value}
```

For example:

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device/setMessage/message/hello
```

The call path is:

```text
HTTP method=setMessage, property=message, value=hello
  → DevicePanel.write_property()
  → Driver.device_data_write("setMessage", property_context, "hello")
  → physical-device driver
```

The default `PrintDriver` only prints values. A physical-device driver should
override `device_data_write()` and validate `method_name` itself:

```python
def device_data_write(self, method_name, property_context, value):
    if method_name != "setMessage":
        raise ValueError(f"unsupported method: {method_name}")
    if property_context.name != "message":
        raise ValueError(f"unsupported property: {property_context.name}")
    self.client.write_message(value)
```

The HTTP layer validates the Device and property, but the driver ultimately handles
the method name. A production driver must validate `method_name`; an HTTP 200 alone
does not prove that the physical device completed the operation.

## Configuration

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

`common.protocol` is the protocol name registered with EdgeCore. EdgeCore assigns
only Devices with a matching protocol to the Mapper. With a custom driver, its
`protocol_name`, the Device's `spec.protocol.protocolName`, and this setting must
all match.

## Container deployment

Build and deploy:

```bash
docker build -t kubeedge/python-mapper-framework:dev .
kubectl apply -f deployment.yaml
```
