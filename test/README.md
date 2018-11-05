# Guide on testing

- [Test with `TestManager` Module](#test-with-testmanager-module)
    - [Compile](#compile)
    - [Configure](#configure)
    - [Run](#run)
    - [Verify](#verify)

## Test with `TestManager` Module

### Compile

```shell
# generate the `edge_core` binary 
make
# or
make edge_core
```

### Install dependency

You need install **mosquitto** to support mqtt.

xref: https://mosquitto.org/download/

```shell
# For ubuntu
apt install -y mosquitto
```

### Configure

#### Modify the configuration files accordingly

##### `modules.yaml` (add the `testManager`)

```yaml
modules:
    enabled: [eventbus, websocket, metaManager, edged, twin, edgefunction, testManager]
```

##### `edge.yaml` (modify `certfile`, `keyfile`, etc.)

```yaml
mqtt:
    server: tcp://127.0.0.1:1883

edgehub:
    websocket:
        url: ws://127.0.0.1:20000/fake_group_id/events
        certfile: /tmp/edge.crt
        keyfile: /tmp/edge.key
        handshake-timeout: 30 #second
        write-deadline: 15 # second
        read-deadline: 15 # second
    controller:
        heartbeat: 15  # second
        refresh-ak-sk-interval: 10 # minute
        auth-info-files-path: /var/IEF/secret
        placement-url: https://10.154.193.32:7444/v1/placement_external/message_queue
        project-id: e632aba927ea4ac2b575ec1603d56f10
        node-id: fb4ebb70-2783-42b8-b3ef-63e2fd6d242e

edged:
    register-node-namespace: default
    hostname-override: 93e05fa9-b782-4a59-9d02-9f6e639b4205
    interface-name: eth0
    node-status-update-frequency: 10 # second
    device-plugin-enabled: false
    gpu-plugin-enabled: false
    image-gc-high-threshold: 80 # percent
    image-gc-low-threshold: 40 # percent
    maximum-dead-containers-per-container: 1
    version: 2.0.0
```

### Run

```bash
# run mosquitto
mosquitto -d -p 1883
# run edge_core
./edge_core
# or
nohup ./edge_core > edge_core.log 2>&1 &
```

### Verify

#### POST request
```bash
curl -i -v -X POST http://127.0.0.1:12345/pod -d '{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "nginx",
    "labels": {
      "name": "nginx"
    }
  },
  "spec": {
    "containers": [
      {
        "name": "nginx",
        "image": "nginx",
        "imagePullPolicy": "IfNotPresent"
      }
    ]
  }
}'
```
#### Check the database

```bash
# Enter the database
sqlite3 edge.db

# Query the database and you shall see the posted pod info
select * from meta;

# or you can check the pod container using `docker ps`
```
### DELETE request
clean after the testing or the container will remain on the machine

```bash
curl -i -v -X DELETE http://127.0.0.1:12345/pod -d '{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "nginx",
    "labels": {
      "name": "nginx"
    }
  },
  "spec": {
    "containers": [
      {
        "name": "nginx",
        "image": "nginx",
        "imagePullPolicy": "IfNotPresent"
      }
    ]
  }
}'
```