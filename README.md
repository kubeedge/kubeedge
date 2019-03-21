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

To use KubeEdge, you will need to have **docker** installed both of Cloud and Edge parts. If you don't, please follow these steps to install docker.

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

KubeEdge's Cloud(edgecontroller) connects to Kubernetes master to sync updates of node/pod status. If you don't have Kubernetes setup, please follow these steps to install Kubernetes using kubeadm.

#### Install kubeadm/kubectl

For Ubuntu:

```shell
apt-get update && apt-get install -y apt-transport-https curl
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb https://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl
```

For CentOS:

```shell
at <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
exclude=kube*
EOF

# Set SELinux in permissive mode (effectively disabling it)
setenforce 0
sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config

yum install -y kubelet kubeadm kubectl --disableexcludes=kubernetes

systemctl enable --now kubelet
```

#### Install Kubernetes

To initialize Kubernetes master, follow the below step:

```shell
kubeadm init
```

To use Kubernetes command line tool **kubectl**, you need to make the below configuration.

```shell
mkdir -p $HOME/.kube
cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
chown $(id -u):$(id -g) $HOME/.kube/config
```

After initializing Kubernetes master, we need to expose insecure port 8080 for edgecontroller/kubectl to work with http connection to Kubernetes apiserver.
Please follow below steps to enable http port in Kubernetes apiserver.

```shell
vi /etc/kubernetes/manifests/kube-apiserver.yaml
# Add the following flags in spec: containers: -command section
- --insecure-port=8080
- --insecure-bind-address=0.0.0.0
```

The Edge part of KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
1) internalMqttMode: internal mqtt broker is enabled.
2) bothMqttMode: internal as well as external broker are enabled.
3) externalMqttMode: only external broker is enabled.

Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml#L4) to select the desired mode.

To use KubeEdge in double mqtt or external mode, you will need to have **mosquitto** installed on the Edge node. If you do not already have it, you may install as follows.

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

KubeEdge has certificate based authentication/authorization between cloud and edge. Certificates can be generated using openssl. Please follow the steps below to generate certificates.

#### Install openssl

If openssl is not already present using below command to install openssl.

```shell
apt-get install openssl
```

#### Generate Certificates

RootCA certificate and a cert/key pair is required to have a setup for KubeEdge. Same cert/key pair can be used in both cloud and edge.

```shell
# Generete Root Key
openssl genrsa -des3 -out rootCA.key 4096
# Generate Root Certificate
openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt
# Generate Key
openssl genrsa -out kubeedge.key 2048
# Generate csr, Fill required details after running the command
openssl req -new -key kubeedge.key -out kubeedge.csr
# Generate Certificate
openssl x509 -req -in kubeedge.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out kubeedge.crt -days 500 -sha256 
```

### Clone KubeEdge

Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

### Build Cloud

+ Modify the `$GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller/conf/controller.yaml` configuration file to modify `cloudhub.ca`, `cloudhub.cert`, `cloudhub.key` to generated Certificate path

+ Build Cloud
    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
    make # or `make edgecontroller`
    ```

### Build Edge

+ Modify the `$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml` configuration file
  + Replace `edgehub.websocket.certfile` and `edgehub.websocket.keyfile` with your own certification path
  + In websocket:url you need to give IP address of the master.
  + replace `fb4ebb70-2783-42b8-b3ef-63e2fd6d242e` with edge node ip in edge.yaml in 3 place.
    + in websecket:URL
    + in controller:node-id
    + in edged:hostname-override

+ Build edge
    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
    make # or `make edge_core`
    ```

KubeEdge can also be cross compiled to run on ARM based processors.
Please click [Cross Compilation](docs/setup/cross-compilation.md) for the instructions.

## Run KubeEdge

### Run Cloud

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
# run edge controller
# `conf/` should be in the same directory as the cloned KubeEdge repository
# verify the configurations before running cloud(edgecontroller)
./edgecontroller
```

### Run Edge

We have provided a sample node.json to add a node in kubernetes. Please make sure edge-node is added in kubernetes. Run below steps to add edge-node.

+ Compile the `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` file and change `metadata.name` to the IP of the edge node
+ Deploy node
    ```shell
    kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json
    ```

+ Run Edge

    ```shell
    # run mosquitto
    mosquitto -d -p 1883

    # run edge_core
    # `conf/` should be in the same directory as the cloned KubeEdge repository
    # verify the configurations before running edge(edge_core)
    ./edge_core
    # or
    nohup ./edge_core > edge_core.log 2>&1 &
    ```

+ After the Cloud and Edge parts have started, you can use below command to check the edge node status.

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
Some sample applications and demos to illustrate possible use cases of KubeEdge platform can be found at [this](https://github.com/kubeedge/examples) repository.

## Support

<!--
We don't have a troubleshooting guide yet.  When we do, uncomment the following and add the link.
If you need support, start with the [troubleshooting guide], and work your way through the process that we've outlined.
 
--> 
If you have questions, feel free to reach out to us in the following ways:

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)

- [slack](https://kubeedge.slack.com)
