# KubeEdge
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)

![logo](./docs/images/KubeEdge_logo.png)

KubeEdge 是一个开源的系统，可将本机容器化应用编排和管理扩展到边缘端设备。 它基于Kubernetes构建，为网络和应用程序提供核心基础架构支持，并在云端和边缘端部署应用，同步元数据。 KubeEdge 还支持 **MQTT** 协议，允许开发人员编写客户逻辑，并在边缘端启用设备通信的资源约束。

## 优势

### 边缘计算

通过在边缘端运行业务逻辑，可以在本地保护和处理大量数据。KubeEdge 减少了边和云之间的带宽请求，加快响应速度，并保护客户数据隐私。 

### 简化开发

开发人员可以编写常规的基于http或mqtt的应用程序，容器化并在边缘或云端任何地方运行。

### Kubernetes原生支持

使用 KubeEdge 用户可以在边缘节点上编排应用、管理设备并监控应用程序/设备状态，就如同在云端操作 Kubernetes 集群一样。

### 丰富的应用程序

用户可以轻松地将复杂的机器学习、图像识别、事件处理等高层应用程序部署到边缘端。

## 介绍

KubeEdge 由以下组件构成:

- **Edged:** Edged 是运行在边缘节点的代理，用于管理用户应用程序。
- **EdgeHub:** EdgeHub 是一个 Web Socket 客户端，负责与**华为云 IEF服务**交互，包括同步云端资源更新、报告边缘主机和设备状态变化等功能。
- **EventBus:** EventBus 是一个与 MQTT 服务器（mosquitto）交互的 MQTT 客户端，为其他组件提供订阅和发布功能。
- **DeviceTwin:** DeviceTwin 负责存储设备状态并将设备状态同步到云，它还为应用程序提供查询接口。
- **MetaManager:** MetaManager 是消息处理器，位于 Edged 和 Edgehub 之间，它负责向轻量级数据库（SQLite）存储/检索元数据。

### 架构

![架构图](docs/images/kubeedge_arch.png)

## 路线图

### Release 1.0

KubeEdge将为 IoT / Edge 工作负载提供基础架构和基本功能。其中包括：

- 使用 K8s 通过 kubectl 从云端向边缘节点部署应用
- 使用 K8s ConfigMap和Secret 通过 kubectl 从云端对边缘节点和 Pod 中的应用进行配置管理和密钥管理。
- 云和边缘节点之间的双向和多路网络通信
- K8s Pod 和 Node 状态通过云端 kubectl 查询，从边缘端收集/报告数据
- 边缘节点在脱机时自动恢复，并重新连接云端
- 支持IoT设备通过Device twin 和 MQTT 协议与边缘节点通信

### Release 2.0 和未来计划

- 使用 KubeEdge 和 Istio 构建服务网格
- 在边缘端提供函数即服务（Function as a Service，FaaS）
- 在边缘端节点支持更多类型的设备协议，如 AMQP、BlueTooth、ZigBee 等等
- 评估并启用具有数千个边缘节点和数百万设备的超大规模边缘集群
- 启用应用的智能调度，扩大边缘节点的规模
- ……

## 使用

### 先决条件

使用 KubeEdge 确保环境中已经安装了 **mosquitto**（作为 MQTT 代理） 和 **docker** 。如果没有，请参考下面的步骤安装 docker 和 mosquitto。

#### 安装 docker

Ubuntu系统：

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

CentOS系统：

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

#### 安装 mosquitto

Ubuntu系统：

```shell
apt install mosquitto
```

CentOS系统：

```shell
yum install mosquitto
```

参考 [mosquitto official website](https://mosquitto.org/download/) 获得更多的信息。

### 构建

克隆 kube-edge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
make # or `make edge_core`
```

### 集成华为云[Intelligent EdgeFabric (IEF)](https://www.huaweicloud.com/product/ief.html)

**注意；** HuaweiCloud IEF 目前只在中国地区可用。

1. 在[华为云](https://www.huaweicloud.com)创建一个账号
2. 跳转到 [IEF](https://www.huaweicloud.com/product/ief.html) 并创建一个边缘节点
3. 下载节点配置文件（<node_name>.tar.gz）
4. 运行 `bash -x hack/setup_for_IEF.sh /PATH/TO/<node_name>.tar.gz` 修改 `conf/`文件夹下的配置文件

### 运行

```shell
# run mosquitto
mosquitto -d -p 1883

# run edge_core
# `conf/` should be in the same directory as the binary
./edge_core
# or
nohup ./edge_core > edge_core.log 2>&1 &
```

如果您使用华为云 IEF, 那么您创建的边缘节点应该正在运行（可在IEF控制台页面中查看）。

## 社区

**Slack channel:** 

kubeedge.slack.com

用户可以通过单击邀请链接 [link](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc) 加入此频道。

## 文档

通过该链接 [link](https://github.com/kubeedge/kubeedge/tree/master/docs/modules) 可用找到有关 KubeEdge 的各个模块的详细信息。

## 支持

如果您需要支持，请从 [故障排除指南] 开始，然后按照我们概述的流程进行操作。

如果您有任何疑问，请与我们联系。

您可以随时与这些人联系：

- @m1093782566
- @islinwb
- @Lion-Wei