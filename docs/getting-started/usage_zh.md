# 使用

## 先决条件
+ [安装 docker](https://docs.docker.com/install/)
+ [安装 kubeadm/kubectl](https://kubernetes.io/docs/setup/independent/install-kubeadm/)
+ [初始化 Kubernetes](https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/)
+ 在完成 Kubernetes master 的初始化后， 我们需要暴露 Kubernetes apiserver 的 http 端口8080用于与 edgecontroller/kubectl 交互。请按照以下步骤在 Kubernetes apiserver 中启用 http 端口。

    ```shell
    vi /etc/kubernetes/manifests/kube-apiserver.yaml
    # Add the following flags in spec: containers: -command section
    - --insecure-port=8080
    - --insecure-bind-address=0.0.0.0
    ```

### 克隆KubeEdge

```shell
git clone https://github.com/kubeedge/kubeedge.git $GOPATH/src/github.com/kubeedge/kubeedge
cd $GOPATH/src/github.com/kubeedge/kubeedge
```

### 配置MQTT模式
KubeEdge 的边缘部分在 deviceTwin 和设备之间使用 MQTT 进行通信。KubeEdge 支持3个 MQTT 模式：
1) internalMqttMode: 启用内部  mqtt 代理。
2) bothMqttMode: 同时启用内部和外部代理。
3) externalMqttMode: 仅启用外部代理。

可以使用 [edge.yaml](https://github.com/kubeedge/kubeedge/blob/master/edge/conf/edge.yaml#L4) 中的 mode 字段去配置期望的模式。

使用 KubeEdge 的 mqtt 内部或外部模式，您都需要确保在边缘节点上安装 [mosquitto](https://mosquitto.org/) 或 [emqx edge](https://www.emqx.io/downloads/edge) 作为 MQTT Broker。

### 生成证书

KubeEdge 在云和边缘之间基于证书进行身份验证/授权。证书可以使用 openssl 生成。请按照以下步骤生成证书。

```bash
# $GOPATH/src/github.com/kubeedge/kubeedge/build/tools/certgen.sh genCertAndKey edge
```

证书和密钥会分别自动生成在`/etc/kubeedge/ca` 和 `/etc/kubeedge/certs` 
目录下。

## 运行KubeEdge

### 运行Cloud

#### 以二进制文件方式运行

+ 构建 Cloud

  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud
  make # or `make edgecontroller`
  ```

+ 修改 `$GOPATH/src/github.com/kubeedge/kubeedge/cloud/conf/controller.yaml` 配置文件，将 `cloudhub.ca`、`cloudhub.cert`、`cloudhub.key`修改为生成的证书路径

+ 创建 device model 和 device CRDs
    ```shell
    cd $GOPATH/src/github.com/kubeedge/kubeedge/build/crds/devices
    kubectl create -f devices_v1alpha1_devicemodel.yaml
    kubectl create -f devices_v1alpha1_device.yaml
    ```

+ 运行二进制文件
  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/cloud
  # run edge controller
  # `conf/` should be in the same directory as the cloned KubeEdge repository
  # verify the configurations before running cloud(edgecontroller)
  ./edgecontroller
  ```

#### [以 k8s deployment 方式运行](../../build/cloud/README_zh.md)

### 运行Edge

#### 部署 Edge node
我们提供了一个示例 node.json 来在 Kubernetes 中添加一个节点。
请确保在 Kubernetes 中添加了边缘节点 edge-node。运行以下步骤以添加边缘节点 edge-node。

+ 编译 `$GOPATH/src/github.com/kubeedge/kubeedge/build/node.json` 文件，将 `metadata.name` 修改为edge node name
+ 部署node
    ```shell
    kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/node.json
    ```
+ 将证书文件传输到edge node

#### 运行Edge

##### 以二进制文件方式运行

+ 构建 Edge

  ```shell
  cd $GOPATH/src/github.com/kubeedge/kubeedge/edge
  make # or `make edge_core`
  ```

  KubeEdge 可以跨平台编译，运行在基于ARM的处理器上。
  请点击 [Cross Compilation](../setup/cross-compilation.md) 获得相关说明。

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

  请使用具有root权限的用户运行 edge。

##### [以容器方式运行](../../build/edge/README_zh.md)

#### [以 k8s deployment 方式运行](../../build/edge/kubernetes/README_zh.md)

#### 检查状态
在 Cloud 和 Edge 被启动之后, 您能通过如下的命令去检查边缘节点的状态。

```shell
kubectl get nodes
```

请确保您创建的边缘节点状态是 **ready**。

如果您使用华为云 IEF, 那么您创建的边缘节点应该正在运行（可在 IEF 控制台页面中查看）。

## 部署应用

请按照以下步骤部署应用程序示例。

```shell
kubectl apply -f $GOPATH/src/github.com/kubeedge/kubeedge/build/deployment.yaml
```

**提示：** 目前对于边缘端，必须在 Pod 配置中使用 hostPort，不然 Pod 会一直处于 ContainerCreating 状态。 hostPort 必须等于 containerPort 而且不能为 0。

然后可以使用下面的命令检查应用程序是否正常运行。

```shell
kubectl get pods
```

## 运行测试

### 运行Edge单元测试

 ```shell
 make edge_test
 ```

 单独运行包的单元测试。

 ```shell
 export GOARCHAIUS_CONFIG_PATH=$GOPATH/src/github.com/kubeedge/kubeedge/edge
 cd <path to package to be tested>
 go test -v
 ```

### 运行Edge集成测试

```shell
make edge_integration_test
```

### 集成测试框架的详细信息和用例

请单击链接 [link](https://github.com/kubeedge/kubeedge/tree/master/edge/test/integration) 找到 KubeEdge 集成测试框架的详细信息和用例。
