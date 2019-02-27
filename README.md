# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)


<img src="./docs/images/KubeEdge_logo.png">

KubeEdge is an open source system extending native containerized application orchestration and device management to hosts at the Edge. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. It also supports **MQTT** and allows developers to author custom logic and enable resource constrained device communication at the Edge. Kubeedge consists of a cloud part and an edge part. The edge part has already been open sourced and the cloud part is coming soon!

## Advantages

#### Edge Computing

With business logic running at the Edge, much larger volumes of data can be secured & processed locally where the data is produced. This reduces the network bandwidth requirements and consumption between Edge and Cloud. This increases responsiveness, decreases costs, and protects customers' data privacy. 

#### Simplified development

Developers can write regular http or mqtt based applications, containerize these, and run them anywhere - either at the Edge or in the Cloud - whichever is more appropriate.

#### Kubernetes-native support

With KubeEdge, users can orchestrate apps, manage devices and monitor app and device status on Edge nodes just like a traditional Kubernetes cluster in the Cloud

#### Abundant applications

It is easy to get and deploy existing complicated machine learning, image recognition, event processing and other high level applications to the Edge.

## Introduction

KubeEdge is composed of the following components:

- **Edged:** an agent that runs on edge nodes and manages containerized applications.
- **EdgeHub:** a web socket client responsible for interacting with Cloud Service for the edge computing (like Edge Controller as in the KubeEdge Architecture). This includes syncing cloud-side resource updates to the edge, and reporting edge-side host and device status changes to the cloud.
- **EventBus:** an MQTT client to interact with MQTT servers (mosquitto), offering publish and subscribe capabilities to other components.
- **DeviceTwin:** responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- **MetaManager:** the message processor between edged and edgehub. It is also responsible for storing/retrieving metadata to/from a lightweight database (SQLite). 

### Architecture

<img src="./docs/images/kubeedge_arch.png">

## Roadmap

### Release 1.0
KubeEdge will provide the fundamental infrastructure and basic functionality for IOT/Edge workloads. This includes: 
- An open source implementation of the cloud part.
- Kubernetes application deployment through kubectl from Cloud to Edge nodes.
- Kubernetes configmap and secret deployment through kubectl from Cloud to Edge nodes and their applications.
- Bi-directional multiplexed network communication between Cloud and Edge nodes.
- Kubernetes Pod and Node status querying with kubectl at Cloud with data collected/reported from the Edge.
- Edge node autonomy when disconnected, and automatic post-reconnection recovery to the Cloud.
- Device twin and MQTT protocol for communication between IOT devices and Edge nodes.

### Release 2.0 and the Future
- Istio-based service mesh across Edge and Cloud.
- Enable function as a service at the Edge
- Support more types of device protocols to Edge nodes such as AMQP, BlueTooth, ZigBee, etc.
- Evaluate and enable much larger scale Edge clusters with thousands of Edge nodes and millions of devices.
- Enable intelligent scheduling of applications to large scale Edge clusters.

## Usage

### Prerequisites

To use KubeEdge, you will need to have **docker** installed. If you don't, please follow these steps to install docker.

#### Install docker

For Ubuntu:

```shell
# Install Docker from Ubuntu's repositories:
apt-get update
apt-get install -y docker.io

# or install Docker CE 18.06 from Docker's repositories for Ubuntu or Debian:
apt-get update && apt-get install apt-transport-https ca-certificates curl software-properties-common
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -
add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"
apt-get update && apt-get install docker-ce=18.06.0~ce~3-0~ubuntu
```

For CentOS:

```shell
# Install Docker from CentOS/RHEL repository:
yum install -y docker

# or install Docker CE 18.06 from Docker's CentOS repositories:
yum install yum-utils device-mapper-persistent-data lvm2
yum-config-manager \
    --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo
yum update && yum install docker-ce-18.06.1.ce
```
KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
1) internalMqttMode: internal mqtt broker is enabled
2) bothMqttMode: internal as well as external broker are enabled
3) externalMqttMode: only external broker is enabled

Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml) to select the desired mode

To use kubeedge in double mqtt or external mode, you will need to have **mosquitto** installed. If you do not already have it, you may install as folllows.
#### Install mosquitto

For Ubuntu:

```shell
apt install mosquitto
```

For CentOS:

```shell
yum install mosquitto
```

See [mosquitto official website](https://mosquitto.org/download/) for more information.

### Build Edge

Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
make # or `make edge_core`
```
KubeEdge can also be cross compiled to run on ARM based processors.
Please click [Cross Compilation](docs/setup/cross-compilation.md) for the instructions.

### Run Edge

```shell
# run mosquitto
mosquitto -d -p 1883

# run edge_core
# `conf/` should be in the same directory as the binary
./edge_core
# or
nohup ./edge_core > edge_core.log 2>&1 &
```

If you are using HuaweiCloud IEF, then the edge node you created should be running (check it in the IEF console page).


### Run Edge Unit Tests

 ```shell
 make edge_test
 ```
 To run unit tests of a package individually 
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

Please find the [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) to use cases of intergration test framework for kubeedge 

## Community

**Slack channel:** 

Users can join this channel by clicking the invitation [link](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc).

## Documentation

Please find [link](https://github.com/kubeedge/kubeedge/tree/master/docs/modules) for detailed information about individual modules of KubeEdge. You can also find the [guides](https://github.com/kubeedge/kubeedge/tree/master/docs/guides/try_kubeedge_with_ief.md) for trying kubeedge with IEF.

## Support

<!--
We don't have a troubleshooting guide yet.  When we do, uncomment the following and add the link.
If you need support, start with the [troubleshooting guide], and work your way through the process that we've outlined.
 
--> 
If you have questions, feel free to reach out to us in the following ways:

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)

- [slack](kubeedge.slack.com)
