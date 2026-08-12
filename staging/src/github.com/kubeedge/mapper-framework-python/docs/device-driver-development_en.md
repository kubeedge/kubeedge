# Python Mapper Framework Device Driver Development Guide

## 1. What the framework provides

Python Mapper Framework follows the responsibility boundaries of KubeEdge's Go
`mapper-framework`. Device developers focus on the device protocol and do not need
to implement DMI gRPC or Unix socket code.

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
   ├── Device lifecycle
   ├── collectCycle polling
   ├── ReportDeviceStatus / ReportDeviceStates
   └── optional HTTP API
          │
          ▼
      BaseDeviceDriver
          │
          ▼
      serial / Modbus / TCP / MQTT / HTTP / vendor SDK
```

The framework implements the EdgeCore DMI client, Mapper Unix-socket gRPC server,
in-memory DeviceModel/Device cache, registration/update/removal/reconnect, desired
value conversion, `Any` and protobuf wrapper decoding, property collection and Twin
reporting, state reporting, HTTP reads/writes, and a hardware-free `PrintDriver`.

## 2. Implement one Driver

Subclass `BaseDeviceDriver` and declare a protocol name:

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
        self.serial = open(port, "r+b", buffering=0)

    def get_device_data(self, property_context: PropertyContext) -> Any:
        return read_register(self.serial, property_context.visitor_config["register"])

    def set_device_data(self, value: Any, property_context: PropertyContext) -> None:
        write_register(self.serial, property_context.visitor_config["register"], value)

    def device_data_write(self, method_name: str, property_context: PropertyContext, value: Any) -> None:
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

| Method | Called when | Responsibility |
| --- | --- | --- |
| `init_device()` | Device registration or update | Connect, authenticate, initialize registers |
| `get_device_data(context)` | Periodic collection or HTTP read | Read a property and return a Python value |
| `set_device_data(value, context)` | Cloud desired or HTTP write | Encode and write a value |
| `device_data_write(method, context, value)` | HTTP method write | Defaults to `set_device_data`; override for commands |
| `get_device_states()` | State reporting cycle | Return a state or JSON string |
| `stop_device()` | Removal, update, Mapper exit | Stop work and release connections |

One Driver instance is created per Device, so `self` can hold a device-specific
serial connection, TCP connection, or SDK client.

## 3. Configuration

`Device.spec.protocol.configData` is decoded to `self.device.protocol_config`:

```yaml
protocol:
  protocolName: serial-demo
  configData:
    port: /dev/ttyUSB0
    baudrate: "9600"
    timeout: "1"
```

Each property's `visitors.configData` is decoded to
`property_context.visitor_config`:

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

`CustomizedValue.data` is `map<string, google.protobuf.Any>`. Common wrappers
(`StringValue`, `BoolValue`, integer wrappers, floating-point wrappers, and
`BytesValue`) are decoded to regular Python values. Unknown custom protobuf types
become diagnostic dictionaries rather than crashing a gRPC callback.

## 4. Desired-value flow

On Device registration or update the framework finds the DeviceModel, creates and
initializes the driver, iterates `Device.spec.properties`, prints
`CLOUD_DESIRED_VALUE`, converts the desired string by DeviceModel type, skips
ReadOnly properties, invokes `set_device_data()` for ReadWrite properties, and
starts `collectCycle`/`reportCycle` tasks.

| DeviceModel type | Python type received by Driver |
| --- | --- |
| `STRING` or unknown | `str` |
| `INT`, `INTEGER`, `INT32`, `INT64` | `int` |
| `FLOAT`, `FLOAT32`, `DOUBLE`, `FLOAT64` | `float` |
| `BOOL`, `BOOLEAN` | `bool` |

Desired values are strings in DMI protobuf; the framework converts them. Do not
treat a string such as `"false"` as a Python boolean in your driver.

## 5. Device Methods

Declare methods in the cloud Device's `spec.methods`, not in DeviceModel:

```yaml
methods:
  - name: setMessage
    description: Set message property
    propertyNames:
      - message
```

`setMessage` is an application-defined name. `message` must be a property of the
same Device; methods may reference several properties and a property may belong to
several methods. Inspect parsed methods after registration:

```bash
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device
```

An empty `methods` list commonly means the Device YAML has no `spec.methods`, the
resource has not been delivered again, or the update did not reach EdgeCore/Mapper.
Invoke a method using:

```text
GET /api/v1/devicemethod/{namespace}/{device}/{method}/{property}/{value}
```

The HTTP server calls `DevicePanel.write_property()`, then
`Driver.device_data_write()`, then the physical device. `set_device_data()` is for
cloud `properties[].desired.value`; `device_data_write()` is for an explicit HTTP
device action. HTTP success only means the Driver did not raise; verify the device
response and raise on failure.

## 6. Collection and reporting

When `reportToCloud: true` and the Driver implements `get_device_data()`, the
framework reads on `collectCycle` and calls EdgeCore `ReportDeviceStatus`. Reports
include `propertyName`, `observedDesired.value`, `reported.value`, and
`reported.metadata.type`. Return only the device value from the Driver.

`get_device_states()` and the Device state's `reportToCloud`/`reportCycle` control
state reports. Database persistence and HTTP push methods are not implemented; add
them in a Driver or your own application thread when needed.

## 7. Register and start Drivers

Use Python's `MODULE:CLASS` form:

```bash
python -m mapper_framework --config config.yaml --driver my_mapper:SerialDeviceDriver
python -m mapper_framework --config config.yaml \
  --driver drivers.modbus:ModbusDriver --driver drivers.mqtt:MqttDriver
```

The built-in `PrintDriver` has `protocol_name="python-demo"`. A custom Driver
registered later with the same protocol replaces it.

## 8. Lifecycle and failure handling

After `RegisterDevice`, a Device enters the active table and collection begins only
after `init_device()` succeeds. On failure it is marked `DEVICE_START_PENDING` and
retried after `reconnect_interval_seconds`. `UpdateDevice` stops the old Driver,
creates a new one, then reinitializes and reapplies desired values. `RemoveDevice`
stops collection, calls `stop_device()`, and clears Device, session, and report
caches; therefore make `stop_device()` idempotent.

All hardware operations need finite timeouts: serial read timeout; TCP connect and
read/write timeouts; HTTP connect/read timeouts; and vendor SDK calls in cancellable
or controlled threads. Never use unbounded blocking reads or allow one Device
failure to terminate the Mapper. The framework catches collection/reporting thread
exceptions and continues with other Devices.

## 9. HTTP API and security

The HTTP server defaults to `0.0.0.0:7777` for local diagnostics and simple I/O:

```bash
curl http://127.0.0.1:7777/api/v1/ping
curl http://127.0.0.1:7777/api/v1/device/default/python-demo-device/message
curl http://127.0.0.1:7777/api/v1/devicemethod/default/python-demo-device
curl http://127.0.0.1:7777/api/v1/meta/model/default/python-demo-device
```

These endpoints access the physical device through `get_device_data()` or
`device_data_write()`. In production, add authentication and access control through
a Service, NetworkPolicy, or reverse proxy.

## 10. Container access to physical devices

The supplied Deployment mounts only the DMI socket. A serial device also needs a
device plugin or a minimal hostPath mount:

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

Prefer Kubernetes Device Plugins and stable udev paths over `privileged: true`.
Only one Mapper instance should own a physical device, so the Deployment defaults
to `replicas: 1`.

## 11. Development checklist

1. Verify DeviceModel, Device, and protocol filtering with `PrintDriver`.
2. Test `init_device`, `set_device_data`, and `stop_device` order with a fake Driver.
3. Test frame parsing with a fake serial or TCP server.
4. Test string, integer, float, and boolean desired values.
5. Confirm ReadOnly properties are not written.
6. Test `collectCycle` and `reportToCloud`.
7. Simulate disconnection, timeout, partial frames, and reconnect.
8. Confirm UpdateDevice releases the old connection.
9. Confirm RemoveDevice leaves no threads or file descriptors.
10. Confirm Mapper restart reconnects to the device.
