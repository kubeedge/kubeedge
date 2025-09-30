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
- The HTTP server (REST) and gRPC server are started automatically.
- An example of the driver:
```
#include "driver/driver.h"
#include "log/log.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "common/const.h"
#include <time.h>
#include <math.h>

#define MAX_SIM_CLIENTS 64
static void *sim_clients[MAX_SIM_CLIENTS];
static double sim_baseline[MAX_SIM_CLIENTS];
static double sim_threshold[MAX_SIM_CLIENTS];
static int sim_threshold_offset[MAX_SIM_CLIENTS];

static int sim_index_of(void *client)
{
    if (!client)
        return -1;
    for (int i = 0; i < MAX_SIM_CLIENTS; ++i)
    {
        if (sim_clients[i] == client)
            return i;
    }
    return -1;
}

static void sim_register_client(void *client)
{
    if (!client)
        return;
    for (int i = 0; i < MAX_SIM_CLIENTS; ++i)
    {
        if (sim_clients[i] == NULL)
        {
            sim_clients[i] = client;
            sim_baseline[i] = 25.0;
            sim_threshold[i] = 50.0;
            sim_threshold_offset[i] = -1;
            return;
        }
    }
}
static void sim_unregister_client(void *client)
{
    if (!client)
        return;
    for (int i = 0; i < MAX_SIM_CLIENTS; ++i)
    {
        if (sim_clients[i] == client)
        {
            sim_clients[i] = NULL;
            sim_baseline[i] = 0.0;
            sim_threshold[i] = 0.0;
            sim_threshold_offset[i] = -1;
            return;
        }
    }
}
static double *sim_find_baseline_ptr(void *client)
{
    int idx = sim_index_of(client);
    if (idx < 0)
        return NULL;
    return &sim_baseline[idx];
}
static double *sim_find_threshold_ptr(void *client)
{
    int idx = sim_index_of(client);
    if (idx < 0)
        return NULL;
    return &sim_threshold[idx];
}
static int *sim_find_threshold_offset_ptr(void *client)
{
    int idx = sim_index_of(client);
    if (idx < 0)
        return NULL;
    return &sim_threshold_offset[idx];
}

static int parse_number_field(const char *json, const char *key, double *out_val)
{
    if (!json || !key || !out_val)
        return 0;
    const char *p = json;
    size_t klen = strlen(key);
    while ((p = strstr(p, key)) != NULL)
    {
        const char *q = p - 1;
        if (q >= json && (*q == '\"' || *q == '\'' || *q == ' ' || *q == '{' || *q == ','))
        {
            const char *colon = strchr(p + klen, ':');
            if (!colon)
            {
                p += klen;
                continue;
            }
            const char *v = colon + 1;
            while (*v && (*v == ' ' || *v == '\t' || *v == '\"' || *v == '\''))
            {
                if (*v == '\"' || *v == '\'')
                {
                    v++;
                    break;
                }
                v++;
            }
            char tmp[64] = {0};
            int i = 0;
            while (*v && i < (int)sizeof(tmp) - 1 && (*v == '+' || *v == '-' || (*v >= '0' && *v <= '9') || *v == '.'))
            {
                tmp[i++] = *v++;
            }
            if (i == 0)
                return 0;
            tmp[i] = '\0';
            *out_val = atof(tmp);
            return 1;
        }
        p += klen;
    }
    return 0;
}

CustomizedClient *NewClient(const ProtocolConfig *protocol)
{
    CustomizedClient *c = calloc(1, sizeof(*c));
    if (!c)
    {
        log_error("driver: NewClient calloc failed");
        return NULL;
    }
    if (protocol)
    {
        c->protocolConfig.protocolName = protocol->protocolName ? strdup(protocol->protocolName) : NULL;
        c->protocolConfig.configData = protocol->configData ? strdup(protocol->configData) : NULL;
    }
    pthread_mutex_init(&c->deviceMutex, NULL);
    sim_register_client((void *)c);
    srand((unsigned int)(time(NULL) ^ (uintptr_t)c));

    if (protocol && protocol->configData)
    {
        double v = 0;
        if (parse_number_field(protocol->configData, "threshold", &v))
        {
            double *tp = sim_find_threshold_ptr((void *)c);
            if (tp)
                *tp = v;
        }
        if (parse_number_field(protocol->configData, "threshold_offset", &v))
        {
            int *toff = sim_find_threshold_offset_ptr((void *)c);
            if (toff)
                *toff = (int)v;
        }
    }

    return c;
}

// Destructor for CustomizedClient
void FreeClient(CustomizedClient *client)
{
    if (!client)
        return;
    free(client->protocolConfig.protocolName);
    free(client->protocolConfig.configData);
    pthread_mutex_destroy(&client->deviceMutex);
    sim_unregister_client((void *)client);
    free(client);
}

// Initialize the device
int InitDevice(CustomizedClient *client)
{
    if (!client)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);
    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

// Read data from the device
int GetDeviceData(CustomizedClient *client, const VisitorConfig *visitor, void **out_data)
{
    if (!client || !visitor || !out_data)
        return -1;

    pthread_mutex_lock(&client->deviceMutex);

    int *toff_ptr = sim_find_threshold_offset_ptr((void *)client);
    int proto_toff = toff_ptr ? *toff_ptr : -1;
    double *tp = sim_find_threshold_ptr((void *)client);
    double proto_threshold = tp ? *tp : 50.0;

    int effective_toff = proto_toff;
    double effective_threshold = proto_threshold;
    if (visitor && visitor->configData)
    {
        double vv = 0;
        if (parse_number_field(visitor->configData, "threshold_offset", &vv))
        {
            effective_toff = (int)vv;
        }
        if (parse_number_field(visitor->configData, "threshold", &vv))
        {
            effective_threshold = vv;
        }
    }
    int voffset = visitor->offset;
    if (effective_toff >= 0 && voffset == effective_toff)
    {
        char tbuf[64];
        snprintf(tbuf, sizeof(tbuf), "%.2f", effective_threshold);
        *out_data = strdup(tbuf);
        pthread_mutex_unlock(&client->deviceMutex);
        int rc = 0;
        return rc;
    }

    double baseline = 25.0;
    double *bp = sim_find_baseline_ptr((void *)client);
    if (bp)
        baseline = *bp;
    time_t now = time(NULL);
    double slow = sin((double)now / 60.0) * 0.5;
    double jitter = ((double)(rand() % 100) - 50.0) / 200.0;
    double value = baseline + slow + jitter;
    char buf[64];
    snprintf(buf, sizeof(buf), "%.2f", value);
    *out_data = strdup(buf);

    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

// Write data to the device
int DeviceDataWrite(CustomizedClient *client, const VisitorConfig *visitor, const char *deviceMethodName, const char *propertyName, const void *data)
{
    if (!client || !visitor)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);
    int *toff_ptr = sim_find_threshold_offset_ptr((void *)client);
    int toff = toff_ptr ? *toff_ptr : -1;
    int voffset = visitor->offset;
    if (data)
    {
        char *endptr = NULL;
        double val = strtod((const char *)data, &endptr);
        if (endptr != (const char *)data)
        {
            if (toff >= 0 && voffset == toff)
            {
                double *tp = sim_find_threshold_ptr((void *)client);
                if (tp)
                    *tp = val;
                log_info("driver: DeviceDataWrite set threshold to %.2f (client=%p offset=%d)", val, (void *)client, voffset);
            }
            else
            {
                double *bp = sim_find_baseline_ptr((void *)client);
                if (bp)
                    *bp = val;
                log_info("driver: DeviceDataWrite adjusted baseline to %.2f (client=%p)", val, (void *)client);
            }
        }
        else
        {
            log_info("driver: DeviceDataWrite received non-numeric data, ignored (client=%p)", (void *)client);
        }
    }
    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

// Set data on the device
int SetDeviceData(CustomizedClient *client, const void *data, const VisitorConfig *visitor)
{
    log_info("driver: SetDeviceData called client=%p data=%p visitor=%p", (void *)client, data, (void *)visitor);
    if (!client || !visitor)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);
    int *toff_ptr = sim_find_threshold_offset_ptr((void *)client);
    int toff = toff_ptr ? *toff_ptr : -1;
    int voffset = visitor->offset;
    if (data)
    {
        char *endptr = NULL;
        double val = strtod((const char *)data, &endptr);
        if (endptr != (const char *)data)
        {
            if (toff >= 0 && voffset == toff)
            {
                double *tp = sim_find_threshold_ptr((void *)client);
                if (tp)
                    *tp = val;
                log_info("driver: SetDeviceData set threshold to %.2f (client=%p offset=%d)", val, (void *)client, voffset);
            }
            else
            {
                double *bp = sim_find_baseline_ptr((void *)client);
                if (bp)
                    *bp = val;
                log_info("driver: SetDeviceData adjusted baseline to %.2f (client=%p)", val, (void *)client);
            }
        }
        else
        {
            log_info("driver: SetDeviceData received non-numeric data, ignored (client=%p)", (void *)client);
        }
    }
    pthread_mutex_unlock(&client->deviceMutex);
    int rc = 0;
    log_info("driver: SetDeviceData -> rc=%d", rc);
    return rc;
}

// Stop the device
int StopDevice(CustomizedClient *client)
{
    if (!client)
        return -1;
    pthread_mutex_lock(&client->deviceMutex);
    pthread_mutex_unlock(&client->deviceMutex);
    return 0;
}

// Get the current state of the device
const char *GetDeviceStates(CustomizedClient *client)
{
    if (!client)
        return DEVICE_STATUS_UNKNOWN;
    pthread_mutex_lock(&client->deviceMutex);
    pthread_mutex_unlock(&client->deviceMutex);
    return DEVICE_STATUS_OK;
}
```