# Guide on testing

- [Test with `TestManager` Module](#test-with-testmanager-module)
    - [TestManager Overview](#Overview-About-testManager-Module)
    - [Compile](#compile)
    - [Configure](#Configure)
    - [Run](#run)
    - [Verify](#verify)

## Overview About testManager Module

* testManager is a utility module to mimic the cloud and push messages for different kinds of actions that could happen from the cloud. Typical device , node and application lifecycle management functions are expected to be performed in the cloud and pushed to the edge node. These functions commonly encompass configurations related to 
    
     - Kubernetes Secrets and Configuration Maps.
     - Application deployment/sync
     - Binding devices to edge nodes via memberships.
     - Syncing of different resources between cloud and edge ( like app status, device status etc)
     - Node sync, etc..
     
Below info can help user how to use the testManager for testing the kubeedge.

* testManager module starts http server on port 12345 to let users interact with kubeedge and perform operations which would be typically performed from cloud.
It exposes its API's to do the following

    - /pods         : Deploying a pod on to kubeedge node.
    - /devices      : Bind a Device to kubeedge node.
    - /secrets      : Configure secrets on kubeedge node.
    - /configmaps   : Configure configmaps on kubeedge node.
    
Using above API's user can perform the resource operations against running edge node.
    
testManager facilitates validating the capabilities of the edge platform by performing **curl** operations against a running edge node. 

Following sections will explain the procedure to test the kubeedge with testManager. 

## Test with `TestManager` Module

### Compile

```shell
# generate the `edge_core` binary 
make
# or
make edge_core
```
### Configure

##### Modify the configuration files accordingly

##### in `modules.yaml` (add the `testManager`)

Kubeedge uses [beehive](https://github.com/kubeedge/kubeedge/blob/master/docs/modules/beehive.md) framework as the inter-module communication, all modules in the kubeedge need to register with beehive.
`testManager` is a module like other KubeEdge modules. So, it has to be configured as shown below.

```yaml
modules:
    enabled: [eventbus, websocket, metaManager, edged, twin, testManager]
```
#### Test kubeedge with Internal MQTT Server

##### `edge.yaml` (`mode: 0 (default mode)` modify `certfile`, `keyfile`, etc.)

```yaml
mqtt:
    server: tcp://127.0.0.1:1883 # external mqtt broker url.
    internal-server: tcp://127.0.0.1:1884 # internal mqtt broker url.
    mode: 0 # 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only.
    qos: 0 # 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce.
    retain: false # if the flag set true, server will store the message and can be delivered to future subscribers.
    session-queue-size: 100 # A size of how many sessions will be handled. default to 100.

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
    docker-address: unix:///var/run/docker.sock
    version: 2.0.0
```

```bash
# run edge_core
./edge_core
# or
nohup ./edge_core > edge_core.log 2>&1 &
```

### Test kubeedge with External MQTT Server

**Install dependency:**

You need install **mosquitto** to support mqtt.

xref: https://mosquitto.org/download/

```shell
# For ubuntu
apt install -y mosquitto
```

##### `edge.yaml` (modify `mode: 2`, `certfile`, `keyfile`, etc.)

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

**Add Device**
```bash
curl -X PUT \
  http://127.0.0.1:12345/devices \
  -H 'content-type: application/json' \
  -d '{
	"added_devices": [{
		"id": "kubeedge-device-1",
		"name": "edgedevice",
		"description": "integrationtest",
		"state": "online"
	}]
}'
``` 

#### Verify the DB 
```bash
# Enter the database
sqlite3 edge.db

# Query the database and you shall see the posted Device info
select * from device;
```

**Remove Device**
```bash
curl -X DELETE \
  http://127.0.0.1:12345/devices \
  -H 'content-type: application/json' \
  -d '{
	"removed_devices": [{
		"id": "kubeedge-device-1",
		"name": "edgedevice",
		"description": "integrationtest",
		"state": "online"
	}]
}'
``` 

#### Add Pod
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
#### Query Pods
```bash
curl -i -v -X GET http://127.0.0.1:12345/pod 

#or (To display response in json format)

curl -i -v -X GET http://127.0.0.1:12345/pod | python -m json.tool
```

#### Check the database

```bash
# Enter the database
sqlite3 edge.db

# Query the database and you shall see the posted application deployment info
select * from meta;

# or you can check the pod container using `docker ps`
```
### Remove Pod 
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