# Mapper Framework-C

The Mapper module acts as a translator between KubeEdge and devices. It enables KubeEdge to:
- Interact with devices via various protocols
- Retrieve device status and properties
- Read and write device data at the edge

This repository provides a C/C++-based mapper framework for quickly building custom protocol mappers.

---

## Create your own mapper (recommended flow)

### 1) Generate a new mapper project (sibling folder)
Create a sibling folder (same level as mapper-framework-c) by copying the full framework code as a scaffold.

```sh
# Run at the root of mapper-framework-c
make generate MyMapper
# If not provided, defaults to 'mapper_default'
```

The generated structure (simplified):
```
MyMapper
├── CMakeLists.txt
├── config.yaml
├── Dockerfile
├── Makefile
├── hack/make-rules/
│   ├── generate.sh
│   └── build.sh
├── common/       # data types and model
├── config/       # config parser
├── data/         # db and publish implementations
├── device/       # device & twin management
├── driver/       # driver interfaces
├── grpcclient/   # DMI client
├── grpcserver/   # DMI server
├── httpserver/   # REST API server
├── log/          # logging
└── util/         # misc utils
```

### 2) Design your DeviceModel and Device CRDs
If unfamiliar with the CRDs, refer to KubeEdge’s proposal:
https://github.com/kubeedge/kubeedge/blob/master/docs/proposals/device-crd-v1beta1.md

### 3) Customize your driver
Implement protocol-specific logic for:
- Initialize/stop device
- Read/write device data
- Protocol and visitor configs

---

## Build

### A) Local build with CMake (for development)

```sh
mkdir -p build && cd build
cmake ..
make -j
```

### B) Build a Docker image (optional)

```sh
# Build image for the generated sibling folder
make build MyMapper

# Resulting image tag (by default):
# IMAGE=MyMapper TAG=latest
```

---

## Run

After building locally, simply run the binary from the build directory:

```sh
cd build
./main ../config.yaml
```

Notes:
- On startup the local gRPC server binds to a Unix socket (e.g., /tmp/mapper_dmi.sock).
- If the EdgeCore DMI socket is not reachable, the first registration RPC may time out (~5s) and then proceed.

To run the Docker image (if built):

```sh
IMAGE=MyMapper TAG=latest make run
```

---

## Publish (environment variables and demos)

These environment variables enable publishing device data to various sinks:

- PUBLISH_METHOD: http | mqtt | otel
- PUBLISH_CONFIG: channel-specific JSON

### HTTP publish example

```sh
export PUBLISH_METHOD=http
export PUBLISH_CONFIG='{"endpoint":"http://127.0.0.1:8080/ingest","method":"POST"}'
cd build && ./main ../config.yaml
```

Simple HTTP sink for testing (save as http_sink.py):

```python
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        l = int(self.headers.get('Content-Length','0'))
        body = self.rfile.read(l) if l>0 else b''
        print("POST", self.path, body.decode('utf-8', 'ignore'))
        self.send_response(200); self.end_headers()
        self.wfile.write(b"ok")
    def do_PUT(self): self.do_POST()
addr = sys.argv[1] if len(sys.argv)>1 else '0.0.0.0'
port = int(sys.argv[2]) if len(sys.argv)>2 else 8080
print(f"Listening on http://{addr}:{port}")
HTTPServer((addr,port), H).serve_forever()
```

Run sink:

```sh
python http_sink.py 0.0.0.0 8080
```

### MQTT publish example

```sh
# Subscribe to all device topics (adjust broker/port as needed)
mosquitto_sub -h 127.0.0.1 -p 1885 -t 'kubeedge/device/#' -v -d

export PUBLISH_METHOD=mqtt
export PUBLISH_CONFIG='{"brokerUrl":"127.0.0.1","port":1885,"clientId":"mapper_client","topicPrefix":"kubeedge/device","qos":1,"keepAlive":60}'
cd build && ./main ../config.yaml
```

### OTEL (HTTP) publish example

```sh
export PUBLISH_METHOD=otel
export PUBLISH_CONFIG='{"endpoint":"http://127.0.0.1:4318/v1/metrics","serviceName":"mapper-c"}'
cd build && ./main ../config.yaml
```

Simple HTTP sink (save as http_sink.py):

```python
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        l = int(self.headers.get('Content-Length','0'))
        body = self.rfile.read(l) if l>0 else b''
        print("POST", self.path, body.decode('utf-8', 'ignore'))
        self.send_response(200); self.end_headers()
        self.wfile.write(b"ok")
    def do_PUT(self): self.do_POST()
addr = sys.argv[1] if len(sys.argv)>1 else '0.0.0.0'
port = int(sys.argv[2]) if len(sys.argv)>2 else 4318
print(f"Listening on http://{addr}:{port}")
HTTPServer((addr,port), H).serve_forever()
```

Run sink:

```sh
python http_sink.py 0.0.0.0 4318
```

---

## Demos

### Modbus simulator (example)

Python (pymodbus) TCP server on 0.0.0.0:1502:

```python
import asyncio
import logging
from pymodbus.datastore import ModbusSlaveContext, ModbusServerContext, ModbusSequentialDataBlock
from pymodbus.server import StartAsyncTcpServer

logging.basicConfig(level=logging.INFO)

store = ModbusSlaveContext(
    di=ModbusSequentialDataBlock(0, [0]*100),
    co=ModbusSequentialDataBlock(0, [0]*100),
    hr=ModbusSequentialDataBlock(0, [0]*100),
    ir=ModbusSequentialDataBlock(0, [0]*100),
)
context = ModbusServerContext(slaves=store, single=True)

store.setValues(3, 1, [30])
store.setValues(3, 2, [30])

async def run():
    print("Modbus TCP slave listening on 0.0.0.0:1502 (unit id any)")
    await StartAsyncTcpServer(context=context, address=("0.0.0.0", 1502))

if __name__ == "__main__":
    asyncio.run(run())
```

Run:

```sh
# Install pymodbus in a virtualenv or system python
# python -m venv venv && source venv/bin/activate && pip install pymodbus
python modbus_server.py
```

### Patch desired value via kubectl

```sh
kubectl -n default patch device demo-1 --type='json' -p='[
  {"op":"replace","path":"/spec/properties/1/desired/value","value":"74"}
]'
```

---

## Additional notes

- For quick local tests, ensure the DMI socket path is reachable or unset it in config.yaml.
- The HTTP server (REST) and gRPC server are started automatically