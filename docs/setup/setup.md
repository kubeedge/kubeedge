# Setup KubeEdge

## Prerequisites

To use KubeEdge, you will need to have **docker** installed. If you don't, please follow these steps to install docker.

## Install docker

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
KubeEdge's Cloud(edgecontroller) connects to Kubernetes master to sync updates of node/pod status. If you don't have Kubernetes setup, please follow these steps to install Kubernetes using kubeadm.  

## Install kubeadm/kubectl 

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

## Install Kubernetes

To initialize Kubernetes master, follow the below step:

```shell
kubeadm init
```
After initializing Kubernetes master, we need to expose insecure port 8080 for edgecontroller/kubectl to work with http connection to api-server
Please follow below steps to enable http port in apiserver

```shell 
vi /etc/kubernetes/manifests/kube-apiserver.yaml
# Add the following flags in spec: containers: -command section
- --insecure-port=8080
- --insecure-bind-address=0.0.0.0
```
KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
- `0 - internalMqttMode`: internal mqtt broker is enabled
- `1 - bothMqttMode`: internal as well as external broker are enabled
- `2 - externalMqttMode`: only external broker is enabled

Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml) to select the desired mode

```yaml
mqtt:
    server: tcp://127.0.0.1:1883 # external mqtt broker url.
    internal-server: tcp://127.0.0.1:1884 # internal mqtt broker url.
    mode: 0 # 0: internal mqtt broker enable only. 1: internal and external mqtt broker enable. 2: external mqtt broker enable only.
    qos: 0 # 0: QOSAtMostOnce, 1: QOSAtLeastOnce, 2: QOSExactlyOnce.
    retain: false # if the flag set true, server will store the message and can be delivered to future subscribers.
    session-queue-size: 100 # A size of how many sessions will be handled. default to 100.
```

To use kubeedge in double mqtt or external mode, make sure you have **mosquitto** in your environment. If you do not already have it, you may install as follows.  

## Install mosquitto

For ubuntu:

```shell
apt install mosquitto
```

For centOS:

```shell
yum install mosquitto
```

See [mosquitto official website](https://mosquitto.org/download/) for more information.

## Authentication  
KubeEdge has certificate based authentication/authorization between cloud and edge. Certificates can be generated using openssl. Please follow the steps below to generate certificates.  

### Install openssl

If openssl is not already present using below command to install openssl

```shell
apt-get install openssl
```
### Generate Certificates

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
## Build  
### Clone KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

### Build Cloud

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
make # or `make edgecontroller`
```

### Build Edge

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
make # or `make edgecontroller`
```

KubeEdge can also be cross compiled to run on ARM based processors.
Please click [Cross Compilation](cross-compilation.html) for the instructions.

## Run KubeEdge  

### Run Cloud

```shell
cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
# run edge controller
# `conf/` should be in the same directory as the binary
# verify the configurations before running cloud(edgecontroller)
./edgecontroller
```

### Run Edge

We have provided a sample node.json to add a node in kubernetes. Please make sure edge-node is added in kubernetes. Run below steps to add edge-node
  
```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json
```

```shell
# run mosquitto
mosquitto -d -p 1883

# run edge_core
# `conf/` should be in the same directory as the binary
# verify the configurations before running edge(edge_core)
./edge_core
# or
nohup ./edge_core > edge_core.log 2>&1 &
```

If you are using HuaweiCloud IEF, then the edge node you created should be running (check it in the IEF console page).

## Deploy Application

Try out a sample application deployment by following below steps

```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```
## Run Edge Unit Tests

 ```shell
 make edge_test
 ```
 To run unit tests of a package individually 
 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge/edge
 cd <path to package to be tested>
 go test -v
 
 ``` 
## Run Edge Integration Tests

```shell 
make edge_integration_test
```

### Details and use cases of integration test framework

Please find the [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) to use cases of intergration test framework for kubeedge 