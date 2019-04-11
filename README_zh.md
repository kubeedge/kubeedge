# KubeEdge
[![Build Status](https://travis-ci.org/kubeedge/kubeedge.svg?branch=master)](https://travis-ci.org/kubeedge/kubeedge)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubeedge/kubeedge)](https://goreportcard.com/report/github.com/kubeedge/kubeedge)
[![LICENSE](https://img.shields.io/github/license/kubeedge/kubeedge.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/blob/master/LICENSE)
[![Releases](https://img.shields.io/github/release/kubeedge/kubeedge/all.svg?style=flat-square)](https://github.com/kubeedge/kubeedge/releases)
[![Documentation Status](https://readthedocs.org/projects/kubeedge/badge/?version=latest)](https://kubeedge.readthedocs.io/en/latest/?badge=latest)


![logo](./docs/images/KubeEdge_logo.png)

KubeEdge 是一个开源的系统，可将本机容器化应用编排和管理扩展到边缘端设备。 它基于Kubernetes构建，为网络和应用程序提供核心基础架构支持，并在云端和边缘端部署应用，同步元数据。KubeEdge 还支持 **MQTT** 协议，允许开发人员编写客户逻辑，并在边缘端启用设备通信的资源约束。KubeEdge 包含云端和边缘端两部分。

## 优势

### 边缘计算

通过在边缘端运行业务逻辑，可以在本地保护和处理大量数据。KubeEdge 减少了边和云之间的带宽请求，加快响应速度，并保护客户数据隐私。

### 简化开发

开发人员可以编写常规的基于 http 或 mqtt 的应用程序，容器化并在边缘或云端任何地方运行。

### Kubernetes 原生支持

使用 KubeEdge 用户可以在边缘节点上编排应用、管理设备并监控应用程序/设备状态，就如同在云端操作 Kubernetes 集群一样。

### 丰富的应用程序

用户可以轻松地将复杂的机器学习、图像识别、事件处理等高层应用程序部署到边缘端。

## 介绍

KubeEdge 由以下组件构成:

- **Edged:** Edged 是运行在边缘节点的代理，用于管理容器化的应用程序。
- **EdgeHub:** EdgeHub 是一个 Web Socket 客户端，负责与边缘计算的云服务（例如 KubeEdge 架构图中的 Edge Controller）交互，包括同步云端资源更新、报告边缘主机和设备状态变化到云端等功能。
- **CloudHub:** CloudHub 是一个 Web Socket 服务端，负责监听云端的变化, 缓存并发送消息到 EdgeHub。
- **EdgeController:** EdgeController 是一个扩展的 Kubernetes 控制器，管理边缘节点和 Pods 的元数据确保数据能够传递到指定的边缘节点。
- **EventBus:** EventBus 是一个与 MQTT 服务器（mosquitto）交互的 MQTT 客户端，为其他组件提供订阅和发布功能。
- **DeviceTwin:** DeviceTwin 负责存储设备状态并将设备状态同步到云，它还为应用程序提供查询接口。
- **MetaManager:** MetaManager 是消息处理器，位于 Edged 和 Edgehub 之间，它负责向轻量级数据库（SQLite）存储/检索元数据。

### 架构

![架构图](docs/images/kubeedge_arch.png)

## 路线图

### Release 1.0

KubeEdge将为 IoT / Edge 工作负载提供基础架构和基本功能。其中包括：

- 云端和边缘端的开源实现。
- 使用 Kubernetes kubectl 从云端向边缘节点部署应用。
- 使用 Kubernetes kubectl 从云端对边缘节点的应用进行配置管理和密钥管理。
- 云和边缘节点之间的双向和多路网络通信。
- Kubernetes Pod 和 Node 状态通过云端 kubectl 查询，从边缘端收集/报告数据。
- 边缘节点在脱机时自动恢复，并重新连接云端。
- 支持IoT设备通过Device twin 和 MQTT 协议与边缘节点通信。

### Release 2.0 和未来计划

- 使用 KubeEdge 和 Istio 构建服务网格。
- 提高 Kubedge 基础设施的性能和可靠性。
- 在边缘端提供函数即服务（Function as a Service，FaaS）。
- 在边缘端节点支持更多类型的设备协议，如 AMQP、BlueTooth、ZigBee 等等。
- 评估并启用具有数千个边缘节点和数百万设备的超大规模边缘集群。
- 启用应用的智能调度，扩大边缘节点的规模。

## 使用

### 先决条件
+ [安装 docker](https://docs.docker.com/install/)
+ [安装 kubeadm/kubectl](https://docs.docker.com/install/)
+ [初始化 Kubernetes](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/)
+ 在完成 Kubernetes master 的初始化后， 我们需要暴露 Kubernetes apiserver 的 http 端口8080用于与 edgecontroller/kubectl 交互。请按照以下步骤在 Kubernetes apiserver 中启用 http 端口。

    ```shell
    vi /etc/kubernetes/manifests/kube-apiserver.yaml
    # Add the following flags in spec: containers: -command section
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
    ```

#### 配置 MQTT 模式
KubeEdge 的边缘部分在 deviceTwin 和设备之间使用 MQTT 进行通信。KubeEdge 支持3个 MQTT 模式：
1) internalMqttMode: 启用内部  mqtt 代理。
2) bothMqttMode: 同时启用内部和外部代理。
3) externalMqttMode: 仅启用外部代理。

可以使用 [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml#L4) 中的 mode 字段去配置期望的模式。

使用 KubeEdge 的 mqtt 内部或外部模式，您都需要确保在边缘节点上安装 [mosquitto](https://mosquitto.org/) 或 [emqx edge](https://www.emqx.io/downloads/emq/edge?osType=Linux#download) 作为 MQTT Broker。

#### 生成证书

KubeEdge 在云和边缘之间基于证书进行身份验证/授权。证书可以使用 openssl 生成。请按照以下步骤生成证书。

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

## 运行 KubeEdge

### 克隆 KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

### 运行 Cloud

#### [以 k8s deployment 方式运行](./build/cloud/README_zh.md)

#### 以二进制文件方式运行
+ 构建 Cloud

  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
  make # or `make edgecontroller`
  ```

+ 修改 `$GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller/conf/controller.yaml` 配置文件，将 `cloudhub.ca`、`cloudhub.cert`、`cloudhub.key`修改为生成的证书路径

+ 运行二进制文件
  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud/edgecontroller
  # run edge controller
  # `conf/` should be in the same directory as the cloned KubeEdge repository
  # verify the configurations before running cloud(edgecontroller)
  ./edgecontroller
  ```

### 运行 Edge

#### 部署 Edge node
我们提供了一个示例 node.json 来在 Kubernetes 中添加一个节点。
请确保在 Kubernetes 中添加了边缘节点 edge-node。运行以下步骤以添加边缘节点 edge-node。

+ 编译 `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` 文件，将 `metadata.name` 修改为edge node name
+ 部署node
    ```shell
    kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json
    ```
+ 将证书文件传输到edge node

#### 运行 Edge

##### [以容器方式运行](./build/edge/README_zh.md)

##### 以二进制文件方式运行

+ 构建 Edge

  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
  make # or `make edge_core`
  ```

  KubeEdge 可以跨平台编译，运行在基于ARM的处理器上。
  请点击 [Cross Compilation](docs/setup/cross-compilation.md) 获得相关说明。

+ 修改`$GOPATH/src/github.com/kubeedge/kubeedge/edge/conf/edge.yaml`配置文件
  + 将 `edgehub.websocket.certfile` 和 `edgehub.websocket.keyfile` 替换为自己的证书路径
  + 将 `edgehub.websocket.url` 中的 `0.0.0.0` 修改为 master node 的IP
  + 用 edge node name 替换 yaml文件中的 `fb4eb70-2783-42b8-b3f-63e2fd6d242e`

+ 运行二进制文件
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

### 检查状态
在 Cloud 和 Edge 被启动之后, 您能通过如下的命令去检查边缘节点的状态。

```shell
kubectl get nodes
```

请确保您创建的边缘节点状态是 **ready**。

如果您使用华为云 IEF, 那么您创建的边缘节点应该正在运行（可在 IEF 控制台页面中查看）。

### 部署应用

请按照以下步骤部署应用程序示例。

```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

**提示：** 目前对于边缘端，必须在 Pod 配置中使用 hostPort，不然 Pod 会一直处于 ContainerCreating 状态。 hostPort 必须等于 containerPort 而且不能为 0。

然后可以使用下面的命令检查应用程序是否正常运行。

```shell
kubectl get pods
```

### 运行 Edge 单元测试

 ```shell
 make edge_test
 ```

 单独运行包的单元测试。

 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge/edge
 cd <path to package to be tested>
 go test -v
 ```

### 运行 Edge 集成测试

```shell
make edge_integration_test
```

### 集成测试框架的详细信息和用例

请单击链接 [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) 找到 KubeEdge 集成测试框架的详细信息和用例。

## 社区

**Slack channel:** 

用户可以通过单击邀请链接 [link](https://join.slack.com/t/kubeedge/shared_invite/enQtNDg1MjAwMDI0MTgyLTQ1NzliNzYwNWU5MWYxOTdmNDZjZjI2YWE2NDRlYjdiZGYxZGUwYzkzZWI2NGZjZWRkZDVlZDQwZWI0MzM1Yzc) 加入此频道。

## 文档

通过该链接 [https://docs.kubeedge.io](https://docs.kubeedge.io) 可用找到有关 KubeEdge 的各个模块的详细信息。
一些说明 KubeEdge 平台的使用案例的示例应用程序和演示可以在当前仓库 [this](https://github.com/kubeedge/examples) 中找到。

## 支持

<!--
如果您需要支持，请从 [故障排除指南] 开始，然后按照我们概述的流程进行操作。
-->
如果您有任何疑问，请以下方式与我们联系：

- [mailing list](https://groups.google.com/forum/#!forum/kubeedge)

- [slack](https://kubeedge.slack.com)
