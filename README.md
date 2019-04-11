# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)


<img src="./docs/images/KubeEdge_logo.png">

KubeEdge is an open source system extending native containerized application orchestration and device management to hosts at the Edge. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. It also supports **MQTT** and allows developers to author custom logic and enable resource constrained device communication at the Edge. KubeEdge consists of a cloud part and an edge part.

## Advantages

#### Edge Computing

With business logic running at the Edge, much larger volumes of data can be secured & processed locally where the data is produced. Edge nodes can run autonomously which effectively reduces the network bandwidth requirements and consumptions between Edge and Cloud. With data processed at the Edge, the responsiveness is increased dramatically and data privacy is protected.

#### Simplified development

Developers can write regular http or mqtt based applications, containerize them, and run them anywhere - either at the Edge or in the Cloud - whichever is more appropriate.

#### Kubernetes-native support

With KubeEdge, users can orchestrate apps, manage devices and monitor app and device status on Edge nodes just like a traditional Kubernetes cluster in the Cloud. Locations of edge nodes are transparent to customers.

#### Abundant applications

It is easy to get and deploy existing complicated machine learning, image recognition, event processing and other high level applications to the Edge.

## Introduction

KubeEdge is composed of the following components:

- **Edged:** an agent that runs on edge nodes and manages containerized applications.
- **EdgeHub:** a web socket client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge Architecture). This includes syncing cloud-side resource updates to the edge, and reporting edge-side host and device status changes to the cloud.
- **CloudHub:** a web socket server responsible for watching changes at the cloud side, caching and sending messages to EdgeHub.
- **EdgeController:** an extended kubernetes controller which manages edge nodes and pods metadata so that the data can be targeted to a specific edge node.
- **EventBus:** a MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- **ServiceBus:** a HTTP client to interact with HTTP servers (REST), offering HTTP client capabilities to components of cloud to reach HTTP servers running at edge.
- **DeviceTwin:** responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- **MetaManager:** the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite). 

### Architecture

<img src="./docs/images/kubeedge_arch.png">

## Roadmap

### Release 1.0
KubeEdge will provide the fundamental infrastructure and basic functionality for IOT/Edge workloads. This includes: 
- An open source implementation of the cloud and edge parts.
- Kubernetes application deployment through kubectl from Cloud to Edge nodes.
- Kubernetes configmap and secret deployment through kubectl from Cloud to Edge nodes and their applications.
- Bi-directional multiplexed network communication between Cloud and Edge nodes.
- Kubernetes Pod and Node status querying with kubectl at Cloud with data collected/reported from the Edge.
- Edge node autonomy when disconnected, and automatic post-reconnection recovery to the Cloud.
- Device twin and MQTT protocol for communication between IOT devices and Edge nodes.

### Release 2.0 and the Future
- Istio-based service mesh across Edge and Cloud where micro-services can communicate freely in the mesh.
- Enhance performance and reliability of KubeEdge infrastructure.
- Enable function as a service at the Edge.
- Support more types of device protocols to Edge nodes such as AMQP, BlueTooth, ZigBee, etc.
- Evaluate and enable much larger scale Edge clusters with thousands of Edge nodes and millions of devices.
- Enable intelligent scheduling of applications to large scale Edge clusters.

## Usage

### Prerequisites

+ [Install docker](https://docs.docker.com/install/)

+ [Install kubeadm/kubectl](https://docs.docker.com/install/)

+ [Creating cluster with kubeadm](<https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/>)

+ After initializing Kubernetes master, we need to expose insecure port 8080 for edgecontroller/kubectl to work with http connection to Kubernetes apiserver.
  Please follow below steps to enable http port in Kubernetes apiserver.

    ```shell
    vi /etc/kubernetes/manifests/kube-apiserver.yaml
    # Add the following flags in spec: containers: -command section
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
    ```

+ (**Optional**)KubeEdge also supports https connection to Kubernetes apiserver. Follow the steps in [Kubernetes Documentation](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) to create the kubeconfig file.

  Enter the path to kubeconfig file in controller.yaml
  ```yaml
  controller:
    kube:
      ...
      kubeconfig: "path_to_kubeconfig_file" #Enter path to kubeconfig file to enable https connection to k8s apiserver
  ```

#### Configuring MQTT mode

The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
1) internalMqttMode: internal mqtt broker is enabled.
2) bothMqttMode: internal as well as external broker are enabled.
3) externalMqttMode: only external broker is enabled.

Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml#L4) to select the desired mode.

To use KubeEdge in double mqtt or external mode, you need to make sure that [mosquitto])(https://mosquitto.org/) or [emqx edge](https://www.emqx.io/downloads/emq/edge?osType=Linux#download) is installed on the edge node as an MQTT Broker.

#### Generate Certificates

RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

```shell
# Generete Root Key
openssl genrsa -des3 -out rootCA.key 4096
# Generate Root Certificate
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt
# Generate Key
openssl genrsa -out edge.key 2048
# Generate csr, Fill required details after running the command
openssl req -new -key edge.key -out edge.csr
# Generate Certificate
openssl x509 -req -in edge.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out edge.crt -days 500 -sha256 
```

## Run KubeEdge

### Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

### Run Cloud

#### [Run as Kubernetes deployment](./build/cloud/README.md)

#### Run as a binary

+ Build Cloud

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
    make # or `make edgecontroller`
    ```

+ The path to the generated certificates should be updated in `$GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller/conf/controller.yaml`. Please update the correct paths for the following :
    + cloudhub.ca
    + cloudhub.cert
    + cloudhub.key

+ Run cloud
    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
    # run edge controller
    # `conf/` should be in the same directory as the cloned KubeEdge repository
    # verify the configurations before running cloud(edgecontroller)
    ./edgecontroller
    ```

### Run Edge

#### Deploy the Edge node
We have provided a sample node.json to add a node in kubernetes. Please make sure edge-node is added in kubernetes. Run below steps to add edge-node.

+ Modify the `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` file and change `metadata.name` to the name of the edge node
+ Deploy node
    ```shell
    kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json
    ```
+ Transfer the certificate file to the edge node

#### Run edge

##### [Run as container](./build/edge/README.md)

##### Run as a binary
+ Build Edge

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make # or `make edge_core`
    ```

    KubeEdge can also be cross compiled to run on ARM based processors.
    Please follow the instructions given below or click [Cross Compilation](docs/setup/cross-compilation.md) for detailed instructions.

    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make edge_cross_build
    ```

    KubeEdge can also be compiled with a small binary size. Please follow the below steps to build a binary of lesser size:

    ```shell
    apt-get install upx-ucl
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make edge_small_build
    ```

    **Note:** If you are using the smaller version of the binary, it is compressed using upx, therefore the possible side effects of using upx compressed binaries like more RAM usage, 
    lower performance, whole code of program being loaded instead of it being on-demand, not allowing sharing of memory which may cause the code to be loaded to memory 
    more than once etc. are applicable here as well.

+ Modify the `$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml` configuration file
    + Replace `edgehub.websocket.certfile` and `edgehub.websocket.keyfile` with your own certificate path
    + Update the IP address of the master in the `websocket.url` field. 
    + replace `fb4ebb70-2783-42b8-b3ef-63e2fd6d242e`q with edge node name in edge.yaml for the below fields :
        + `websocket:URL`
        + `controller:node-id`
        + `edged:hostname-override`

+ Run edge

    ```shell
    # run mosquitto
    mosquitto -d -p 1883
    # or run emqx edge
    # emqx start
    
    # run edge_core
    # `conf/` should be in the same directory as the cloned KubeEdge repository
    # verify the configurations before running edge(edge_core)
    ./edge_core
    # or
    nohup ./edge_core > edge_core.log 2>&1 &
    ```

### Check status

After the Cloud and Edge parts have started, you can use below command to check the edge node status.

```shell
kubectl get nodes
```

Please make sure the status of edge node you created is **ready**.

If you are using HuaweiCloud IEF, then the edge node you created should be running (check it in the IEF console page).

### Deploy Application

Try out a sample application deployment by following below steps.

```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

**Note:** Currently, for edge node, we must use hostPort in the Pod container spec so that the pod comes up normally, or the pod will be always in ContainerCreating status. The hostPort must be equal to containerPort and can not be 0.

Then you can use below command to check if the application is normally running.

```shell
kubectl get pods
```

### Run Edge Unit Tests

 ```shell
 make edge_test
 ```

 To run unit tests of a package individually.

 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge/edge
 cd <path to package to be tested>
 go test -v
 ```

### Run Edge Integration Tests

```shell
make edge_integration_test
```

### Details and use cases of integration test framework

Please find the [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) to use cases of intergration test framework for KubeEdge.

## Community

**Slack channel:** 

Users can join this channel by clicking the invitation [link](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc).

## Documentation

The detailed documentation for KubeEdge and its modules can be found at [https://docs.kubeedge.io](https://docs.kubeedge.io). 
Some sample applications and demos to illustrate possible use cases of KubeEdge platform can be found at this [examples](https://github.com/kubeedge/examples) repository.

## Support

<!--
We don't have a troubleshooting guide yet.  When we do, uncomment the following and add the link.
If you need support, start with the [troubleshooting guide], and work your way through the process that we've outlined.

--> 
If you have questions, feel free to reach out to us in the following ways:

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)

- [slack](https://kubeedge.slack.com)
