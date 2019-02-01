# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)

<img src="./docs/images/KubeEdge_logo.png">

KubeEdge is an open source system extending native containerized application orchestration and device management to hosts at Edge. It is built upon Kubernetes and provides core infrastructure support for network, app. deployment and metadata synchronization between cloud and edge. It also supports **MQTT** and allows developers to author customer logic and enable resource constraint devices communication at Edge. Basically kubeedge consists of cloud part and edge part, the edge part has been already open sourced and cloud part is coming soon!

## Advantages

#### Edge Computing

With business logic running at Edge, volumes of data can be secured & processed locally. It reduces the bandwidth request between Edge and Cloud; increases the response speak; and protects customers' data privacy. 

#### Simplify development

Developers can write regular http or mqtt based applications; containerize and run anywhere at Edge or Cloud.

#### Kubernetes-native support

With KubeEdge, users can orchestrate apps, manage devices and monitor app/device status against Edge nodes like a normal K8s cluster in the Cloud

#### Abundant applications

You can easily get and deploy complicated machine learning, image recognition, event processing and other high level applications to your Edge side.

## Introduction

KubeEdge is composed of these components:

- **Edged:** Edged is an agent running on edge node for managing user's application.
- **EdgeHub:** EdgeHub is a web socket client, which is responsible for interacting with **Huawei Cloud IEF service**, including sync cloud side resources update, report edged side host and device status changes.
- **EventBus:** EventBus is a MQTT client to interact with MQTT server(mosquitto), offer subscribe and publish capability to other components.
- **DeviceTwin:** DeviceTwin is responsible for storing device status and syncing device status to the cloud. It also provides query interfaces for applications.
- **MetaManager:** MetaManager is the message processor and between edged and edgehub. It's also responsible for storing/retrieving metadata to/from a lightweight database(SQLite). 

### Architecture

<img src="./docs/images/kubeedge_arch.png">

## Roadmap

### Release 1.0
KubeEdge will provide the fundamental infrastructure and basic functionalities for IOT/Edge workload. This includes: 
- Open source cloud part implementation.
- K8s Application deployment through kubectl from Cloud to Edge node(s)
- K8s configmap, secret deployment through kubectl from Cloud to Edge node(s) and their applications in Pod
- Bi-directional and multiplex network communication between Cloud and edge nodes
- K8s Pod and Node status querying with kubectl at Cloud with data collected/reported from Edge
- Edge node autonomy when its getting offline and recover post reconnection to Cloud
- Device twin and MQTT protocol for IOT devices talking to Edge node

### Release 2.0 and Future
- Build service mesh with KubeEdge and Istio 
- Enable function as a service at Edge
- Support more types of device protocols to Edge node such as AMQP, BlueTooth, ZigBee, etc.
- Evaluate and enable super large scale of Edge clusters with thousands of Edge nodes and millions of devices
- Enable intelligent scheduling of apps. to large scale of Edge nodes
- etc.

## Usage

### Prerequisites

To use KubeEdge, make sure you have **docker** in your environment, if don't have, please reference the following steps to install docker.

#### Install docker

For ubuntu:

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

For centOS:

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

Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/conf/edge.yaml) to select the desired mode

To use kubeedge in double mqtt or external mode, make sure you have **mosquitto** in your environment. Please reference the following steps to install mosquitto if it is not already present in your environment.
#### Install mosquitto

For ubuntu:

```shell
apt install mosquitto
```

For centOS:

```shell
yum install mosquitto
```

See [mosquitto official website](https://mosquitto.org/download/) for more information.

### Build

Clone kube-edge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
make # or `make edge_core`
```

### Run

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


### Run Unit Tests

 ```shell
 make test
 ```
 To run unit tests of a package individually 
 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge
 cd <path to package to be tested>
 go test -v
 
 ```
## Community

**Slack channel:** 

Users can join this channel by clicking the invitation [link](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc).

## Documentation

Please find [link](https://github.com/kubeedge/kubeedge/tree/master/docs/modules) for detailed information about individual modules of KubeEdge. You can also find the [guides](https://github.com/kubeedge/kubeedge/tree/master/docs/guides/try_kubeedge_with_ief.md) for trying kubeedge with IEF.

## Support

If you need support, start with the [troubleshooting guide], and work your way through the process that we've outlined.

That said, if you have questions, reach out to us, feel free to reach out in the following ways:

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)

- [slack](kubeedge.slack.com)
