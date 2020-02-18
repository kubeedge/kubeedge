# Install KubeEdge

In this section, we would cover the below topics

1. [Abstract](#Abstract)
2. [Pre-Requisite](#pre-requisite)
3. [Setup KubeEdge](#setup-kubeEdge)
4. [Configure KubeEdge](#configure-kubeEdge)
5. [Run KubeEdge](#run-kubeedge)
6. [KubeEdge Pre-Check](#kubeedge-pre-check)

## Abstract

KubeEdge is composed of cloud and edge sides. It is built upon Kubernetes and provides core infrastructure support for networking, application deployment and metadata synchronization between cloud and edge. So if we want to setup KubeEdge, we need to setup Kubernetes cluster (exisiting cluster can be used), cloud side and edge side.

+ on `cloud side`, we need to install Docker, Kubernetes cluster and cloudcore.
+ on `edge side`, we need to install Docker, MQTT (We can also use internal MQTT broker) and edgecore.

## Pre-Requisite

+ Please refer [compatibility-matrix](https://github.com/kubeedge/kubeedge#compatibility-matrix) to understand **Go dependency** and **Kubernetes compatibility** and determine what version of Docker, Kubernetes, Golang can be installed.

### Cloud side (KubeEdge Master)

+ [Install golang](https://golang.org/dl/) (If you want to compile KubeEdge)

+ [Install Docker](https://docs.docker.com/install/), or other runtime, such as [containerd](https://github.com/containerd/containerd)

+ [Install kubeadm/ kubectl](https://kubernetes.io/docs/setup/independent/install-kubeadm/)

+ [Creating Kubernetes cluster with kubeadm](<https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/>)

If you are creating Kubernetes cluster for just testing KubeEdge, maybe start with Flannel.

Check Kubernetes Master Status: It should be `ready`.

```shell
kubectl get nodes

NAME               STATUS   ROLES    AGE    VERSION
kubeedge-master   Ready    master   4d3h   v1.16.1
```

**Note:** If you choose to decide installing KubeEdge using KubeEdge Installer keadm, do pass the flag

```shell
keadm init --pod-network-cidr
```

### Edge side (KubeEdge Worker Node)

+ [Install golang](https://golang.org/dl/) (If you want to compile KubeEdge)

+ [Install Docker](https://docs.docker.com/install/), or other runtime, such as [containerd](https://github.com/containerd/containerd)

+ [Install mosquitto](https://mosquitto.org/download/) : If you are just trying KubeEdge, this step can be skipped.

**Note:** Do not install **kubelet** and **kube-proxy** on edge side

## Setup KubeEdge

Currently there are four different ways of installing KubeEdge

1. From [Source](kubeedge_install_source.md)
2. From [KubeEdge Installer keadm](kubeedge_install_keadm.md)
3. From Binary
4. From Container

## Configure KubeEdge

At this point, we assume that you have completed the installation of KubeEdge and want to configure either Cloudcore or Edgecore

Refer [KubeEdge Configuration](kubeedge_configure.md)

## Run KubeEdge

Refer [KubeEdge Run](kubeedge_run.md)

## KubeEdge Pre-Check

Refer [KubeEdge Pre-Check](kubeedge_precheck.md)
