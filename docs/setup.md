# Setup KubeEdge

## Prerequisites

To use KubeEdge, make sure you have **docker** in your environment, if don't have, please reference the following steps to install docker.

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
KubeEdge uses MQTT for communication between deviceTwin and devices. KubeEdge supports 3 MQTT modes:
1) internalMqttMode: internal mqtt broker is enabled
2) bothMqttMode: internal as well as external broker are enabled
3) externalMqttMode: only external broker is enabled

Use mode field in [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/conf/edge.yaml) to select the desired mode

To use kubeedge in double mqtt or external mode, make sure you have **mosquitto** in your environment. Please reference the following steps to install mosquitto if it is not already present in your environment.
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

## Build

Clone kubeedge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
make # or `make edge_core`
```  

## Run

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


## Run Unit Tests

 ```shell
 make test
 ```
 To run unit tests of a package individually 
 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge
 cd <path to package to be tested>
 go test -v
 
 ```
